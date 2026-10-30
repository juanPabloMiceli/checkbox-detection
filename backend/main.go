package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"checkbox-detector/internal/checkbox"
	"checkbox-detector/internal/config"
	"checkbox-detector/internal/httpapi"
	"gocv.io/x/gocv"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "HTTP listen address")
	configPath := flag.String("config", "", "Optional detector JSON configuration")
	options := httpapi.DefaultOptions()
	flag.IntVar(&options.MaxConcurrent, "max-concurrent", options.MaxConcurrent, "Maximum simultaneous uploads and computations")
	flag.IntVar(&options.MaxQueued, "max-queued", options.MaxQueued, "Maximum waiting requests; zero rejects immediately when full")
	flag.DurationVar(&options.QueueTimeout, "queue-timeout", options.QueueTimeout, "Maximum time a request can wait for capacity")
	flag.DurationVar(&options.ProcessTimeout, "process-timeout", options.ProcessTimeout, "Computation deadline, checked between stages")
	drainTimeout := flag.Duration("shutdown-timeout", 2*time.Minute, "Maximum graceful drain time")
	nativeThreads := flag.Int("opencv-threads", 1, "Native OpenCV threads per computation")
	flag.Parse()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, *addr, *configPath, options, *drainTimeout, *nativeThreads); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, addr, configPath string, options httpapi.Options, drainTimeout time.Duration, nativeThreads int) error {
	if drainTimeout <= 0 || nativeThreads < 1 {
		return fmt.Errorf("shutdown-timeout and opencv-threads must be positive")
	}
	cfg, err := config.LoadFile(configPath)
	if err != nil {
		return err
	}
	detector, err := checkbox.New(cfg)
	if err != nil {
		return err
	}
	handler, err := httpapi.NewHandler(detector, options)
	if err != nil {
		return err
	}
	// Set native process-wide parallelism once, before any request is accepted.
	gocv.SetNumThreads(nativeThreads)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      3 * time.Minute,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("Checkbox detector listening at http://%s/detect (active=%d queued=%d)", listener.Addr(), options.MaxConcurrent, options.MaxQueued)
	return serve(ctx, server, listener, handler, drainTimeout)
}

// serve stops admission before draining HTTP requests on cancellation. A fresh
// shutdown context lets active native processing finish until drainTimeout;
// expiration closes the remaining connections and returns an error.
func serve(ctx context.Context, server *http.Server, listener net.Listener, handler *httpapi.Handler, drainTimeout time.Duration) error {
	completed := make(chan error, 1)
	go func() { completed <- server.Serve(listener) }()
	select {
	case err := <-completed:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		log.Print("Shutting down: rejecting queued/new work and draining active requests")
		handler.BeginShutdown()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), drainTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			server.Close()
			return fmt.Errorf("shutdown drain: %w", err)
		}
		err := <-completed
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
