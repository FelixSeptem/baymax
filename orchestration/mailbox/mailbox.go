package mailbox

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	DefaultQueryPageSize = 50
	MaxQueryPageSize     = 200
)

type ErrUnsupportedBackend struct {
	Backend string
}

func (e ErrUnsupportedBackend) Error() string {
	return fmt.Sprintf("mailbox backend must be one of [memory,file], got %q", strings.TrimSpace(e.Backend))
}

type StoreInitResult struct {
	Store          Store  `json:"-"`
	Requested      string `json:"requested,omitempty"`
	Backend        string `json:"backend"`
	Fallback       bool   `json:"fallback"`
	FallbackReason string `json:"fallback_reason,omitempty"`
}

func NewStoreWithFallback(backend, path string, policy Policy) (StoreInitResult, error) {
	return newStoreWithFallback(backend, path, policy)
}

type Mailbox struct {
	store     Store
	now       func() time.Time
	observers []LifecycleObserver
}

type Option func(*Mailbox)

func WithClock(now func() time.Time) Option {
	return func(m *Mailbox) {
		if now != nil {
			m.now = now
		}
	}
}

func WithLifecycleObserver(observer LifecycleObserver) Option {
	return func(m *Mailbox) {
		if observer != nil {
			m.observers = append(m.observers, observer)
		}
	}
}

func New(store Store, opts ...Option) (*Mailbox, error) {
	if store == nil {
		return nil, errors.New("mailbox store is required")
	}
	m := &Mailbox{
		store:     store,
		now:       time.Now,
		observers: []LifecycleObserver{},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(m)
		}
	}
	if configurable, ok := m.store.(interface {
		SetLifecycleTracing(enabled bool)
	}); ok {
		configurable.SetLifecycleTracing(len(m.observers) > 0)
	}
	return m, nil
}

func (m *Mailbox) Publish(ctx context.Context, envelope Envelope) (PublishResult, error) {
	return m.store.Publish(ctx, envelope, m.nowTime())
}

func (m *Mailbox) Consume(ctx context.Context, consumerID string) (Record, bool, error) {
	record, ok, err := m.store.Consume(ctx, consumerID, m.nowTime(), 0, false)
	m.emitLifecycle(ctx)
	return record, ok, err
}

func (m *Mailbox) ConsumeWithLease(
	ctx context.Context,
	consumerID string,
	inflightTimeout time.Duration,
	reclaimOnConsume bool,
) (Record, bool, error) {
	record, ok, err := m.store.Consume(ctx, consumerID, m.nowTime(), inflightTimeout, reclaimOnConsume)
	m.emitLifecycle(ctx)
	return record, ok, err
}

func (m *Mailbox) Ack(ctx context.Context, messageID, consumerID string) (Record, error) {
	record, err := m.store.Ack(ctx, messageID, consumerID, m.nowTime())
	m.emitLifecycle(ctx)
	return record, err
}

func (m *Mailbox) Nack(ctx context.Context, messageID, consumerID, reason string) (Record, error) {
	record, err := m.store.Nack(ctx, messageID, consumerID, reason, m.nowTime(), ActionOptions{})
	m.emitLifecycle(ctx)
	return record, err
}

func (m *Mailbox) NackWithOptions(
	ctx context.Context,
	messageID, consumerID, reason string,
	opts ActionOptions,
) (Record, error) {
	record, err := m.store.Nack(ctx, messageID, consumerID, reason, m.nowTime(), opts)
	m.emitLifecycle(ctx)
	return record, err
}

func (m *Mailbox) Requeue(ctx context.Context, messageID, consumerID, reason string) (Record, error) {
	record, err := m.store.Requeue(ctx, messageID, consumerID, reason, m.nowTime(), ActionOptions{})
	m.emitLifecycle(ctx)
	return record, err
}

func (m *Mailbox) RequeueWithOptions(
	ctx context.Context,
	messageID, consumerID, reason string,
	opts ActionOptions,
) (Record, error) {
	record, err := m.store.Requeue(ctx, messageID, consumerID, reason, m.nowTime(), opts)
	m.emitLifecycle(ctx)
	return record, err
}

