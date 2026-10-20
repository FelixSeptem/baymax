package runner

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

// ActiveRunSnapshot is a bounded, source-owned view of one active Run.
type ActiveRunSnapshot struct {
	RunID     string         `json:"run_id"`
	SessionID string         `json:"session_id,omitempty"`
	State     types.RunState `json:"state"`
	Stream    bool           `json:"stream"`
	StartedAt time.Time      `json:"started_at"`
}

type activeRunState struct {
	activeRunMu                sync.RWMutex
	activeRuns                 map[string]*ActiveRunControl
	activeRunControl           bool
	activeRunLimit             int
	activeRunIngressBuffer     int
	runtimeInputFollowUpBuffer int
}

// ActiveRunControl exposes cancellation and bounded Realtime ingress for one Run.
// The control is owned by its Engine and is removed when the Run terminates.
type ActiveRunControl struct {
	engine             *Engine
	runID              string
	sessionID          string
	stream             bool
	startedAt          time.Time
	ctx                context.Context
	cancel             context.CancelFunc
	ingress            chan types.RealtimeEventEnvelope
	closed             chan struct{}
	closeOnce          sync.Once
	mu                 sync.RWMutex
	state              types.RunState
	runtime            *realtimeSessionRuntime
	cancelAdmitted     bool
	closedFlag         bool
	inputMu            sync.Mutex
	pendingSteering    *types.RuntimeInputEnvelope
	followUps          []types.RuntimeInputEnvelope
	inputSeen          map[string]struct{}
	followUpLimit      int
	runtimeInputClosed bool
}

type hostRunReservationKey struct{}

type hostRunExecution struct {
	mu      sync.Mutex
	engine  *Engine
	request types.RunRequest
	control *ActiveRunControl
	used    bool
}

type retryCausationHandler struct {
	next        types.EventHandler
	causationID string
}

func (h retryCausationHandler) OnEvent(ctx context.Context, ev types.Event) {
	if ev.Type == "run.finished" && ev.Payload != nil {
		ev.Payload["terminal_causation_id"] = h.causationID
		if outcome, ok := ev.Payload["terminal_outcome"].(map[string]any); ok {
			outcome["causation_id"] = h.causationID
		}
	}
	h.next.OnEvent(ctx, ev)
}

// AdmitHostRun reserves the source-owned active control before execution. The
// returned execution remains dormant until the host has delivered admission.
func (e *Engine) AdmitHostRun(ctx context.Context, req types.RunRequest, mode types.HostRunExecutionMode) (types.HostRunStartAdmission, error) {
	stream := false
	switch mode {
	case types.HostRunExecutionModeRun:
	case types.HostRunExecutionModeStream:
		stream = true
	default:
		return types.HostRunStartAdmission{Status: types.HostAdmissionStatusRejected, ReasonCode: types.HostReasonInvalidEnvelope}, fmt.Errorf("unsupported host run execution mode %q", mode)
	}
	if strings.TrimSpace(req.RunID) == "" {
		req.RunID = e.newRunID()
	}
	control, _, err := e.beginActiveRun(context.WithoutCancel(ctx), req.RunID, req.SessionID, stream)
	if err != nil {
		status := types.HostAdmissionStatusRejected
		if errors.Is(err, ErrActiveRunDuplicate) {
			status = types.HostAdmissionStatusDuplicate
		}
		return types.HostRunStartAdmission{Status: status, ReasonCode: err.Error(), RunID: req.RunID}, nil
	}
	return types.HostRunStartAdmission{
		Status:    types.HostAdmissionStatusAccepted,
		RunID:     req.RunID,
		Execution: &hostRunExecution{engine: e, request: req, control: control},
	}, nil
}

func (x *hostRunExecution) Execute(h types.EventHandler) (types.RunResult, error) {
	x.mu.Lock()
	if x.used {
		x.mu.Unlock()
		return types.RunResult{RunID: x.request.RunID}, ErrActiveRunClosed
	}
	x.used = true
	x.mu.Unlock()
	ctx := context.WithValue(x.control.ctx, hostRunReservationKey{}, x.control)
	if x.control.stream {
		return x.engine.Stream(ctx, x.request, h)
	}
	return x.engine.Run(ctx, x.request, h)
}

func (x *hostRunExecution) Release() {
	x.mu.Lock()
	if x.used {
		x.mu.Unlock()
		return
	}
	x.used = true
	x.mu.Unlock()
	x.engine.finishActiveRun(x.control)
}

