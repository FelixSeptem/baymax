package evalcontract

import (
	"reflect"
	"sort"
	"testing"
)

func TestCompareFirstErrorAttributionClassifiesEachSemanticDrift(t *testing.T) {
	tests := []struct {
		name   string
		reason string
		mutate func(*FirstErrorAttribution)
	}{
		{
			name:   "first-error step",
			reason: ReasonFirstErrorStepDrift,
			mutate: func(in *FirstErrorAttribution) { in.FirstError.StepID = "step-3" },
		},
		{
			name:   "first-error ordinal",
			reason: ReasonFirstErrorStepDrift,
			mutate: func(in *FirstErrorAttribution) { ordinal := 3; in.FirstError.Ordinal = &ordinal },
		},
		{
			name:   "first-error kind",
			reason: ReasonFirstErrorKindDrift,
			mutate: func(in *FirstErrorAttribution) { in.FirstError.Kind = "tool_input" },
		},
		{
			name:   "trajectory prefix",
			reason: ReasonFirstErrorPrefixDrift,
			mutate: func(in *FirstErrorAttribution) { in.PrefixDigest = "sha256:other-prefix" },
		},
		{
			name:   "root-cause owner",
			reason: ReasonFirstErrorOwnerDrift,
			mutate: func(in *FirstErrorAttribution) { in.Cause.Owner = "tool" },
		},
		{
			name:   "primary cause",
			reason: ReasonFirstErrorCauseDrift,
			mutate: func(in *FirstErrorAttribution) { in.Cause.Primary = "invalid_arguments" },
		},
		{
			name:   "ranked secondary causes",
			reason: ReasonFirstErrorCauseDrift,
			mutate: func(in *FirstErrorAttribution) { in.Cause.Secondary[0].Code = "retrieval_scope_error" },
		},
		{
			name:   "acceptable action boundary",
			reason: ReasonTrajectoryActionBoundaryDrift,
			mutate: func(in *FirstErrorAttribution) { in.Boundary.AcceptableActions[0].Digest = "sha256:other-action" },
		},
		{
			name:   "required evidence boundary",
			reason: ReasonTrajectoryRequiredEvidenceDrift,
			mutate: func(in *FirstErrorAttribution) {
				in.Boundary.RequiredEvidence = nil
			},
		},
		{
			name:   "evidence reference",
			reason: ReasonFirstErrorEvidenceDrift,
			mutate: func(in *FirstErrorAttribution) {
				in.Evidence[1].Reference.Digest = "sha256:other-policy"
			},
		},
		{
			name:   "recoverability",
			reason: ReasonFirstErrorRecoverabilityDrift,
			mutate: func(in *FirstErrorAttribution) { in.Recoverability = "non_recoverable" },
		},
		{
			name:   "confidence",
			reason: ReasonFirstErrorConfidenceDrift,
			mutate: func(in *FirstErrorAttribution) { in.ConfidenceBasisPoints = 7000 },
		},
		{
			name:   "evaluation correlation",
			reason: ReasonFirstErrorCorrelationDrift,
			mutate: func(in *FirstErrorAttribution) { in.Correlation.RunID = "run-2" },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseline := validFirstErrorAttribution()
			candidate := validFirstErrorAttribution()
			tt.mutate(&candidate)

			comparison, err := CompareFirstErrorAttribution(baseline, candidate)
			if err != nil {
				t.Fatal(err)
			}
			assertFirstErrorDriftReasons(t, comparison, tt.reason)
		})
	}
}

