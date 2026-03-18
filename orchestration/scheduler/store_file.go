package scheduler

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
	state schedulerState
}

func NewFileStore(path string) (*FileStore, error) {
	cleaned := strings.TrimSpace(path)
	if cleaned == "" {
		return nil, fmt.Errorf("scheduler file backend path is required")
	}
	store := &FileStore{
		path:  cleaned,
		state: newSchedulerState("file"),
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *FileStore) Backend() string {
	return "file"
}

func (s *FileStore) Enqueue(_ context.Context, task Task, now time.Time) (TaskRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, err := s.state.enqueue(task, now)
	if err != nil {
		return TaskRecord{}, err
	}
	if err := s.persist(); err != nil {
		return TaskRecord{}, err
	}
	return record, nil
}

func (s *FileStore) Claim(_ context.Context, workerID string, now time.Time, leaseTimeout time.Duration) (ClaimedTask, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	claimed, ok, err := s.state.claim(strings.TrimSpace(workerID), now, leaseTimeout)
	if err != nil || !ok {
		return claimed, ok, err
	}
	if err := s.persist(); err != nil {
		return ClaimedTask{}, false, err
	}
	return claimed, true, nil
}

func (s *FileStore) Heartbeat(_ context.Context, taskID, attemptID, leaseToken string, now time.Time, leaseTimeout time.Duration) (ClaimedTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	claimed, err := s.state.heartbeat(taskID, attemptID, leaseToken, now, leaseTimeout)
	if err != nil {
		return ClaimedTask{}, err
	}
	if err := s.persist(); err != nil {
		return ClaimedTask{}, err
	}
	return claimed, nil
}

func (s *FileStore) ExpireLeases(_ context.Context, now time.Time) ([]ClaimedTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	expired := s.state.expireLeases(now)
	if len(expired) == 0 {
		return nil, nil
	}
	if err := s.persist(); err != nil {
		return nil, err
	}
	return expired, nil
}

func (s *FileStore) Requeue(_ context.Context, taskID, _ string, now time.Time) (TaskRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, err := s.state.requeue(taskID, now)
	if err != nil {
		return TaskRecord{}, err
	}
	if err := s.persist(); err != nil {
		return TaskRecord{}, err
	}
	return record, nil
}

func (s *FileStore) CommitTerminal(_ context.Context, commit TerminalCommit) (CommitResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result, err := s.state.commitTerminal(commit)
	if err != nil {
		return CommitResult{}, err
	}
	if err := s.persist(); err != nil {
		return CommitResult{}, err
	}
	return result, nil
}

func (s *FileStore) Get(_ context.Context, taskID string) (TaskRecord, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.state.get(taskID)
	return record, ok, nil
}

func (s *FileStore) Stats(_ context.Context) (Stats, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.Stats, nil
}

func (s *FileStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("mkdir scheduler backend directory: %w", err)
	}
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read scheduler file backend: %w", err)
	}
	decoded := newSchedulerState("file")
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return fmt.Errorf("decode scheduler file backend: %w", err)
	}
	if decoded.Tasks == nil {
		decoded.Tasks = map[string]*TaskRecord{}
	}
	if decoded.TerminalCommits == nil {
		decoded.TerminalCommits = map[string]TerminalCommit{}
	}
	decoded.Stats.Backend = "file"
	s.state = decoded
	return nil
}

func (s *FileStore) persist() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("mkdir scheduler backend directory: %w", err)
	}
	raw, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode scheduler backend: %w", err)
	}
	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, raw, 0o600); err != nil {
		return fmt.Errorf("write scheduler tmp backend file: %w", err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("commit scheduler backend file: %w", err)
	}
	return nil
}
