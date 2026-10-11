package eventadapter

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

func fact() types.Event {
	return types.Event{Type: "run.finished", RunID: "run", TraceID: "shared", Time: time.Unix(100, 0), Payload: map[string]any{"state": "completed"}}
}
func options(sink Sink) Options {
	return Options{Projection: types.NativeExecutionOptions{Source: types.ProtocolSourceRunner, ExpectedRunID: "run"}, Sink: sink}
}
func TestObserverSerialDeliveryAndSnapshot(t *testing.T) {
	var got []types.NativeExecutionEvent
	o, err := New(options(SinkFunc(func(ctx context.Context, event types.NativeExecutionEvent) error {
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > time.Second || time.Until(deadline) < 500*time.Millisecond {
			t.Fatal("default deadline")
		}
		got = append(got, event)
		return nil
	})))
	if err != nil {
		t.Fatal(err)
	}
	ev := fact()
	o.OnEvent(context.Background(), ev)
	ev.Type = "tool.finished"
	o.OnEvent(context.Background(), ev)
	o.OnEvent(context.Background(), ev)
	ev.Payload["state"] = "failed"
	if len(got) != 3 || got[0].Event.EventID != got[1].Event.EventID || got[0].Event.Payload["state"] != "completed" {
		t.Fatal("identity loss or alias")
	}
	s := o.Status()
	if s.State != StateHealthy || s.Received != 3 || s.Delivered != 3 || s.TerminalDelivered != 1 || s.Incomplete {
		t.Fatalf("status %#v", s)
	}
}
func TestObserverAdmission(t *testing.T) {
	for _, opts := range []Options{options(nil), {Projection: types.NativeExecutionOptions{Source: types.ProtocolSourceRealtime}, Sink: SinkFunc(func(context.Context, types.NativeExecutionEvent) error { return nil })}, {Projection: types.NativeExecutionOptions{Source: types.ProtocolSourceRunner}, Sink: SinkFunc(func(context.Context, types.NativeExecutionEvent) error { return nil }), DeliveryTimeout: -1}} {
		if _, err := New(opts); err == nil {
			t.Fatal("invalid observer admitted")
		}
	}
}

func TestObserverContextOverrideAndBoundedFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	opts := options(SinkFunc(func(delivery context.Context, _ types.NativeExecutionEvent) error {
		calls++
		deadline, ok := delivery.Deadline()
		if !ok || time.Until(deadline) > 3*time.Second || time.Until(deadline) < time.Second {
			t.Fatal("positive deadline override not applied")
		}
		return nil
	}))
	opts.Context, opts.DeliveryTimeout = ctx, 3*time.Second
	o, err := New(opts)
	if err != nil {
		t.Fatal(err)
	}
	defer o.Close()
	o.OnEvent(context.Background(), fact())
	cancel()
	ev := fact()
	ev.TraceID = strings.Repeat("界", 200)
	o.OnEvent(context.Background(), ev)
	ev.RunID = "conflicting"
	o.OnEvent(context.Background(), ev)
	s := o.Status()
	if calls != 1 || s.Reason != ReasonContext || len(s.FirstEventReference) > 256 || s.Skipped != 2 || s.Delivered != 1 || !s.Incomplete {
		t.Fatalf("bounded first failure: %#v, calls=%d", s, calls)
	}
	// Lifetime observation counters saturate instead of wrapping to zero.
	o.mu.Lock()
	o.status.Received, o.status.Skipped = math.MaxUint64, math.MaxUint64
	o.mu.Unlock()
	o.OnEvent(context.Background(), fact())
	if s := o.Status(); s.Received != math.MaxUint64 || s.Skipped != math.MaxUint64 {
		t.Fatal("counter overflow")
	}
}
func TestObserverFailureAndIndependentContext(t *testing.T) {
	for _, reason := range []string{ReasonSink, ReasonTimeout, types.NativeExecutionReasonCorrelation} {
		t.Run(reason, func(t *testing.T) {
			calls := 0
			opts := options(SinkFunc(func(ctx context.Context, _ types.NativeExecutionEvent) error {
				calls++
				if reason == ReasonTimeout {
					<-ctx.Done()
					return ctx.Err()
				}
				return errors.New("private body")
			}))
			opts.DeliveryTimeout = 5 * time.Millisecond
			o, err := New(opts)
			if err != nil {
				t.Fatal(err)
			}
			ev := fact()
			if reason == types.NativeExecutionReasonCorrelation {
				ev.RunID = "other"
			}
			o.OnEvent(context.Background(), ev)
			o.OnEvent(context.Background(), fact())
			s := o.Status()
			if s.Reason != reason || s.State != StateFailed || s.Delivered != 0 || s.Received != 2 || !s.Incomplete || s.Skipped != 1 {
				t.Fatalf("status %#v", s)
			}
			if reason == types.NativeExecutionReasonCorrelation {
				if calls != 0 || s.Unconfirmed != 0 {
					t.Fatal("projection delivered")
				}
			} else if calls != 1 || s.Unconfirmed != 1 {
				t.Fatal("write ambiguity lost or retried")
			}
		})
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	o, _ := New(options(SinkFunc(func(ctx context.Context, _ types.NativeExecutionEvent) error { calls++; return ctx.Err() })))
	ev := fact()
	ev.Payload["state"] = "canceled"
	o.OnEvent(canceled, ev)
	if calls != 1 || o.Status().Delivered != 1 {
		t.Fatal("run cancellation stopped observation")
	}
}
func TestObserverConcurrentAndReentrantFailFast(t *testing.T) {
	var o *Observer
	o, _ = New(options(SinkFunc(func(_ context.Context, _ types.NativeExecutionEvent) error {
		o.OnEvent(context.Background(), fact())
		return nil
	})))
	o.OnEvent(context.Background(), fact())
	if s := o.Status(); s.Reason != ReasonConcurrent || s.Unconfirmed != 1 || s.Delivered != 0 || s.Received != 2 {
		t.Fatalf("reentrant %#v", s)
	}
	entered := make(chan struct{})
	done := make(chan struct{})
	o, _ = New(options(SinkFunc(func(ctx context.Context, _ types.NativeExecutionEvent) error {
		close(entered)
		<-ctx.Done()
		return ctx.Err()
	})))
	go func() { o.OnEvent(context.Background(), fact()); close(done) }()
	<-entered
	o.OnEvent(context.Background(), fact())
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("in-flight delivery not canceled")
	}
	if s := o.Status(); s.Reason != ReasonConcurrent || s.Received != 2 || s.Unconfirmed != 1 {
		t.Fatalf("concurrent %#v", s)
	}
}
func TestObserverCloseAndMissingTerminal(t *testing.T) {
	calls := 0
	o, _ := New(options(SinkFunc(func(context.Context, types.NativeExecutionEvent) error { calls++; return nil })))
	ev := fact()
	ev.Payload["state"] = "input_required"
	o.OnEvent(context.Background(), ev)
	if !o.Status().Incomplete || o.Status().TerminalDelivered != 0 {
		t.Fatal("pause implies complete")
	}
	o.Close()
	o.Close()
	o.OnEvent(context.Background(), fact())
	if s := o.Status(); calls != 1 || s.State != StateClosed || s.Reason != ReasonClosed || s.Skipped != 1 || !s.Incomplete {
		t.Fatalf("close %#v", s)
	}
}
