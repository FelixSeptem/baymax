package runner

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

var (
	ErrDynamicActionUnknown      = errors.New("dynamic action checkpoint not found")
	ErrDynamicActionStale        = errors.New("dynamic action checkpoint is stale")
	ErrDynamicActionConflict     = errors.New("dynamic action resume conflicts with an existing admission")
	ErrDynamicActionTerminal     = errors.New("dynamic action checkpoint belongs to a terminal run")
	ErrDynamicActionExpired      = errors.New("dynamic action checkpoint has expired")
	ErrDynamicActionNotResumable = errors.New("dynamic action is not resumable")
)

const dynamicActionCheckpointMaxAge = 24 * time.Hour

type dynamicActionCheckpoint struct {
	checkpoint types.RunCheckpoint
	request    types.RunRequest
	outcomes   []types.ToolCallOutcome
	stream     bool
	resumed    bool
	decision   types.DynamicActionDecision
	result     types.RunResult
}

func normalizeDynamicAction(outcome types.ToolCallOutcome, runID, sessionID string, iteration int) (types.DynamicActionReference, error) {
	if outcome.Result.PendingAction == nil {
		return types.DynamicActionReference{}, nil
	}
	ref := outcome.Result.PendingAction.Normalized()
	if err := ref.Validate(); err != nil {
		return types.DynamicActionReference{}, err
	}
	if !ref.Resumable {
		return ref, ErrDynamicActionNotResumable
	}
	if ref.RunID != strings.TrimSpace(runID) || ref.SessionID != strings.TrimSpace(sessionID) || ref.Iteration != iteration || ref.CallID != strings.TrimSpace(outcome.CallID) {
		return types.DynamicActionReference{}, fmt.Errorf("dynamic action correlation mismatch")
	}
	return ref, nil
}

func (e *Engine) dynamicActionFromOutcomes(outcomes []types.ToolCallOutcome, runID, sessionID string, iteration int) (types.DynamicActionReference, bool, error) {
	for _, outcome := range outcomes {
		if outcome.Result.PendingAction == nil {
			continue
		}
		ref, err := normalizeDynamicAction(outcome, runID, sessionID, iteration)
		if err != nil {
			if errors.Is(err, ErrDynamicActionNotResumable) {
				return ref, false, nil
			}
			return types.DynamicActionReference{}, false, err
		}
		return ref, true, nil
	}
	return types.DynamicActionReference{}, false, nil
}

func (e *Engine) pauseForDynamicAction(ctx context.Context, req types.RunRequest, h types.EventHandler, control *ActiveRunControl, ref types.DynamicActionReference, outcomes []types.ToolCallOutcome, iteration int, stream bool, seq *int64, gateStats *actionGateStats) types.RunResult {
	if control == nil {
		// Plain Run/Stream deliberately do not expose host controls by default.
		// Once a tool registers a dynamic action, retain a dedicated control for
		// the paused source Run so same-Run resume can be admitted without making
		// ordinary runs subject to host-control limits.
		control, _, _ = e.beginActiveRun(context.WithoutCancel(ctx), ref.RunID, ref.SessionID, stream)
	}
	checkpoint := types.RunCheckpoint{
		Version:      types.DynamicActionResumeProtocolVersion,
		CheckpointID: fmt.Sprintf("%s:%s", ref.RunID, ref.CallID),
		RunID:        ref.RunID, SessionID: ref.SessionID, State: types.RunStateInputRequired,
		Mode: map[bool]string{true: "stream", false: "run"}[stream], Iteration: iteration,
		PendingAction: &ref, Digest: ref.Digest, CreatedAt: e.now().UTC(),
	}
	if err := checkpoint.Validate(); err != nil {
		return types.RunResult{RunID: ref.RunID, Iterations: iteration, Error: classified(types.ErrTool, err.Error(), false)}
	}
	e.dynamicActionMu.Lock()
	e.dynamicCheckpoints[ref.RunID] = dynamicActionCheckpoint{checkpoint: checkpoint, request: req, outcomes: append([]types.ToolCallOutcome(nil), outcomes...), stream: stream}
	e.dynamicActionMu.Unlock()
	if control != nil {
		control.mu.Lock()
		control.state = types.RunStateInputRequired
		control.mu.Unlock()
	}
	if gateStats != nil {
		gateStats.Checks++
	}
	e.emitTimeline(ctx, h, ref.RunID, iteration, seq, types.ActionPhaseRun, types.ActionStatusPending, "dynamic_action.input_required_same_run")
	e.emit(ctx, h, types.Event{Version: types.EventSchemaVersionV1, Type: "run.input_required", RunID: ref.RunID, Iteration: iteration, Time: e.now(), Payload: map[string]any{
		"reason": "dynamic_action", "action_kind": ref.Kind, "action_token": ref.Token, "checkpoint_id": checkpoint.CheckpointID, "checkpoint_version": checkpoint.Version, "checkpoint_digest": checkpoint.Digest,
	}})
	result := types.RunResult{RunID: ref.RunID, Iterations: iteration, ToolCalls: nil, TerminalOutcome: &types.TerminalOutcome{
		RunID: ref.RunID, SessionID: ref.SessionID, State: types.RunStateInputRequired, FailureFamily: types.FailureFamilyNone, Phase: types.ExecutionPhasePostStart, Resumable: true,
		PendingAction: &ref, CheckpointID: checkpoint.CheckpointID, CheckpointVersion: checkpoint.Version, CheckpointDigest: checkpoint.Digest,
	}}
	e.emit(ctx, h, types.Event{Version: types.EventSchemaVersionV1, Type: "run.finished", RunID: ref.RunID, Iteration: iteration, Time: e.now(), Payload: runFinishedPayload(result, "input_required", "", runFinishMeta{
		GateChecks: gateStatsValue(gateStats), DynamicActionCount: 1, DynamicActionReferenceDigest: ref.Digest,
		DynamicActionCheckpointID: checkpoint.CheckpointID, DynamicActionCheckpointVersion: checkpoint.Version,
		DynamicActionCheckpointDigest: checkpoint.Digest, DynamicActionPauseReason: "dynamic_action.input_required_same_run",
	})})
	return result
}