// Snapshot returns the current bounded lifecycle projection.
func (c *ActiveRunControl) Snapshot() ActiveRunSnapshot {
	if c == nil {
		return ActiveRunSnapshot{}
	}
	c.mu.RLock()
	state := c.state
	out := ActiveRunSnapshot{RunID: c.runID, SessionID: c.sessionID, State: state, Stream: c.stream, StartedAt: c.startedAt}
	c.mu.RUnlock()
	return out
}

var (
	ErrActiveRunDuplicate       = errors.New("active run already registered")
	ErrActiveRunUnknown         = errors.New("active run not found")
	ErrActiveRunClosed          = errors.New("active run control is closed")
	ErrActiveRunBackpressure    = errors.New("active run realtime ingress is full")
	ErrRuntimeInputUnknown      = errors.New("runtime input run not found")
	ErrRuntimeInputBackpressure = errors.New("runtime input lane is full")
	ErrRuntimeInputClosed       = errors.New("runtime input lane is closed")
)

// WithActiveRunControlLimit bounds the number of concurrently controllable Runs.
func WithActiveRunControlLimit(limit int) Option {
	return func(e *Engine) {
		if limit > 0 {
			e.activeRunControl = true
			e.activeRunLimit = limit
		}
	}
}

// WithActiveRunIngressBuffer sets the per-Run Realtime ingress capacity.
func WithActiveRunIngressBuffer(size int) Option {
	return func(e *Engine) {
		if size > 0 {
			e.activeRunControl = true
			e.activeRunIngressBuffer = size
		}
	}
}

// WithRuntimeInputFollowUpBuffer sets the per-Run bounded follow-up capacity.
func WithRuntimeInputFollowUpBuffer(size int) Option {
	return func(e *Engine) {
		if size > 0 {
			e.activeRunControl = true
			e.runtimeInputFollowUpBuffer = size
		}
	}
}

func (e *Engine) initActiveRunControl() {
	if e.activeRuns == nil {
		e.activeRuns = make(map[string]*ActiveRunControl)
	}
	if e.activeRunLimit <= 0 {
		e.activeRunLimit = 128
	}
	if e.activeRunIngressBuffer <= 0 {
		e.activeRunIngressBuffer = 16
	}
	if e.runtimeInputFollowUpBuffer <= 0 {
		e.runtimeInputFollowUpBuffer = 8
	}
}

func (e *Engine) beginActiveRun(ctx context.Context, runID, sessionID string, stream bool) (*ActiveRunControl, context.Context, error) {
	if e == nil {
		return nil, ctx, ErrActiveRunUnknown
	}
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return nil, ctx, fmt.Errorf("run_id is required")
	}
	e.activeRunMu.Lock()
	e.initActiveRunControl()
	if _, exists := e.activeRuns[runID]; exists {
		e.activeRunMu.Unlock()
		return nil, ctx, ErrActiveRunDuplicate
	}
	if len(e.activeRuns) >= e.activeRunLimit {
		e.activeRunMu.Unlock()
		return nil, ctx, ErrActiveRunBackpressure
	}
	derived, cancel := context.WithCancel(ctx)
	ctrl := &ActiveRunControl{
		engine: e, runID: runID, sessionID: strings.TrimSpace(sessionID), stream: stream,
		startedAt: time.Now().UTC(), ctx: derived, cancel: cancel,
		ingress: make(chan types.RealtimeEventEnvelope, e.activeRunIngressBuffer),
		closed:  make(chan struct{}), state: types.RunStateWorking,
		inputSeen: make(map[string]struct{}), followUpLimit: e.runtimeInputFollowUpBuffer,
	}
	e.activeRuns[runID] = ctrl
	e.activeRunMu.Unlock()
	return ctrl, derived, nil
}

func (e *Engine) activateActiveRun(ctx context.Context, runID, sessionID string, stream bool) (*ActiveRunControl, context.Context, error) {
	reserved, _ := ctx.Value(hostRunReservationKey{}).(*ActiveRunControl)
	if reserved == nil {
		if !e.activeRunControl {
			return nil, ctx, nil
		}
		return e.beginActiveRun(ctx, runID, sessionID, stream)
	}
	if reserved.engine != e || reserved.runID != strings.TrimSpace(runID) || reserved.sessionID != strings.TrimSpace(sessionID) || reserved.stream != stream {
		return nil, ctx, fmt.Errorf("host run reservation correlation mismatch")
	}
	return reserved, ctx, nil
}

