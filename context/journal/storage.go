package journal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var ErrBackendNotReady = errors.New("context journal backend not ready")

type Entry struct {
	Time          time.Time `json:"time"`
	RunID         string    `json:"run_id,omitempty"`
	SessionID     string    `json:"session_id,omitempty"`
	Phase         string    `json:"phase"`
	PrefixVersion string    `json:"prefix_version,omitempty"`
	PrefixHash    string    `json:"prefix_hash,omitempty"`
	Status        string    `json:"status,omitempty"`
	Violation     string    `json:"violation,omitempty"`
}

type Storage interface {
	Append(ctx context.Context, entry Entry) error
}

func NewStorage(backend, path string) (Storage, error) {
	switch strings.ToLower(strings.TrimSpace(backend)) {
	case "", "file":
		return NewFileStorage(path), nil
	case "db":
		return nil, fmt.Errorf("%w: db backend is not implemented in context-prefix-and-journal-baseline", ErrBackendNotReady)
	default:
		return nil, fmt.Errorf("unsupported context journal backend %q", backend)
	}
}

type FileStorage struct {
	path string
	mu   sync.Mutex
}

func NewFileStorage(path string) *FileStorage {
	return &FileStorage{path: strings.TrimSpace(path)}
}

func (s *FileStorage) Append(_ context.Context, entry Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(s.path) == "" {
		return errors.New("context journal path is required")
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create context journal dir: %w", err)
	}
	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open context journal: %w", err)
	}
	defer func() { _ = f.Close() }()
	row, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal context journal entry: %w", err)
	}
	if _, err := f.Write(append(row, '\n')); err != nil {
		return fmt.Errorf("append context journal entry: %w", err)
	}
	return nil
}
