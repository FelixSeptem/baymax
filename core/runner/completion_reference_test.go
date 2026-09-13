package runner

import (
	"context"
	"reflect"
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
)

func TestAdmitCompletionReferenceUsesExistingFollowUpLane(t *testing.T) {
	e := New(nil)
	ctrl, _, err := e.beginActiveRun(context.Background(), "run-1", "session-1", false)
	if err != nil {
		t.Fatal(err)
	}
	defer e.finishActiveRun(ctrl)
	admission, err := e.AdmitCompletionReference(types.CompletionReference{MessageID: "msg-1", IdempotencyKey: "idem-1", CorrelationID: "corr-1", TaskID: "task-1", AttemptID: "attempt-1", SessionID: "session-1", RunID: "run-1"})
	if err != nil || admission.Status != types.RuntimeInputAdmissionStatusAccepted {
		t.Fatalf("admission=%#v err=%v", admission, err)
	}
	if ctrl.PendingRuntimeInputCount() != 1 {
		t.Fatalf("pending=%d", ctrl.PendingRuntimeInputCount())
	}
}

func TestAdmitCompletionReferenceDeduplicatesAndRejectsMissingReference(t *testing.T) {
	e := New(nil)
	ctrl, _, err := e.beginActiveRun(context.Background(), "run-1", "session-1", false)
	if err != nil {
		t.Fatal(err)
	}
	defer e.finishActiveRun(ctrl)
	ref := types.CompletionReference{MessageID: "msg-1", IdempotencyKey: "idem-1", SessionID: "session-1", RunID: "run-1"}
	first, _ := e.AdmitCompletionReference(ref)
	second, _ := e.AdmitCompletionReference(ref)
	if first.Status != types.RuntimeInputAdmissionStatusAccepted || second.Status != types.RuntimeInputAdmissionStatusDuplicate {
		t.Fatalf("first=%#v second=%#v", first, second)
	}
	if _, err := e.AdmitCompletionReference(types.CompletionReference{MessageID: "msg-2", IdempotencyKey: "idem-2"}); err == nil {
		t.Fatal("missing reference unexpectedly accepted")
	}
}

func TestAdmitCompletionReferenceDoesNotResurrectClosedRun(t *testing.T) {
	e := New(nil)
	ctrl, _, err := e.beginActiveRun(context.Background(), "run-closed", "session-closed", false)
	if err != nil {
		t.Fatal(err)
	}
	e.finishActiveRun(ctrl)
	_, err = e.AdmitCompletionReference(types.CompletionReference{MessageID: "msg-closed", IdempotencyKey: "idem-closed", SessionID: "session-closed", RunID: "run-closed"})
	if err == nil {
		t.Fatal("closed run unexpectedly accepted completion")
	}
}

func TestCompletionReferencePromotesOnlyAtExistingBoundaryForRunAndStream(t *testing.T) {
	for _, stream := range []bool{false, true} {
		t.Run(map[bool]string{false: "run", true: "stream"}[stream], func(t *testing.T) {
			calls := 0
			model := &fakeModel{generate: func(_ context.Context, _ types.ModelRequest) (types.ModelResponse, error) {
				calls++
				return types.ModelResponse{FinalAnswer: "done"}, nil
			}, stream: func(_ context.Context, _ types.ModelRequest, _ func(types.ModelEvent) error) error {
				calls++
				return nil
			}}
			e := New(model)
			ctrl, _, err := e.beginActiveRun(context.Background(), "run-boundary", "session-boundary", stream)
			if err != nil {
				t.Fatal(err)
			}
			defer e.finishActiveRun(ctrl)
			if _, err := e.AdmitCompletionReference(types.CompletionReference{MessageID: "msg-1", IdempotencyKey: "idem-1", SessionID: ctrl.sessionID, RunID: ctrl.runID}); err != nil {
				t.Fatal(err)
			}
			if calls != 0 {
				t.Fatalf("completion invoked model before boundary: %d", calls)
			}
			if _, err := e.PromoteRuntimeFollowUp(context.Background(), ctrl.runID, ctrl.sessionID, nil, stream); err != nil {
				t.Fatal(err)
			}
			if calls != 1 {
				t.Fatalf("model calls=%d, want 1", calls)
			}
		})
	}
}

func TestCompletionReferenceSnapshotRestoreIsIdempotent(t *testing.T) {
	e := New(nil)
	ctrl, _, err := e.beginActiveRun(context.Background(), "run-recover", "session-recover", false)
	if err != nil {
		t.Fatal(err)
	}
	defer e.finishActiveRun(ctrl)
	ref := types.CompletionReference{MessageID: "msg-1", IdempotencyKey: "idem-1", SessionID: ctrl.sessionID, RunID: ctrl.runID}
	if _, err := e.AdmitCompletionReference(ref); err != nil {
		t.Fatal(err)
	}
	snapshot := ctrl.SnapshotCompletionReferences()
	if len(snapshot.Pending) != 1 {
		t.Fatalf("snapshot=%#v", snapshot)
	}
	if err := e.RestoreCompletionReferences(snapshot); err != nil {
		t.Fatal(err)
	}
	if ctrl.PendingRuntimeInputCount() != 1 {
		t.Fatalf("pending=%d, want 1", ctrl.PendingRuntimeInputCount())
	}
}

