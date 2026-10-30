package main

import (
	"bytes"
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"checkbox-detector/internal/checkbox"
	"checkbox-detector/internal/httpapi"
)

type processFunc func(context.Context, []byte) (checkbox.DetectionResult, error)

func (f processFunc) Process(ctx context.Context, data []byte) (checkbox.DetectionResult, error) {
	return f(ctx, data)
}

func TestServerDrainsAnActiveRequest(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	defer unblock()
	processor := processFunc(func(context.Context, []byte) (checkbox.DetectionResult, error) {
		close(started)
		<-release
		return checkbox.DetectionResult{Boxes: []checkbox.Detection{}}, nil
	})
	handler, err := httpapi.NewHandler(processor, httpapi.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: handler}
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	completed := make(chan error, 1)
	go func() { completed <- serve(ctx, server, listener, handler, 3*time.Second) }()
	responseCode := make(chan int, 1)
	go func() {
		client := http.Client{Timeout: 3 * time.Second}
		response, err := client.Post("http://"+listener.Addr().String()+"/detect", "image/png", bytes.NewBufferString("image"))
		if err != nil {
			responseCode <- 0
			return
		}
		defer response.Body.Close()
		responseCode <- response.StatusCode
	}()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("request did not start")
	}
	cancel()
	deadline := time.After(2 * time.Second)
	for {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))
		if response.Code == 503 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("server did not begin draining")
		case <-time.After(time.Millisecond):
		}
	}
	select {
	case err := <-completed:
		t.Fatalf("server exited before active request finished: %v", err)
	default:
	}
	unblock()
	select {
	case code := <-responseCode:
		if code != 200 {
			t.Fatalf("active response was interrupted: %d", code)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("response did not finish")
	}
	select {
	case err := <-completed:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("server did not shut down")
	}
}

func TestServerClosesRequestsWhenDrainDeadlineExpires(t *testing.T) {
	started := make(chan struct{})
	requestCanceled := make(chan struct{})
	processor := processFunc(func(ctx context.Context, _ []byte) (checkbox.DetectionResult, error) {
		close(started)
		<-ctx.Done()
		close(requestCanceled)
		return checkbox.DetectionResult{}, ctx.Err()
	})
	handler, err := httpapi.NewHandler(processor, httpapi.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: handler}
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	completed := make(chan error, 1)
	go func() { completed <- serve(ctx, server, listener, handler, 30*time.Millisecond) }()
	go func() {
		client := http.Client{Timeout: 3 * time.Second}
		response, err := client.Post("http://"+listener.Addr().String()+"/detect", "image/png", bytes.NewBufferString("image"))
		if err == nil {
			response.Body.Close()
		}
	}()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("request did not start")
	}
	cancel()
	select {
	case err := <-completed:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("want drain deadline error, got %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("drain was not bounded")
	}
	select {
	case <-requestCanceled:
	case <-time.After(3 * time.Second):
		t.Fatal("forced close did not cancel active request")
	}
}
