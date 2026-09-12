package runner

import (
	"context"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

func runtimeInputForControl(kind types.RuntimeInputKind, id string, ctrl *ActiveRunControl) types.RuntimeInputEnvelope {
	return types.RuntimeInputEnvelope{
		Version:   types.RuntimeInputProtocolVersionV1,
		InputID:   id,
		Kind:      kind,
		Time:      time.Unix(1, 0).UTC(),
		SessionID: ctrl.sessionID,
		RunID:     ctrl.runID,
		Payload:   "adjust the next decision",
	}
}

func TestActiveRunRuntimeInputUsesBoundedSourceOwnedLanes(t *testing.T) {
	e := New(nil, WithActiveRunControlLimit(1), WithRuntimeInputFollowUpBuffer(1))
	ctrl, _, err := e.beginActiveRun(context.Background(), "run-input", "session-input", false)
	if err != nil {
		t.Fatal(err)
	}
	defer e.finishActiveRun(ctrl)

	steering := runtimeInputForControl(types.RuntimeInputKindSteering, "steer-1", ctrl)
	first, err := e.AdmitRuntimeInput(steering)
	if err != nil || first.Status != types.RuntimeInputAdmissionStatusAccepted {
		t.Fatalf("steering admission=%+v err=%v", first, err)
	}
	duplicate, err := e.AdmitRuntimeInput(steering)
	if err != nil || duplicate.Status != types.RuntimeInputAdmissionStatusDuplicate {
		t.Fatalf("duplicate admission=%+v err=%v", duplicate, err)
	}
	secondSteering := runtimeInputForControl(types.RuntimeInputKindSteering, "steer-2", ctrl)
	full, err := e.AdmitRuntimeInput(secondSteering)
	if err == nil || full.ReasonCode != types.RuntimeInputReasonBackpressure {
		t.Fatalf("full steering admission=%+v err=%v", full, err)
	}

	followUp := runtimeInputForControl(types.RuntimeInputKindFollowUp, "follow-1", ctrl)
	accepted, err := e.AdmitRuntimeInput(followUp)
	if err != nil || accepted.Status != types.RuntimeInputAdmissionStatusAccepted {
		t.Fatalf("follow-up admission=%+v err=%v", accepted, err)
	}
	fullFollowUp, err := e.AdmitRuntimeInput(runtimeInputForControl(types.RuntimeInputKindFollowUp, "follow-2", ctrl))
	if err == nil || fullFollowUp.ReasonCode != types.RuntimeInputReasonBackpressure {
		t.Fatalf("full follow-up admission=%+v err=%v", fullFollowUp, err)
	}
}

func TestActiveRunRuntimeInputSteeringDrainsOnlyAtSafePoint(t *testing.T) {
	e := New(nil)
	ctrl, _, err := e.beginActiveRun(context.Background(), "run-safe-point", "session-safe-point", false)
	if err != nil {
		t.Fatal(err)
	}
	defer e.finishActiveRun(ctrl)
	input := runtimeInputForControl(types.RuntimeInputKindSteering, "steer-safe", ctrl)
	if _, err := e.AdmitRuntimeInput(input); err != nil {
		t.Fatal(err)
	}
	if got := ctrl.PendingRuntimeInputCount(); got != 1 {
		t.Fatalf("pending before safe point=%d, want 1", got)
	}
	applied := ctrl.DrainRuntimeInputSafePoint()
	if len(applied) != 1 || applied[0].InputID != input.InputID {
		t.Fatalf("applied=%+v", applied)
	}
	if got := ctrl.PendingRuntimeInputCount(); got != 0 {
		t.Fatalf("pending after safe point=%d, want 0", got)
	}
}

func TestPromoteRuntimeFollowUpCreatesDistinctCausalRun(t *testing.T) {
	var modelRunID string
	e := New(&fakeModel{generate: func(_ context.Context, req types.ModelRequest) (types.ModelResponse, error) {
		modelRunID = req.RunID
		return types.ModelResponse{FinalAnswer: "followed"}, nil
	}})
	ctrl, _, err := e.beginActiveRun(context.Background(), "run-terminal-source", "session-follow", false)
	if err != nil {
		t.Fatal(err)
	}
	input := runtimeInputForControl(types.RuntimeInputKindFollowUp, "follow-1", ctrl)
	if _, err := e.AdmitRuntimeInput(input); err != nil {
		t.Fatal(err)
	}
	result, err := e.PromoteRuntimeFollowUp(context.Background(), ctrl.runID, ctrl.sessionID, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.RunID == ctrl.runID || modelRunID != result.RunID {
		t.Fatalf("promoted result=%+v model_run_id=%q", result, modelRunID)
	}
	if result.TerminalOutcome == nil || result.TerminalOutcome.CausationID != ctrl.runID {
		t.Fatalf("terminal causation=%+v, want %q", result.TerminalOutcome, ctrl.runID)
	}
	if got := ctrl.PendingRuntimeInputCount(); got != 0 {
		t.Fatalf("follow-up remained pending=%d", got)
	}
	e.finishActiveRun(ctrl)
}

func TestCancelSettlesPendingRuntimeInputBeforeSafePoint(t *testing.T) {
	e := New(nil)
	ctrl, _, err := e.beginActiveRun(context.Background(), "run-cancel-input", "session-cancel-input", false)
	if err != nil {
		t.Fatal(err)
	}
	defer e.finishActiveRun(ctrl)
	if _, err := e.AdmitRuntimeInput(runtimeInputForControl(types.RuntimeInputKindSteering, "steer-cancel", ctrl)); err != nil {
		t.Fatal(err)
	}
	status, err := e.CancelRun(ctrl.runID, ctrl.sessionID)
	if err != nil || status != types.HostAdmissionStatusAccepted {
		t.Fatalf("cancel status=%v err=%v", status, err)
	}
	if applied := ctrl.DrainRuntimeInputSafePoint(); len(applied) != 0 {
		t.Fatalf("cancelled input applied at safe point: %+v", applied)
	}
	if got := ctrl.PendingRuntimeInputCount(); got != 0 {
		t.Fatalf("pending input after cancel=%d", got)
	}
}

func TestRuntimeInputAdmissionRejectsUnknownSessionAndClosedRuns(t *testing.T) {
	e := New(nil)
	unknown := types.RuntimeInputEnvelope{Version: types.RuntimeInputProtocolVersionV1, InputID: "unknown", Kind: types.RuntimeInputKindSteering, Time: time.Unix(1, 0).UTC(), SessionID: "session", RunID: "missing", Payload: "input"}
	admission, err := e.AdmitRuntimeInput(unknown)
	if err == nil || admission.ReasonCode != types.RuntimeInputReasonStale {
		t.Fatalf("unknown admission=%+v err=%v", admission, err)
	}
	ctrl, _, err := e.beginActiveRun(context.Background(), "run-session-check", "session-a", false)
	if err != nil {
		t.Fatal(err)
	}
	defer e.finishActiveRun(ctrl)
	wrongSession := runtimeInputForControl(types.RuntimeInputKindSteering, "wrong-session", ctrl)
	wrongSession.SessionID = "session-b"
	admission, err = e.AdmitRuntimeInput(wrongSession)
	if err == nil || admission.ReasonCode != types.RuntimeInputReasonRejected {
		t.Fatalf("session mismatch admission=%+v err=%v", admission, err)
	}
	if _, err := e.CancelRun(ctrl.runID, ctrl.sessionID); err != nil {
		t.Fatal(err)
	}
	closed := runtimeInputForControl(types.RuntimeInputKindSteering, "closed", ctrl)
	admission, err = e.AdmitRuntimeInput(closed)
	if err == nil || admission.ReasonCode != types.RuntimeInputReasonDisconnected {
		t.Fatalf("closed admission=%+v err=%v", admission, err)
	}
}