func (e *Engine) prepareActiveRun(ctx context.Context, runID, sessionID string, stream bool) (*ActiveRunControl, context.Context, error) {
	return e.activateActiveRun(ctx, runID, sessionID, stream)
}

func (e *Engine) activeRunFailure(runID string, err error) (types.RunResult, error) {
	return types.RunResult{RunID: runID, Error: classified(types.ErrPolicyTimeout, err.Error(), false)}, err
}

func (e *Engine) bindActiveRun(control *ActiveRunControl, runtime *realtimeSessionRuntime) {
	if control != nil {
		control.attachRealtime(runtime)
	}
}

func (e *Engine) activeRunSafePoint(ctx context.Context, h types.EventHandler, control *ActiveRunControl, runtime *realtimeSessionRuntime, iteration int) error {
	if control == nil {
		return nil
	}
	if err := control.drainRealtime(ctx, h, iteration); err != nil {
		return err
	}
	if runtime != nil && runtime.isInterrupted() {
		return control.waitRealtimeResume(ctx, h, iteration)
	}
	return nil
}

// applyRuntimeInputSafePoint moves source-owned steering into the next model
// request only after the existing atomic Runner boundary has completed.
func (e *Engine) applyRuntimeInputSafePoint(ctx context.Context, h types.EventHandler, req *types.RunRequest, control *ActiveRunControl) {
	if req == nil || control == nil {
		return
	}
	for _, input := range control.DrainRuntimeInputSafePoint() {
		req.Messages = append(req.Messages, types.Message{Role: "user", Content: input.Payload})
		e.emit(ctx, h, types.Event{Version: types.EventSchemaVersionV1, Type: types.EventTypeRuntimeInputApplied, RunID: control.runID, Time: e.now(), Payload: map[string]any{
			"input_id":       input.InputID,
			"input_kind":     string(input.Kind),
			"apply_boundary": string(input.ApplyBoundary),
		}})
	}
}

func (e *Engine) finishActiveRun(ctrl *ActiveRunControl) {
	if ctrl == nil {
		return
	}
	ctrl.closeOnce.Do(func() {
		ctrl.settleRuntimeInputs()
		ctrl.mu.Lock()
		ctrl.closedFlag = true
		ctrl.mu.Unlock()
		e.activeRunMu.Lock()
		if current, ok := e.activeRuns[ctrl.runID]; ok && current == ctrl {
			delete(e.activeRuns, ctrl.runID)
		}
		e.activeRunMu.Unlock()
		ctrl.cancel()
		close(ctrl.closed)
	})
}

// ActiveRun returns a currently active source control by Run ID.
func (e *Engine) ActiveRun(runID string) (*ActiveRunControl, bool) {
	if e == nil {
		return nil, false
	}
	e.activeRunMu.RLock()
	ctrl, ok := e.activeRuns[strings.TrimSpace(runID)]
	e.activeRunMu.RUnlock()
	return ctrl, ok
}

// ActiveRuns returns a bounded lifecycle snapshot of active Runs.
func (e *Engine) ActiveRuns() []ActiveRunSnapshot {
	if e == nil {
		return nil
	}
	e.activeRunMu.RLock()
	controls := make([]*ActiveRunControl, 0, len(e.activeRuns))
	for _, ctrl := range e.activeRuns {
		controls = append(controls, ctrl)
	}
	e.activeRunMu.RUnlock()
	out := make([]ActiveRunSnapshot, 0, len(controls))
	for _, ctrl := range controls {
		out = append(out, ctrl.Snapshot())
	}
	return out
}

