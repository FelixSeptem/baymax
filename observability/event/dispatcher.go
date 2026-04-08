package event

import (
	"context"
	"sync"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

type DispatcherOptions struct {
	Fanout             int
	SlowHandlerTimeout time.Duration
}

type Dispatcher struct {
	handlers           []types.EventHandler
	fanout             int
	slowHandlerTimeout time.Duration
}

func NewDispatcher(handlers ...types.EventHandler) *Dispatcher {
	return NewDispatcherWithOptions(DispatcherOptions{}, handlers...)
}

func NewDispatcherWithOptions(options DispatcherOptions, handlers ...types.EventHandler) *Dispatcher {
	filtered := make([]types.EventHandler, 0, len(handlers))
	for i := range handlers {
		if handlers[i] == nil {
			continue
		}
		filtered = append(filtered, handlers[i])
	}
	fanout := options.Fanout
	if fanout <= 0 {
		fanout = 1
	}
	return &Dispatcher{
		handlers:           filtered,
		fanout:             fanout,
		slowHandlerTimeout: options.SlowHandlerTimeout,
	}
}

func (d *Dispatcher) Emit(ctx context.Context, ev types.Event) {
	if d == nil || len(d.handlers) == 0 {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if d.fanout <= 1 || len(d.handlers) == 1 {
		for i := range d.handlers {
			d.emitToHandler(ctx, d.handlers[i], ev)
		}
		return
	}
	semaphore := make(chan struct{}, d.fanout)
	var wait sync.WaitGroup
	for i := range d.handlers {
		handler := d.handlers[i]
		semaphore <- struct{}{}
		wait.Add(1)
		go func() {
			defer wait.Done()
			defer func() { <-semaphore }()
			d.emitToHandler(ctx, handler, ev)
		}()
	}
	wait.Wait()
}

func (d *Dispatcher) emitToHandler(ctx context.Context, handler types.EventHandler, ev types.Event) {
	if handler == nil {
		return
	}
	if d.slowHandlerTimeout <= 0 {
		handler.OnEvent(ctx, ev)
		return
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		handler.OnEvent(ctx, ev)
	}()
	timer := time.NewTimer(d.slowHandlerTimeout)
	defer timer.Stop()
	select {
	case <-done:
	case <-timer.C:
	case <-ctx.Done():
	}
}
