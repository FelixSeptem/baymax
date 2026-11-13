package mailbox

import (
	"context"
	"strings"
	"sync"
	"time"
)

type MemoryStore struct {
	mu    sync.Mutex
	state mailboxState
}

func NewMemoryStore(policy Policy) *MemoryStore {
	return &MemoryStore{
		state: newMailboxState("memory", policy),
	}
}

func (s *MemoryStore) Backend() string {
	return "memory"
}

func (s *MemoryStore) MarkFallback(reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.setFallback(reason)
}

func (s *MemoryStore) Publish(_ context.Context, envelope Envelope, now time.Time) (PublishResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.publish(envelope, now)
}

func (s *MemoryStore) Consume(
	_ context.Context,
	consumerID string,
	now time.Time,
	inflightTimeout time.Duration,
	reclaimOnConsume bool,
) (Record, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok, _, err := s.state.consume(consumerID, now, inflightTimeout, reclaimOnConsume)
	return record, ok, err
}

func (s *MemoryStore) Heartbeat(
	_ context.Context,
	messageID, consumerID string,
	now time.Time,
	inflightTimeout time.Duration,
) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.heartbeat(messageID, consumerID, now, inflightTimeout)
}

func (s *MemoryStore) Ack(_ context.Context, messageID, consumerID string, now time.Time) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.ack(messageID, consumerID, now)
}

func (s *MemoryStore) Nack(
	_ context.Context,
	messageID, consumerID, reason string,
	now time.Time,
	opts ActionOptions,
) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.nack(messageID, consumerID, reason, now, opts)
}

func (s *MemoryStore) Requeue(
	_ context.Context,
	messageID, consumerID, reason string,
	now time.Time,
	opts ActionOptions,
) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.requeue(messageID, consumerID, reason, now, opts)
}

func (s *MemoryStore) Stats(_ context.Context) (Stats, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.Stats, nil
}

func (s *MemoryStore) Snapshot(_ context.Context) (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.snapshot(), nil
}

func (s *MemoryStore) Restore(_ context.Context, snapshot Snapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.restore(snapshot)
}

func (s *MemoryStore) continuationSnapshot(ctx context.Context, fn func(Snapshot) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return fn(s.state.snapshot())
}

func (s *MemoryStore) continuationMutate(ctx context.Context, fn func(*mailboxState) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	candidate, err := cloneMailboxState(s.state)
	if err != nil {
		return err
	}
	if err := fn(&candidate); err != nil {
		return err
	}
	s.state = candidate
	return nil
}

func (s *MemoryStore) DrainLifecycleEvents() []LifecycleEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.drainLifecycleEvents()
}

func (s *MemoryStore) SetLifecycleTracing(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.setTraceEvents(enabled)
}

func newStoreWithFallback(backend, path string, policy Policy, opts ...FileStoreOption) (StoreInitResult, error) {
	return newStoreWithFallbackConfig(backend, path, policy, RedisStoreConfig{}, opts...)
}

// newStoreWithFallbackConfig is the backend-aware initializer. The redis
// backend fails fast on invalid wiring (no injected client) instead of
// silently degrading: a cross-instance deployment that lost its coordinator
// must not quietly become single-instance.
func newStoreWithFallbackConfig(backend, path string, policy Policy, redisCfg RedisStoreConfig, opts ...FileStoreOption) (StoreInitResult, error) {
	normalizedBackend := strings.ToLower(strings.TrimSpace(backend))
	switch normalizedBackend {
	case "", "memory":
		return StoreInitResult{
			Store:   NewMemoryStore(policy),
			Backend: "memory",
		}, nil
	case "file":
		store, err := NewFileStore(path, policy, opts...)
		if err != nil {
			fallback := NewMemoryStore(policy)
			fallback.MarkFallback("mailbox.backend.file_init_failed")
			return StoreInitResult{
				Store:          fallback,
				Backend:        "memory",
				Requested:      "file",
				Fallback:       true,
				FallbackReason: "mailbox.backend.file_init_failed",
			}, nil
		}
		return StoreInitResult{
			Store:     store,
			Backend:   "file",
			Requested: "file",
		}, nil
	case "redis":
		store, err := NewRedisStore(redisCfg, policy)
		if err != nil {
			return StoreInitResult{}, err
		}
		return StoreInitResult{
			Store:     store,
			Backend:   "redis",
			Requested: "redis",
		}, nil
	default:
		return StoreInitResult{}, ErrUnsupportedBackend{Backend: backend}
	}
}
