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
	state          schedulerState
	persistOpts    fileStoreOptions
	pendingPersist int
	dirtySince     time.Time
}

func NewFileStore(path string, opts ...FileStoreOption) (*FileStore, error) {
	cleaned := strings.TrimSpace(path)
	if cleaned == "" {
		return nil, fmt.Errorf("scheduler file backend path is required")
	}
	store := &FileStore{
		path:        cleaned,
		state:       newSchedulerState("file"),
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

func (s *FileStore) SetGovernance(cfg GovernanceConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.setGovernance(cfg)
}

func (s *FileStore) SetRecoveryBoundary(cfg RecoveryBoundaryConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.setRecoveryBoundary(cfg)
}

func (s *FileStore) SetAsyncAwait(cfg AsyncAwaitConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.setAsyncAwait(cfg)
}

func (s *FileStore) SetTaskBoardControl(cfg TaskBoardControlConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.setTaskBoardControl(cfg)
}

func (s *FileStore) Enqueue(_ context.Context, task Task, now time.Time) (TaskRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, err := s.state.enqueue(task, now)
	if err != nil {
		return TaskRecord{}, err
	}
	if err := s.maybePersistLocked(false); err != nil {
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
	if err := s.maybePersistLocked(false); err != nil {
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
	if err := s.maybePersistLocked(false); err != nil {
		return ClaimedTask{}, err
	}
	return claimed, nil
}

func (s *FileStore) ControlTask(_ context.Context, req TaskBoardControlRequest, now time.Time) (TaskBoardControlResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result, err := s.state.controlTask(req, now)
	if err != nil {
		return TaskBoardControlResult{}, err
	}
	if err := s.maybePersistLocked(false); err != nil {
		return TaskBoardControlResult{}, err
	}
	return result, nil
}

func (s *FileStore) ExpireLeases(_ context.Context, now time.Time) ([]ClaimedTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	expired := s.state.expireLeases(now)
	if len(expired) == 0 {
		return nil, nil
	}
	if err := s.maybePersistLocked(false); err != nil {
		return nil, err
	}
	return expired, nil
}

func (s *FileStore) ExpireAwaitingReports(_ context.Context, now time.Time) ([]ClaimedTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	expired := s.state.expireAwaitingReports(now)
	if len(expired) == 0 {
		return nil, nil
	}
	if err := s.maybePersistLocked(false); err != nil {
		return nil, err
	}
	return expired, nil
}

func (s *FileStore) MarkAwaitingReport(_ context.Context, taskID, attemptID, remoteTaskID string, now time.Time, reportTimeout time.Duration) (TaskRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, err := s.state.markAwaitingReport(taskID, attemptID, remoteTaskID, now, reportTimeout)
	if err != nil {
		return TaskRecord{}, err
	}
	if err := s.maybePersistLocked(false); err != nil {
		return TaskRecord{}, err
	}
	return record, nil
}

func (s *FileStore) ListAwaitingReport(_ context.Context, now time.Time, limit int) ([]TaskRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.listAwaitingReport(now, limit), nil
}

func (s *FileStore) RecordAsyncReconcileStats(_ context.Context, pollTotal, errorTotal int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if pollTotal <= 0 && errorTotal <= 0 {
		return nil
	}
	s.state.recordAsyncReconcileStats(pollTotal, errorTotal)
	return s.maybePersistLocked(false)
}

func (s *FileStore) Requeue(_ context.Context, taskID, _ string, now time.Time) (TaskRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, err := s.state.requeue(taskID, now)
	if err != nil {
		return TaskRecord{}, err
	}
	if err := s.maybePersistLocked(false); err != nil {
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
	if err := s.maybePersistLocked(false); err != nil {
		return CommitResult{}, err
	}
	return result, nil
}

func (s *FileStore) CommitAsyncReportTerminal(_ context.Context, commit TerminalCommit) (CommitResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result, err := s.state.commitAsyncReportTerminal(commit)
	if err != nil {
		return CommitResult{}, err
	}
	if err := s.maybePersistLocked(false); err != nil {
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
	return s.state.snapshotStats(), nil
}

func (s *FileStore) Snapshot(_ context.Context) (StoreSnapshot, error) {
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

func (s *FileStore) Restore(_ context.Context, snapshot StoreSnapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.state.restore(snapshot); err != nil {
		return err
	}
	return s.maybePersistLocked(true)
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
	snapshot := decoded.snapshot()
	snapshot.Backend = "file"
	snapshot.Stats.Backend = "file"
	if err := s.state.restore(snapshot); err != nil {
		return fmt.Errorf("decode scheduler file backend: %w", err)
	}
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
