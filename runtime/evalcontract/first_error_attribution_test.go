package evalcontract

import (
	"reflect"
	"strings"
	"testing"
)

func TestNormalizeFirstErrorAttributionCanonicalizesFirstErrorAndBoundary(t *testing.T) {
	in := validFirstErrorAttribution()

	normalized, identity, err := NormalizeFirstErrorAttribution(in)
	if err != nil {
		t.Fatal(err)
	}
	if identity == "" {
		t.Fatal("expected stable attribution identity")
	}
	if normalized.Version != FirstErrorAttributionVersionV1 ||
		normalized.Correlation.CorpusItemID != "item-1" ||
		normalized.Correlation.BadcaseID != "badcase-1" ||
		normalized.Correlation.RunID != "run-1" ||
		normalized.FirstError.StepID != "step-2" ||
		normalized.FirstError.Kind != "tool_selection" ||
		normalized.Cause.Owner != "model" ||
		normalized.Cause.Primary != "tool_mismatch" ||
		normalized.Recoverability != "recoverable" ||
		normalized.PrefixDigest != "sha256:prefix" {
		t.Fatalf("unexpected canonical attribution: %#v", normalized)
	}
	if got := normalized.Cause.Secondary; len(got) != 2 || got[0].Rank != 1 || got[0].Code != "retrieval_noise" || got[1].Rank != 2 || got[1].Code != "policy_preference" {
		t.Fatalf("secondary causes are not canonical: %#v", got)
	}
	if got := normalized.Boundary.AcceptableActions; len(got) != 2 || got[0].ID != "ask-user" || got[1].ID != "search" {
		t.Fatalf("acceptable actions are not canonical: %#v", got)
	}
	if got := normalized.Evidence; len(got) != 2 || got[0].Reference.Owner != "policy" || got[1].Reference.Owner != "runtime" {
		t.Fatalf("evidence is not canonical: %#v", got)
	}
}

func TestNormalizeFirstErrorAttributionEquivalentInputOrderingProducesSameIdentity(t *testing.T) {
	left := validFirstErrorAttribution()
	right := validFirstErrorAttribution()

	reverseRankedCauses(right.Cause.Secondary)
	reverseReferences(right.Boundary.AcceptableActions)
	reverseReferences(right.Boundary.ForbiddenActions)
	reverseReferences(right.Boundary.RequiredEvidence)
	reverseReferences(right.Boundary.SafetyConstraints)
	reverseEvidence(right.Evidence)

	normalizedLeft, leftIdentity, err := NormalizeFirstErrorAttribution(left)
	if err != nil {
		t.Fatal(err)
	}
	normalizedRight, rightIdentity, err := NormalizeFirstErrorAttribution(right)
	if err != nil {
		t.Fatal(err)
	}
	if leftIdentity != rightIdentity || !reflect.DeepEqual(normalizedLeft, normalizedRight) {
		t.Fatalf("equivalent ordering changed normalization:\nleft:  %#v (%s)\nright: %#v (%s)", normalizedLeft, leftIdentity, normalizedRight, rightIdentity)
	}
}

func TestNormalizeFirstErrorAttributionRejectsMissingIdentity(t *testing.T) {
		tests := []struct {
		name   string
		mutate func(*FirstErrorAttribution)
	}{
		{
			name: "corpus item association",
			mutate: func(in *FirstErrorAttribution) { in.Correlation.CorpusItemID = "" },
		},
		{name: "badcase association", mutate: func(in *FirstErrorAttribution) { in.Correlation.BadcaseID = "" }},
		{name: "first-error step", mutate: func(in *FirstErrorAttribution) { in.FirstError.StepID = "" }},
		{name: "first-error ordinal", mutate: func(in *FirstErrorAttribution) { in.FirstError.Ordinal = nil }},
		{name: "first-error kind", mutate: func(in *FirstErrorAttribution) { in.FirstError.Kind = "" }},
		{name: "root-cause owner", mutate: func(in *FirstErrorAttribution) { in.Cause.Owner = "" }},
		{name: "primary cause", mutate: func(in *FirstErrorAttribution) { in.Cause.Primary = "" }},
		{name: "prefix digest", mutate: func(in *FirstErrorAttribution) { in.PrefixDigest = "" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := validFirstErrorAttribution()
			tt.mutate(&in)

			normalized, identity, err := NormalizeFirstErrorAttribution(in)
			if err == nil || err.Error() != ReasonFirstErrorSchemaDrift {
				t.Fatalf("expected %s, got normalized=%#v identity=%q err=%v", ReasonFirstErrorSchemaDrift, normalized, identity, err)
			}
			if identity != "" || !reflect.DeepEqual(normalized, FirstErrorAttribution{}) {
				t.Fatalf("schema rejection emitted partial attribution: %#v %q", normalized, identity)
			}
		})
	}
}