// CancelRun propagates cancellation to an active source Run.
func (e *Engine) CancelRun(runID, sessionID string) (types.HostAdmissionStatus, error) {
	ctrl, ok := e.ActiveRun(runID)
	if !ok {
		return types.HostAdmissionStatusRejected, ErrActiveRunUnknown
	}
	if sid := strings.TrimSpace(sessionID); sid != "" && sid != ctrl.sessionID {
		return types.HostAdmissionStatusRejected, fmt.Errorf("session mismatch")
	}
	ctrl.mu.Lock()
	if ctrl.closedFlag {
		ctrl.mu.Unlock()
		return types.HostAdmissionStatusRejected, ErrActiveRunClosed
	}
	if ctrl.cancelAdmitted {
		ctrl.mu.Unlock()
		return types.HostAdmissionStatusDuplicate, nil
	}
	if e.nativeOptions != nil && ctrl.state == types.RunStateInputRequired {
		ref, err := e.NativeCheckpoint(context.Background(), runID)
		if err == nil {
			err = e.CancelNativeCheckpoint(context.Background(), ref, nil)
		}
		if err != nil && !errors.Is(err, types.ErrCheckpointInProgress) {
			ctrl.mu.Unlock()
			return types.HostAdmissionStatusRejected, err
		}
	}
	ctrl.cancelAdmitted = true
	ctrl.mu.Unlock()
	ctrl.settleRuntimeInputs()
	ctrl.cancel()
	return types.HostAdmissionStatusAccepted, nil
}

// IngestRealtime submits one bounded Realtime control envelope without blocking.
func (e *Engine) IngestRealtime(runID, sessionID string, ev types.RealtimeEventEnvelope) (types.HostAdmissionStatus, error) {
	ctrl, ok := e.ActiveRun(runID)
	if !ok {
		return types.HostAdmissionStatusRejected, ErrActiveRunUnknown
	}
	if sid := strings.TrimSpace(sessionID); sid != "" && sid != ctrl.sessionID {
		return types.HostAdmissionStatusRejected, fmt.Errorf("session mismatch")
	}
	if strings.TrimSpace(ev.RunID) == "" {
		ev.RunID = ctrl.runID
	}
	if strings.TrimSpace(ev.SessionID) == "" {
		ev.SessionID = ctrl.sessionID
	}
	if err := ev.Validate(); err != nil {
		return types.HostAdmissionStatusRejected, err
	}
	ctrl.mu.Lock()
	if ctrl.closedFlag {
		ctrl.mu.Unlock()
		return types.HostAdmissionStatusRejected, ErrActiveRunClosed
	}
	if ctrl.runtime == nil {
		ctrl.mu.Unlock()
		return types.HostAdmissionStatusRejected, ErrActiveRunClosed
	}
	status, err := ctrl.runtime.reserveControlEvent(ev)
	if err != nil || status != types.HostAdmissionStatusAccepted {
		ctrl.mu.Unlock()
		return status, err
	}
	select {
	case ctrl.ingress <- ev:
		ctrl.mu.Unlock()
		return types.HostAdmissionStatusAccepted, nil
	default:
		ctrl.runtime.releaseControlEvent(ev)
		ctrl.mu.Unlock()
		return types.HostAdmissionStatusRejected, ErrActiveRunBackpressure
	}
}

// RetryRun starts a causally related Run from a terminal failed/canceled source.
// The source result is never mutated; retry support is intentionally delegated
// to this Engine's normal Run/Stream path.
func (e *Engine) RetryRun(ctx context.Context, previous types.RunRef, req types.RunRequest, h types.EventHandler, stream bool) (types.RunResult, error) {
	if err := previous.ValidateProtocolReference(); err != nil {
		return types.RunResult{}, err
	}
	if previous.State != types.RunStateFailed && previous.State != types.RunStateCanceled {
		return types.RunResult{}, fmt.Errorf("retry requires failed or canceled run state, got %q", previous.State)
	}
	runID := strings.TrimSpace(req.RunID)
	if runID == "" {
		runID = e.newRunID()
	}
	if runID == strings.TrimSpace(previous.RunID) {
		return types.RunResult{}, fmt.Errorf("retry run_id must differ from source run")
	}
	retryRef, err := types.NewRetryRunRef(previous, runID)
	if err != nil {
		return types.RunResult{}, err
	}
	req.RunID = retryRef.RunID
	if strings.TrimSpace(req.SessionID) == "" {
		req.SessionID = retryRef.SessionID
	}
	var result types.RunResult
	var runErr error
	retryHandler := h
	if h != nil {
		retryHandler = retryCausationHandler{next: h, causationID: retryRef.CausationID}
	}
	if stream {
		result, runErr = e.Stream(ctx, req, retryHandler)
	} else {
		result, runErr = e.Run(ctx, req, retryHandler)
	}
	attachTerminalOutcome(&result, req.SessionID)
	if result.TerminalOutcome != nil {
		result.TerminalOutcome.CausationID = retryRef.CausationID
	}
	return result, runErr
}

