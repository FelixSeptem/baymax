package sourceadapter

import (
	"context"
	"errors"
	"math"
	"sync"
	"sync/atomic"
)

type HealthClass string

const (
	Healthy          HealthClass = "healthy"
	Invalid          HealthClass = "invalid"
	Foreign          HealthClass = "foreign"
	Conflict         HealthClass = "conflict"
	Deferred         HealthClass = "retryable"
	DependencyFailed HealthClass = "dependency_failed"
	Interrupted      HealthClass = "interrupted"
	Closed           HealthClass = "closed"
)

type Health struct {
	Class       HealthClass
	Reference   string
	Confirmed   uint64
	Unconfirmed uint64
}

type Adapter struct {
	metadata Metadata
	source   Source
	consumer Consumer
	store    Store
	options  Options
	ctx      context.Context
	cancel   context.CancelFunc
	busy     atomic.Bool
	mu       sync.Mutex
	closed   bool
	health   Health
}

func New(m Metadata, source Source, consumer Consumer, store Store, options Options) (*Adapter, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	if missing(source) || missing(consumer) || missing(store) || options.OperationTimeout < 0 || options.PageLimit < 0 || options.PageLimit > MaxPageSize {
		return nil, ErrInvalid
	}
	if options.OperationTimeout == 0 {
		options.OperationTimeout = DefaultTimeout
	}
	if options.PageLimit == 0 {
		options.PageLimit = MaxPageSize
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Adapter{metadata: m, source: source, consumer: consumer, store: store, options: options, ctx: ctx, cancel: cancel, health: Health{Class: Healthy}}, nil
}

func (a *Adapter) Dispose() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.closed = true
	a.health.Class = Closed
	a.cancel()
}

func (a *Adapter) Abort() { a.Dispose() }

func (a *Adapter) Health() Health { a.mu.Lock(); defer a.mu.Unlock(); return a.health }

func (a *Adapter) begin(parent context.Context) (context.Context, func(), error) {
	if parent == nil {
		return nil, nil, ErrInvalid
	}
	a.mu.Lock()
	closed := a.closed
	a.mu.Unlock()
	if closed {
		return nil, nil, ErrClosed
	}
	if !a.busy.CompareAndSwap(false, true) {
		return nil, nil, ErrBusy
	}
	ctx, cancel := context.WithTimeout(parent, a.options.OperationTimeout)
	stop := context.AfterFunc(a.ctx, cancel)
	if a.ctx.Err() != nil {
		cancel()
	}
	return ctx, func() { stop(); cancel(); a.busy.Store(false) }, nil
}

func classify(err error) HealthClass {
	switch {
	case err == nil:
		return Healthy
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return Interrupted
	case errors.Is(err, ErrInvalid):
		return Invalid
	case errors.Is(err, ErrForeign):
		return Foreign
	case errors.Is(err, ErrConflict):
		return Conflict
	case errors.Is(err, ErrRetryable):
		return Deferred
	default:
		return DependencyFailed
	}
}

func saturate(n *uint64) {
	if *n != math.MaxUint64 {
		*n++
	}
}

func (a *Adapter) observe(err error, ref string, confirmed bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.closed {
		a.health.Class = classify(err)
	}
	if validRef(ref, MaxReferenceBytes) {
		a.health.Reference = ref
	}
	if confirmed {
		saturate(&a.health.Confirmed)
	} else if err != nil {
		saturate(&a.health.Unconfirmed)
	}
}

func dependency(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err != nil {
		for _, known := range []error{ErrInvalid, ErrForeign, ErrConflict} {
			if errors.Is(err, known) {
				return known
			}
		}
		return ErrDependency
	}
	return nil
}

func (a *Adapter) state(ctx context.Context) (State, error) {
	s, err := a.store.Load(ctx)
	if err = dependency(ctx, err); err != nil {
		return State{}, err
	}
	if err := s.Validate(a.metadata); err != nil {
		return State{}, err
	}
	return s, nil
}

