package scheduler

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
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

func WithGovernance(cfg GovernanceConfig) Option {
	return func(s *Scheduler) {
		s.governance = normalizeGovernanceConfig(cfg)
	}
}

func WithRecoveryBoundary(cfg RecoveryBoundaryConfig) Option {
	return func(s *Scheduler) {
		s.recoveryBoundary = normalizeRecoveryBoundaryConfig(cfg)
	}
}

func WithAsyncAwait(cfg AsyncAwaitConfig) Option {
	return func(s *Scheduler) {
		s.asyncAwait = normalizeAsyncAwaitConfig(cfg)
	}
}

func WithTaskBoardControl(cfg TaskBoardControlConfig) Option {
	return func(s *Scheduler) {
		s.taskBoardControl = normalizeTaskBoardControlConfig(cfg)
	}
}

type Scheduler struct {
	store            QueueStore
	leaseTimeout     time.Duration
	guardrails       Guardrails
	governance       GovernanceConfig
	asyncAwait       AsyncAwaitConfig
	taskBoardControl TaskBoardControlConfig
	recoveryBoundary RecoveryBoundaryConfig
	timeline         types.EventHandler
	now              func() time.Time
	seq              atomic.Int64
	reconcileSeq     atomic.Int64

	queryCacheMu    sync.RWMutex
	queryCacheEpoch uint64
	queryCache      map[string][]TaskRecord
}

func New(store QueueStore, opts ...Option) (*Scheduler, error) {
	if store == nil {
		return nil, errors.New("scheduler queue store is required")
	}
	s := &Scheduler{
		store:            store,
		leaseTimeout:     2 * time.Second,
		governance:       defaultGovernanceConfig(),
		asyncAwait:       defaultAsyncAwaitConfig(),
		taskBoardControl: defaultTaskBoardControlConfig(),
		recoveryBoundary: defaultRecoveryBoundaryConfig(),
		now:              time.Now,
		queryCache:       map[string][]TaskRecord{},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(s)
		}
	}
	if s.leaseTimeout <= 0 {
		return nil, fmt.Errorf("lease timeout must be > 0")
	}
	s.governance = normalizeGovernanceConfig(s.governance)
	s.asyncAwait = normalizeAsyncAwaitConfig(s.asyncAwait)
	s.taskBoardControl = normalizeTaskBoardControlConfig(s.taskBoardControl)
	if configurable, ok := store.(interface {
		SetGovernance(GovernanceConfig)
	}); ok {
		configurable.SetGovernance(s.governance)
	}
	if configurable, ok := store.(interface {
		SetRecoveryBoundary(RecoveryBoundaryConfig)
	}); ok {
		configurable.SetRecoveryBoundary(s.recoveryBoundary)
	}
	if configurable, ok := store.(interface {
		SetAsyncAwait(AsyncAwaitConfig)
	}); ok {
		configurable.SetAsyncAwait(s.asyncAwait)
	}
	if configurable, ok := store.(interface {
		SetTaskBoardControl(TaskBoardControlConfig)
	}); ok {
		configurable.SetTaskBoardControl(s.taskBoardControl)
	}
	return s, nil
}

func (s *Scheduler) Enqueue(ctx context.Context, task Task) (TaskRecord, error) {
	now := s.nowTime()
	record, err := s.store.Enqueue(ctx, task, now)
	if err != nil {
		return TaskRecord{}, err
	}
	s.invalidateTaskBoardQueryCache()
	s.emitTimeline(ctx, record, Attempt{}, types.ActionStatusPending, ReasonEnqueue)
	if isTaskDelayed(record.Task, now) {
		remainingMs := record.Task.NotBefore.Sub(now).Milliseconds()
		if remainingMs < 0 {
			remainingMs = 0
		}
		s.emitTimelineWithExtras(ctx, record, Attempt{}, types.ActionStatusPending, ReasonDelayedEnqueue, map[string]any{
			"not_before_unix_ms": record.Task.NotBefore.UnixMilli(),
			"remaining_delay_ms": remainingMs,
		})
		s.emitTimelineWithExtras(ctx, record, Attempt{}, types.ActionStatusPending, ReasonDelayedWait, map[string]any{
			"not_before_unix_ms": record.Task.NotBefore.UnixMilli(),
			"remaining_delay_ms": remainingMs,
		})
	}
	return record, nil
}

