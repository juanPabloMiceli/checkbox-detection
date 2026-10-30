package httpapi

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"checkbox-detector/internal/checkbox"
)

func waitFor(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.NewTimer(3 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()
	for !condition() {
		select {
		case <-deadline.C:
			t.Fatal("condition was not reached")
		case <-tick.C:
		}
	}
}

func rawRequest(ctx context.Context) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/detect", bytes.NewBufferString("encoded image"))
	r.Header.Set("Content-Type", "image/png")
	return r.WithContext(ctx)
}

func asyncRequest(h http.Handler, ctx context.Context) <-chan *httptest.ResponseRecorder {
	completed := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		response := httptest.NewRecorder()
		h.ServeHTTP(response, rawRequest(ctx))
		completed <- response
	}()
	return completed
}

func receiveResponse(t *testing.T, completed <-chan *httptest.ResponseRecorder) *httptest.ResponseRecorder {
	t.Helper()
	select {
	case response := <-completed:
		return response
	case <-time.After(3 * time.Second):
		t.Fatal("request did not complete")
		return nil
	}
}

func blockingProcessor(t *testing.T) (Processor, <-chan struct{}, func()) {
	t.Helper()
	started := make(chan struct{}, 16)
	release := make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	t.Cleanup(unblock)
	processor := processorFunc(func(context.Context, []byte) (checkbox.DetectionResult, error) {
		started <- struct{}{}
		<-release
		return checkbox.DetectionResult{Boxes: []checkbox.Detection{}}, nil
	})
	return processor, started, unblock
}

func TestAdmissionBoundsActiveAndQueuedRequests(t *testing.T) {
	processor, started, unblock := blockingProcessor(t)
	options := DefaultOptions()
	options.MaxQueued = 1
	handler := testHandler(t, processor, options)
	first := asyncRequest(handler, context.Background())
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("processor did not start")
	}
	second := asyncRequest(handler, context.Background())
	waitFor(t, func() bool { return len(handler.admission.queued) == 1 })
	select {
	case <-started:
		t.Fatal("queued request started before capacity was released")
	default:
	}
	third := httptest.NewRecorder()
	body := &observedBody{}
	r := rawRequest(context.Background())
	r.Body = body
	handler.ServeHTTP(third, r)
	if third.Code != 503 || third.Header().Get("Retry-After") != "1" || body.read.Load() {
		t.Fatalf("capacity was not enforced before upload: %d, read=%v", third.Code, body.read.Load())
	}
	unblock()
	if receiveResponse(t, first).Code != 200 || receiveResponse(t, second).Code != 200 {
		t.Fatal("admitted work did not finish")
	}
	if len(handler.admission.active) != 0 || len(handler.admission.queued) != 0 {
		t.Fatal("capacity leaked")
	}
}

type observedBody struct{ read atomic.Bool }

func (b *observedBody) Read([]byte) (int, error) {
	b.read.Store(true)
	return 0, context.Canceled
}
func (*observedBody) Close() error { return nil }

func TestAdmissionCancellationAndQueueTimeoutReleaseCapacity(t *testing.T) {
	processor, started, unblock := blockingProcessor(t)
	options := DefaultOptions()
	options.MaxQueued = 1
	options.QueueTimeout = 100 * time.Millisecond
	handler := testHandler(t, processor, options)
	first := asyncRequest(handler, context.Background())
	waitFor(t, func() bool { return len(started) > 0 })
	<-started
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	waiting := asyncRequest(handler, ctx)
	waitFor(t, func() bool { return len(handler.admission.queued) == 1 })
	cancel()
	if receiveResponse(t, waiting).Code != 408 {
		t.Fatal("canceled waiter did not return 408")
	}
	if len(handler.admission.queued) != 0 {
		t.Fatal("canceled waiter retained queue capacity")
	}
	timedOut := receiveResponse(t, asyncRequest(handler, context.Background()))
	if timedOut.Code != 503 || len(handler.admission.queued) != 0 {
		t.Fatalf("queue deadline failed: %d", timedOut.Code)
	}
	unblock()
	receiveResponse(t, first)
}

func TestShutdownWakesQueuedWorkAndPreservesActiveWork(t *testing.T) {
	processor, started, unblock := blockingProcessor(t)
	handler := testHandler(t, processor)
	first := asyncRequest(handler, context.Background())
	waitFor(t, func() bool { return len(started) > 0 })
	<-started
	waiting := asyncRequest(handler, context.Background())
	waitFor(t, func() bool { return len(handler.admission.queued) == 1 })
	handler.BeginShutdown()
	handler.BeginShutdown()
	if receiveResponse(t, waiting).Code != 503 {
		t.Fatal("shutdown did not wake queued request")
	}
	if receiveResponse(t, asyncRequest(handler, context.Background())).Code != 503 {
		t.Fatal("shutdown admitted new work")
	}
	for route, want := range map[string]int{"/healthz": 200, "/readyz": 503} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, route, nil))
		if response.Code != want {
			t.Fatalf("%s: got %d", route, response.Code)
		}
	}
	select {
	case <-first:
		t.Fatal("shutdown interrupted active work before drain")
	default:
	}
	unblock()
	if receiveResponse(t, first).Code != 200 {
		t.Fatal("active work did not drain")
	}
}

func TestDeadlineDoesNotReleaseAnActiveNativeSlot(t *testing.T) {
	deadlineObserved := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	defer unblock()
	processor := processorFunc(func(ctx context.Context, _ []byte) (checkbox.DetectionResult, error) {
		<-ctx.Done()
		close(deadlineObserved)
		// Simulate a native call which cannot be interrupted by context.
		<-release
		return checkbox.DetectionResult{Boxes: []checkbox.Detection{}}, nil
	})
	options := DefaultOptions()
	options.MaxQueued = 0
	options.ProcessTimeout = 20 * time.Millisecond
	handler := testHandler(t, processor, options)
	first := asyncRequest(handler, context.Background())
	select {
	case <-deadlineObserved:
	case <-time.After(3 * time.Second):
		t.Fatal("processing context had no deadline")
	}
	if len(handler.admission.active) != 1 {
		t.Fatal("deadline released a still-running native slot")
	}
	if receiveResponse(t, asyncRequest(handler, context.Background())).Code != 503 {
		t.Fatal("new work ran alongside canceled native work")
	}
	unblock()
	if receiveResponse(t, first).Code != 408 || len(handler.admission.active) != 0 {
		t.Fatal("late native completion did not release capacity and return timeout")
	}
}

func TestHandlerRejectsInvalidCapacitySettings(t *testing.T) {
	processor, _, _ := blockingProcessor(t)
	for _, change := range []func(*Options){
		func(o *Options) { o.MaxConcurrent = 0 }, func(o *Options) { o.MaxConcurrent = 1025 },
		func(o *Options) { o.MaxQueued = -1 }, func(o *Options) { o.QueueTimeout = 0 }, func(o *Options) { o.ProcessTimeout = 0 },
	} {
		options := DefaultOptions()
		change(&options)
		if _, err := NewHandler(processor, options); err == nil {
			t.Fatal("invalid capacity settings accepted")
		}
	}
	if _, err := NewHandler(nil, DefaultOptions()); err == nil {
		t.Fatal("nil processor accepted")
	}
}