func (a *Adapter) RunOnce(parent context.Context) (err error) {
	ctx, end, err := a.begin(parent)
	if err != nil {
		return err
	}
	defer end()
	defer func() {
		if err != nil {
			a.observe(err, "", false)
		}
	}()
	s, err := a.state(ctx)
	if err != nil {
		return err
	}
	p, err := a.source.Poll(ctx, cloneCursor(s.Confirmed), a.options.PageLimit)
	if err = dependency(ctx, err); err != nil {
		return err
	}
	return a.deliver(ctx, s, p)
}

func (a *Adapter) Offer(parent context.Context, p Page) (err error) {
	ctx, end, err := a.begin(parent)
	if err != nil {
		return err
	}
	defer end()
	defer func() {
		if err != nil {
			a.observe(err, "", false)
		}
	}()
	s, err := a.state(ctx)
	if err != nil {
		return err
	}
	return a.deliver(ctx, s, p)
}

func (a *Adapter) page(s State, p Page) ([]Fact, error) {
	if p.Metadata != a.metadata {
		return nil, ErrForeign
	}
	if p.After != nil {
		if err := p.After.validate(a.metadata); err != nil {
			return nil, err
		}
	}
	if !sameCursor(p.After, s.Confirmed) {
		return nil, ErrConflict
	}
	if len(p.Facts) > a.options.PageLimit {
		return nil, ErrInvalid
	}
	facts := make([]Fact, len(p.Facts))
	seen := make(map[string]Receipt, len(p.Facts))
	for i, f := range p.Facts {
		r, err := receipt(a.metadata, f)
		if err != nil {
			return nil, err
		}
		if old, ok := seen[r.ExternalID]; ok && old != r {
			return nil, ErrConflict
		}
		seen[r.ExternalID] = r
		for _, old := range s.Recent {
			if old.ExternalID == r.ExternalID && old != r {
				return nil, ErrConflict
			}
		}
		if s.Pending != nil && ((i == 0 && *s.Pending != r) || (s.Pending.ExternalID == r.ExternalID && *s.Pending != r)) {
			return nil, ErrConflict
		}
		f.Payload = append([]byte(nil), f.Payload...)
		facts[i] = f
	}
	return facts, nil
}

func (a *Adapter) save(ctx context.Context, previous, next State) (State, error) {
	if err := ctx.Err(); err != nil {
		return State{}, err
	}
	s, err := a.store.CompareAndSwap(ctx, previous.Revision, cloneState(next))
	// An admitted store commit may finish concurrently with Abort. Never infer
	// rollback from cancellation or from an external store's uncertain error.
	if err != nil {
		return State{}, dependency(ctx, err)
	}
	if err := s.Validate(a.metadata); err != nil {
		return State{}, err
	}
	if !equalState(s, next) {
		return State{}, ErrConflict
	}
	return s, nil
}

func equalState(a, b State) bool {
	if a.Schema != b.Schema || a.Metadata != b.Metadata || a.Revision != b.Revision || !sameCursor(a.Confirmed, b.Confirmed) || len(a.Recent) != len(b.Recent) || (a.Pending == nil) != (b.Pending == nil) {
		return false
	}
	if a.Pending != nil && *a.Pending != *b.Pending {
		return false
	}
	for i := range a.Recent {
		if a.Recent[i] != b.Recent[i] {
			return false
		}
	}
	return true
}

func (a *Adapter) deliver(ctx context.Context, s State, p Page) error {
	facts, err := a.page(s, p)
	if err != nil {
		return err
	}
	for _, f := range facts {
		next, err := Prepare(s, f)
		if err != nil {
			return err
		}
		s, err = a.save(ctx, s, next)
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		d, err := a.consumer.Consume(ctx, f)
		if err = dependency(ctx, err); err != nil {
			return err
		}
		next, err = Confirm(s, d)
		if err != nil {
			return err
		}
		s, err = a.save(ctx, s, next)
		if err != nil {
			return err
		}
		a.observe(nil, f.Reference, true)
	}
	return nil
}
