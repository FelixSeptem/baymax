package event

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

type eventHandlerFunc func(context.Context, types.Event)

func (f eventHandlerFunc) OnEvent(ctx context.Context, ev types.Event) {
	if f == nil {
		return
	}
	f(ctx, ev)
}

func TestDispatcherDefaultSequentialCompatibility(t *testing.T) {
	var (
		mu    sync.Mutex
		order []string
	)
	handler1 := eventHandlerFunc(func(_ context.Context, _ types.Event) {
		time.Sleep(40 * time.Millisecond)
		mu.Lock()
		order = append(order, "h1")
		mu.Unlock()
	})
	handler2 := eventHandlerFunc(func(_ context.Context, _ types.Event) {
		mu.Lock()
		order = append(order, "h2")
		mu.Unlock()
	})

	dispatcher := NewDispatcher(handler1, handler2)
	start := time.Now()
	dispatcher.Emit(context.Background(), types.Event{Type: "test.default"})
	elapsed := time.Since(start)
	if elapsed < 35*time.Millisecond {
		t.Fatalf("default dispatcher should keep sequential behavior, elapsed=%v", elapsed)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(order) != 2 {
		t.Fatalf("handler order len=%d, want 2", len(order))
	}
	if order[0] != "h1" || order[1] != "h2" {
		t.Fatalf("handler execution order=%v, want [h1 h2]", order)
	}
}

func TestDispatcherFanoutIsConfigurable(t *testing.T) {
	var running int32
	var maxRunning int32

	buildHandler := func() types.EventHandler {
		return eventHandlerFunc(func(_ context.Context, _ types.Event) {
			current := atomic.AddInt32(&running, 1)
			for {
				peak := atomic.LoadInt32(&maxRunning)
				if current <= peak || atomic.CompareAndSwapInt32(&maxRunning, peak, current) {
					break
				}
			}
			time.Sleep(50 * time.Millisecond)
			atomic.AddInt32(&running, -1)
		})
	}

	dispatcher := NewDispatcherWithOptions(
		DispatcherOptions{Fanout: 2},
		buildHandler(),
		buildHandler(),
		buildHandler(),
		buildHandler(),
	)
	dispatcher.Emit(context.Background(), types.Event{Type: "test.fanout"})

	if got := atomic.LoadInt32(&maxRunning); got != 2 {
		t.Fatalf("max concurrent handlers=%d, want 2", got)
	}
}

func TestDispatcherSlowHandlerTimeoutIsolation(t *testing.T) {
	block := make(chan struct{})
	defer close(block)

	var secondCalled int32
	handler1 := eventHandlerFunc(func(_ context.Context, _ types.Event) {
		<-block
	})
	handler2 := eventHandlerFunc(func(_ context.Context, _ types.Event) {
		atomic.AddInt32(&secondCalled, 1)
	})

	dispatcher := NewDispatcherWithOptions(
		DispatcherOptions{
			Fanout:             1,
			SlowHandlerTimeout: 30 * time.Millisecond,
		},
		handler1,
		handler2,
	)
	start := time.Now()
	dispatcher.Emit(context.Background(), types.Event{Type: "test.timeout-isolation"})
	elapsed := time.Since(start)

	if elapsed > 120*time.Millisecond {
		t.Fatalf("emit should not be blocked by slow handler when timeout isolation is enabled, elapsed=%v", elapsed)
	}
	if atomic.LoadInt32(&secondCalled) != 1 {
		t.Fatalf("second handler should be invoked even when first handler is slow")
	}
}