func TestLateCompletionAfterTimeoutCannotMutateAuthoritativeRun(t *testing.T) {
	e := New(nil)
	ctrl, _, err := e.beginActiveRun(context.Background(), "run-timeout-late", "session-timeout-late", false)
	if err != nil {
		t.Fatal(err)
	}
	defer e.finishActiveRun(ctrl)
	if status, err := e.CancelRun(ctrl.runID, ctrl.sessionID); err != nil || status != types.HostAdmissionStatusAccepted {
		t.Fatalf("timeout cancellation status=%v err=%v", status, err)
	}
	ref := types.CompletionReference{MessageID: "late-timeout", IdempotencyKey: "late-timeout-idem", TaskID: "task-timeout", AttemptID: "attempt-timeout", SessionID: ctrl.sessionID, RunID: ctrl.runID}
	admission, err := e.AdmitCompletionReference(ref)
	if err == nil || admission.Status != types.RuntimeInputAdmissionStatusRejected {
		t.Fatalf("late completion admission=%#v err=%v, want rejected", admission, err)
	}
	if admission.ReasonCode != types.RuntimeInputReasonDisconnected && admission.ReasonCode != types.RuntimeInputReasonStale && admission.ReasonCode != types.RuntimeInputReasonTerminal {
		t.Fatalf("late completion reason=%q", admission.ReasonCode)
	}
	if got := ctrl.PendingRuntimeInputCount(); got != 0 {
		t.Fatalf("late completion pending=%d", got)
	}
}

func TestCompletionRecoveryAppliesAtMostOnceAndPreservesTerminalSource(t *testing.T) {
	for _, stream := range []bool{false, true} {
		mode := "run"
		if stream {
			mode = "stream"
		}
		t.Run(mode, func(t *testing.T) {
			calls := 0
			model := &fakeModel{generate: func(_ context.Context, _ types.ModelRequest) (types.ModelResponse, error) {
				calls++
				return types.ModelResponse{FinalAnswer: "recovered"}, nil
			}, stream: func(_ context.Context, _ types.ModelRequest, onEvent func(types.ModelEvent) error) error {
				calls++
				return onEvent(types.ModelEvent{Type: types.ModelEventTypeFinalAnswer, TextDelta: "recovered"})
			}}
			e := New(model)
			ctrl, _, err := e.beginActiveRun(context.Background(), "run-recovery-"+mode, "session-recovery", stream)
			if err != nil {
				t.Fatal(err)
			}
			defer e.finishActiveRun(ctrl)
			original := types.RunRef{RunID: ctrl.runID, SessionID: ctrl.sessionID, State: types.RunStateFailed, Source: types.ProtocolSourceRunner}
			ref := types.CompletionReference{MessageID: "recover-completion-" + mode, IdempotencyKey: "recover-idem-" + mode, TaskID: "task-recover", AttemptID: "attempt-recover", SessionID: ctrl.sessionID, RunID: ctrl.runID}
			if admission, err := e.AdmitCompletionReference(ref); err != nil || admission.Status != types.RuntimeInputAdmissionStatusAccepted {
				t.Fatalf("admission=%#v err=%v", admission, err)
			}
			snapshot := ctrl.SnapshotCompletionReferences()
			if err := e.RestoreCompletionReferences(snapshot); err != nil {
				t.Fatal(err)
			}
			if err := e.RestoreCompletionReferences(snapshot); err != nil {
				t.Fatal(err)
			}
			promoted, err := e.PromoteRuntimeFollowUp(context.Background(), ctrl.runID, ctrl.sessionID, nil, stream)
			if err != nil {
				t.Fatal(err)
			}
			if calls != 1 || promoted.TerminalOutcome == nil || promoted.TerminalOutcome.CausationID != ctrl.runID {
				t.Fatalf("calls=%d result=%#v", calls, promoted)
			}
			if _, err := e.PromoteRuntimeFollowUp(context.Background(), ctrl.runID, ctrl.sessionID, nil, stream); err == nil {
				t.Fatal("second promotion unexpectedly applied")
			}
			if !reflect.DeepEqual(original, types.RunRef{RunID: ctrl.runID, SessionID: ctrl.sessionID, State: types.RunStateFailed, Source: types.ProtocolSourceRunner}) {
				t.Fatal("terminal source reference mutated")
			}
		})
	}
}
