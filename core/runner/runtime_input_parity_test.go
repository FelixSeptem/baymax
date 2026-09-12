package runner

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

func TestRuntimeInputCancelAndTerminalPrecedenceAcrossRunAndStream(t *testing.T) {
	for _, stream := range []bool{false, true} {
		mode := "run"
		if stream {
			mode = "stream"
		}
		t.Run(mode, func(t *testing.T) {
			e := New(nil, WithRuntimeInputFollowUpBuffer(1))
			ctrl, _, err := e.beginActiveRun(context.Background(), "input-precedence-"+mode, "session-precedence", stream)
			if err != nil {
				t.Fatal(err)
			}
			input := runtimeInputForControl(types.RuntimeInputKindSteering, "steer-"+mode, ctrl)
			if _, err := e.AdmitRuntimeInput(input); err != nil {
				t.Fatal(err)
			}
			follow := runtimeInputForControl(types.RuntimeInputKindFollowUp, "follow-"+mode, ctrl)
			if _, err := e.AdmitRuntimeInput(follow); err != nil {
				t.Fatal(err)
			}
			status, err := e.CancelRun(ctrl.runID, ctrl.sessionID)
			if err != nil || status != types.HostAdmissionStatusAccepted {
				t.Fatalf("cancel status=%v err=%v", status, err)
			}
			if got := ctrl.DrainRuntimeInputSafePoint(); len(got) != 0 {
				t.Fatalf("cancelled steering applied: %#v", got)
			}
			if got := ctrl.PendingRuntimeInputCount(); got != 0 {
				t.Fatalf("cancelled input remained pending: %d", got)
			}
			if _, err := e.PromoteRuntimeFollowUp(context.Background(), ctrl.runID, ctrl.sessionID, nil, stream); err == nil || !strings.Contains(err.Error(), types.RuntimeInputReasonStale) {
				t.Fatalf("promotion after terminal cancel err=%v, want stale source-owned outcome", err)
			}
			e.finishActiveRun(ctrl)
		})
	}
}

func TestRuntimeInputRunStreamSafePointAndPromotionParity(t *testing.T) {
	type observation struct {
		status     types.RuntimeInputAdmissionStatus
		reason     string
		kind       types.RuntimeInputKind
		applied    string
		promotedID string
		causation  string
	}
	observed := make(map[bool]observation, 2)
	for _, stream := range []bool{false, true} {
		e := New(&fakeModel{
			generate: func(context.Context, types.ModelRequest) (types.ModelResponse, error) {
				return types.ModelResponse{FinalAnswer: "promoted"}, nil
			},
			stream: func(_ context.Context, _ types.ModelRequest, onEvent func(types.ModelEvent) error) error {
				return onEvent(types.ModelEvent{Type: types.ModelEventTypeFinalAnswer, TextDelta: "promoted"})
			},
		}, WithRuntimeInputFollowUpBuffer(1))
		ctrl, _, err := e.beginActiveRun(context.Background(), "input-parity-"+map[bool]string{false: "run", true: "stream"}[stream], "session-parity", stream)
		if err != nil {
			t.Fatal(err)
		}
		input := runtimeInputForControl(types.RuntimeInputKindSteering, "steer-parity", ctrl)
		admission, err := e.AdmitRuntimeInput(input)
		if err != nil {
			t.Fatal(err)
		}
		applied := ctrl.DrainRuntimeInputSafePoint()
		if len(applied) != 1 {
			t.Fatalf("%v safe point applied=%#v", stream, applied)
		}
		follow := runtimeInputForControl(types.RuntimeInputKindFollowUp, "follow-parity", ctrl)
		if _, err := e.AdmitRuntimeInput(follow); err != nil {
			t.Fatal(err)
		}
		result, err := e.PromoteRuntimeFollowUp(context.Background(), ctrl.runID, ctrl.sessionID, nil, stream)
		if err != nil {
			t.Fatal(err)
		}
		observed[stream] = observation{status: admission.Status, reason: admission.ReasonCode, kind: admission.Kind, applied: applied[0].Payload, promotedID: result.RunID, causation: result.TerminalOutcome.CausationID}
		e.finishActiveRun(ctrl)
	}
	run, stream := observed[false], observed[true]
	if run.status != stream.status || run.reason != stream.reason || run.kind != stream.kind || run.applied != stream.applied {
		t.Fatalf("runtime input parity drift run=%#v stream=%#v", run, stream)
	}
	if run.causation == "" || stream.causation == "" {
		t.Fatalf("runtime input promotion lost causation run=%#v stream=%#v", run, stream)
	}
	if run.promotedID == "input-parity-run" || stream.promotedID == "input-parity-stream" {
		t.Fatalf("follow-up promotion reused terminal source run: run=%q stream=%q", run.promotedID, stream.promotedID)
	}
}

func TestRuntimeInputIdentityRemainsStableAcrossSafePointBoundary(t *testing.T) {
	e := New(nil)
	ctrl, _, err := e.beginActiveRun(context.Background(), "input-identity", "session-identity", false)
	if err != nil {
		t.Fatal(err)
	}
	defer e.finishActiveRun(ctrl)
	input := runtimeInputForControl(types.RuntimeInputKindSteering, "identity", ctrl)
	input.Time = time.Unix(123, 0).UTC()
	first, err := e.AdmitRuntimeInput(input)
	if err != nil || first.Status != types.RuntimeInputAdmissionStatusAccepted {
		t.Fatalf("first admission=%#v err=%v", first, err)
	}
	if got := ctrl.DrainRuntimeInputSafePoint(); len(got) != 1 || got[0].NormalizedIdentity() != input.NormalizedIdentity() {
		t.Fatalf("safe-point identity drift: %#v", got)
	}
	duplicate, err := e.AdmitRuntimeInput(input)
	if err != nil || duplicate.Status != types.RuntimeInputAdmissionStatusDuplicate {
		t.Fatalf("post-apply duplicate=%#v err=%v", duplicate, err)
	}
}
