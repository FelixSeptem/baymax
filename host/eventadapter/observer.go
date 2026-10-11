// Package eventadapter observes one serial source callback stream. It never
// controls execution. Independent concurrent sources need independent observers.
package eventadapter

import (
	"context"
	"errors"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

const (
	StateHealthy           = "healthy"
	StateFailed            = "failed"
	StateClosed            = "closed"
	ReasonSink             = "observation.sink_error"
	ReasonTimeout          = "observation.timeout"
	ReasonConcurrent       = "observation.concurrent"
	ReasonClosed           = "observation.closed"
	ReasonContext          = "observation.context_canceled"
	DefaultDeliveryTimeout = time.Second
)

// Sink must cooperate with ctx and must not reenter the same Observer. An error
// can follow a successful application write: it means unconfirmed delivery.
// Events are detached snapshots owned by the sink; no automatic retry occurs.
type Sink interface {
	Consume(context.Context, types.NativeExecutionEvent) error
}
type SinkFunc func(context.Context, types.NativeExecutionEvent) error

func (f SinkFunc) Consume(ctx context.Context, event types.NativeExecutionEvent) error {
	return f(ctx, event)
}

type Options struct {
	Projection types.NativeExecutionOptions
	Sink       Sink
	// Context governs observation independently of the business Run context.
	// Nil uses Background. Close cancels cooperative in-flight delivery.
	Context context.Context
	// Zero selects one second. Negative durations are rejected.
	DeliveryTimeout time.Duration
}

// Status is fixed-size except for a reference bounded to 256 bytes. It contains
// no payload or raw sink errors. Counters saturate. Incomplete means failure or
// no acknowledged source terminal; it is not an execution outcome or commit gate.
type Status struct {
	State               string `json:"state"`
	Reason              string `json:"reason,omitempty"`
	FirstEventReference string `json:"first_event_reference,omitempty"`
	Received            uint64 `json:"received"`
	Delivered           uint64 `json:"delivered"`
	Unconfirmed         uint64 `json:"unconfirmed"`
	Skipped             uint64 `json:"skipped"`
	TerminalDelivered   uint64 `json:"terminal_delivered"`
	Incomplete          bool   `json:"incomplete"`
}

type Observer struct {
	projector *types.NativeExecutionProjector
	sink      Sink
	ctx       context.Context
	cancel    context.CancelFunc
	timeout   time.Duration
	active    atomic.Bool
	mu        sync.Mutex
	status    Status
}

var _ types.EventHandler = (*Observer)(nil)

func New(opts Options) (*Observer, error) {
	if opts.Sink == nil || opts.DeliveryTimeout < 0 {
		return nil, errors.New("observation.invalid_options")
	}
	if f, ok := opts.Sink.(SinkFunc); ok && f == nil {
		return nil, errors.New("observation.invalid_options")
	}
	if opts.Projection.Source == types.ProtocolSourceRealtime {
		return nil, errors.New(types.NativeExecutionReasonUnsupported)
	}
	projector, err := types.NewNativeExecutionProjector(opts.Projection)
	if err != nil {
		return nil, err
	}
	ctx := opts.Context
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)
	timeout := opts.DeliveryTimeout
	if timeout == 0 {
		timeout = DefaultDeliveryTimeout
	}
	return &Observer{projector: projector, sink: opts.Sink, ctx: ctx, cancel: cancel, timeout: timeout, status: Status{State: StateHealthy}}, nil
}

func increment(value *uint64) {
	if *value < math.MaxUint64 {
		*value++
	}
}

// OnEvent ignores the Run context for delivery. The source must not mutate its
// payload concurrently during this synchronous call. Arbitrary uncooperative
// Go sinks cannot be forcibly interrupted; no detached goroutine masks this.
func (o *Observer) OnEvent(_ context.Context, event types.Event) {
	o.mu.Lock()
	increment(&o.status.Received)
	if o.status.State != StateHealthy {
		increment(&o.status.Skipped)
		if o.status.State == StateClosed {
			o.failLocked(ReasonClosed, event)
		}
		o.mu.Unlock()
		return
	}
	if !o.active.CompareAndSwap(false, true) {
		increment(&o.status.Skipped)
		o.failLocked(ReasonConcurrent, event)
		o.mu.Unlock()
		return
	}
	o.mu.Unlock()
	defer o.active.Store(false)
	projected, err := o.projector.ProjectEvent(event)
	if err != nil {
		o.mu.Lock()
		o.failLocked(types.NativeExecutionReasonOf(err), event)
		o.mu.Unlock()
		return
	}
	ctx, cancel := context.WithTimeout(o.ctx, o.timeout)
	defer cancel()
	o.mu.Lock()
	if o.status.State != StateHealthy || ctx.Err() != nil {
		increment(&o.status.Skipped)
		if o.status.State == StateHealthy {
			o.failLocked(ReasonContext, event)
		}
		o.mu.Unlock()
		return
	}
	// Admission happens under the short lock; the sink itself never holds it.
	o.mu.Unlock()
	err = o.sink.Consume(ctx, projected)
	ctxErr := ctx.Err()
	o.mu.Lock()
	defer o.mu.Unlock()
	if err != nil || ctxErr != nil || o.status.State != StateHealthy {
		increment(&o.status.Unconfirmed)
		reason := ReasonSink
		if errors.Is(ctxErr, context.DeadlineExceeded) {
			reason = ReasonTimeout
		} else if errors.Is(ctxErr, context.Canceled) {
			reason = ReasonContext
		}
		o.failLocked(reason, event)
		return
	}
	increment(&o.status.Delivered)
	if projected.IsTerminal() {
		increment(&o.status.TerminalDelivered)
	}
}

func (o *Observer) failLocked(reason string, event types.Event) {
	if o.status.Reason != "" {
		return
	}
	o.status.Reason = reason
	ref := strings.ToValidUTF8(event.TraceID, "")
	if ref == "" {
		ref = strings.ToValidUTF8(event.CallID, "")
	}
	if len(ref) > 256 {
		ref = strings.ToValidUTF8(ref[:256], "")
	}
	o.status.FirstEventReference = ref
	if o.status.State != StateClosed {
		o.status.State = StateFailed
	}
	o.cancel()
}

func (o *Observer) Status() Status {
	o.mu.Lock()
	defer o.mu.Unlock()
	result := o.status
	result.Incomplete = result.Reason != "" || result.TerminalDelivered == 0
	return result
}

// Close stops only observation and is idempotent. A delivery already admitted
// before Close may have written; its receipt remains unconfirmed.
func (o *Observer) Close() {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.status.State == StateHealthy {
		o.status.State = StateClosed
	}
	o.cancel()
}
