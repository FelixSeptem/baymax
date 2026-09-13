package evalcontract

import "testing"

func TestNormalizeContinuityProjectionSortsAndRejectsBodies(t *testing.T) {
	projection := ContinuityProjection{
		Version:   ContinuityComparisonVersionV1,
		RunID:     "run-1",
		SessionID: "session-1",
		Phase:     ContinuityPhaseBaseline,
		Facts: []ContinuityFact{
			{Kind: ContinuityKindObjective, Owner: "runner", ID: " objective ", Digest: " d "},
			{Kind: ContinuityKindIdentityAgent, Owner: "runner", ID: "agent"},
		},
	}
	normalized, digest, err := NormalizeContinuityProjection(projection)
	if err != nil {
		t.Fatal(err)
	}
	if digest == "" || normalized.Facts[0].Kind != ContinuityKindIdentityAgent || normalized.Facts[1].ID != "objective" {
		t.Fatalf("unexpected normalization: %#v %q", normalized, digest)
	}
	projection.Facts[0].Body = "secret transcript"
	if _, _, err := NormalizeContinuityProjection(projection); err == nil || err.Error() != ReasonContinuityPrivacyViolation {
		t.Fatalf("expected privacy violation, got %v", err)
	}
}

func TestCompareContinuityDetectsAxisDriftAndDuplicate(t *testing.T) {
	baseline := ContinuityProjection{Version: ContinuityComparisonVersionV1, RunID: "run-1", Phase: ContinuityPhaseBaseline, Facts: []ContinuityFact{
		{Kind: ContinuityKindObjective, Owner: "runner", ID: "objective", Digest: "one"},
		{Kind: ContinuityKindWorkspaceBinding, Owner: "scheduler", ID: "workspace", Digest: "before"},
	}}
	candidate := ContinuityProjection{Version: ContinuityComparisonVersionV1, RunID: "run-1", Phase: ContinuityPhaseCandidate, Facts: []ContinuityFact{
		{Kind: ContinuityKindObjective, Owner: "runner", ID: "objective", Digest: "two"},
		{Kind: ContinuityKindWorkspaceBinding, Owner: "scheduler", ID: "workspace-new", Digest: "before"},
	}}
	result, err := CompareContinuity(baseline, candidate)
	if err != nil {
		t.Fatal(err)
	}
	if result.Passed || len(result.Drifts) != 2 || result.Drifts[0].Class != ReasonContinuityObjectiveDrift || result.Drifts[1].Class != ReasonContinuityWorkspaceBindingDrift {
		t.Fatalf("unexpected drift result: %#v", result)
	}
	candidate.Facts = append(candidate.Facts, candidate.Facts[0])
	if _, err := CompareContinuity(baseline, candidate); err == nil || err.Error() != ReasonContinuityDuplicateReference {
		t.Fatalf("expected duplicate rejection, got %v", err)
	}
}

func TestCompareContinuityRequiresRunAndPhase(t *testing.T) {
	_, err := CompareContinuity(ContinuityProjection{Version: ContinuityComparisonVersionV1, Phase: ContinuityPhaseBaseline}, ContinuityProjection{Version: ContinuityComparisonVersionV1, RunID: "r", Phase: ContinuityPhaseCandidate})
	if err == nil || err.Error() != ReasonContinuitySchemaDrift {
		t.Fatalf("expected schema drift, got %v", err)
	}
}