func (c *ActiveRunControl) attachRealtime(runtime *realtimeSessionRuntime) {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.runtime = runtime
	c.mu.Unlock()
}

func (c *ActiveRunControl) drainRealtime(ctx context.Context, h types.EventHandler, iteration int) error {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	runtime := c.runtime
	c.mu.RUnlock()
	if runtime == nil {
		return nil
	}
	for {
		select {
		case ev := <-c.ingress:
			c.mu.Lock()
			runtime.consumeReservedControlEvent(ev)
			c.mu.Unlock()
			if err := runtime.ingestControlEvents(ctx, h, iteration, []types.RealtimeEventEnvelope{ev}); err != nil {
				return err
			}
			c.mu.Lock()
			if runtime.isInterrupted() {
				c.state = types.RunStateInputRequired
			} else {
				c.state = types.RunStateWorking
			}
			c.mu.Unlock()
		default:
			return nil
		}
	}
}

func (c *ActiveRunControl) waitRealtimeResume(ctx context.Context, h types.EventHandler, iteration int) error {
	if c == nil {
		return nil
	}
	for {
		c.mu.RLock()
		runtime := c.runtime
		interrupted := runtime != nil && runtime.isInterrupted()
		c.mu.RUnlock()
		if runtime == nil || !interrupted {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.closed:
			return ErrActiveRunClosed
		case ev := <-c.ingress:
			c.mu.Lock()
			runtime.consumeReservedControlEvent(ev)
			c.mu.Unlock()
			if err := runtime.ingestControlEvents(ctx, h, iteration, []types.RealtimeEventEnvelope{ev}); err != nil {
				return err
			}
			c.mu.Lock()
			if runtime.isInterrupted() {
				c.state = types.RunStateInputRequired
			} else {
				c.state = types.RunStateWorking
			}
			c.mu.Unlock()
		}
	}
}

type actionGateViolationError struct {
	classified *types.ClassifiedError
	err        error
}

func (e *actionGateViolationError) Error() string {
	if e == nil {
		return ""
	}
	if e.err != nil {
		return e.err.Error()
	}
	if e.classified != nil {
		return e.classified.Message
	}
	return "action gate violation"
}

