package composer

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/orchestration/scheduler"
	"github.com/FelixSeptem/baymax/orchestration/workflow"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

type RecoverRequest struct {
	RunID string
}

type RecoverResult struct {
	RunID                   string
	ConflictPolicy          string
	RecoveredA2AInFlight    int
	ReplayedTerminalCommits int
}

func (c *Composer) initRecovery(cfg runtimeconfig.Config) error {
	enabled := cfg.Recovery.Enabled
	configuredBackend := strings.TrimSpace(strings.ToLower(cfg.Recovery.Backend))
	if configuredBackend == "" {
		configuredBackend = runtimeconfig.RecoveryBackendMemory
	}
	backend := configuredBackend
	path := strings.TrimSpace(cfg.Recovery.Path)
	conflictPolicy := strings.TrimSpace(strings.ToLower(cfg.Recovery.ConflictPolicy))
	if conflictPolicy == "" {
		conflictPolicy = runtimeconfig.RecoveryConflictPolicyFailFast
	}
	store := c.recoveryStore
	fallback := false
	fallbackReason := ""

	switch {
	case !c.managedRecoveryStore && store != nil:
		enabled = true
		configuredBackend = "custom"
		backend = strings.TrimSpace(store.Backend())
		if backend == "" {
			backend = "custom"
		}
		path = ""
	case !enabled:
		store = nil
		backend = "disabled"
	case store == nil:
		switch configuredBackend {
		case runtimeconfig.RecoveryBackendFile:
			fileStore, err := NewFileRecoveryStore(path)
			if err != nil {
				store = NewMemoryRecoveryStore()
				backend = runtimeconfig.RecoveryBackendMemory
				fallback = true
				fallbackReason = "recovery.backend.file_init_failed"
			} else {
				store = fileStore
			}
		default:
			backend = runtimeconfig.RecoveryBackendMemory
			store = NewMemoryRecoveryStore()
		}
	}

	c.schedulerMu.Lock()
	c.recoveryStore = store
	c.recoveryEnabled = enabled
	c.recoveryConfiguredBackend = configuredBackend
	c.recoveryBackend = backend
	c.recoveryPath = path
	c.recoveryFallback = fallback
	c.recoveryFallbackReason = fallbackReason
	c.recoveryConflictPolicy = conflictPolicy
	c.recoverySignature = c.recoveryConfigSignature(cfg)
	c.schedulerMu.Unlock()
	return nil
}

func (c *Composer) refreshRecoveryForNextAttempt() {
	if c == nil || !c.managedRecoveryStore || c.runtimeMgr == nil {
		return
	}
	cfg := c.runtimeMgr.EffectiveConfig()
	signature := c.recoveryConfigSignature(cfg)

	c.schedulerMu.RLock()
	if c.recoverySignature == signature {
		c.schedulerMu.RUnlock()
		return
	}
	c.schedulerMu.RUnlock()

	_ = c.initRecovery(cfg)
}

func (c *Composer) recoveryConfigSignature(cfg runtimeconfig.Config) string {
	return strings.Join([]string{
		fmt.Sprintf("%t", cfg.Recovery.Enabled),
		strings.TrimSpace(strings.ToLower(cfg.Recovery.Backend)),
		strings.TrimSpace(cfg.Recovery.Path),
		strings.TrimSpace(strings.ToLower(cfg.Recovery.ConflictPolicy)),
	}, "|")
}

func (c *Composer) maybePersistRecoverySnapshot(ctx context.Context, runID string) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return
	}
	if !c.recoveryEnabled || c.recoveryStore == nil {
		return
	}
	if _, err := c.PersistRecoverySnapshot(ctx, runID, ""); err != nil {
		c.markRecoveryFallback(runID, "recovery.snapshot.persist_failed")
	}
}

