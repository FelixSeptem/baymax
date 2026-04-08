package mailbox

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type FileStoreOption func(*fileStoreOptions)

type fileStoreOptions struct {
	PersistDebounce  time.Duration
	PersistBatchSize int
}

func WithPersistDebounce(debounce time.Duration) FileStoreOption {
	return func(opts *fileStoreOptions) {
		if opts == nil {
			return
		}
		opts.PersistDebounce = debounce
	}
}

func WithPersistBatchSize(size int) FileStoreOption {
	return func(opts *fileStoreOptions) {
		if opts == nil {
			return
		}
		opts.PersistBatchSize = size
	}
}

func normalizeFileStoreOptions(opts []FileStoreOption) fileStoreOptions {
	normalized := fileStoreOptions{
		PersistDebounce:  0,
		PersistBatchSize: 1,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&normalized)
		}
	}
	if normalized.PersistDebounce < 0 {
		normalized.PersistDebounce = 0
	}
	if normalized.PersistBatchSize <= 0 {
		normalized.PersistBatchSize = 1
	}
	return normalized
}

func (o fileStoreOptions) batchingEnabled() bool {
	return o.PersistDebounce > 0 || o.PersistBatchSize > 1
}

type FileStore struct {
	mu             sync.Mutex
	path           string
	state          mailboxState
	persistOpts    fileStoreOptions
	pendingPersist int
	dirtySince     time.Time
}

func NewFileStore(path string, policy Policy, opts ...FileStoreOption) (*FileStore, error) {
	cleaned := strings.TrimSpace(path)
	if cleaned == "" {
		return nil, fmt.Errorf("mailbox file backend path is required")
	}
	store := &FileStore{
		path:        cleaned,
		state:       newMailboxState("file", policy),
		persistOpts: normalizeFileStoreOptions(opts),
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *FileStore) Backend() string {
	return "file"
}

func (s *FileStore) MarkFallback(reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.setFallback(reason)
}

func (s *FileStore) Publish(_ context.Context, envelope Envelope, now time.Time) (PublishResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result, err := s.state.publish(envelope, now)
	if err != nil {
		return PublishResult{}, err
	}
	if err := s.maybePersistLocked(false); err != nil {
		return PublishResult{}, err
	}
	return result, nil
}

func (s *FileStore) Consume(
	_ context.Context,
	consumerID string,
	now time.Time,
	inflightTimeout time.Duration,
	reclaimOnConsume bool,
) (Record, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok, mutated, err := s.state.consume(consumerID, now, inflightTimeout, reclaimOnConsume)
	if err != nil {
		return record, ok, err
	}
	if ok || mutated {
		if err := s.maybePersistLocked(false); err != nil {
			return Record{}, false, err
		}
	}
	return record, ok, nil
}

func (s *FileStore) Heartbeat(
	_ context.Context,
	messageID, consumerID string,
	now time.Time,
	inflightTimeout time.Duration,
) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, err := s.state.heartbeat(messageID, consumerID, now, inflightTimeout)
	if err != nil {
		return Record{}, err
	}
	if err := s.maybePersistLocked(false); err != nil {
		return Record{}, err
	}
	return record, nil
}

func (s *FileStore) Ack(_ context.Context, messageID, consumerID string, now time.Time) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, err := s.state.ack(messageID, consumerID, now)
	if err != nil {
		return Record{}, err
	}
	if err := s.maybePersistLocked(false); err != nil {
		return Record{}, err
	}
	return record, nil
}

func (s *FileStore) Nack(
	_ context.Context,
	messageID, consumerID, reason string,
	now time.Time,
	opts ActionOptions,
) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, err := s.state.nack(messageID, consumerID, reason, now, opts)
	if err != nil {
		return Record{}, err
	}
	if err := s.maybePersistLocked(false); err != nil {
		return Record{}, err
	}
	return record, nil
}

func (s *FileStore) Requeue(
	_ context.Context,
	messageID, consumerID, reason string,
	now time.Time,
	opts ActionOptions,
) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, err := s.state.requeue(messageID, consumerID, reason, now, opts)
	if err != nil {
		return Record{}, err
	}
	if err := s.maybePersistLocked(false); err != nil {
		return Record{}, err
	}
	return record, nil
}

func (s *FileStore) Stats(_ context.Context) (Stats, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.Stats, nil
}

func (s *FileStore) Snapshot(_ context.Context) (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.snapshot(), nil
}

// Flush forces pending batched mutations to be durably persisted.
// It defines the explicit durability boundary when debounce/group-commit is enabled.
func (s *FileStore) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.flushPersistLocked()
}

func (s *FileStore) Restore(_ context.Context, snapshot Snapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.state.restore(snapshot); err != nil {
		return err
	}
	return s.maybePersistLocked(true)
}

func (s *FileStore) DrainLifecycleEvents() []LifecycleEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.drainLifecycleEvents()
}

func (s *FileStore) SetLifecycleTracing(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.setTraceEvents(enabled)
}

func (s *FileStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("mkdir mailbox backend directory: %w", err)
	}
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read mailbox file backend: %w", err)
	}
	var snapshot Snapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return fmt.Errorf("decode mailbox file backend: %w", err)
	}
	if err := s.state.restore(snapshot); err != nil {
		return fmt.Errorf("decode mailbox file backend: %w", err)
	}
	return nil
}

func (s *FileStore) persist() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("mkdir mailbox backend directory: %w", err)
	}
	raw, err := json.MarshalIndent(s.state.snapshot(), "", "  ")
	if err != nil {
		return fmt.Errorf("encode mailbox backend: %w", err)
	}
	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, raw, 0o600); err != nil {
		return fmt.Errorf("write mailbox tmp backend file: %w", err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("commit mailbox backend file: %w", err)
	}
	return nil
}

func (s *FileStore) maybePersistLocked(force bool) error {
	if !s.persistOpts.batchingEnabled() {
		return s.persist()
	}
	s.pendingPersist++
	if s.dirtySince.IsZero() {
		s.dirtySince = time.Now()
	}
	if force {
		return s.flushPersistLocked()
	}
	if s.pendingPersist >= s.persistOpts.PersistBatchSize {
		return s.flushPersistLocked()
	}
	if s.persistOpts.PersistDebounce > 0 && time.Since(s.dirtySince) >= s.persistOpts.PersistDebounce {
		return s.flushPersistLocked()
	}
	return nil
}

func (s *FileStore) flushPersistLocked() error {
	if s.persistOpts.batchingEnabled() && s.pendingPersist == 0 {
		return nil
	}
	if err := s.persist(); err != nil {
		return err
	}
	s.pendingPersist = 0
	s.dirtySince = time.Time{}
	return nil
}