func TestNormalizeFirstErrorAttributionRejectsInvalidConfidence(t *testing.T) {
	tests := []struct {
		name       string
		confidence int
	}{
		{name: "below zero", confidence: -1},
		{name: "above 10000", confidence: 10001},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := validFirstErrorAttribution()
			in.ConfidenceBasisPoints = tt.confidence

			assertFirstErrorAttributionReason(t, in, ReasonFirstErrorSchemaDrift)
		})
	}
}

func TestNormalizeFirstErrorAttributionRejectsCollectionAndSerializedSizeBounds(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*FirstErrorAttribution)
	}{
		{
			name: "secondary causes above 16",
			mutate: func(in *FirstErrorAttribution) {
				in.Cause.Secondary = rankedCauses(17)
			},
		},
		{
			name: "evidence above 32",
			mutate: func(in *FirstErrorAttribution) {
				in.Boundary.RequiredEvidence = nil
				in.Evidence = evidenceReferences(33)
			},
		},
		{
			name: "acceptable actions above 32",
			mutate: func(in *FirstErrorAttribution) {
				in.Boundary.AcceptableActions = boundaryReferences("tool", "acceptable", 33)
			},
		},
		{
			name: "forbidden actions above 32",
			mutate: func(in *FirstErrorAttribution) {
				in.Boundary.ForbiddenActions = boundaryReferences("tool", "forbidden", 33)
			},
		},
		{
			name: "required evidence above 32",
			mutate: func(in *FirstErrorAttribution) {
				in.Boundary.RequiredEvidence = boundaryReferences("runtime", "required", 33)
			},
		},
		{
			name: "safety constraints above 32",
			mutate: func(in *FirstErrorAttribution) {
				in.Boundary.SafetyConstraints = boundaryReferences("policy", "constraint", 33)
			},
		},
		{
			name: "canonical serialization above 64 KiB",
			mutate: func(in *FirstErrorAttribution) {
				in.PrefixDigest = strings.Repeat("d", 64*1024)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := validFirstErrorAttribution()
			tt.mutate(&in)

			assertFirstErrorAttributionReason(t, in, ReasonFirstErrorSchemaDrift)
		})
	}
}

func TestNormalizeFirstErrorAttributionRejectsActionBoundaryConflict(t *testing.T) {
	in := validFirstErrorAttribution()
	in.Boundary.ForbiddenActions = append(in.Boundary.ForbiddenActions, AttributionReference{
		Kind:   " tool ",
		Owner:  " tool ",
		ID:     " search ",
		Digest: " sha256:search ",
	})

	assertFirstErrorAttributionReason(t, in, ReasonTrajectoryActionBoundaryConflict)
}

func TestNormalizeFirstErrorAttributionRejectsMissingRequiredEvidence(t *testing.T) {
	in := validFirstErrorAttribution()
	in.Boundary.RequiredEvidence = append(in.Boundary.RequiredEvidence, AttributionReference{
		Kind:  "event",
		Owner: "runtime",
		ID:    "event-missing",
	})

	assertFirstErrorAttributionReason(t, in, ReasonTrajectoryRequiredEvidenceMissing)
}