func (c *Composer) CaptureRecoverySnapshot(ctx context.Context, runID, workflowID string) (RecoverySnapshot, error) {
	s := c.Scheduler()
	if s == nil {
		return RecoverySnapshot{}, newRecoveryError(RecoveryErrorStoreUnavailable, "scheduler is not initialized", nil)
	}
	schedulerSnapshot, err := s.Snapshot(ctx)
	if err != nil {
		return RecoverySnapshot{}, newRecoveryError(RecoveryErrorStoreUnavailable, "capture scheduler snapshot", err)
	}

	workflowID = strings.TrimSpace(workflowID)
	var checkpoint workflow.Checkpoint
	if workflowID != "" {
		if loader, ok := c.workflow.(interface {
			LoadCheckpoint(context.Context, string) (workflow.Checkpoint, bool, error)
		}); ok {
			cp, found, loadErr := loader.LoadCheckpoint(ctx, workflowID)
			if loadErr != nil {
				return RecoverySnapshot{}, loadErr
			}
			if found {
				checkpoint = cp
			}
		}
	}

	resolvedRunID := strings.TrimSpace(runID)
	if resolvedRunID == "" {
		resolvedRunID = strings.TrimSpace(checkpoint.RunID)
	}
	if resolvedRunID == "" {
		for i := range schedulerSnapshot.Tasks {
			if candidate := strings.TrimSpace(schedulerSnapshot.Tasks[i].Task.RunID); candidate != "" {
				resolvedRunID = candidate
				break
			}
		}
	}
	if resolvedRunID == "" {
		return RecoverySnapshot{}, newRecoveryError(RecoveryErrorSnapshotCorrupt, "recovery snapshot requires run_id", nil)
	}

	conflictPolicy := strings.TrimSpace(strings.ToLower(c.recoveryConflictPolicy))
	if conflictPolicy == "" {
		conflictPolicy = runtimeconfig.RecoveryConflictPolicyFailFast
	}
	snapshot := RecoverySnapshot{
		Version:   RecoverySnapshotVersion,
		UpdatedAt: c.now(),
		Run: RecoveryRunSnapshot{
			RunID: resolvedRunID,
		},
		Workflow: RecoveryWorkflowSnapshot{
			Checkpoint: checkpoint,
		},
		Scheduler: schedulerSnapshot,
		A2A: RecoveryA2ASnapshot{
			InFlight: extractRecoveryA2AInFlightStates(schedulerSnapshot),
		},
		Replay: RecoveryReplayCursor{
			Sequence:            c.now().UnixNano(),
			TerminalCommitCount: len(schedulerSnapshot.TerminalCommits),
		},
		ConflictPolicy: conflictPolicy,
	}
	return normalizeRecoverySnapshot(snapshot, resolvedRunID)
}

func (c *Composer) PersistRecoverySnapshot(ctx context.Context, runID, workflowID string) (RecoverySnapshot, error) {
	if !c.recoveryEnabled || c.recoveryStore == nil {
		return RecoverySnapshot{}, newRecoveryError(RecoveryErrorStoreUnavailable, "recovery is disabled", nil)
	}
	snapshot, err := c.CaptureRecoverySnapshot(ctx, runID, workflowID)
	if err != nil {
		return RecoverySnapshot{}, err
	}
	if err := c.recoveryStore.Save(ctx, snapshot); err != nil {
		return RecoverySnapshot{}, err
	}
	return snapshot, nil
}