func TestCompareFirstErrorAttributionPreservesAllIndependentDrifts(t *testing.T) {
	baseline := validFirstErrorAttribution()
	candidate := validFirstErrorAttribution()
	candidate.FirstError.StepID = "step-3"
	candidate.FirstError.Kind = "policy"
	candidate.Cause.Owner = "policy"
	candidate.Cause.Primary = "policy_denied"
	candidate.Recoverability = "non_recoverable"
	candidate.ConfidenceBasisPoints = 2000
	candidate.PrefixDigest = "sha256:other-prefix"

	comparison, err := CompareFirstErrorAttribution(baseline, candidate)
	if err != nil {
		t.Fatal(err)
	}
	assertFirstErrorDriftReasons(t, comparison,
		ReasonFirstErrorCauseDrift,
		ReasonFirstErrorConfidenceDrift,
		ReasonFirstErrorKindDrift,
		ReasonFirstErrorOwnerDrift,
		ReasonFirstErrorPrefixDrift,
		ReasonFirstErrorRecoverabilityDrift,
		ReasonFirstErrorStepDrift,
	)
}

func TestCompareFirstErrorAttributionRunStreamParity(t *testing.T) {
	run := validFirstErrorAttribution()
	run.Correlation.ExecutionMode = "run"
	stream := validFirstErrorAttribution()
	stream.Correlation.ExecutionMode = "stream"

	equivalent, err := CompareFirstErrorAttribution(run, stream)
	if err != nil {
		t.Fatal(err)
	}
	if !equivalent.Passed || len(equivalent.Drifts) != 0 || equivalent.BaselineIdentity != equivalent.CandidateIdentity {
		t.Fatalf("equivalent Run/Stream attribution drifted: %#v", equivalent)
	}

	stream.FirstError.Kind = "tool_input"
	drifted, err := CompareFirstErrorAttribution(run, stream)
	if err != nil {
		t.Fatal(err)
	}
	assertFirstErrorDriftReasons(t, drifted, ReasonFirstErrorKindDrift, ReasonFirstErrorRunStreamParityDrift)
}

func TestCompareFirstErrorAttributionIsIdempotentAndDoesNotMutateInputs(t *testing.T) {
	baseline := validFirstErrorAttribution()
	candidate := validFirstErrorAttribution()
	candidate.Cause.Primary = "invalid_arguments"
	baselineBefore := cloneFirstErrorAttribution(t, baseline)
	candidateBefore := cloneFirstErrorAttribution(t, candidate)

	first, err := CompareFirstErrorAttribution(baseline, candidate)
	if err != nil {
		t.Fatal(err)
	}
	second, err := CompareFirstErrorAttribution(baseline, candidate)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("repeated comparison changed output:\nfirst:  %#v\nsecond: %#v", first, second)
	}
	if !reflect.DeepEqual(baseline, baselineBefore) || !reflect.DeepEqual(candidate, candidateBefore) {
		t.Fatalf("comparison mutated input:\nbaseline:  %#v\ncandidate: %#v", baseline, candidate)
	}
}

func assertFirstErrorDriftReasons(t *testing.T, comparison FirstErrorComparison, expected ...string) {
	t.Helper()
	if comparison.Passed {
		t.Fatalf("expected drift, got pass: %#v", comparison)
	}
	actual := make([]string, len(comparison.Drifts))
	for i, drift := range comparison.Drifts {
		actual[i] = drift.Reason
	}
	sort.Strings(actual)
	sort.Strings(expected)
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("drift reasons = %v, want %v", actual, expected)
	}
}

func cloneFirstErrorAttribution(t *testing.T, in FirstErrorAttribution) FirstErrorAttribution {
	t.Helper()
	clone := in
	clone.Cause.Secondary = append([]RankedCause(nil), in.Cause.Secondary...)
	clone.Boundary.AcceptableActions = append([]AttributionReference(nil), in.Boundary.AcceptableActions...)
	clone.Boundary.ForbiddenActions = append([]AttributionReference(nil), in.Boundary.ForbiddenActions...)
	clone.Boundary.RequiredEvidence = append([]AttributionReference(nil), in.Boundary.RequiredEvidence...)
	clone.Boundary.SafetyConstraints = append([]AttributionReference(nil), in.Boundary.SafetyConstraints...)
	clone.Evidence = append([]AttributionEvidence(nil), in.Evidence...)
	return clone
}