func gateStatsValue(stats *actionGateStats) int {
	if stats == nil {
		return 0
	}
	return stats.Checks
}

func (c *ActiveRunControl) dynamicActionPaused() bool {
	if c == nil {
		return false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state == types.RunStateInputRequired && !c.closedFlag
}

// ResumeDynamicAction admits a decision against a bounded checkpoint and
// continues the original Run/Stream correlation. It deliberately does not
// route through RetryRun or PromoteRuntimeFollowUp.
func (e *Engine) ResumeDynamicAction(ctx context.Context, decision types.DynamicActionDecision, h types.EventHandler, stream bool) (types.RunResult, error) {
	if e == nil {
		return types.RunResult{}, ErrDynamicActionUnknown
	}
	if err := decision.Validate(); err != nil {
		return types.RunResult{}, err
	}
	e.dynamicActionMu.Lock()
	cp, ok := e.dynamicCheckpoints[strings.TrimSpace(decision.RunID)]
	if !ok {
		e.dynamicActionMu.Unlock()
		return types.RunResult{}, ErrDynamicActionUnknown
	}
	if cp.resumed && cp.decision.IdempotencyKey == decision.IdempotencyKey && cp.decision.Decision == decision.Decision {
		result := cp.result
		e.dynamicActionMu.Unlock()
		return result, nil
	}
	if !cp.checkpoint.CreatedAt.IsZero() && e.now().UTC().Sub(cp.checkpoint.CreatedAt) > dynamicActionCheckpointMaxAge {
		e.dynamicActionMu.Unlock()
		return types.RunResult{}, ErrDynamicActionExpired
	}
	if cp.checkpoint.State != types.RunStateInputRequired {
		e.dynamicActionMu.Unlock()
		return types.RunResult{}, ErrDynamicActionTerminal
	}
	if cp.stream != stream || cp.checkpoint.SessionID != strings.TrimSpace(decision.SessionID) || cp.checkpoint.CheckpointID != strings.TrimSpace(decision.CheckpointID) || cp.checkpoint.Version != strings.TrimSpace(decision.CheckpointVersion) || cp.checkpoint.Digest != strings.TrimSpace(decision.CheckpointDigest) || cp.checkpoint.PendingAction == nil || cp.checkpoint.PendingAction.Token != strings.TrimSpace(decision.Token) {
		e.dynamicActionMu.Unlock()
		return types.RunResult{}, ErrDynamicActionStale
	}
	if cp.resumed {
		e.dynamicActionMu.Unlock()
		return types.RunResult{}, ErrDynamicActionConflict
	}
	cp.resumed, cp.decision = true, decision
	if decision.Decision != types.DynamicActionDecisionConfirm {
		cp.result = types.RunResult{RunID: cp.checkpoint.RunID, Iterations: cp.checkpoint.Iteration, TerminalOutcome: &types.TerminalOutcome{RunID: cp.checkpoint.RunID, SessionID: cp.checkpoint.SessionID, State: types.RunStateCanceled, FailureFamily: types.FailureFamilyCanceled, Phase: types.ExecutionPhasePostStart, SourceReason: string(decision.Decision)}}
		cp.checkpoint.State = types.RunStateCanceled
		e.dynamicCheckpoints[decision.RunID] = cp
		e.dynamicActionMu.Unlock()
		e.emitDynamicActionResolution(ctx, h, cp, decision)
		if ctrl, exists := e.ActiveRun(decision.RunID); exists {
			e.finishActiveRun(ctrl)
		}
		return cp.result, nil
	}
	e.dynamicCheckpoints[decision.RunID] = cp
	request, outcomes, control := cp.request, append([]types.ToolCallOutcome(nil), cp.outcomes...), (*ActiveRunControl)(nil)
	if ctrl, exists := e.ActiveRun(decision.RunID); exists {
		control = ctrl
	}
	e.dynamicActionMu.Unlock()
	request.RunID = decision.RunID
	e.dynamicActionMu.Lock()
	// The first resumed model step consumes the checkpointed tool results.
	e.dynamicCheckpoints[decision.RunID] = cp
	e.dynamicActionMu.Unlock()
	resumeCtx := ctx
	if control != nil {
		control.mu.Lock()
		control.state = types.RunStateWorking
		control.mu.Unlock()
		resumeCtx = context.WithValue(resumeCtx, hostRunReservationKey{}, control)
	}
	// Store the continuation inputs for the normal loop to consume once.
	e.dynamicActionMu.Lock()
	cp.outcomes = outcomes
	e.dynamicCheckpoints[decision.RunID] = cp
	e.dynamicActionMu.Unlock()
	if stream {
		cp.result, _ = e.streamReactWithPending(resumeCtx, request, outcomes, h)
	} else {
		cp.result, _ = e.runWithPending(resumeCtx, request, outcomes, h)
	}
	e.dynamicActionMu.Lock()
	latest := e.dynamicCheckpoints[decision.RunID]
	latest.result = cp.result
	if cp.result.TerminalOutcome != nil {
		latest.checkpoint.State = cp.result.TerminalOutcome.State
	}
	e.dynamicCheckpoints[decision.RunID] = latest
	e.dynamicActionMu.Unlock()
	return cp.result, nil
}

func (e *Engine) emitDynamicActionResolution(ctx context.Context, h types.EventHandler, cp dynamicActionCheckpoint, decision types.DynamicActionDecision) {
	if h == nil {
		return
	}
	e.emit(ctx, h, types.Event{Version: types.EventSchemaVersionV1, Type: "run.dynamic_action.resolved", RunID: cp.checkpoint.RunID, Iteration: cp.checkpoint.Iteration, Time: e.now(), Payload: map[string]any{
		"decision": string(decision.Decision), "checkpoint_id": cp.checkpoint.CheckpointID, "checkpoint_version": cp.checkpoint.Version,
		"checkpoint_digest": cp.checkpoint.Digest, "dynamic_action_resume_attempt": 1, "dynamic_action_resume_admission": "accepted",
	}})
	e.emit(ctx, h, types.Event{Version: types.EventSchemaVersionV1, Type: "run.finished", RunID: cp.checkpoint.RunID, Iteration: cp.checkpoint.Iteration, Time: e.now(), Payload: map[string]any{
		"state": string(types.RunStateCanceled), "reason_code": string(decision.Decision), "gate_checks": 1,
		"dynamic_action_count": 1, "dynamic_action_checkpoint_id": cp.checkpoint.CheckpointID, "dynamic_action_checkpoint_version": cp.checkpoint.Version,
		"dynamic_action_checkpoint_digest": cp.checkpoint.Digest, "dynamic_action_resume_attempt": 1, "dynamic_action_resume_admission": "accepted",
	}})
}

func (e *Engine) runWithPending(ctx context.Context, req types.RunRequest, outcomes []types.ToolCallOutcome, h types.EventHandler) (types.RunResult, error) {
	// The normal Run loop consumes the stored checkpoint outcomes at its first
	// model boundary. Keeping this helper separate makes the resume path explicit.
	return e.Run(context.WithValue(ctx, dynamicPendingOutcomesKey{}, outcomes), req, h)
}

func (e *Engine) streamReactWithPending(ctx context.Context, req types.RunRequest, outcomes []types.ToolCallOutcome, h types.EventHandler) (types.RunResult, error) {
	return e.streamReact(context.WithValue(ctx, dynamicPendingOutcomesKey{}, outcomes), req, h)
}

type dynamicPendingOutcomesKey struct{}
