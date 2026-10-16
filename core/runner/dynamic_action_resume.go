package runner

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
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
	ErrDynamicActionInProgress   = errors.New("dynamic action resume is in progress")
)

const dynamicActionCheckpointMaxAge = 24 * time.Hour

// dynamicActionState keeps checkpoint ownership and its admission guard together.
type dynamicActionState struct {
	dynamicActionMu    sync.Mutex
	dynamicCheckpoints map[string]dynamicActionCheckpoint
	dynamicGeneration  uint64
	hostResumeRequired bool
	hostResumeVerifier types.HostResumeVerifier
}

// WithHostResumeVerifier enables host revalidation for every dynamic-action
// decision on this Engine. Nil explicitly enables a fail-closed guard.
func WithHostResumeVerifier(verifier types.HostResumeVerifier) Option {
	return func(e *Engine) {
		e.hostResumeRequired = true
		e.hostResumeVerifier = verifier
	}
}

type dynamicActionCheckpoint struct {
	checkpoint types.RunCheckpoint
	request    types.RunRequest
	outcomes   []types.ToolCallOutcome
	stream     bool
	resumed    bool
	decision   types.DynamicActionDecision
	result     types.RunResult
	completed  bool
	resultErr  error
	generation uint64
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
	e.dynamicActionMu.Lock()
	e.dynamicGeneration++
	generation := e.dynamicGeneration
	e.dynamicActionMu.Unlock()
	checkpoint := types.RunCheckpoint{
		Version:      types.DynamicActionResumeProtocolVersion,
		CheckpointID: fmt.Sprintf("dynamic-action:%d", generation),
		RunID:        ref.RunID, SessionID: ref.SessionID, State: types.RunStateInputRequired,
		Mode: map[bool]string{true: "stream", false: "run"}[stream], Iteration: iteration,
		PendingAction: &ref, Digest: ref.Digest, CreatedAt: e.now().UTC(),
	}
	checkpoint = (types.HostResumeRequest{Checkpoint: checkpoint}).Detached().Checkpoint
	if err := checkpoint.Validate(); err != nil {
		return types.RunResult{RunID: ref.RunID, Iterations: iteration, Error: classified(types.ErrTool, err.Error(), false)}
	}
	e.dynamicActionMu.Lock()
	e.dynamicCheckpoints[ref.RunID] = dynamicActionCheckpoint{checkpoint: checkpoint, request: req, outcomes: append([]types.ToolCallOutcome(nil), outcomes...), stream: stream, generation: generation}
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
	decision = decision.Normalized()
	e.dynamicActionMu.Lock()
	cp, duplicate, err := e.dynamicResumePreflight(decision, stream)
	e.dynamicActionMu.Unlock()
	if err != nil || duplicate {
		if duplicate {
			err = types.NewDynamicActionContinuationError(err)
		}
		return detachedDynamicActionResult(cp.result), err
	}
	if ctx.Err() != nil {
		return types.RunResult{}, types.HostResumeRevalidationFailed
	}
	if e.hostResumeRequired {
		r := types.HostResumeRequest{Version: types.HostResumeAdmissionVersion, Checkpoint: cp.checkpoint, Decision: decision}.Detached()
		if err := e.revalidateHostResume(ctx, r); err != nil {
			return types.RunResult{}, err
		}
	}
	// Host code has returned. Recheck the source generation and cancellation
	// before the one decision commit; never keep source locks across host code.
	e.dynamicActionMu.Lock()
	latest, duplicate, err := e.dynamicResumePreflight(decision, stream)
	if err != nil || duplicate {
		e.dynamicActionMu.Unlock()
		if duplicate {
			err = types.NewDynamicActionContinuationError(err)
		}
		return detachedDynamicActionResult(latest.result), err
	}
	if latest.generation != cp.generation {
		e.dynamicActionMu.Unlock()
		return types.RunResult{}, ErrDynamicActionStale
	}
	cp = latest
	control, exists := e.ActiveRun(decision.RunID)
	if !exists {
		e.dynamicActionMu.Unlock()
		return types.RunResult{}, ErrDynamicActionTerminal
	}
	// CancelRun linearizes under this same control mutex. A cancellation that
	// wins first cannot be bypassed by an old successful host check.
	control.mu.Lock()
	if control.cancelAdmitted || control.closedFlag || control.ctx.Err() != nil || control.state != types.RunStateInputRequired {
		control.mu.Unlock()
		e.dynamicActionMu.Unlock()
		return types.RunResult{}, ErrDynamicActionTerminal
	}
	if ctx.Err() != nil {
		control.mu.Unlock()
		e.dynamicActionMu.Unlock()
		return types.RunResult{}, types.HostResumeRevalidationFailed
	}
	cp.resumed, cp.decision = true, decision
	if decision.Decision != types.DynamicActionDecisionConfirm {
		cp.result = types.RunResult{RunID: cp.checkpoint.RunID, Iterations: cp.checkpoint.Iteration, TerminalOutcome: &types.TerminalOutcome{RunID: cp.checkpoint.RunID, SessionID: cp.checkpoint.SessionID, State: types.RunStateCanceled, FailureFamily: types.FailureFamilyCanceled, Phase: types.ExecutionPhasePostStart, SourceReason: string(decision.Decision)}}
		cp.checkpoint.State = types.RunStateCanceled
		cp.completed = true
		control.state = types.RunStateCanceled
		e.dynamicCheckpoints[decision.RunID] = cp
		control.mu.Unlock()
		e.dynamicActionMu.Unlock()
		e.emitDynamicActionResolved(ctx, h, cp, decision)
		e.emitDynamicActionCanceled(ctx, h, cp, decision)
		e.finishActiveRun(control)
		return detachedDynamicActionResult(cp.result), nil
	}
	e.dynamicCheckpoints[decision.RunID] = cp
	control.state = types.RunStateWorking
	control.mu.Unlock()
	request, outcomes := cp.request, append([]types.ToolCallOutcome(nil), cp.outcomes...)
	e.dynamicActionMu.Unlock()
	e.emitDynamicActionResolved(ctx, h, cp, decision)
	request.RunID = decision.RunID
	resumeCtx := context.WithValue(ctx, dynamicResumeIterationKey{}, cp.checkpoint.Iteration)
	resumeCtx = context.WithValue(resumeCtx, hostRunReservationKey{}, control)
	// Preserve both request cancellation and source control cancellation while
	// keeping this cooperative continuation in the calling goroutine.
	resumeCtx, cancel := context.WithCancel(resumeCtx)
	stop := context.AfterFunc(control.ctx, cancel)
	defer func() { stop(); cancel() }()
	if stream {
		cp.result, cp.resultErr = e.streamReactWithPending(resumeCtx, request, outcomes, h)
	} else {
		cp.result, cp.resultErr = e.runWithPending(resumeCtx, request, outcomes, h)
	}
	e.dynamicActionMu.Lock()
	latest = e.dynamicCheckpoints[decision.RunID]
	if latest.generation == cp.generation {
		latest.result, latest.resultErr, latest.completed = detachedDynamicActionResult(cp.result), cp.resultErr, true
		if cp.result.TerminalOutcome != nil {
			latest.checkpoint.State = cp.result.TerminalOutcome.State
		}
		e.dynamicCheckpoints[decision.RunID] = latest
	}
	e.dynamicActionMu.Unlock()
	return cp.result, types.NewDynamicActionContinuationError(cp.resultErr)
}

