package runner

import (
	"context"
	"github.com/FelixSeptem/baymax/core/types"
	"testing"
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