func (s *Scheduler) SpawnChild(ctx context.Context, req SpawnRequest) (TaskRecord, error) {
	if req.ParentRemainingBudget <= 0 {
		record := TaskRecord{Task: req.Task}
		s.emitTimeline(ctx, record, Attempt{}, types.ActionStatusFailed, ReasonBudgetReject)
		return TaskRecord{}, &BudgetError{
			Code:    BudgetRejectParentBudgetExhausted,
			Message: fmt.Sprintf("parent remaining budget %s must be > 0", req.ParentRemainingBudget),
		}
	}
	if req.ChildTimeout <= 0 {
		record := TaskRecord{Task: req.Task}
		s.emitTimeline(ctx, record, Attempt{}, types.ActionStatusFailed, ReasonBudgetReject)
		return TaskRecord{}, &BudgetError{
			Code:    BudgetRejectTimeout,
			Message: fmt.Sprintf("child timeout %s must be > 0", req.ChildTimeout),
		}
	}
	if req.ChildTimeout > req.ParentRemainingBudget {
		req.ChildTimeout = req.ParentRemainingBudget
		req.TimeoutResolution.ParentBudgetClamped = true
	}
	req.TimeoutResolution.EffectiveOperationProfile = strings.TrimSpace(req.TimeoutResolution.EffectiveOperationProfile)
	req.TimeoutResolution.Source = strings.TrimSpace(req.TimeoutResolution.Source)
	req.TimeoutResolution.Trace = strings.TrimSpace(req.TimeoutResolution.Trace)
	req.TimeoutResolution.ResolvedTimeout = req.ChildTimeout
	req.Task.TimeoutResolution = req.TimeoutResolution

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
	now := s.nowTime()
	claimed, ok, err := s.store.Claim(ctx, workerID, now, s.leaseTimeout)
	if err != nil || !ok {
		return claimed, ok, err
	}
	s.invalidateTaskBoardQueryCache()
	if isTaskDelayed(claimed.Record.Task, claimed.Record.CreatedAt) {
		waitMs := now.Sub(claimed.Record.CreatedAt).Milliseconds()
		if waitMs < 0 {
			waitMs = 0
		}
		s.emitTimelineWithExtras(ctx, claimed.Record, claimed.Attempt, types.ActionStatusPending, ReasonDelayedReady, map[string]any{
			"not_before_unix_ms": claimed.Record.Task.NotBefore.UnixMilli(),
			"delayed_wait_ms":    waitMs,
		})
	}
	s.emitTimeline(ctx, claimed.Record, claimed.Attempt, types.ActionStatusRunning, ReasonClaim)
	if s.governance.QoS == QoSModePriority {
		s.emitTimelineWithExtras(ctx, claimed.Record, claimed.Attempt, types.ActionStatusRunning, ReasonQoSClaim, map[string]any{
			"qos_mode":      string(s.governance.QoS),
			"task_priority": normalizedPriority(claimed.TaskPriority),
		})
		if claimed.FairnessYielded {
			s.emitTimelineWithExtras(ctx, claimed.Record, claimed.Attempt, types.ActionStatusPending, ReasonFairnessYield, map[string]any{
				"qos_mode":         string(s.governance.QoS),
				"task_priority":    normalizedPriority(claimed.TaskPriority),
				"fairness_window":  s.governance.Fairness.MaxConsecutiveClaimsPerPriority,
				"fairness_yielded": true,
			})
		}
	}
	return claimed, true, nil
}

func (s *Scheduler) Heartbeat(ctx context.Context, taskID, attemptID, leaseToken string) (ClaimedTask, error) {
	claimed, err := s.store.Heartbeat(ctx, taskID, attemptID, leaseToken, s.nowTime(), s.leaseTimeout)
	if err != nil {
		return ClaimedTask{}, err
	}
	s.invalidateTaskBoardQueryCache()
	s.emitTimeline(ctx, claimed.Record, claimed.Attempt, types.ActionStatusRunning, ReasonHeartbeat)
	return claimed, nil
}