// dynamicResumePreflight requires dynamicActionMu. A duplicate is identified
// before terminal/expiry checks, but only with the entire normalized identity.
func (e *Engine) dynamicResumePreflight(d types.DynamicActionDecision, stream bool) (dynamicActionCheckpoint, bool, error) {
	cp, ok := e.dynamicCheckpoints[d.RunID]
	if !ok {
		return cp, false, ErrDynamicActionUnknown
	}
	if cp.resumed && cp.decision == d && cp.stream == stream {
		if !cp.completed {
			return dynamicActionCheckpoint{}, false, ErrDynamicActionInProgress
		}
		return cp, true, cp.resultErr
	}
	if cp.stream != stream || cp.checkpoint.SessionID != d.SessionID || cp.checkpoint.CheckpointID != d.CheckpointID || cp.checkpoint.Version != d.CheckpointVersion || cp.checkpoint.Digest != d.CheckpointDigest || cp.checkpoint.PendingAction == nil || cp.checkpoint.PendingAction.Token != d.Token {
		return dynamicActionCheckpoint{}, false, ErrDynamicActionStale
	}
	if !cp.checkpoint.CreatedAt.IsZero() && e.now().UTC().Sub(cp.checkpoint.CreatedAt) > dynamicActionCheckpointMaxAge {
		return dynamicActionCheckpoint{}, false, ErrDynamicActionExpired
	}
	if cp.checkpoint.State != types.RunStateInputRequired {
		return dynamicActionCheckpoint{}, false, ErrDynamicActionTerminal
	}
	if cp.resumed {
		return dynamicActionCheckpoint{}, false, ErrDynamicActionConflict
	}
	return cp, false, nil
}

func (e *Engine) revalidateHostResume(ctx context.Context, r types.HostResumeRequest) (err error) {
	defer func() {
		if recover() != nil {
			err = types.HostResumeRevalidationFailed
		}
	}()
	if err = r.Validate(); err != nil {
		return err
	}
	if e.hostResumeVerifier == nil {
		return types.HostResumeRevalidationFailed
	}
	x, callbackErr := e.hostResumeVerifier.RevalidateHostResume(ctx, r.Detached())
	if callbackErr != nil || ctx.Err() != nil {
		return types.HostResumeRevalidationFailed
	}
	return types.ValidateHostResumeResult(r, x)
}

func (e *Engine) emitDynamicActionResolved(ctx context.Context, h types.EventHandler, cp dynamicActionCheckpoint, decision types.DynamicActionDecision) {
	if h == nil {
		return
	}
	e.emit(ctx, h, types.Event{Version: types.EventSchemaVersionV1, Type: "run.dynamic_action.resolved", RunID: cp.checkpoint.RunID, Iteration: cp.checkpoint.Iteration, Time: e.now(), Payload: map[string]any{
		"decision": string(decision.Decision), "checkpoint_id": cp.checkpoint.CheckpointID, "checkpoint_version": cp.checkpoint.Version,
		"checkpoint_digest": cp.checkpoint.Digest, "dynamic_action_resume_attempt": 1, "dynamic_action_resume_admission": "accepted",
	}})
}

func (e *Engine) emitDynamicActionCanceled(ctx context.Context, h types.EventHandler, cp dynamicActionCheckpoint, decision types.DynamicActionDecision) {
	if h == nil {
		return
	}
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

type dynamicResumeIterationKey struct{}