func (c *Composer) Recover(ctx context.Context, req RecoverRequest) (RecoverResult, error) {
	c.refreshSchedulerForNextAttempt()
	c.refreshRecoveryForNextAttempt()

	runID := strings.TrimSpace(req.RunID)
	if runID == "" {
		return RecoverResult{}, newRecoveryError(RecoveryErrorSnapshotNotFound, "recover requires run_id", nil)
	}
	if !c.recoveryEnabled || c.recoveryStore == nil {
		return RecoverResult{}, newRecoveryError(RecoveryErrorStoreUnavailable, "recovery is disabled", nil)
	}
	s := c.Scheduler()
	if s == nil {
		return RecoverResult{}, newRecoveryError(RecoveryErrorStoreUnavailable, "scheduler is not initialized", nil)
	}

	c.emitRecoveryTimeline(ctx, runID, RecoveryReasonRestore, types.ActionStatusRunning, "", "")
	snapshot, found, err := c.recoveryStore.Load(ctx, runID)
	if err != nil {
		c.markRecoveryConflict(runID, "load_failed")
		c.emitRecoveryTimeline(ctx, runID, RecoveryReasonConflict, types.ActionStatusFailed, "", "")
		return RecoverResult{}, err
	}
	if !found {
		c.markRecoveryConflict(runID, "snapshot_not_found")
		c.emitRecoveryTimeline(ctx, runID, RecoveryReasonConflict, types.ActionStatusFailed, "", "")
		return RecoverResult{}, newRecoveryError(
			RecoveryErrorSnapshotNotFound,
			fmt.Sprintf("recovery snapshot for run %q is not found", runID),
			nil,
		)
	}
	normalized, err := normalizeRecoverySnapshot(snapshot, runID)
	if err != nil {
		c.markRecoveryConflict(runID, "snapshot_corrupt")
		c.emitRecoveryTimeline(ctx, runID, RecoveryReasonConflict, types.ActionStatusFailed, "", "")
		return RecoverResult{}, err
	}
	if normalized.ConflictPolicy != runtimeconfig.RecoveryConflictPolicyFailFast {
		c.markRecoveryConflict(runID, "policy_unsupported")
		c.emitRecoveryTimeline(ctx, runID, RecoveryReasonConflict, types.ActionStatusFailed, "", "")
		return RecoverResult{}, newRecoveryError(
			RecoveryErrorPolicyUnsupported,
			fmt.Sprintf("unsupported recovery conflict policy %q", normalized.ConflictPolicy),
			nil,
		)
	}

	current, currentErr := s.Snapshot(ctx)
	if currentErr != nil {
		c.markRecoveryConflict(runID, "scheduler_snapshot_failed")
		c.emitRecoveryTimeline(ctx, runID, RecoveryReasonConflict, types.ActionStatusFailed, "", "")
		return RecoverResult{}, currentErr
	}
	if hasSchedulerSnapshotState(current) && !schedulerSnapshotsEqual(current, normalized.Scheduler) {
		c.markRecoveryConflict(runID, "state_mismatch")
		c.emitRecoveryTimeline(ctx, runID, RecoveryReasonConflict, types.ActionStatusFailed, "", "")
		return RecoverResult{}, newRecoveryError(
			RecoveryErrorConflict,
			"recovery fail_fast: scheduler snapshot conflicts with current runtime state",
			nil,
		)
	}

	checkpoint := normalized.Workflow.Checkpoint
	if strings.TrimSpace(checkpoint.WorkflowID) != "" {
		if restorer, ok := c.workflow.(interface {
			RestoreCheckpoint(context.Context, workflow.Checkpoint) error
		}); ok {
			if err := restorer.RestoreCheckpoint(ctx, checkpoint); err != nil {
				c.markRecoveryConflict(runID, "workflow_restore_failed")
				c.emitRecoveryTimeline(ctx, runID, RecoveryReasonConflict, types.ActionStatusFailed, "", "")
				return RecoverResult{}, err
			}
		}
	}

	if err := s.Restore(ctx, normalized.Scheduler); err != nil {
		c.markRecoveryConflict(runID, "scheduler_restore_failed")
		c.emitRecoveryTimeline(ctx, runID, RecoveryReasonConflict, types.ActionStatusFailed, "", "")
		return RecoverResult{}, err
	}
	if err := reconcileA2AInFlight(normalized.A2A.InFlight, normalized.Scheduler); err != nil {
		c.markRecoveryConflict(runID, "a2a_correlation_mismatch")
		c.emitRecoveryTimeline(ctx, runID, RecoveryReasonConflict, types.ActionStatusFailed, "", "")
		return RecoverResult{}, newRecoveryError(RecoveryErrorConflict, "recovery fail_fast: a2a in-flight correlation mismatch", err)
	}

	c.rebuildCollabStatsFromSchedulerSnapshot(runID, normalized.Scheduler)
	c.markRecoveryRecovered(runID, normalized.Replay.TerminalCommitCount)
	c.emitRecoveryTimeline(ctx, runID, RecoveryReasonRestore, types.ActionStatusSucceeded, "", "")
	c.emitRecoveryTimeline(ctx, runID, RecoveryReasonReplay, types.ActionStatusSucceeded, "", "")
	return RecoverResult{
		RunID:                   runID,
		ConflictPolicy:          normalized.ConflictPolicy,
		RecoveredA2AInFlight:    len(normalized.A2A.InFlight),
		ReplayedTerminalCommits: normalized.Replay.TerminalCommitCount,
	}, nil
}

