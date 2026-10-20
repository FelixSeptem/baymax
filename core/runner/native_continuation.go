package runner

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

// NativeContinuationOptions binds executable dependencies explicitly. Clients,
// callbacks and credentials are never serialized. Revision must change whenever
// the model/tool/context implementation or policy changes incompatibly.
type NativeContinuationOptions struct {
	Store              types.DurableRunCheckpointStore
	OwnerNamespace     string
	DependencyProfile  string
	DependencyRevision string
	Binding            types.NativeCheckpointRef
	Approval           bool
	Budget             bool
	// BudgetOwner reads current source budget facts and authorizes a proposed
	// increase. Host revalidation alone cannot replenish a source budget.
	BudgetOwner func(context.Context, types.NativeCheckpointRef, types.DurableBudget, types.DurableBudget) (types.DurableBudget, error)
}

// ContinuationCapabilities exposes direct restore only when this Engine is
// explicitly bound to a durable source and its three pause producers are
// enabled. It never infers capability from the presence of DTOs alone.
func (e *Engine) ContinuationCapabilities(ctx context.Context) (types.ContinuationCapabilities, error) {
	if err := ctx.Err(); err != nil {
		return types.ContinuationCapabilities{}, err
	}
	if e == nil || e.nativeOptions == nil || e.nativeOptions.Store == nil || !e.nativeOptions.Store.CrossProcessRestore() {
		return types.ContinuationCapabilities{}, types.ErrCheckpointUnavailable
	}
	o := e.nativeOptions
	if o.OwnerNamespace == "" || o.DependencyProfile == "" || o.DependencyRevision == "" || !o.Approval || !o.Budget || o.BudgetOwner == nil {
		return types.ContinuationCapabilities{}, types.ErrCheckpointUnsupported
	}
	if err := e.validateNativeDependencies(types.RunRequest{}); err != nil {
		return types.ContinuationCapabilities{}, err
	}
	c := types.ContinuationCapabilities{ProfileVersion: types.NativeContinuationVersion, SourceNamespace: o.OwnerNamespace, ContractVersion: "baymax.runtime.contract.v1", ModuleRevision: "runner-native-continuation", APIVersion: "host_runtime_capability_manifest.v1", RetentionPolicy: "24h or original deadline", ExpiryPolicy: "earliest source deadline or 24h", CheckpointCreation: true, CrossProcessRestore: true, PauseReasons: []string{"input_required", "approval_required", "budget_exhausted"}, EvidenceReferences: map[string]string{"checkpoint_creation": "test:TestNativeContinuationCrossProcess", "cross_process_restore": "test:TestNativeContinuationCrossProcess", "input_required": "test:TestNativeContinuationCrossProcess/input_required", "approval_required": "test:TestNativeContinuationCrossProcess/approval_required", "budget_exhausted": "test:TestNativeContinuationCrossProcess/budget_exhausted"}}
	c.ModuleRevision = nativeBuildRevision()
	if err := c.Validate(); err != nil {
		return types.ContinuationCapabilities{}, types.ErrCheckpointUnsupported
	}
	return c, nil
}

func nativeBuildRevision() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" && setting.Value != "" && len(setting.Value) <= 256 {
				return setting.Value
			}
		}
		if info.Main.Version != "" && len(info.Main.Version) <= 256 {
			return info.Main.Version
		}
	}
	return "unversioned-development-build"
}

type nativeContinuationState struct {
	nativeOptions    *NativeContinuationOptions
	nativeMu         sync.Mutex
	nativeRefs       map[string]types.NativeCheckpointRef
	nativeLegacyRefs map[string]types.NativeCheckpointRef
}

func WithNativeContinuation(options NativeContinuationOptions) Option {
	return func(e *Engine) {
		o := options
		e.nativeOptions = &o
		e.nativeRefs = make(map[string]types.NativeCheckpointRef)
		e.nativeLegacyRefs = make(map[string]types.NativeCheckpointRef)
	}
}

type nativeContinuationKey struct{}