func (e *actionGateViolationError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func (e *actionGateViolationError) ClassifiedError() *types.ClassifiedError {
	if e == nil {
		return nil
	}
	return e.classified
}

func (e *Engine) enforceActionGateForToolCalls(
	ctx context.Context,
	h types.EventHandler,
	req types.RunRequest,
	runID string,
	iteration int,
	seq *int64,
	calls []types.ToolCall,
	stats *actionGateStats,
) (*types.ClassifiedError, error) {
	classifiedActionGate := func(
		call types.ToolCall,
		class types.ErrorClass,
		msg string,
		retryable bool,
		reasonCode string,
	) *types.ClassifiedError {
		ce := classified(class, msg, retryable)
		code := strings.ToLower(strings.TrimSpace(reasonCode))
		if code == "" {
			code = "action.gate.denied"
		}
		details := map[string]any{
			"policy_kind": "action_gate",
			"decision":    "deny",
			"reason_code": code,
		}
		if namespaceTool, ok := namespaceToolKey(call.Name); ok {
			details["namespace_tool"] = namespaceTool
		}
		if callID := strings.TrimSpace(call.CallID); callID != "" {
			details["call_id"] = callID
		}
		if tool := strings.TrimSpace(call.Name); tool != "" {
			details["tool"] = tool
		}
		if trace, ok := e.evaluateRuntimePolicyTrace([]runtimeconfig.RuntimePolicyCandidate{
			{
				Stage:    runtimeconfig.RuntimePolicyStageActionGate,
				Code:     code,
				Source:   runtimeconfig.RuntimePolicyStageActionGate,
				Decision: runtimeconfig.RuntimePolicyDecisionDeny,
			},
		}); ok {
			if strings.TrimSpace(trace.Version) != "" {
				details["policy_precedence_version"] = strings.TrimSpace(trace.Version)
			}
			if strings.TrimSpace(trace.WinnerStage) != "" {
				details["winner_stage"] = strings.TrimSpace(trace.WinnerStage)
			}
			if strings.TrimSpace(trace.DenySource) != "" {
				details["deny_source"] = strings.TrimSpace(trace.DenySource)
			}
			if strings.TrimSpace(trace.TieBreakReason) != "" {
				details["tie_break_reason"] = strings.TrimSpace(trace.TieBreakReason)
			}
			if len(trace.PolicyDecisionPath) > 0 {
				details["policy_decision_path"] = append([]runtimeconfig.RuntimePolicyCandidate(nil), trace.PolicyDecisionPath...)
			}
		}
		ce.Details = details
		return ce
	}

	for _, call := range calls {
		check := types.ActionGateCheck{
			RunID:     runID,
			SessionID: req.SessionID,
			Iteration: iteration,
			CallID:    call.CallID,
			ToolName:  call.Name,
			Input:     req.Input,
			Args:      call.Args,
		}
		evaluation, err := e.evaluateActionGateDecision(ctx, check)
		if err != nil {
			ce := classifiedActionGate(
				call,
				types.ErrTool,
				fmt.Sprintf("action gate evaluate failed: %v", err),
				false,
				"action.gate.evaluate_failed",
			)
			e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseTool, types.ActionStatusFailed, "gate.denied")
			if stats != nil {
				stats.Checks++
				stats.DeniedCount++
			}
			return ce, err
		}
		if !evaluation.Checked {
			continue
		}
		if stats != nil {
			stats.Checks++
			if evaluation.RuleHit {
				stats.RuleHitCount++
				stats.RuleLastID = evaluation.RuleID
			}
		}
		if evaluation.RuleHit {
			e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseTool, types.ActionStatusPending, "gate.rule_match")
		}
		switch evaluation.Decision {
		case types.ActionGateDecisionAllow:
			continue
		case types.ActionGateDecisionDeny:
			if stats != nil {
				stats.DeniedCount++
			}
			e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseTool, types.ActionStatusFailed, "gate.denied")
			msg := fmt.Sprintf("action gate denied tool call: %s", strings.TrimSpace(call.Name))
			return classifiedActionGate(call, types.ErrTool, msg, false, "action.gate.denied"), errors.New(msg)
		case types.ActionGateDecisionRequireConfirm:
			if x := nativeExecutionFrom(ctx); x != nil && e.nativeOptions.Approval {
				if x.approved {
					continue
				}
				return classifiedActionGate(call, types.ErrTool, "native approval required", false, "native.approval_required"), nativeApprovalPause{}
			}
			e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseTool, types.ActionStatusPending, "gate.require_confirm")
			if e.actionGateResolver == nil {
				if stats != nil {
					stats.DeniedCount++
				}
				e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseTool, types.ActionStatusFailed, "gate.denied")
				msg := fmt.Sprintf("action gate requires confirmation but resolver is not configured: %s", strings.TrimSpace(call.Name))
				return classifiedActionGate(call, types.ErrTool, msg, false, "action.gate.resolver_missing"), errors.New(msg)
			}
			timeout := e.actionGateTimeout()
			confirmCtx, cancel := context.WithTimeout(ctx, timeout)
			approved, confirmErr := e.actionGateResolver.Confirm(confirmCtx, types.ActionGateConfirmRequest{
				Check:   check,
				Timeout: timeout,
			})
			cancel()
			if confirmErr != nil || errors.Is(confirmCtx.Err(), context.DeadlineExceeded) {
				if errors.Is(confirmErr, context.DeadlineExceeded) || errors.Is(confirmCtx.Err(), context.DeadlineExceeded) {
					if stats != nil {
						stats.TimeoutCount++
						stats.DeniedCount++
					}
					e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseTool, types.ActionStatusCanceled, "gate.timeout")
					msg := fmt.Sprintf("action gate confirmation timed out for tool: %s", strings.TrimSpace(call.Name))
					return classifiedActionGate(call, types.ErrPolicyTimeout, msg, true, "action.gate.timeout"), context.DeadlineExceeded
				}
				if stats != nil {
					stats.DeniedCount++
				}
				e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseTool, types.ActionStatusFailed, "gate.denied")
				msg := fmt.Sprintf("action gate confirmation failed for tool %s: %v", strings.TrimSpace(call.Name), confirmErr)
				return classifiedActionGate(call, types.ErrTool, msg, false, "action.gate.confirmation_failed"), confirmErr
			}
			if !approved {
				if stats != nil {
					stats.DeniedCount++
				}
				e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseTool, types.ActionStatusFailed, "gate.denied")
				msg := fmt.Sprintf("action gate confirmation denied tool call: %s", strings.TrimSpace(call.Name))
				return classifiedActionGate(call, types.ErrTool, msg, false, "action.gate.confirmation_denied"), errors.New(msg)
			}
		default:
			if stats != nil {
				stats.DeniedCount++
			}
			e.emitTimeline(ctx, h, runID, iteration, seq, types.ActionPhaseTool, types.ActionStatusFailed, "gate.denied")
			msg := fmt.Sprintf("action gate returned unsupported decision %q for tool %s", evaluation.Decision, strings.TrimSpace(call.Name))
			return classifiedActionGate(call, types.ErrTool, msg, false, "action.gate.decision_unsupported"), errors.New(msg)
		}
	}
	return nil, nil
}

