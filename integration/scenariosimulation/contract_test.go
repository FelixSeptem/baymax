package scenariosimulation

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNormalizeScenarioIsDeterministicAndIgnoresUnknownAdditiveFields(t *testing.T) {
	first := Scenario{
		Version: ScenarioVersionV1,
		ID:      "approval-recovery",
		Events: []PlannedEvent{
			{ID: "recover", Sequence: 2, Kind: EventRecovery, Owner: "runtime"},
			{ID: "approve", Sequence: 1, Kind: EventApproval, Owner: "host"},
		},
		Unknown: map[string]any{"future_extension": "ignored"},
	}
	second := Scenario{
		Version: ScenarioVersionV1,
		ID:      "approval-recovery",
		Events: []PlannedEvent{
			{ID: "approve", Sequence: 1, Kind: EventApproval, Owner: "host"},
			{ID: "recover", Sequence: 2, Kind: EventRecovery, Owner: "runtime"},
		},
	}

	normalizedFirst, firstDigest, err := NormalizeScenario(first)
	if err != nil {
		t.Fatalf("normalize first scenario: %v", err)
	}
	normalizedSecond, secondDigest, err := NormalizeScenario(second)
	if err != nil {
		t.Fatalf("normalize second scenario: %v", err)
	}
	if firstDigest != secondDigest {
		t.Fatalf("equivalent scenarios have different digests: %q != %q", firstDigest, secondDigest)
	}
	if normalizedFirst.Events[0].ID != "approve" || normalizedSecond.Events[0].ID != "approve" {
		t.Fatalf("events were not ordered by sequence: %#v %#v", normalizedFirst.Events, normalizedSecond.Events)
	}
}

func TestHistoricalResultPayloadWithoutSimulationFieldsRemainsReadable(t *testing.T) {
	const legacy = `{"version":"run_result.v1","scenario_id":"legacy","execution":{"status":"pass"}}`
	var result RunResult
	if err := json.Unmarshal([]byte(legacy), &result); err != nil {
		t.Fatalf("decode historical payload: %v", err)
	}
	if result.ScenarioID != "legacy" || result.Evidence.Status != "" || result.Outcome.Status != "" || result.Admission.Status != "" {
		t.Fatalf("historical nullable/default fields changed: %#v", result)
	}
}

func TestCompletionProjectionPreservesLateAndDuplicateOwnerClassifications(t *testing.T) {
	if got := ClassifyCompletion(CompletionReference{ID: "c", Committed: true}); got.Status != VerdictPass {
		t.Fatalf("committed completion = %#v", got)
	}
	if got := ClassifyCompletion(CompletionReference{ID: "c", Late: true}); got.Reason != ReasonLateCompletion {
		t.Fatalf("late completion = %#v", got)
	}
	if got := ClassifyCompletion(CompletionReference{ID: "c", Duplicate: true}); got.Reason != ReasonDuplicateCompletion {
		t.Fatalf("duplicate completion = %#v", got)
	}
}

func TestNormalizeScenarioRejectsMalformedAndOversizedInputsWithoutPartialResult(t *testing.T) {
	tests := []struct {
		name string
		in   Scenario
		want string
	}{
		{name: "missing identity", in: Scenario{Version: ScenarioVersionV1}, want: ReasonScenarioSchemaDrift},
		{name: "unsupported version", in: Scenario{Version: "scenario.v9", ID: "valid"}, want: ReasonScenarioVersionDrift},
		{name: "duplicate event identity", in: Scenario{Version: ScenarioVersionV1, ID: "duplicate", Events: []PlannedEvent{{ID: "same", Sequence: 1, Kind: EventApproval, Owner: "host"}, {ID: "same", Sequence: 2, Kind: EventRecovery, Owner: "runtime"}}}, want: ReasonScenarioSchemaDrift},
		{name: "too many events", in: Scenario{Version: ScenarioVersionV1, ID: "oversized", Events: make([]PlannedEvent, MaxScenarioEvents+1)}, want: ReasonScenarioSchemaDrift},
		{name: "oversized unknown extension", in: Scenario{Version: ScenarioVersionV1, ID: "oversized-extension", Unknown: map[string]any{"future": strings.Repeat("x", MaxSerializedBytes+1)}}, want: ReasonScenarioSchemaDrift},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, digest, err := NormalizeScenario(tc.in)
			if err == nil || err.Error() != tc.want {
				t.Fatalf("NormalizeScenario() error = %v, want %q", err, tc.want)
			}
			if got.ID != "" || len(got.Events) != 0 || digest != "" {
				t.Fatalf("malformed scenario emitted partial result: %#v digest=%q", got, digest)
			}
		})
	}
}

func TestVerifyRunResultKeepsExecutionOutcomeEvidenceAndAdmissionIndependent(t *testing.T) {
	result := RunResult{
		Version:    RunResultVersionV1,
		ScenarioID: "completed-without-outcome",
		Execution:  Verdict{Status: VerdictPass},
	}
	got, err := VerifyRunResult(result, []string{"checkpoint-1"}, true)
	if err == nil || err.Error() != ReasonEvidenceIncomplete {
		t.Fatalf("VerifyRunResult() error = %v, want %q", err, ReasonEvidenceIncomplete)
	}
	if got.Execution.Status != VerdictPass || got.Evidence.Status != VerdictIndeterminate || got.Outcome.Status != VerdictIndeterminate || got.Admission.Status != VerdictIndeterminate {
		t.Fatalf("verdict axes were conflated: %#v", got)
	}
}

func TestValidateEvidenceRejectsPrivacyAndConflictingReferences(t *testing.T) {
	tests := []struct {
		name string
		refs []EvidenceReference
		want string
	}{
		{name: "body-bearing", refs: []EvidenceReference{{Owner: "tool", ID: "call-1", Body: "private output"}}, want: ReasonPrivacyViolation},
		{name: "conflicting duplicate", refs: []EvidenceReference{{Owner: "tool", ID: "call-1", Digest: "sha256:a"}, {Owner: "tool", ID: "call-1", Digest: "sha256:b"}}, want: ReasonEvidenceConflict},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			verdict, err := ValidateEvidence(tc.refs, nil)
			if err == nil || err.Error() != tc.want || verdict.Reason != tc.want {
				t.Fatalf("ValidateEvidence() = %#v, %v; want reason %q", verdict, err, tc.want)
			}
		})
	}
}