// This is private source state, independently bounded from the public ref.
// It intentionally contains model context/tool outcomes necessary to continue;
// applications must protect the store as execution data, not telemetry.
type nativeMaterial struct {
	Request      types.RunRequest        `json:"request"`
	Outcomes     []types.ToolCallOutcome `json:"outcomes,omitempty"`
	PendingCalls []types.ToolCall        `json:"pending_calls,omitempty"`
	Iteration    int                     `json:"iteration"`
	ToolCalls    int                     `json:"tool_calls"`
	Calls        []types.ToolCallSummary `json:"calls,omitempty"`
	Usage        types.TokenUsage        `json:"usage"`
	StartedAt    time.Time               `json:"started_at"`
	Deadline     time.Time               `json:"deadline,omitempty"`
	ConfigDigest string                  `json:"config_digest"`
}

type nativeExecution struct {
	nativeMaterial
	restored       bool
	approved       bool
	predecessor    *types.ResumeClaim
	successor      *types.PersistedRunCheckpoint
	budgetRevision string
}

func nativeExecutionFrom(ctx context.Context) *nativeExecution {
	x, _ := ctx.Value(nativeContinuationKey{}).(*nativeExecution)
	return x
}

func (e *Engine) nativeConfigDigest() string {
	b, _ := json.Marshal(e.runtimeEffectiveConfigSnapshot())
	d := sha256.Sum256(b)
	return hex.EncodeToString(d[:])
}

func (e *Engine) prepareNativeContinuation(ctx context.Context, req types.RunRequest) (context.Context, types.RunRequest, error) {
	if e.nativeOptions == nil || nativeExecutionFrom(ctx) != nil {
		return ctx, req, nil
	}
	o := e.nativeOptions
	if o.Store == nil || o.OwnerNamespace == "" || o.DependencyProfile == "" || o.DependencyRevision == "" {
		return ctx, req, types.ErrCheckpointUnavailable
	}
	// These owners have additional private state that this profile cannot
	// reconstruct. Reject rather than silently dropping ephemeral dependencies.
	if err := e.validateNativeDependencies(req); err != nil {
		return ctx, req, err
	}
	if ctx.Err() != nil {
		return ctx, req, ctx.Err()
	}
	if raw, err := json.Marshal(nativeMaterial{Request: req}); err != nil || len(raw) > types.NativeContinuationMaxMaterialBytes {
		return ctx, req, types.ErrCheckpointMalformed
	}
	if req.SessionID == "" {
		return ctx, req, types.ErrCheckpointMalformed
	}
	if req.RunID == "" {
		req.RunID = e.newRunID()
	}
	x := &nativeExecution{nativeMaterial: nativeMaterial{Request: req, StartedAt: e.now().UTC(), ConfigDigest: e.nativeConfigDigest()}, budgetRevision: o.DependencyRevision}
	if deadline, ok := ctx.Deadline(); ok {
		x.Deadline = deadline.UTC()
	}
	return context.WithValue(ctx, nativeContinuationKey{}, x), req, nil
}

func (e *Engine) validateNativeDependencies(req types.RunRequest) error {
	if req.Realtime != nil || req.HostContextProjection != nil || req.HostContextResolver != nil || e.skillLoader != nil || len(e.lifecycleHooks) != 0 || e.activeRunControl {
		return types.ErrCheckpointUnsupported
	}
	cfg := e.runtimeEffectiveConfigSnapshot()
	if cfg.Runtime.React.PlanNotebook.Enabled || e.runtimeMgr != nil && cfg.ContextAssembler.Enabled {
		return types.ErrCheckpointUnsupported
	}
	return nil
}

// NativeCheckpoint returns the source-published ref on this Engine. Persisted
// callers should retain RunResult.NativeCheckpoint for a future process.
func (e *Engine) NativeCheckpoint(ctx context.Context, runID string) (types.NativeCheckpointRef, error) {
	if err := ctx.Err(); err != nil {
		return types.NativeCheckpointRef{}, err
	}
	e.nativeMu.Lock()
	defer e.nativeMu.Unlock()
	ref, ok := e.nativeRefs[runID]
	if !ok {
		return ref, types.ErrCheckpointNotFound
	}
	return ref, nil
}

