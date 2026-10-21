package workflow

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

// CheckpointStore retains the legacy same-process workflow resume state.
// Source-owned cross-process continuations are defined in continuation.go.
type CheckpointStore interface {
	Load(ctx context.Context, workflowID string) (Checkpoint, bool, error)
	Save(ctx context.Context, cp Checkpoint) error
}

type CheckpointStep struct {
	Status   StepStatus `json:"status"`
	Attempts int        `json:"attempts"`
}

type Checkpoint struct {
	WorkflowID  string                    `json:"workflow_id"`
	RunID       string                    `json:"run_id,omitempty"`
	Status      string                    `json:"workflow_status,omitempty"`
	ResumeCount int                       `json:"workflow_resume_count,omitempty"`
	Steps       map[string]CheckpointStep `json:"steps,omitempty"`
	UpdatedAt   time.Time                 `json:"updated_at"`
}

type MemoryCheckpointStore struct {
	mu   sync.Mutex
	data map[string]Checkpoint
}

func NewMemoryCheckpointStore() *MemoryCheckpointStore {
	return &MemoryCheckpointStore{data: map[string]Checkpoint{}}
}

func (s *MemoryCheckpointStore) Load(_ context.Context, workflowID string) (Checkpoint, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp, ok := s.data[strings.TrimSpace(workflowID)]
	if !ok {
		return Checkpoint{}, false, nil
	}
	return cp, true, nil
}

func (s *MemoryCheckpointStore) Save(_ context.Context, cp Checkpoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := strings.TrimSpace(cp.WorkflowID)
	if key == "" {
		return errors.New("workflow checkpoint requires workflow_id")
	}
	s.data[key] = cp
	return nil
}

type FileCheckpointStore struct {
	root string
}

func NewFileCheckpointStore(root string) (*FileCheckpointStore, error) {
	path := strings.TrimSpace(root)
	if path == "" {
		return nil, errors.New("workflow checkpoint root is required")
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir workflow checkpoint root: %w", err)
	}
	return &FileCheckpointStore{root: path}, nil
}

func (s *FileCheckpointStore) Load(_ context.Context, workflowID string) (Checkpoint, bool, error) {
	path := s.filePath(workflowID)
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Checkpoint{}, false, nil
	}
	if err != nil {
		return Checkpoint{}, false, fmt.Errorf("read workflow checkpoint: %w", err)
	}
	var cp Checkpoint
	if err := json.Unmarshal(raw, &cp); err != nil {
		return Checkpoint{}, false, fmt.Errorf("decode workflow checkpoint: %w", err)
	}
	return cp, true, nil
}

func (s *FileCheckpointStore) Save(_ context.Context, cp Checkpoint) error {
	if strings.TrimSpace(cp.WorkflowID) == "" {
		return errors.New("workflow checkpoint requires workflow_id")
	}
	if err := os.MkdirAll(s.root, 0o755); err != nil {
		return fmt.Errorf("mkdir workflow checkpoint root: %w", err)
	}
	raw, err := json.Marshal(cp)
	if err != nil {
		return fmt.Errorf("encode workflow checkpoint: %w", err)
	}
	if err := os.WriteFile(s.filePath(cp.WorkflowID), raw, 0o600); err != nil {
		return fmt.Errorf("write workflow checkpoint: %w", err)
	}
	return nil
}

func (s *FileCheckpointStore) filePath(workflowID string) string {
	clean := strings.ToLower(strings.TrimSpace(workflowID))
	clean = strings.ReplaceAll(clean, " ", "_")
	clean = strings.ReplaceAll(clean, "/", "_")
	clean = strings.ReplaceAll(clean, "\\", "_")
	return filepath.Join(s.root, clean+".json")
}
