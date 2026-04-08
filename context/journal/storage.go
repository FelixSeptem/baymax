package journal

import (
	"bytes"
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

type FileStorageOptions struct {
	ReuseHandle    bool
	BatchFlushSize int
}

func NewStorage(backend, path string) (Storage, error) {
	return NewStorageWithOptions(backend, path, FileStorageOptions{})
}

func NewStorageWithOptions(backend, path string, opts FileStorageOptions) (Storage, error) {
	switch strings.ToLower(strings.TrimSpace(backend)) {
	case "", "file":
		return NewFileStorageWithOptions(path, opts), nil
	case "db":
		return nil, fmt.Errorf("%w: db backend is not implemented in context-prefix-and-journal-baseline", ErrBackendNotReady)
	default:
		return nil, fmt.Errorf("unsupported context journal backend %q", backend)
	}
}

type FileStorage struct {
	path   string
	opts   FileStorageOptions
	mu     sync.Mutex
	handle *os.File
}

func NewFileStorage(path string) *FileStorage {
	return NewFileStorageWithOptions(path, FileStorageOptions{})
}

func NewFileStorageWithOptions(path string, opts FileStorageOptions) *FileStorage {
	normalized := opts
	if normalized.BatchFlushSize <= 0 {
		normalized.BatchFlushSize = 1
	}
	return &FileStorage{
		path: strings.TrimSpace(path),
		opts: normalized,
	}
}

func (s *FileStorage) Append(ctx context.Context, entry Entry) error {
	return s.AppendBatch(ctx, []Entry{entry})
}

func (s *FileStorage) AppendBatch(ctx context.Context, entries []Entry) error {
	if len(entries) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(s.path) == "" {
		return errors.New("context journal path is required")
	}
	if err := s.ensurePathLocked(); err != nil {
		return err
	}
	size := s.opts.BatchFlushSize
	if size <= 1 || len(entries) <= size {
		return s.appendBatchLocked(entries)
	}
	for i := 0; i < len(entries); i += size {
		end := i + size
		if end > len(entries) {
			end = len(entries)
		}
		if err := s.appendBatchLocked(entries[i:end]); err != nil {
			return err
		}
	}
	return nil
}

func (s *FileStorage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.handle == nil {
		return nil
	}
	if err := s.handle.Close(); err != nil {
		return fmt.Errorf("close context journal: %w", err)
	}
	s.handle = nil
	return nil
}

func (s *FileStorage) ensurePathLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create context journal dir: %w", err)
	}
	return nil
}

func (s *FileStorage) appendBatchLocked(entries []Entry) error {
	var output bytes.Buffer
	for i := range entries {
		row, err := json.Marshal(entries[i])
		if err != nil {
			return fmt.Errorf("marshal context journal entry: %w", err)
		}
		output.Write(row)
		output.WriteByte('\n')
	}
	fd, release, err := s.acquireWriteHandleLocked()
	if err != nil {
		return err
	}
	if _, err := fd.Write(output.Bytes()); err != nil {
		release()
		return fmt.Errorf("append context journal entry: %w", err)
	}
	release()
	return nil
}

func (s *FileStorage) acquireWriteHandleLocked() (*os.File, func(), error) {
	if s.opts.ReuseHandle {
		if s.handle == nil {
			fd, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
			if err != nil {
				return nil, nil, fmt.Errorf("open context journal: %w", err)
			}
			s.handle = fd
		}
		return s.handle, func() {}, nil
	}
	fd, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, nil, fmt.Errorf("open context journal: %w", err)
	}
	return fd, func() { _ = fd.Close() }, nil
}