// CancelNativeCheckpoint durably fences an unclaimed continuation. Successful
// admission is a cancellation request fact; callers must not invent a terminal
// RunResult from it. A claimed attempt uses cooperative live Run cancellation.
func (e *Engine) CancelNativeCheckpoint(ctx context.Context, ref types.NativeCheckpointRef, h types.EventHandler) error {
	if e == nil || e.nativeOptions == nil || e.nativeOptions.Store == nil {
		return types.ErrCheckpointUnavailable
	}
	o := e.nativeOptions
	if ref.OwnerNamespace != o.OwnerNamespace || !nativeBindingsEqual(ref, o.Binding) {
		return types.ErrCheckpointForeign
	}
	p, err := o.Store.Load(ctx, ref)
	if err != nil {
		return err
	}
	_, err = o.Store.Control(ctx, types.CheckpointControl{Checkpoint: ref, Generation: p.Generation, Action: types.CheckpointControlCancel, Reason: "source.cancel_requested"})
	if err != nil {
		return err
	}
	e.emitNativeLifecycle(ctx, h, "run.checkpoint.cancel_requested", ref, "")
	return nil
}

func nativeCheckpointID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", types.ErrCheckpointUnavailable
	}
	return "continuation-" + hex.EncodeToString(b[:]), nil
}

func (e *Engine) persistNativePause(ctx context.Context, cp types.RunCheckpoint, outcomes []types.ToolCallOutcome, calls []types.ToolCall, reason types.PauseReason, stream bool) (*types.NativeCheckpointRef, error) {
	x := nativeExecutionFrom(ctx)
	if x == nil {
		return nil, nil
	}
	o := e.nativeOptions
	if o == nil || o.Store == nil {
		return nil, types.ErrCheckpointUnavailable
	}
	if len(outcomes)+len(calls)+len(x.Calls) > types.NativeContinuationMaxEntries {
		return nil, types.ErrCheckpointMalformed
	}
	x.Request.RunID, x.Request.SessionID = cp.RunID, cp.SessionID
	x.Iteration = cp.Iteration
	x.Outcomes, x.PendingCalls = outcomes, calls
	b, err := json.Marshal(x.nativeMaterial)
	if err != nil || len(b) > types.NativeContinuationMaxMaterialBytes {
		return nil, types.ErrCheckpointMalformed
	}
	id, err := nativeCheckpointID()
	if err != nil {
		return nil, err
	}
	now := e.now().UTC()
	expiry := now.Add(24 * time.Hour)
	if !x.Deadline.IsZero() && x.Deadline.Before(expiry) {
		expiry = x.Deadline
	}
	ref := o.Binding
	ref.SchemaVersion, ref.CheckpointID, ref.Digest = types.NativeContinuationVersion, id, "unsealed"
	ref.SourceComponent, ref.OwnerNamespace, ref.RunID, ref.SessionID = "runner", o.OwnerNamespace, cp.RunID, cp.SessionID
	ref.PauseReason, ref.SourcePosition, ref.CreatedAt, ref.ExpiresAt = reason, fmt.Sprintf("iteration:%d", cp.Iteration), now, expiry
	if x.predecessor != nil {
		ref.PredecessorCheckpointID = x.predecessor.Request.Checkpoint.CheckpointID
		ref.ResumeAttemptID = x.predecessor.AttemptID
	}
	policy := resolvePolicy(x.Request.Policy)
	policy = e.applyRuntimeDefaults(policy, x.Request.Policy, e.runtimeEffectiveConfigSnapshot())
	p, err := types.SealRunCheckpoint(types.PersistedRunCheckpoint{
		Ref: ref, Checkpoint: cp, Generation: 1, Material: b, DependencyProfile: o.DependencyProfile, DependencyRevision: o.DependencyRevision,
		Mode: map[bool]string{false: "run", true: "stream"}[stream], Iteration: cp.Iteration, CompletedEntries: len(x.Calls), PendingEntries: len(calls), Deadline: x.Deadline,
		Budget: types.DurableBudget{Revision: x.budgetRevision, Limit: int64(policy.MaxIterations), Consumed: int64(cp.Iteration), ToolLimit: int64(policy.ToolCallLimit), ToolConsumed: int64(x.ToolCalls), Deadline: x.Deadline},
	})
	if err != nil {
		return nil, err
	}
	if x.predecessor != nil {
		x.successor = &p
	} else if err := o.Store.Save(ctx, p); err != nil {
		return nil, err
	}
	if x.predecessor == nil {
		e.publishNativeRef(p)
	}
	return &p.Ref, nil
}

func (e *Engine) publishNativeRef(p types.PersistedRunCheckpoint) {
	e.nativeMu.Lock()
	e.nativeRefs[p.Ref.RunID] = p.Ref
	e.nativeLegacyRefs[p.Checkpoint.CheckpointID] = p.Ref
	e.nativeMu.Unlock()
}