func TestNormalizeFirstErrorAttributionRejectsConflictingDuplicateEvidence(t *testing.T) {
	tests := []struct {
		name      string
		duplicate AttributionEvidence
	}{
		{
			name: "digest conflict",
			duplicate: AttributionEvidence{Reference: AttributionReference{
				Kind: "event", Owner: "runtime", ID: "event-1", Digest: "sha256:other", Version: "event.v1",
			}},
		},
		{
			name: "version conflict",
			duplicate: AttributionEvidence{Reference: AttributionReference{
				Kind: "event", Owner: "runtime", ID: "event-1", Digest: "sha256:event", Version: "event.v2",
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := validFirstErrorAttribution()
			in.Evidence = append(in.Evidence, tt.duplicate)

			assertFirstErrorAttributionReason(t, in, ReasonFirstErrorEvidenceConflict)
		})
	}
}

func TestNormalizeFirstErrorAttributionRejectsBodyBearingEvidence(t *testing.T) {
	in := validFirstErrorAttribution()
	in.Evidence[0].Body = "raw tool output"

	assertFirstErrorAttributionReason(t, in, ReasonFirstErrorPrivacyViolation)
}

func validFirstErrorAttribution() FirstErrorAttribution {
	ordinal := 2
	return FirstErrorAttribution{
		Version: " EVAL_FIRST_ERROR_ATTRIBUTION.V1 ",
		Correlation: AttributionCorrelation{
			CorpusItemID: " item-1 ",
			BadcaseID:    " badcase-1 ",
			RunID:        " run-1 ",
		},
		FirstError: FirstErrorIdentity{
			StepID:  " step-2 ",
			Ordinal: &ordinal,
			Kind:    " TOOL_SELECTION ",
		},
		Cause: FirstErrorCause{
			Owner:   " MODEL ",
			Primary: " TOOL_MISMATCH ",
			Secondary: []RankedCause{
				{Rank: 2, Code: " POLICY_PREFERENCE "},
				{Rank: 1, Code: " RETRIEVAL_NOISE "},
			},
		},
		Recoverability:        " RECOVERABLE ",
		ConfidenceBasisPoints: 8750,
		PrefixDigest:          " sha256:prefix ",
		Boundary: TrajectoryDecisionBoundary{
			AcceptableActions: []AttributionReference{
				{Kind: "tool", Owner: "tool", ID: " search ", Digest: " sha256:search "},
				{Kind: "interaction", Owner: "runtime", ID: " ask-user ", Digest: " sha256:ask "},
			},
			ForbiddenActions: []AttributionReference{
				{Kind: "tool", Owner: "tool", ID: "delete", Digest: "sha256:delete"},
				{Kind: "policy", Owner: "policy", ID: "bypass-review", Digest: "sha256:bypass"},
			},
			RequiredEvidence: []AttributionReference{
				{Kind: "event", Owner: "runtime", ID: "event-1", Digest: "sha256:event", Version: "event.v1"},
				{Kind: "policy", Owner: "policy", ID: "policy-1", Digest: "sha256:policy", Version: "policy.v1"},
			},
			SafetyConstraints: []AttributionReference{
				{Kind: "policy", Owner: "policy", ID: "no-destructive-write", Digest: "sha256:no-write"},
				{Kind: "policy", Owner: "policy", ID: "review-required", Digest: "sha256:review"},
			},
		},
		Evidence: []AttributionEvidence{
			{Reference: AttributionReference{Kind: "event", Owner: "runtime", ID: " event-1 ", Digest: " sha256:event ", Version: " event.v1 "}},
			{Reference: AttributionReference{Kind: "policy", Owner: "policy", ID: " policy-1 ", Digest: " sha256:policy ", Version: " policy.v1 "}},
		},
	}
}

func assertFirstErrorAttributionReason(t *testing.T, in FirstErrorAttribution, reason string) {
	t.Helper()
	normalized, identity, err := NormalizeFirstErrorAttribution(in)
	if err == nil || err.Error() != reason {
		t.Fatalf("expected %s, got normalized=%#v identity=%q err=%v", reason, normalized, identity, err)
	}
	if identity != "" || !reflect.DeepEqual(normalized, FirstErrorAttribution{}) {
		t.Fatalf("rejection emitted partial attribution: %#v %q", normalized, identity)
	}
}

func rankedCauses(count int) []RankedCause {
	causes := make([]RankedCause, count)
	for i := range causes {
		causes[i] = RankedCause{Rank: i + 1, Code: "cause-" + strings.Repeat("x", i+1)}
	}
	return causes
}

func evidenceReferences(count int) []AttributionEvidence {
	evidence := make([]AttributionEvidence, count)
	for i := range evidence {
		evidence[i] = AttributionEvidence{Reference: AttributionReference{
			Kind: "event", Owner: "runtime", ID: "event-" + strings.Repeat("x", i+1), Digest: "sha256:event",
		}}
	}
	return evidence
}

func boundaryReferences(kind, prefix string, count int) []AttributionReference {
	references := make([]AttributionReference, count)
	for i := range references {
		references[i] = AttributionReference{Kind: kind, Owner: kind, ID: prefix + "-" + strings.Repeat("x", i+1)}
	}
	return references
}

func reverseRankedCauses(values []RankedCause) {
	for left, right := 0, len(values)-1; left < right; left, right = left+1, right-1 {
		values[left], values[right] = values[right], values[left]
	}
}

func reverseReferences(values []AttributionReference) {
	for left, right := 0, len(values)-1; left < right; left, right = left+1, right-1 {
		values[left], values[right] = values[right], values[left]
	}
}

func reverseEvidence(values []AttributionEvidence) {
	for left, right := 0, len(values)-1; left < right; left, right = left+1, right-1 {
		values[left], values[right] = values[right], values[left]
	}
}
