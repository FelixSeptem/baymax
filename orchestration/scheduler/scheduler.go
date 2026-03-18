package scheduler

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync/atomic"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

type Option func(*Scheduler)

func WithTimelineEmitter(handler types.EventHandler) Option {
	return func(s *Scheduler) {
		s.timeline = handler
	}
}

func WithLeaseTimeout(timeout time.Duration) Option {
	return func(s *Scheduler) {
		if timeout > 0 {
			s.leaseTimeout = timeout
		}
	}
}

func WithGuardrails(guardrails Guardrails) Option {
	return func(s *Scheduler) {
		s.guardrails = guardrails
	}
}

type Scheduler struct {
	store        QueueStore
	leaseTimeout time.Duration
	guardrails   Guardrails
	timeline     types.EventHandler
	now          func() time.Time
	seq          atomic.Int64
}

func New(store QueueStore, opts ...Option) (*Scheduler, error) {
	if store == nil {
		return nil, errors.New("scheduler queue store is required")
	}
	s := &Scheduler{
		store:        store,
		leaseTimeout: 2 * time.Second,
		now:          time.Now,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(s)
		}
	}
	if s.leaseTimeout <= 0 {
		return nil, fmt.Errorf("lease timeout must be > 0")
	}
	return s, nil
}

func (s *Scheduler) Enqueue(ctx context.Context, task Task) (TaskRecord, error) {
	record, err := s.store.Enqueue(ctx, task, s.nowTime())
	if err != nil {
		return TaskRecord{}, err
	}
	s.emitTimeline(ctx, record, Attempt{}, types.ActionStatusPending, ReasonEnqueue)
	return record, nil
}

func (s *Scheduler) SpawnChild(ctx context.Context, req SpawnRequest) (TaskRecord, error) {
	if err := s.guardrails.ValidateSpawn(req); err != nil {
		record := TaskRecord{Task: req.Task}
		s.emitTimeline(ctx, record, Attempt{}, types.ActionStatusFailed, ReasonBudgetReject)
		return TaskRecord{}, err
	}
	record, err := s.Enqueue(ctx, req.Task)
	if err != nil {
		return TaskRecord{}, err
	}
	s.emitTimeline(ctx, record, Attempt{}, types.ActionStatusPending, ReasonSpawn)
	return record, nil
}

func (s *Scheduler) Claim(ctx context.Context, workerID string) (ClaimedTask, bool, error) {
	claimed, ok, err := s.store.Claim(ctx, workerID, s.nowTime(), s.leaseTimeout)
	if err != nil || !ok {
		return claimed, ok, err
	}
	s.emitTimeline(ctx, claimed.Record, claimed.Attempt, types.ActionStatusRunning, ReasonClaim)
	return claimed, true, nil
}

func (s *Scheduler) Heartbeat(ctx context.Context, taskID, attemptID, leaseToken string) (ClaimedTask, error) {
	claimed, err := s.store.Heartbeat(ctx, taskID, attemptID, leaseToken, s.nowTime(), s.leaseTimeout)
	if err != nil {
		return ClaimedTask{}, err
	}
	s.emitTimeline(ctx, claimed.Record, claimed.Attempt, types.ActionStatusRunning, ReasonHeartbeat)
	return claimed, nil
}

func (s *Scheduler) ExpireLeases(ctx context.Context) ([]ClaimedTask, error) {
	expired, err := s.store.ExpireLeases(ctx, s.nowTime())
	if err != nil {
		return nil, err
	}
	for _, item := range expired {
		s.emitTimeline(ctx, item.Record, item.Attempt, types.ActionStatusFailed, ReasonLeaseExpired)
		s.emitTimeline(ctx, item.Record, item.Attempt, types.ActionStatusPending, ReasonRequeue)
	}
	return expired, nil
}

func (s *Scheduler) Requeue(ctx context.Context, taskID, reason string) (TaskRecord, error) {
	record, err := s.store.Requeue(ctx, taskID, reason, s.nowTime())
	if err != nil {
		return TaskRecord{}, err
	}
	attempt, _ := record.currentAttempt()
	s.emitTimeline(ctx, record, attempt, types.ActionStatusPending, ReasonRequeue)
	return record, nil
}