// ResumeFromCheckpoint restores private source material only after validating
// the retained source and committing its atomic claim. It never invokes Run,
// retry, replay, or a host state reconstruction callback.
func (e *Engine) ResumeFromCheckpoint(ctx context.Context, request types.HostResumeRequest, h types.EventHandler, stream bool) (types.RunResult, error) {
	if e == nil || e.nativeOptions == nil || e.nativeOptions.Store == nil {
		return types.RunResult{}, types.ErrCheckpointUnavailable
	}
	if ctx.Err() != nil {
		return types.RunResult{}, ctx.Err()
	}
	request = request.Detached()
	if request.NativeCheckpoint == nil {
		return types.RunResult{}, types.ErrCheckpointMalformed
	}
	o := e.nativeOptions
	p, err := o.Store.Load(ctx, *request.NativeCheckpoint)
	if err != nil {
		return types.RunResult{}, err
	}
	if err = p.Validate(); err != nil {
		return types.RunResult{}, err
	}
	mode := map[bool]string{false: "run", true: "stream"}[stream]
	if p.Ref.SourceComponent != "runner" || p.Ref.OwnerNamespace != o.OwnerNamespace || p.DependencyProfile != o.DependencyProfile || p.DependencyRevision != o.DependencyRevision {
		return types.RunResult{}, types.ErrCheckpointForeign
	}
	if p.Mode != mode {
		return types.RunResult{}, types.ErrCheckpointConflict
	}
	d := request.Decision.Normalized()
	if d.Validate() != nil || d.RunID != p.Ref.RunID || d.SessionID != p.Ref.SessionID || d.CheckpointID != p.Ref.CheckpointID || d.CheckpointVersion != p.Ref.SchemaVersion || d.CheckpointDigest != p.Ref.Digest || p.Checkpoint.PendingAction == nil || d.Token != p.Checkpoint.PendingAction.Token {
		return types.RunResult{}, types.ErrCheckpointConflict
	}
	var material nativeMaterial
	if json.Unmarshal(p.Material, &material) != nil || material.Request.RunID != p.Ref.RunID || material.Request.SessionID != p.Ref.SessionID || material.Iteration != p.Iteration || material.ConfigDigest != e.nativeConfigDigest() {
		return types.RunResult{}, types.ErrCheckpointCorrupt
	}
	if err = e.validateNativeDependencies(material.Request); err != nil {
		return types.RunResult{}, err
	}
	if !nativeBindingsEqual(p.Ref, o.Binding) {
		return types.RunResult{}, types.ErrCheckpointForeign
	}
	budget := p.Budget
	if p.Ref.PauseReason == types.PauseReasonBudgetExhausted && d.Decision == types.DynamicActionDecisionConfirm {
		if !o.Budget || o.BudgetOwner == nil || request.NativeBudget == nil {
			return types.RunResult{}, types.ErrCheckpointUnsupported
		}
	}
	if p.Ref.PauseReason == types.PauseReasonApprovalRequired && !o.Approval {
		return types.RunResult{}, types.ErrCheckpointUnsupported
	}
	// Use the original compatibility projection for the existing host verifier.
	// A native ref is a binding, never authorization.
	if e.hostResumeRequired {
		hostDecision := d
		hostDecision.CheckpointID = p.Checkpoint.CheckpointID
		hostDecision.CheckpointVersion = p.Checkpoint.Version
		hostDecision.CheckpointDigest = p.Checkpoint.Digest
		if err = e.revalidateHostResume(ctx, types.HostResumeRequest{Version: types.HostResumeAdmissionVersion, Checkpoint: p.Checkpoint, Decision: hostDecision, NativeCheckpoint: &p.Ref, NativeBudget: request.NativeBudget}); err != nil {
			return types.RunResult{}, err
		}
	}
	resumeRequest := types.ResumeRequest{Checkpoint: p.Ref, Decision: d, Mode: mode, DependencyProfile: o.DependencyProfile, DependencyRevision: o.DependencyRevision, Generation: p.Generation, IdempotencyKey: d.IdempotencyKey, AuthorityDigest: p.Ref.AuthorityDigest, PlanDigest: p.Ref.PlanDigest, WriteScopeDigest: p.Ref.WriteScopeDigest, Budget: budget}
	if p.Ref.PauseReason == types.PauseReasonBudgetExhausted && d.Decision == types.DynamicActionDecisionConfirm && request.NativeBudget != nil {
		resumeRequest.ProposedBudget = request.NativeBudget
		if retained, readErr := o.Store.ReadResumeClaim(ctx, p.Ref); readErr == nil {
			expected := resumeRequest
			expected.Budget = retained.Request.Budget
			if retained.Request.IdentityDigest() == expected.IdentityDigest() {
				if retained.Result == nil {
					// ClaimResume distinguishes a live attempt from a recovered
					// attempt without authorizing another budget increase.
					_, claimErr := o.Store.ClaimResume(ctx, expected)
					return types.RunResult{}, claimErr
				}
				return projectNativeResumeResult(retained)
			}
			return types.RunResult{}, types.ErrCheckpointConflict
		} else if !errors.Is(readErr, types.ErrCheckpointNotFound) {
			return types.RunResult{}, readErr
		}
		budget, err = o.BudgetOwner(ctx, p.Ref, p.Budget, *request.NativeBudget)
		if err != nil {
			return types.RunResult{}, types.ErrCheckpointConflict
		}
		if budget.Revision == "" || budget.Revision == p.Budget.Revision || budget.Consumed != p.Budget.Consumed || budget.ToolConsumed != p.Budget.ToolConsumed || budget.Limit < p.Budget.Limit || budget.ToolLimit < p.Budget.ToolLimit || !budget.Deadline.Equal(p.Budget.Deadline) || budget.Limit <= budget.Consumed || budget.ToolLimit > 0 && budget.ToolConsumed+int64(len(material.PendingCalls)) > budget.ToolLimit {
			return types.RunResult{}, types.ErrCheckpointConflict
		}
		resumeRequest.Budget = budget
	}
	claim, err := o.Store.ClaimResume(ctx, resumeRequest)
	if err != nil {
		if errors.Is(err, types.ErrCheckpointInProgress) || errors.Is(err, types.ErrCheckpointRecoveryRequired) {
			e.emitNativeLifecycle(ctx, h, "run.checkpoint.recovery_required", p.Ref, claim.AttemptID)
		}
		return types.RunResult{}, err
	}
	if claim.Duplicate {
		return projectNativeResumeResult(claim)
	}
	e.emitNativeLifecycle(ctx, h, "run.checkpoint.claimed", p.Ref, claim.AttemptID)
	claim, err = o.Store.MarkRunning(ctx, claim)
	if err != nil {
		return types.RunResult{}, err
	}
	x := &nativeExecution{nativeMaterial: material, restored: true, approved: p.Ref.PauseReason == types.PauseReasonApprovalRequired, predecessor: &claim, budgetRevision: budget.Revision}
	policy := resolvePolicy(material.Request.Policy)
	policy = e.applyRuntimeDefaults(policy, material.Request.Policy, e.runtimeEffectiveConfigSnapshot())
	policy.MaxIterations = int(budget.Limit)
	policy.ToolCallLimit = int(budget.ToolLimit)
	x.Request.Policy = &policy
	resumeCtx := context.WithValue(ctx, nativeContinuationKey{}, x)
	resumeCtx = context.WithValue(resumeCtx, dynamicResumeIterationKey{}, material.Iteration)
	resumeCtx = context.WithValue(resumeCtx, dynamicPendingOutcomesKey{}, material.Outcomes)
	if !material.Deadline.IsZero() {
		var cancel context.CancelFunc
		resumeCtx, cancel = context.WithDeadline(resumeCtx, material.Deadline)
		defer cancel()
	}
	control, ok := e.ActiveRun(p.Ref.RunID)
	if !ok {
		control, _, err = e.beginActiveRun(context.WithoutCancel(resumeCtx), p.Ref.RunID, p.Ref.SessionID, stream)
		if err != nil {
			return types.RunResult{}, types.ErrCheckpointInProgress
		}
	}
	control.mu.Lock()
	if control.cancelAdmitted || control.closedFlag {
		control.mu.Unlock()
		return types.RunResult{}, types.ErrCheckpointRecoveryRequired
	}
	control.state = types.RunStateWorking
	control.mu.Unlock()
	resumeCtx = context.WithValue(resumeCtx, hostRunReservationKey{}, control)
	resumeCtx, cancelResume := context.WithCancel(resumeCtx)
	stop := context.AfterFunc(control.ctx, cancelResume)
	defer func() { stop(); cancelResume() }()
	e.emitNativeLifecycle(ctx, h, "run.checkpoint.restore_started", p.Ref, claim.AttemptID)
	buffer := &nativeEventBuffer{target: h}
	var result types.RunResult
	var executionErr error
	switch {
	case d.Decision != types.DynamicActionDecisionConfirm:
		result = types.RunResult{RunID: p.Ref.RunID, Iterations: p.Iteration, TerminalOutcome: &types.TerminalOutcome{RunID: p.Ref.RunID, SessionID: p.Ref.SessionID, State: types.RunStateCanceled, FailureFamily: types.FailureFamilyCanceled, Phase: types.ExecutionPhasePostStart, SourceReason: string(d.Decision)}}
	case stream:
		result, executionErr = e.streamReact(resumeCtx, x.Request, buffer)
	default:
		result, executionErr = e.run(resumeCtx, x.Request, buffer)
	}
	projection, encodeErr := json.Marshal(result)
	if encodeErr != nil {
		return types.RunResult{}, types.ErrCheckpointUnavailable
	}
	state := types.RunStateFailed
	if result.TerminalOutcome != nil {
		state = result.TerminalOutcome.State
	}
	code := ""
	if executionErr != nil {
		code = "native_continuation.execution_failed"
	}
	_, err = o.Store.CommitResumeResult(context.WithoutCancel(ctx), claim, types.ResumeResult{State: state, Projection: projection, Successor: x.successor, ErrorCode: code})
	if err != nil {
		return types.RunResult{RunID: p.Ref.RunID}, err
	}
	if x.successor != nil {
		e.publishNativeRef(*x.successor)
		e.emitNativeLifecycle(ctx, h, "run.checkpoint.persisted", x.successor.Ref, claim.AttemptID)
	} else {
		e.emitNativeLifecycle(ctx, h, "run.checkpoint.terminal", p.Ref, claim.AttemptID)
		control.mu.Lock()
		control.state = state
		control.mu.Unlock()
		e.finishActiveRun(control)
	}
	if old, ok := e.ActiveRun(p.Ref.RunID); ok && !old.dynamicActionPaused() {
		e.finishActiveRun(old)
	}
	buffer.flush(ctx)
	if d.Decision != types.DynamicActionDecisionConfirm {
		e.emit(ctx, h, types.Event{Version: types.EventSchemaVersionV1, Type: "run.finished", RunID: p.Ref.RunID, Iteration: p.Iteration, Time: e.now(), Payload: runFinishedPayload(result, "canceled", "", runFinishMeta{})})
	}
	return result, types.NewDynamicActionContinuationError(executionErr)
}