func (s *Scheduler) ExpireLeases(ctx context.Context) ([]ClaimedTask, error) {
	now := s.nowTime()
	expired, err := s.store.ExpireLeases(ctx, now)
	if err != nil {
		return nil, err
	}
	for _, item := range expired {
		s.emitTimeline(ctx, item.Record, item.Attempt, types.ActionStatusFailed, ReasonLeaseExpired)
		switch item.Record.State {
		case TaskStateQueued:
			s.emitTimeline(ctx, item.Record, item.Attempt, types.ActionStatusPending, ReasonRequeue)
			if !item.Record.NextEligibleAt.IsZero() && item.Record.NextEligibleAt.After(now) {
				s.emitTimelineWithExtras(ctx, item.Record, item.Attempt, types.ActionStatusPending, ReasonRetryBackoff, map[string]any{
					"task_priority": normalizedPriority(item.Record.Task.Priority),
					"backoff_ms":    item.Record.NextEligibleAt.Sub(now).Milliseconds(),
					"max_attempts":  item.Record.Task.MaxAttempts,
				})
			}
		case TaskStateDeadLetter:
			s.emitTimelineWithExtras(ctx, item.Record, item.Attempt, types.ActionStatusFailed, ReasonDeadLetter, map[string]any{
				"task_priority":     normalizedPriority(item.Record.Task.Priority),
				"dead_letter_code":  strings.TrimSpace(item.Record.DeadLetterCode),
				"max_attempts":      item.Record.Task.MaxAttempts,
				"dead_letter_state": string(TaskStateDeadLetter),
				"retry_exhausted":   true,
				"retry_attempts":    len(item.Record.Attempts),
			})
		}
	}
	awaitingExpired, err := s.store.ExpireAwaitingReports(ctx, now)
	if err != nil {
		return nil, err
	}
	for _, item := range awaitingExpired {
		extras := map[string]any{
			"task_priority":    normalizedPriority(item.Record.Task.Priority),
			"timeout_terminal": string(item.Record.State),
		}
		if !item.Record.ReportTimeoutAt.IsZero() {
			extras["report_timeout_at_unix_ms"] = item.Record.ReportTimeoutAt.UnixMilli()
		}
		s.emitTimelineWithExtras(ctx, item.Record, item.Attempt, types.ActionStatusFailed, ReasonAsyncTimeout, extras)
		if item.Record.State == TaskStateDeadLetter {
			s.emitTimelineWithExtras(ctx, item.Record, item.Attempt, types.ActionStatusFailed, ReasonDeadLetter, map[string]any{
				"task_priority":     normalizedPriority(item.Record.Task.Priority),
				"dead_letter_code":  strings.TrimSpace(item.Record.DeadLetterCode),
				"max_attempts":      item.Record.Task.MaxAttempts,
				"dead_letter_state": string(TaskStateDeadLetter),
				"retry_exhausted":   true,
				"retry_attempts":    len(item.Record.Attempts),
			})
		}
	}
	if len(awaitingExpired) > 0 {
		expired = append(expired, awaitingExpired...)
	}
	if len(expired) > 0 {
		s.invalidateTaskBoardQueryCache()
	}
	return expired, nil
}

func (s *Scheduler) MarkAwaitingReport(ctx context.Context, taskID, attemptID string, remoteTaskID ...string) (TaskRecord, error) {
	now := s.nowTime()
	remote := ""
	if len(remoteTaskID) > 0 {
		remote = strings.TrimSpace(remoteTaskID[0])
	}
	record, err := s.store.MarkAwaitingReport(ctx, taskID, attemptID, remote, now, s.asyncAwait.ReportTimeout)
	if err != nil {
		return TaskRecord{}, err
	}
	s.invalidateTaskBoardQueryCache()
	attempt, _ := record.attemptByID(strings.TrimSpace(attemptID))
	extras := map[string]any{
		"report_timeout_ms": s.asyncAwait.ReportTimeout.Milliseconds(),
	}
	if strings.TrimSpace(record.RemoteTaskID) != "" {
		extras["remote_task_id"] = strings.TrimSpace(record.RemoteTaskID)
	}
	s.emitTimelineWithExtras(ctx, record, attempt, types.ActionStatusPending, ReasonAwaitingReport, extras)
	return record, nil
}

func (s *Scheduler) Requeue(ctx context.Context, taskID, reason string) (TaskRecord, error) {
	now := s.nowTime()
	record, err := s.store.Requeue(ctx, taskID, reason, now)
	if err != nil {
		return TaskRecord{}, err
	}
	s.invalidateTaskBoardQueryCache()
	attempt := latestAttempt(record)
	switch record.State {
	case TaskStateQueued:
		s.emitTimeline(ctx, record, attempt, types.ActionStatusPending, ReasonRequeue)
		if !record.NextEligibleAt.IsZero() && record.NextEligibleAt.After(now) {
			s.emitTimelineWithExtras(ctx, record, attempt, types.ActionStatusPending, ReasonRetryBackoff, map[string]any{
				"task_priority": normalizedPriority(record.Task.Priority),
				"backoff_ms":    record.NextEligibleAt.Sub(now).Milliseconds(),
				"max_attempts":  record.Task.MaxAttempts,
			})
		}
	case TaskStateDeadLetter:
		s.emitTimelineWithExtras(ctx, record, attempt, types.ActionStatusFailed, ReasonDeadLetter, map[string]any{
			"task_priority":     normalizedPriority(record.Task.Priority),
			"dead_letter_code":  strings.TrimSpace(record.DeadLetterCode),
			"max_attempts":      record.Task.MaxAttempts,
			"dead_letter_state": string(TaskStateDeadLetter),
			"retry_exhausted":   true,
			"retry_attempts":    len(record.Attempts),
		})
	}
	return record, nil
}