type actionGateEvaluation struct {
	Decision types.ActionGateDecision
	Checked  bool
	RuleHit  bool
	RuleID   string
}

func (e *Engine) evaluateActionGateDecision(ctx context.Context, check types.ActionGateCheck) (actionGateEvaluation, error) {
	if e.actionGateMatcher != nil {
		decision, err := e.actionGateMatcher.Evaluate(ctx, check)
		if err != nil {
			return actionGateEvaluation{}, err
		}
		return actionGateEvaluation{
			Decision: normalizeActionGateDecision(decision),
			Checked:  true,
		}, nil
	}
	cfg := e.actionGateConfig()
	if !cfg.Enabled {
		return actionGateEvaluation{Decision: types.ActionGateDecisionAllow}, nil
	}

	toolName := normalizeToolName(check.ToolName)
	content := strings.ToLower(strings.TrimSpace(check.Input) + " " + marshalToolArgs(check.Args))
	defaultDecision := normalizeActionGateDecision(types.ActionGateDecision(cfg.Policy))
	for _, rule := range cfg.ParameterRules {
		if !ruleAppliesToTool(rule, toolName) {
			continue
		}
		matched, err := evaluateRuleCondition(rule.Condition, check.Args)
		if err != nil {
			return actionGateEvaluation{}, err
		}
		if !matched {
			continue
		}
		decision := defaultDecision
		if strings.TrimSpace(string(rule.Action)) != "" {
			decision = normalizeActionGateDecision(rule.Action)
		}
		return actionGateEvaluation{
			Decision: decision,
			Checked:  true,
			RuleHit:  true,
			RuleID:   strings.TrimSpace(rule.ID),
		}, nil
	}

	if policy, ok := cfg.DecisionByTool[toolName]; ok {
		return actionGateEvaluation{
			Decision: normalizeActionGateDecision(types.ActionGateDecision(policy)),
			Checked:  true,
		}, nil
	}
	segments := strings.Split(toolName, ".")
	if len(segments) > 1 {
		shortName := strings.TrimSpace(segments[len(segments)-1])
		if policy, ok := cfg.DecisionByTool[shortName]; ok {
			return actionGateEvaluation{
				Decision: normalizeActionGateDecision(types.ActionGateDecision(policy)),
				Checked:  true,
			}, nil
		}
	}

	for _, tool := range cfg.ToolNames {
		normalized := normalizeToolName(tool)
		if normalized == "" {
			continue
		}
		if normalized == toolName {
			return actionGateEvaluation{Decision: defaultDecision, Checked: true}, nil
		}
		if len(segments) > 1 && normalized == strings.TrimSpace(segments[len(segments)-1]) {
			return actionGateEvaluation{Decision: defaultDecision, Checked: true}, nil
		}
	}

	for keyword, policy := range cfg.DecisionByWord {
		kw := strings.ToLower(strings.TrimSpace(keyword))
		if kw == "" {
			continue
		}
		if strings.Contains(content, kw) {
			return actionGateEvaluation{
				Decision: normalizeActionGateDecision(types.ActionGateDecision(policy)),
				Checked:  true,
			}, nil
		}
	}
	for _, keyword := range cfg.Keywords {
		kw := strings.ToLower(strings.TrimSpace(keyword))
		if kw == "" {
			continue
		}
		if strings.Contains(content, kw) {
			return actionGateEvaluation{Decision: defaultDecision, Checked: true}, nil
		}
	}
	return actionGateEvaluation{Decision: types.ActionGateDecisionAllow}, nil
}