func projectNativeResumeResult(claim types.ResumeClaim) (types.RunResult, error) {
	if claim.Result == nil {
		return types.RunResult{}, types.ErrCheckpointInProgress
	}
	var result types.RunResult
	if json.Unmarshal(claim.Result.Projection, &result) != nil {
		return types.RunResult{}, types.ErrCheckpointCorrupt
	}
	if claim.Result.ErrorCode != "" {
		return result, types.NewDynamicActionContinuationError(errors.New(claim.Result.ErrorCode))
	}
	return detachedDynamicActionResult(result), nil
}

func (e *Engine) emitNativeLifecycle(ctx context.Context, h types.EventHandler, kind string, ref types.NativeCheckpointRef, attempt string) {
	e.emit(ctx, h, types.Event{Version: types.EventSchemaVersionV1, Type: kind, RunID: ref.RunID, Time: e.now(), Payload: map[string]any{"checkpoint_id": ref.CheckpointID, "checkpoint_digest": ref.Digest, "checkpoint_version": ref.SchemaVersion, "source_position": ref.SourcePosition, "pause_reason": string(ref.PauseReason), "resume_attempt_id": attempt, "predecessor_checkpoint_id": ref.PredecessorCheckpointID, "session_id": ref.SessionID, "parent_run_id": ref.ParentRunID, "host_correlation": ref.HostCorrelation, "authority_digest": ref.AuthorityDigest, "plan_digest": ref.PlanDigest, "write_scope_digest": ref.WriteScopeDigest, "occurrence_id": ref.OccurrenceID, "effect_namespace": ref.EffectNamespace}})
}