func (m *Mailbox) Heartbeat(
	ctx context.Context,
	messageID, consumerID string,
	inflightTimeout time.Duration,
) (Record, error) {
	record, err := m.store.Heartbeat(ctx, messageID, consumerID, m.nowTime(), inflightTimeout)
	m.emitLifecycle(ctx)
	return record, err
}

func (m *Mailbox) Stats(ctx context.Context) (Stats, error) {
	return m.store.Stats(ctx)
}

func (m *Mailbox) Snapshot(ctx context.Context) (Snapshot, error) {
	return m.store.Snapshot(ctx)
}

func (m *Mailbox) Restore(ctx context.Context, snapshot Snapshot) error {
	return m.store.Restore(ctx, snapshot)
}

func (m *Mailbox) Query(ctx context.Context, req QueryRequest) (QueryResult, error) {
	snapshot, err := m.store.Snapshot(ctx)
	if err != nil {
		return QueryResult{}, err
	}
	return querySnapshot(snapshot, req)
}

func (m *Mailbox) PublishCommand(ctx context.Context, envelope Envelope) (PublishResult, error) {
	if envelope.Kind != KindCommand {
		return PublishResult{}, fmt.Errorf("publish command requires kind=command")
	}
	return m.Publish(ctx, envelope)
}

func (m *Mailbox) PublishResult(ctx context.Context, envelope Envelope) (PublishResult, error) {
	if envelope.Kind != KindResult {
		return PublishResult{}, fmt.Errorf("publish result requires kind=result")
	}
	return m.Publish(ctx, envelope)
}

func (m *Mailbox) InvokeSync(
	ctx context.Context,
	command Envelope,
	waitTerminal func(context.Context, Envelope) (Envelope, error),
) (PublishResult, PublishResult, error) {
	if waitTerminal == nil {
		return PublishResult{}, PublishResult{}, errors.New("waitTerminal callback is required")
	}
	command.Kind = KindCommand
	commandPublished, err := m.PublishCommand(ctx, command)
	if err != nil {
		return PublishResult{}, PublishResult{}, err
	}
	terminal, err := waitTerminal(ctx, commandPublished.Record.Envelope)
	if err != nil {
		return commandPublished, PublishResult{}, err
	}
	terminal.Kind = KindResult
	if strings.TrimSpace(terminal.CorrelationID) == "" {
		terminal.CorrelationID = commandPublished.Record.Envelope.MessageID
	}
	if strings.TrimSpace(terminal.RunID) == "" {
		terminal.RunID = commandPublished.Record.Envelope.RunID
	}
	if strings.TrimSpace(terminal.TaskID) == "" {
		terminal.TaskID = commandPublished.Record.Envelope.TaskID
	}
	if strings.TrimSpace(terminal.WorkflowID) == "" {
		terminal.WorkflowID = commandPublished.Record.Envelope.WorkflowID
	}
	if strings.TrimSpace(terminal.TeamID) == "" {
		terminal.TeamID = commandPublished.Record.Envelope.TeamID
	}
	resultPublished, err := m.PublishResult(ctx, terminal)
	if err != nil {
		return commandPublished, PublishResult{}, err
	}
	return commandPublished, resultPublished, nil
}

func (m *Mailbox) nowTime() time.Time {
	if m == nil || m.now == nil {
		return time.Now().UTC()
	}
	return m.now().UTC()
}

func (m *Mailbox) emitLifecycle(ctx context.Context) {
	if m == nil {
		return
	}
	drain, ok := m.store.(interface {
		DrainLifecycleEvents() []LifecycleEvent
	})
	if !ok {
		return
	}
	events := drain.DrainLifecycleEvents()
	if len(events) == 0 {
		return
	}
	for _, event := range events {
		ev := event
		if ev.Time.IsZero() {
			ev.Time = m.nowTime()
		}
		for _, observer := range m.observers {
			if observer != nil {
				observer(ctx, ev)
			}
		}
	}
}
