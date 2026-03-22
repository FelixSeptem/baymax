package scheduler

import (
	"context"
	"strings"
	"sync"
	"time"
)

type MemoryStore struct {
	mu    sync.Mutex
	state schedulerState
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{state: newSchedulerState("memory")}
}

func (s *MemoryStore) SetGovernance(cfg GovernanceConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.setGovernance(cfg)
}

func (s *MemoryStore) SetRecoveryBoundary(cfg RecoveryBoundaryConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.setRecoveryBoundary(cfg)
}

func (s *MemoryStore) SetAsyncAwait(cfg AsyncAwaitConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.setAsyncAwait(cfg)
}

func (s *MemoryStore) Backend() string {
	return "memory"
}

func (s *MemoryStore) Enqueue(_ context.Context, task Task, now time.Time) (TaskRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.enqueue(task, now)
}

func (s *MemoryStore) Claim(_ context.Context, workerID string, now time.Time, leaseTimeout time.Duration) (ClaimedTask, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.claim(strings.TrimSpace(workerID), now, leaseTimeout)
}

func (s *MemoryStore) Heartbeat(_ context.Context, taskID, attemptID, leaseToken string, now time.Time, leaseTimeout time.Duration) (ClaimedTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.heartbeat(taskID, attemptID, leaseToken, now, leaseTimeout)
}

func (s *MemoryStore) ExpireLeases(_ context.Context, now time.Time) ([]ClaimedTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.expireLeases(now), nil
}

func (s *MemoryStore) ExpireAwaitingReports(_ context.Context, now time.Time) ([]ClaimedTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.expireAwaitingReports(now), nil
}

func (s *MemoryStore) MarkAwaitingReport(_ context.Context, taskID, attemptID, remoteTaskID string, now time.Time, reportTimeout time.Duration) (TaskRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.markAwaitingReport(taskID, attemptID, remoteTaskID, now, reportTimeout)
}

func (s *MemoryStore) ListAwaitingReport(_ context.Context, now time.Time, limit int) ([]TaskRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.listAwaitingReport(now, limit), nil
}

func (s *MemoryStore) RecordAsyncReconcileStats(_ context.Context, pollTotal, errorTotal int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.recordAsyncReconcileStats(pollTotal, errorTotal)
	return nil
}

func (s *MemoryStore) Requeue(_ context.Context, taskID, _ string, now time.Time) (TaskRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.requeue(taskID, now)
}

func (s *MemoryStore) CommitTerminal(_ context.Context, commit TerminalCommit) (CommitResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.commitTerminal(commit)
}

func (s *MemoryStore) CommitAsyncReportTerminal(_ context.Context, commit TerminalCommit) (CommitResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.commitAsyncReportTerminal(commit)
}

func (s *MemoryStore) Get(_ context.Context, taskID string) (TaskRecord, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.state.get(taskID)
	return record, ok, nil
}

func (s *MemoryStore) Stats(_ context.Context) (Stats, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.Stats, nil
}

func (s *MemoryStore) Snapshot(_ context.Context) (StoreSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.snapshot(), nil
}

func (s *MemoryStore) Restore(_ context.Context, snapshot StoreSnapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.restore(snapshot)
}
