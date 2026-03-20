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

type FileStore struct {
	mu    sync.Mutex
	path  string
	state mailboxState
}

func NewFileStore(path string, policy Policy) (*FileStore, error) {
	cleaned := strings.TrimSpace(path)
	if cleaned == "" {
		return nil, fmt.Errorf("mailbox file backend path is required")
	}
	store := &FileStore{
		path:  cleaned,
		state: newMailboxState("file", policy),
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
	if err := s.persist(); err != nil {
		return PublishResult{}, err
	}
	return result, nil
}

func (s *FileStore) Consume(_ context.Context, consumerID string, now time.Time) (Record, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok, err := s.state.consume(consumerID, now)
	if err != nil || !ok {
		return record, ok, err
	}
	if err := s.persist(); err != nil {
		return Record{}, false, err
	}
	return record, true, nil
}

func (s *FileStore) Ack(_ context.Context, messageID, consumerID string, now time.Time) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, err := s.state.ack(messageID, consumerID, now)
	if err != nil {
		return Record{}, err
	}
	if err := s.persist(); err != nil {
		return Record{}, err
	}
	return record, nil
}

func (s *FileStore) Nack(_ context.Context, messageID, consumerID, reason string, now time.Time) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, err := s.state.nack(messageID, consumerID, reason, now)
	if err != nil {
		return Record{}, err
	}
	if err := s.persist(); err != nil {
		return Record{}, err
	}
	return record, nil
}

func (s *FileStore) Requeue(_ context.Context, messageID, consumerID, reason string, now time.Time) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, err := s.state.requeue(messageID, consumerID, reason, now)
	if err != nil {
		return Record{}, err
	}
	if err := s.persist(); err != nil {
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

func (s *FileStore) Restore(_ context.Context, snapshot Snapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.state.restore(snapshot); err != nil {
		return err
	}
	return s.persist()
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
