package runner

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/FelixSeptem/baymax/core/types"
)

type CompletionReferenceSnapshot struct {
	Pending []types.CompletionReference `json:"pending,omitempty"`
}

// SnapshotCompletionReferences returns bounded pending completion references for an active Run.
func (e *Engine) SnapshotCompletionReferences(runID string) (CompletionReferenceSnapshot, bool) {
	ctrl, ok := e.ActiveRun(runID)
	if !ok {
		return CompletionReferenceSnapshot{}, false
	}
	return ctrl.SnapshotCompletionReferences(), true
}

// AdmitHostRuntimeInput adapts the host transport contract to the source-owned
// input admission API without giving the host access to the input lanes.
func (e *Engine) AdmitHostRuntimeInput(_ context.Context, _ types.HostCommandEnvelope, input types.RuntimeInputEnvelope) (types.HostAdmissionStatus, string, error) {
	admission, err := e.AdmitRuntimeInput(input)
	if err != nil {
		return mapRuntimeInputAdmissionStatus(admission.Status), admission.ReasonCode, err
	}
	return mapRuntimeInputAdmissionStatus(admission.Status), admission.ReasonCode, nil
}

func mapRuntimeInputAdmissionStatus(status types.RuntimeInputAdmissionStatus) types.HostAdmissionStatus {
	switch status {
	case types.RuntimeInputAdmissionStatusAccepted:
		return types.HostAdmissionStatusAccepted
	case types.RuntimeInputAdmissionStatusDuplicate:
		return types.HostAdmissionStatusDuplicate
	default:
		return types.HostAdmissionStatusRejected
	}
}

// PromoteRuntimeFollowUp settles the next source-owned follow-up at the
// existing idle/terminal boundary by starting a distinct causal Run.
func (e *Engine) PromoteRuntimeFollowUp(ctx context.Context, runID, sessionID string, h types.EventHandler, stream bool) (types.RunResult, error) {
	ctrl, ok := e.ActiveRun(runID)
	if !ok {
		return types.RunResult{}, ErrRuntimeInputUnknown
	}
	if strings.TrimSpace(sessionID) != ctrl.sessionID {
		return types.RunResult{}, fmt.Errorf("session mismatch")
	}
	input, ok := ctrl.popFollowUpAtBoundary()
	if !ok {
		return types.RunResult{}, fmt.Errorf("%s: follow-up unavailable", types.RuntimeInputReasonStale)
	}
	newRunID := e.newRunID()
	req := types.RunRequest{RunID: newRunID, SessionID: ctrl.sessionID, Input: input.Payload}
	var (
		result types.RunResult
		err    error
	)
	if stream {
		result, err = e.Stream(ctx, req, h)
	} else {
		result, err = e.Run(ctx, req, h)
	}
	attachTerminalOutcome(&result, ctrl.sessionID)
	if result.TerminalOutcome != nil {
		result.TerminalOutcome.CausationID = ctrl.runID
	}
	e.emit(ctx, h, types.Event{Version: types.EventSchemaVersionV1, Type: types.EventTypeRuntimeInputPromotion, RunID: ctrl.runID, Time: e.now(), Payload: map[string]any{
		"input_id":        input.InputID,
		"input_kind":      string(input.Kind),
		"promoted_run_id": result.RunID,
		"causation_id":    ctrl.runID,
	}})
	return result, err
}

// AdmitRuntimeInput delegates steering/follow-up admission to the source-owned Run control.
// The host does not receive access to the underlying lanes or terminal state.
func (e *Engine) AdmitRuntimeInput(input types.RuntimeInputEnvelope) (types.RuntimeInputAdmission, error) {
	if err := input.Validate(); err != nil {
		return types.RuntimeInputAdmission{}, err
	}
	ctrl, ok := e.ActiveRun(input.RunID)
	if !ok {
		return runtimeInputRejected(input, types.RuntimeInputReasonStale, ErrRuntimeInputUnknown)
	}
	if strings.TrimSpace(input.SessionID) != ctrl.sessionID {
		return runtimeInputRejected(input, types.RuntimeInputReasonRejected, fmt.Errorf("session mismatch"))
	}
	identity := input.NormalizedIdentity()
	ctrl.inputMu.Lock()
	defer ctrl.inputMu.Unlock()
	if ctrl.runtimeInputClosed {
		return runtimeInputRejected(input, types.RuntimeInputReasonDisconnected, ErrRuntimeInputClosed)
	}
	if _, exists := ctrl.inputSeen[identity]; exists {
		return types.NormalizeRuntimeInputAdmission(input, types.RuntimeInputAdmissionStatusDuplicate, types.RuntimeInputReasonDuplicate)
	}
	switch input.Kind {
	case types.RuntimeInputKindSteering:
		if ctrl.pendingSteering != nil {
			return runtimeInputRejected(input, types.RuntimeInputReasonBackpressure, ErrRuntimeInputBackpressure)
		}
		copy := input
		ctrl.pendingSteering = &copy
	case types.RuntimeInputKindFollowUp:
		if len(ctrl.followUps) >= ctrl.followUpLimit {
			return runtimeInputRejected(input, types.RuntimeInputReasonBackpressure, ErrRuntimeInputBackpressure)
		}
		ctrl.followUps = append(ctrl.followUps, input)
	default:
		return runtimeInputRejected(input, types.RuntimeInputReasonUnsupportedKind, fmt.Errorf("unsupported input kind %q", input.Kind))
	}
	ctrl.inputSeen[identity] = struct{}{}
	return types.NormalizeRuntimeInputAdmission(input, types.RuntimeInputAdmissionStatusAccepted, types.RuntimeInputReasonAccepted)
}