func nativeBindingsEqual(a, b types.NativeCheckpointRef) bool {
	return a.ParentRunID == b.ParentRunID && a.HostCorrelation == b.HostCorrelation && a.AuthorityDigest == b.AuthorityDigest && a.PlanDigest == b.PlanDigest && a.WriteScopeDigest == b.WriteScopeDigest && a.OccurrenceID == b.OccurrenceID && a.EffectNamespace == b.EffectNamespace
}

type nativeEventBuffer struct {
	target types.EventHandler
	events []types.Event
}

func (b *nativeEventBuffer) OnEvent(ctx context.Context, event types.Event) {
	if event.Type == "run.started" {
		return
	}
	if event.Type == "run.finished" || event.Type == "run.input_required" {
		b.events = append(b.events, event)
		return
	}
	if b.target != nil {
		b.target.OnEvent(ctx, event)
	}
}
func (b *nativeEventBuffer) flush(ctx context.Context) {
	if b.target != nil {
		for _, event := range b.events {
			b.target.OnEvent(ctx, event)
		}
	}
}

// pauseNativeBoundary is produced by the runner safe point, not by changing a
// tool-result action kind. Pending calls have never reached the dispatcher.
func (e *Engine) pauseNativeBoundary(ctx context.Context, req types.RunRequest, h types.EventHandler, control *ActiveRunControl, reason types.PauseReason, calls []types.ToolCall, outcomes []types.ToolCallOutcome, iteration int, stream bool, total int, merged []types.ToolCallSummary, usage types.TokenUsage) (types.RunResult, error) {
	x := nativeExecutionFrom(ctx)
	if x == nil {
		return types.RunResult{}, types.ErrCheckpointUnsupported
	}
	x.Request = req
	x.ToolCalls = total
	x.Calls = merged
	x.Usage = usage
	id, err := nativeCheckpointID()
	if err != nil {
		return types.RunResult{}, err
	}
	action := types.DynamicActionReference{Token: id, Kind: "native." + string(reason), Resumable: true, RunID: req.RunID, SessionID: req.SessionID, Iteration: iteration, CallID: id, Source: "runner", Digest: id}
	cp := types.RunCheckpoint{Version: types.DynamicActionResumeProtocolVersion, CheckpointID: id, RunID: req.RunID, SessionID: req.SessionID, State: types.RunStateInputRequired, Mode: map[bool]string{false: "run", true: "stream"}[stream], Iteration: iteration, PendingAction: &action, Digest: id, CreatedAt: e.now().UTC()}
	ref, err := e.persistNativePause(ctx, cp, outcomes, calls, reason, stream)
	if err != nil {
		return types.RunResult{RunID: req.RunID}, err
	}
	if x.predecessor == nil {
		e.emitNativeLifecycle(ctx, h, "run.checkpoint.persisted", *ref, "")
	}
	if control == nil {
		control, _, err = e.beginActiveRun(context.WithoutCancel(ctx), req.RunID, req.SessionID, stream)
		if err != nil {
			return types.RunResult{}, err
		}
	}
	control.mu.Lock()
	control.state = types.RunStateInputRequired
	control.mu.Unlock()
	result := types.RunResult{NativeCheckpoint: ref, RunID: req.RunID, Iterations: iteration, ToolCalls: merged, TokenUsage: usage, TerminalOutcome: &types.TerminalOutcome{RunID: req.RunID, SessionID: req.SessionID, State: types.RunStateInputRequired, FailureFamily: types.FailureFamilyNone, Phase: types.ExecutionPhasePostStart, SourceReason: string(reason), Resumable: true, PendingAction: &action, CheckpointID: cp.CheckpointID, CheckpointVersion: cp.Version, CheckpointDigest: cp.Digest}}
	e.emit(ctx, h, types.Event{Version: types.EventSchemaVersionV1, Type: "run.input_required", RunID: req.RunID, Iteration: iteration, Time: e.now(), Payload: map[string]any{"reason": string(reason), "checkpoint_id": ref.CheckpointID, "checkpoint_digest": ref.Digest}})
	e.emit(ctx, h, types.Event{Version: types.EventSchemaVersionV1, Type: "run.finished", RunID: req.RunID, Iteration: iteration, Time: e.now(), Payload: map[string]any{"state": "input_required", "reason": string(reason), "checkpoint_id": ref.CheckpointID}})
	return result, nil
}

type nativeApprovalPause struct{}

func (nativeApprovalPause) Error() string { return "native approval required" }
