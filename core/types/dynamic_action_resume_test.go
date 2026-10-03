package types

import (
	"encoding/json"
	"strings"
	"testing"
)

func validDynamicActionReference() DynamicActionReference {
	return DynamicActionReference{
		Token: "opaque-token-1", Kind: "application.prepare", Resumable: true,
		RunID: "run-1", SessionID: "session-1", Iteration: 3, CallID: "call-1",
		Source: "application-adapter", Digest: "digest-1",
	}
}

func validRunCheckpoint() RunCheckpoint {
	ref := validDynamicActionReference()
	return RunCheckpoint{
		Version: DynamicActionResumeProtocolVersion, CheckpointID: "checkpoint-1",
		RunID: ref.RunID, SessionID: ref.SessionID, State: RunStateInputRequired,
		Iteration: ref.Iteration, PendingAction: &ref, Digest: ref.Digest,
	}
}

func TestDynamicActionReferenceValidateRequiresBoundedCorrelation(t *testing.T) {
	valid := validDynamicActionReference()
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid reference rejected: %v", err)
	}
	cases := []struct {
		name string
		edit func(*DynamicActionReference)
	}{
		{"empty token", func(ref *DynamicActionReference) { ref.Token = " " }},
		{"oversized token", func(ref *DynamicActionReference) { ref.Token = strings.Repeat("x", DynamicActionMaxTokenBytes+1) }},
		{"oversized kind", func(ref *DynamicActionReference) { ref.Kind = strings.Repeat("x", DynamicActionMaxKindBytes+1) }},
		{"oversized digest", func(ref *DynamicActionReference) { ref.Digest = strings.Repeat("x", DynamicActionMaxDigestBytes+1) }},
		{"oversized correlation", func(ref *DynamicActionReference) {
			ref.CallID = strings.Repeat("x", DynamicActionMaxCorrelationBytes+1)
		}},
		{"missing correlation", func(ref *DynamicActionReference) { ref.RunID = "" }},
		{"negative iteration", func(ref *DynamicActionReference) { ref.Iteration = -1 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ref := valid
			tc.edit(&ref)
			if err := ref.Validate(); err == nil {
				t.Fatalf("Validate() unexpectedly succeeded")
			}
		})
	}
}

func TestDynamicActionProtocolResumePathAndStateValidation(t *testing.T) {
	if err := ValidateDynamicActionRunStateTransition(RunStateWorking, RunStateInputRequired); err != nil {
		t.Fatalf("working -> input_required rejected: %v", err)
	}
	if err := ValidateRunStateTransitionVia(RunStateInputRequired, RunStateWorking, DynamicActionResumePath); err != nil {
		t.Fatalf("dynamic resume path rejected: %v", err)
	}
	if err := ValidateRunStateTransitionVia(RunStateInputRequired, RunStateWorking, "retry"); err == nil {
		t.Fatal("non-dynamic resume path unexpectedly accepted")
	}
	for _, state := range []RunState{RunStateCompleted, RunStateFailed, RunStateCanceled, RunStateWorking} {
		if err := ValidateDynamicActionResumeState(state); err == nil {
			t.Fatalf("resume from %q unexpectedly accepted", state)
		}
	}
	if err := ValidateDynamicActionResumeState(RunStateInputRequired); err != nil {
		t.Fatalf("input_required resume rejected: %v", err)
	}
}

func TestDynamicActionReferenceJSONRoundTripPreservesOptionalField(t *testing.T) {
	want := ToolResult{Content: "ok", PendingAction: func() *DynamicActionReference { ref := validDynamicActionReference(); return &ref }()}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got ToolResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.PendingAction == nil || got.PendingAction.Token != want.PendingAction.Token || got.PendingAction.Iteration != want.PendingAction.Iteration {
		t.Fatalf("pending action not preserved: %#v", got.PendingAction)
	}
}

func TestDynamicActionCheckpointRejectsDigestAndRunMismatch(t *testing.T) {
	checkpoint := validRunCheckpoint()
	if err := checkpoint.Validate(); err != nil {
		t.Fatalf("valid checkpoint rejected: %v", err)
	}
	checkpoint.PendingAction.Digest = "different"
	if err := checkpoint.Validate(); err == nil || !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("digest mismatch error = %v", err)
	}
	checkpoint = validRunCheckpoint()
	checkpoint.PendingAction.RunID = "other-run"
	if err := checkpoint.Validate(); err == nil || !strings.Contains(err.Error(), "correlation mismatch") {
		t.Fatalf("run mismatch error = %v", err)
	}
}

func TestResumeAdmissionDuplicateIsIdempotent(t *testing.T) {
	accepted := RunResumeAdmission{Status: RunResumeAdmissionAccepted, RunID: "run-1", SessionID: "session-1", CheckpointID: "checkpoint-1", IdempotencyKey: "idem-1", Decision: DynamicActionDecisionConfirm}
	duplicate := accepted
	duplicate.Status = RunResumeAdmissionDuplicate
	if err := accepted.Validate(); err != nil {
		t.Fatalf("accepted admission rejected: %v", err)
	}
	if err := duplicate.Validate(); err != nil {
		t.Fatalf("duplicate admission rejected: %v", err)
	}
	if !duplicate.IsDuplicateOf(accepted) {
		t.Fatalf("duplicate admission not idempotent: %#v / %#v", duplicate, accepted)
	}
	unsupported := accepted
	unsupported.Decision = DynamicActionDecisionKind("unknown")
	if err := unsupported.Validate(); err == nil {
		t.Fatalf("unsupported decision unexpectedly accepted")
	}
}

func TestLegacyToolResultWithoutPendingActionStillDecodes(t *testing.T) {
	var result ToolResult
	if err := json.Unmarshal([]byte(`{"content":"legacy","structured":{"ok":true}}`), &result); err != nil {
		t.Fatalf("legacy result decode: %v", err)
	}
	if result.Content != "legacy" || result.PendingAction != nil {
		t.Fatalf("legacy result changed: %#v", result)
	}
}
