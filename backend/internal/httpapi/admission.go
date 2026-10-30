package httpapi

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	errBusy     = errors.New("image processing capacity is full")
	errDraining = errors.New("server is shutting down")
)

// Options controls HTTP resource use, independently of detection thresholds.
// Capacity includes reading an upload, native work, and writing its response.
type Options struct {
	MaxConcurrent  int
	MaxQueued      int
	QueueTimeout   time.Duration
	ProcessTimeout time.Duration
}

func DefaultOptions() Options {
	return Options{MaxConcurrent: 1, MaxQueued: 4, QueueTimeout: 2 * time.Second, ProcessTimeout: time.Minute}
}

func (o Options) validate() error {
	if o.MaxConcurrent < 1 || o.MaxConcurrent > 1024 {
		return fmt.Errorf("max-concurrent must be between 1 and 1024")
	}
	if o.MaxQueued < 0 || o.MaxQueued > 1024 {
		return fmt.Errorf("max-queued must be between 0 and 1024")
	}
	if o.QueueTimeout <= 0 || o.ProcessTimeout <= 0 {
		return fmt.Errorf("queue-timeout and process-timeout must be positive")
	}
	return nil
}

type admission struct {
	active   chan struct{}
	queued   chan struct{}
	draining chan struct{}
	once     sync.Once
	wait     time.Duration
}

func newAdmission(options Options) *admission {
	return &admission{active: make(chan struct{}, options.MaxConcurrent), queued: make(chan struct{}, options.MaxQueued), draining: make(chan struct{}), wait: options.QueueTimeout}
}

func (a *admission) close() { a.once.Do(func() { close(a.draining) }) }

func (a *admission) acquire(ctx context.Context) (func(), error) {
	if err := a.available(ctx); err != nil {
		return nil, err
	}
	select {
	case a.active <- struct{}{}:
		return a.finishAcquire(ctx)
	default:
	}
	// Bound queued requests as well as their waiting time. No upload is read
	// until admission succeeds, so waiting clients cannot fill the Go heap.
	select {
	case a.queued <- struct{}{}:
		defer func() { <-a.queued }()
	default:
		return nil, errBusy
	}
	timer := time.NewTimer(a.wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-a.draining:
		return nil, errDraining
	case <-timer.C:
		return nil, errBusy
	case a.active <- struct{}{}:
		return a.finishAcquire(ctx)
	}
}

func (a *admission) available(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case <-a.draining:
		return errDraining
	default:
		return nil
	}
}

func (a *admission) finishAcquire(ctx context.Context) (func(), error) {
	release := func() { <-a.active }
	if err := a.available(ctx); err != nil {
		release()
		return nil, err
	}
	return release, nil
}
