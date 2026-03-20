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

func (s *MemoryStore) Consume(_ context.Context, consumerID string, now time.Time) (Record, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.consume(consumerID, now)
}

func (s *MemoryStore) Ack(_ context.Context, messageID, consumerID string, now time.Time) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.ack(messageID, consumerID, now)
}

func (s *MemoryStore) Nack(_ context.Context, messageID, consumerID, reason string, now time.Time) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.nack(messageID, consumerID, reason, now)
}

func (s *MemoryStore) Requeue(_ context.Context, messageID, consumerID, reason string, now time.Time) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.requeue(messageID, consumerID, reason, now)
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

func newStoreWithFallback(backend, path string, policy Policy) (StoreInitResult, error) {
	normalizedBackend := strings.ToLower(strings.TrimSpace(backend))
	switch normalizedBackend {
	case "", "memory":
		return StoreInitResult{
			Store:   NewMemoryStore(policy),
			Backend: "memory",
		}, nil
	case "file":
		store, err := NewFileStore(path, policy)
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
	default:
		return StoreInitResult{}, ErrUnsupportedBackend{Backend: backend}
	}
}