func (c *Composer) Resume(ctx context.Context, req RecoverRequest) (RecoverResult, error) {
	return c.Recover(ctx, req)
}

func extractRecoveryA2AInFlightStates(snapshot scheduler.StoreSnapshot) []RecoveryA2AInFlightState {
	if len(snapshot.Tasks) == 0 {
		return nil
	}
	out := make([]RecoveryA2AInFlightState, 0, len(snapshot.Tasks))
	for i := range snapshot.Tasks {
		record := snapshot.Tasks[i]
		task := record.Task
		if strings.TrimSpace(task.PeerID) == "" {
			continue
		}
		if record.State != scheduler.TaskStateQueued && record.State != scheduler.TaskStateRunning {
			continue
		}
		entry := RecoveryA2AInFlightState{
			TaskID:     strings.TrimSpace(task.TaskID),
			AttemptID:  strings.TrimSpace(record.CurrentAttempt),
			WorkflowID: strings.TrimSpace(task.WorkflowID),
			TeamID:     strings.TrimSpace(task.TeamID),
			AgentID:    strings.TrimSpace(task.AgentID),
			PeerID:     strings.TrimSpace(task.PeerID),
			TaskState:  string(record.State),
		}
		entry.CorrelationKey = strings.TrimSpace(entry.TaskID + "|" + entry.AttemptID)
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool {
		left := strings.TrimSpace(out[i].TaskID) + "|" + strings.TrimSpace(out[i].AttemptID)
		right := strings.TrimSpace(out[j].TaskID) + "|" + strings.TrimSpace(out[j].AttemptID)
		return left < right
	})
	return out
}

func reconcileA2AInFlight(inflight []RecoveryA2AInFlightState, snapshot scheduler.StoreSnapshot) error {
	if len(inflight) == 0 {
		return nil
	}
	taskMap := make(map[string]scheduler.TaskRecord, len(snapshot.Tasks))
	for i := range snapshot.Tasks {
		taskID := strings.TrimSpace(snapshot.Tasks[i].Task.TaskID)
		if taskID == "" {
			continue
		}
		taskMap[taskID] = snapshot.Tasks[i]
	}
	for i := range inflight {
		state := inflight[i]
		taskID := strings.TrimSpace(state.TaskID)
		attemptID := strings.TrimSpace(state.AttemptID)
		record, ok := taskMap[taskID]
		if !ok {
			return fmt.Errorf("in-flight task %q does not exist in restored scheduler snapshot", taskID)
		}
		if strings.TrimSpace(record.Task.PeerID) != strings.TrimSpace(state.PeerID) {
			return fmt.Errorf("in-flight task %q peer_id mismatch", taskID)
		}
		if attemptID == "" {
			continue
		}
		if strings.TrimSpace(record.CurrentAttempt) == attemptID {
			continue
		}
		found := false
		for _, attempt := range record.Attempts {
			if strings.TrimSpace(attempt.AttemptID) == attemptID {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("in-flight task %q attempt_id %q is missing", taskID, attemptID)
		}
	}
	return nil
}

func hasSchedulerSnapshotState(snapshot scheduler.StoreSnapshot) bool {
	return len(snapshot.Tasks) > 0 || len(snapshot.Queue) > 0 || len(snapshot.TerminalCommits) > 0
}

func schedulerSnapshotsEqual(left, right scheduler.StoreSnapshot) bool {
	leftNorm := normalizeSchedulerStoreSnapshot(left)
	rightNorm := normalizeSchedulerStoreSnapshot(right)
	leftRaw, leftErr := json.Marshal(leftNorm)
	rightRaw, rightErr := json.Marshal(rightNorm)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftRaw) == string(rightRaw)
}