// AdmitCompletionReference adapts a durable completion to the existing
// source-owned follow-up lane. It carries references only; completion bodies
// remain owned by the mailbox/scheduler path and are never copied here.
func (e *Engine) AdmitCompletionReference(ref types.CompletionReference) (types.RuntimeInputAdmission, error) {
	if strings.TrimSpace(ref.MessageID) == "" || strings.TrimSpace(ref.IdempotencyKey) == "" || strings.TrimSpace(ref.RunID) == "" || strings.TrimSpace(ref.SessionID) == "" {
		return types.RuntimeInputAdmission{}, fmt.Errorf("completion reference requires message, idempotency, run, and session identifiers")
	}
	correlation := strings.TrimSpace(ref.CorrelationID)
	if correlation == "" {
		correlation = strings.TrimSpace(ref.TaskID)
	}
	payload := "completion:" + strings.TrimSpace(ref.MessageID)
	input := types.RuntimeInputEnvelope{Version: types.RuntimeInputProtocolVersionV1, InputID: strings.TrimSpace(ref.IdempotencyKey), Kind: types.RuntimeInputKindFollowUp, Time: e.now(), SessionID: strings.TrimSpace(ref.SessionID), RunID: strings.TrimSpace(ref.RunID), CausationID: strings.TrimSpace(ref.AttemptID), SourceCorrelation: correlation, ApplyBoundary: types.RuntimeInputApplyBoundaryIdleTerminal, Payload: payload}
	return e.AdmitRuntimeInput(input)
}

// SnapshotCompletionReferences exposes only bounded pending reference metadata.
func (c *ActiveRunControl) SnapshotCompletionReferences() CompletionReferenceSnapshot {
	if c == nil {
		return CompletionReferenceSnapshot{}
	}
	c.inputMu.Lock()
	defer c.inputMu.Unlock()
	out := CompletionReferenceSnapshot{Pending: make([]types.CompletionReference, 0, len(c.followUps))}
	for _, input := range c.followUps {
		if !strings.HasPrefix(input.Payload, "completion:") {
			continue
		}
		out.Pending = append(out.Pending, types.CompletionReference{MessageID: strings.TrimPrefix(input.Payload, "completion:"), IdempotencyKey: input.InputID, CorrelationID: input.SourceCorrelation, AttemptID: input.CausationID, SessionID: input.SessionID, RunID: input.RunID})
	}
	sort.Slice(out.Pending, func(i, j int) bool { return out.Pending[i].IdempotencyKey < out.Pending[j].IdempotencyKey })
	return out
}

// RestoreCompletionReferences reuses admission dedupe and never creates a Run.
func (e *Engine) RestoreCompletionReferences(snapshot CompletionReferenceSnapshot) error {
	if len(snapshot.Pending) > 256 {
		return fmt.Errorf("completion reference snapshot exceeds bound")
	}
	for _, ref := range snapshot.Pending {
		admission, err := e.AdmitCompletionReference(ref)
		if err != nil {
			return err
		}
		if admission.Status != types.RuntimeInputAdmissionStatusAccepted && admission.Status != types.RuntimeInputAdmissionStatusDuplicate {
			return fmt.Errorf("completion reference restore not applied: %s", admission.ReasonCode)
		}
	}
	return nil
}

func runtimeInputRejected(input types.RuntimeInputEnvelope, reason string, err error) (types.RuntimeInputAdmission, error) {
	admission, normalizeErr := types.NormalizeRuntimeInputAdmission(input, types.RuntimeInputAdmissionStatusRejected, reason)
	if normalizeErr != nil {
		return types.RuntimeInputAdmission{}, normalizeErr
	}
	return admission, err
}

// DrainRuntimeInputSafePoint applies only the pending steering item. Follow-ups
// remain pending until their existing idle/terminal promotion boundary.
func (c *ActiveRunControl) DrainRuntimeInputSafePoint() []types.RuntimeInputEnvelope {
	if c == nil {
		return nil
	}
	c.inputMu.Lock()
	defer c.inputMu.Unlock()
	if c.pendingSteering == nil {
		return nil
	}
	out := []types.RuntimeInputEnvelope{*c.pendingSteering}
	c.pendingSteering = nil
	return out
}

// PendingRuntimeInputCount reports source-owned bounded input occupancy.
func (c *ActiveRunControl) PendingRuntimeInputCount() int {
	if c == nil {
		return 0
	}
	c.inputMu.Lock()
	defer c.inputMu.Unlock()
	count := len(c.followUps)
	if c.pendingSteering != nil {
		count++
	}
	return count
}

func (c *ActiveRunControl) popFollowUpAtBoundary() (types.RuntimeInputEnvelope, bool) {
	if c == nil {
		return types.RuntimeInputEnvelope{}, false
	}
	c.inputMu.Lock()
	defer c.inputMu.Unlock()
	if len(c.followUps) == 0 {
		return types.RuntimeInputEnvelope{}, false
	}
	input := c.followUps[0]
	copy(c.followUps, c.followUps[1:])
	c.followUps[len(c.followUps)-1] = types.RuntimeInputEnvelope{}
	c.followUps = c.followUps[:len(c.followUps)-1]
	return input, true
}

func (c *ActiveRunControl) settleRuntimeInputs() {
	if c == nil {
		return
	}
	c.inputMu.Lock()
	c.runtimeInputClosed = true
	c.pendingSteering = nil
	c.followUps = nil
	c.inputMu.Unlock()
}