func (s *Scheduler) Complete(ctx context.Context, commit TerminalCommit) (CommitResult, error) {
	if commit.Status != TaskStateSucceeded {
		return CommitResult{}, fmt.Errorf("complete requires succeeded status")
	}
	result, err := s.store.CommitTerminal(ctx, commit)
	if err != nil {
		return CommitResult{}, err
	}
	if !result.Duplicate {
		attempt, _ := result.Record.attemptByID(commit.AttemptID)
		s.emitTimeline(ctx, result.Record, attempt, types.ActionStatusSucceeded, ReasonJoin)
	}
	return result, nil
}

func (s *Scheduler) Fail(ctx context.Context, commit TerminalCommit) (CommitResult, error) {
	if commit.Status != TaskStateFailed {
		return CommitResult{}, fmt.Errorf("fail requires failed status")
	}
	result, err := s.store.CommitTerminal(ctx, commit)
	if err != nil {
		return CommitResult{}, err
	}
	if !result.Duplicate {
		attempt, _ := result.Record.attemptByID(commit.AttemptID)
		s.emitTimeline(ctx, result.Record, attempt, types.ActionStatusFailed, ReasonJoin)
	}
	return result, nil
}

func (s *Scheduler) Get(ctx context.Context, taskID string) (TaskRecord, bool, error) {
	return s.store.Get(ctx, taskID)
}

func (s *Scheduler) Stats(ctx context.Context) (Stats, error) {
	return s.store.Stats(ctx)
}

func (s *Scheduler) Snapshot(ctx context.Context) (StoreSnapshot, error) {
	type snapshotter interface {
		Snapshot(context.Context) (StoreSnapshot, error)
	}
	store, ok := s.store.(snapshotter)
	if !ok {
		return StoreSnapshot{}, errors.New("scheduler store does not support snapshot")
	}
	return store.Snapshot(ctx)
}

func (s *Scheduler) Restore(ctx context.Context, snapshot StoreSnapshot) error {
	type restorer interface {
		Restore(context.Context, StoreSnapshot) error
	}
	store, ok := s.store.(restorer)
	if !ok {
		return errors.New("scheduler store does not support restore")
	}
	return store.Restore(ctx, snapshot)
}

func (s *Scheduler) SnapshotEquivalent(ctx context.Context, snapshot StoreSnapshot) (bool, error) {
	current, err := s.Snapshot(ctx)
	if err != nil {
		return false, err
	}
	return reflect.DeepEqual(current, snapshot), nil
}

func (s *Scheduler) emitTimeline(
	ctx context.Context,
	record TaskRecord,
	attempt Attempt,
	status types.ActionStatus,
	reason string,
) {
	if s == nil || s.timeline == nil {
		return
	}
	reason, ok := CanonicalReason(reason)
	if !ok {
		return
	}
	sequence := s.seq.Add(1)
	payload := map[string]any{
		"phase":    string(types.ActionPhaseRun),
		"status":   string(status),
		"reason":   reason,
		"sequence": sequence,
		"task_id":  record.Task.TaskID,
	}
	if attemptID := strings.TrimSpace(attempt.AttemptID); attemptID != "" {
		payload["attempt_id"] = attemptID
	}
	if runID := strings.TrimSpace(record.Task.RunID); runID != "" {
		payload["run_id"] = runID
	}
	if workflowID := strings.TrimSpace(record.Task.WorkflowID); workflowID != "" {
		payload["workflow_id"] = workflowID
	}
	if teamID := strings.TrimSpace(record.Task.TeamID); teamID != "" {
		payload["team_id"] = teamID
	}
	if stepID := strings.TrimSpace(record.Task.StepID); stepID != "" {
		payload["step_id"] = stepID
	}
	if agentID := strings.TrimSpace(record.Task.AgentID); agentID != "" {
		payload["agent_id"] = agentID
	}
	if peerID := strings.TrimSpace(record.Task.PeerID); peerID != "" {
		payload["peer_id"] = peerID
	}
	if parentRunID := strings.TrimSpace(record.Task.ParentRunID); parentRunID != "" {
		payload["parent_run_id"] = parentRunID
	}
	s.timeline.OnEvent(ctx, types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    types.EventTypeActionTimeline,
		RunID:   strings.TrimSpace(record.Task.RunID),
		Time:    s.nowTime(),
		Payload: payload,
	})
}

func (s *Scheduler) nowTime() time.Time {
	if s == nil || s.now == nil {
		return time.Now()
	}
	return s.now()
}