func normalizeSchedulerStoreSnapshot(snapshot scheduler.StoreSnapshot) scheduler.StoreSnapshot {
	out := snapshot
	sort.Slice(out.Tasks, func(i, j int) bool {
		return strings.TrimSpace(out.Tasks[i].Task.TaskID) < strings.TrimSpace(out.Tasks[j].Task.TaskID)
	})
	out.Queue = normalizeQueue(out.Queue)
	sort.Slice(out.TerminalCommits, func(i, j int) bool {
		left := strings.TrimSpace(out.TerminalCommits[i].TaskID) + "|" + strings.TrimSpace(out.TerminalCommits[i].AttemptID)
		right := strings.TrimSpace(out.TerminalCommits[j].TaskID) + "|" + strings.TrimSpace(out.TerminalCommits[j].AttemptID)
		return left < right
	})
	return out
}

func (c *Composer) emitRecoveryTimeline(
	ctx context.Context,
	runID string,
	reason string,
	status types.ActionStatus,
	taskID string,
	attemptID string,
) {
	if c == nil || c.handler == nil {
		return
	}
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return
	}
	sequence := c.now().UnixNano()
	if sequence <= 0 {
		sequence = time.Now().UnixNano()
	}
	payload := map[string]any{
		"phase":    string(types.ActionPhaseRun),
		"status":   string(status),
		"reason":   strings.TrimSpace(reason),
		"sequence": sequence,
		"run_id":   runID,
	}
	if strings.TrimSpace(taskID) != "" {
		payload["task_id"] = strings.TrimSpace(taskID)
	}
	if strings.TrimSpace(attemptID) != "" {
		payload["attempt_id"] = strings.TrimSpace(attemptID)
	}
	c.handler.OnEvent(ctx, types.Event{
		Version: types.EventSchemaVersionV1,
		Type:    types.EventTypeActionTimeline,
		RunID:   runID,
		Time:    c.now(),
		Payload: payload,
	})
}

func (c *Composer) markRecoveryRecovered(runID string, replayTotal int) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return
	}
	if replayTotal < 0 {
		replayTotal = 0
	}
	c.runMu.Lock()
	defer c.runMu.Unlock()
	stat := c.ensureRunStat(runID)
	stat.RecoveryEnabled = true
	stat.RecoveryRecovered = true
	stat.RecoveryReplayTotal = replayTotal
	stat.RecoveryConflict = false
	stat.RecoveryConflictCode = ""
}

func (c *Composer) markRecoveryConflict(runID, conflictCode string) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return
	}
	c.runMu.Lock()
	defer c.runMu.Unlock()
	stat := c.ensureRunStat(runID)
	stat.RecoveryEnabled = true
	stat.RecoveryConflict = true
	stat.RecoveryConflictCode = strings.TrimSpace(conflictCode)
}

func (c *Composer) markRecoveryFallback(runID, reason string) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return
	}
	c.runMu.Lock()
	defer c.runMu.Unlock()
	stat := c.ensureRunStat(runID)
	stat.RecoveryEnabled = true
	stat.RecoveryFallback = true
	stat.RecoveryFallbackReason = strings.TrimSpace(reason)
}