func (s *Scheduler) ControlTask(ctx context.Context, req TaskBoardControlRequest) (TaskBoardControlResult, error) {
	result, err := s.store.ControlTask(ctx, req, s.nowTime())
	if err != nil {
		return TaskBoardControlResult{}, err
	}
	if !result.Applied || result.Deduplicated {
		return result, nil
	}
	s.invalidateTaskBoardQueryCache()
	switch result.Action {
	case TaskBoardControlActionCancel:
		s.emitTimeline(ctx, result.Record, latestAttempt(result.Record), types.ActionStatusCanceled, ReasonManualCancel)
	case TaskBoardControlActionRetryTerminal:
		s.emitTimeline(ctx, result.Record, latestAttempt(result.Record), types.ActionStatusPending, ReasonManualRetry)
	}
	return result, nil
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
		s.invalidateTaskBoardQueryCache()
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
		s.invalidateTaskBoardQueryCache()
		attempt, _ := result.Record.attemptByID(commit.AttemptID)
		s.emitTimeline(ctx, result.Record, attempt, types.ActionStatusFailed, ReasonJoin)
	}
	return result, nil
}

func (s *Scheduler) CommitAsyncReportTerminal(ctx context.Context, commit TerminalCommit) (CommitResult, error) {
	result, err := s.store.CommitAsyncReportTerminal(ctx, commit)
	if err != nil {
		return CommitResult{}, err
	}
	if result.LateReport {
		return result, nil
	}
	if result.Duplicate {
		return result, nil
	}
	s.invalidateTaskBoardQueryCache()
	attempt, _ := result.Record.attemptByID(commit.AttemptID)
	switch commit.Status {
	case TaskStateSucceeded:
		s.emitTimeline(ctx, result.Record, attempt, types.ActionStatusSucceeded, ReasonJoin)
	case TaskStateFailed:
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
	if err := store.Restore(ctx, snapshot); err != nil {
		return err
	}
	s.invalidateTaskBoardQueryCache()
	return nil
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
	s.emitTimelineWithExtras(ctx, record, attempt, status, reason, nil)
}

func (s *Scheduler) emitTimelineWithExtras(
	ctx context.Context,
	record TaskRecord,
	attempt Attempt,
	status types.ActionStatus,
	reason string,
	extras map[string]any,
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
	for k, v := range extras {
		payload[k] = v
	}
	s.timeline.OnEvent(ctx, types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    types.EventTypeActionTimeline,
		RunID:   strings.TrimSpace(record.Task.RunID),
		Time:    s.nowTime(),
		Payload: payload,
	})
}

func latestAttempt(record TaskRecord) Attempt {
	if len(record.Attempts) == 0 {
		return Attempt{}
	}
	return record.Attempts[len(record.Attempts)-1]
}

func (s *Scheduler) nowTime() time.Time {
	if s == nil || s.now == nil {
		return time.Now()
	}
	return s.now()
}

func (s *Scheduler) taskBoardQueryEpoch() uint64 {
	if s == nil {
		return 0
	}
	s.queryCacheMu.RLock()
	defer s.queryCacheMu.RUnlock()
	return s.queryCacheEpoch
}

func (s *Scheduler) readTaskBoardQueryCache(epoch uint64, key string) ([]TaskRecord, bool) {
	if s == nil || strings.TrimSpace(key) == "" {
		return nil, false
	}
	s.queryCacheMu.RLock()
	defer s.queryCacheMu.RUnlock()
	if s.queryCacheEpoch != epoch {
		return nil, false
	}
	items, ok := s.queryCache[key]
	if !ok {
		return nil, false
	}
	return cloneTaskRecords(items), true
}

func (s *Scheduler) writeTaskBoardQueryCache(epoch uint64, key string, items []TaskRecord) {
	if s == nil || strings.TrimSpace(key) == "" {
		return
	}
	const maxEntries = 32
	s.queryCacheMu.Lock()
	defer s.queryCacheMu.Unlock()
	if s.queryCacheEpoch != epoch {
		return
	}
	if s.queryCache == nil {
		s.queryCache = map[string][]TaskRecord{}
	}
	if len(s.queryCache) >= maxEntries {
		clear(s.queryCache)
	}
	s.queryCache[key] = cloneTaskRecords(items)
}

func (s *Scheduler) invalidateTaskBoardQueryCache() {
	if s == nil {
		return
	}
	s.queryCacheMu.Lock()
	defer s.queryCacheMu.Unlock()
	s.queryCacheEpoch++
	clear(s.queryCache)
}

func cloneTaskRecords(in []TaskRecord) []TaskRecord {
	if len(in) == 0 {
		return nil
	}
	out := make([]TaskRecord, len(in))
	copy(out, in)
	return out
}
