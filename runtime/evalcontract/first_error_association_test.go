package evalcontract

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestFirstErrorAttributionAssociationsAreAdditiveAndPreserveLegacySemantics(t *testing.T) {
	legacyBadcase := Badcase{ID: "badcase-1", Category: "tool", ExpectedDigest: "same"}
	classified, err := ClassifyBadcase(legacyBadcase, "same", true)
	if err != nil || classified.Status != BadcaseStatusReplayable {
		t.Fatalf("legacy badcase behavior changed: %#v err=%v", classified, err)
	}
	assertJSONFieldAbsent(t, legacyBadcase, "first_error_attribution")

	legacyExperiment := validAssociationExperiment()
	legacyResult, err := CompareExperiments(legacyExperiment)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONFieldAbsent(t, legacyExperiment, "first_error_attribution")

	attribution := validFirstErrorAttribution()
	attribution.Correlation.ExperimentID = legacyExperiment.ID
	reference := mustFirstErrorAttributionReference(t, attribution)
	attributedBadcase := legacyBadcase
	attributedBadcase.CorpusItemID = attribution.Correlation.CorpusItemID
	attributedBadcase.FirstErrorAttribution = &reference
	classified, err = ClassifyBadcase(attributedBadcase, "same", true)
	if err != nil || classified.Status != BadcaseStatusReplayable || classified.FirstErrorAttribution == nil || classified.FirstErrorAttribution.ID != reference.ID {
		t.Fatalf("attributed badcase changed reproduction semantics: %#v err=%v", classified, err)
	}

	attributedExperiment := legacyExperiment
	attributedExperiment.FirstErrorAttribution = &reference
	attributedResult, err := CompareExperiments(attributedExperiment)
	if err != nil {
		t.Fatal(err)
	}
	if attributedResult.FirstErrorAttribution == nil || attributedResult.FirstErrorAttribution.ID != reference.ID {
		t.Fatalf("experiment result omitted attribution reference: %#v", attributedResult)
	}
	legacyComparable := legacyResult
	attributedComparable := attributedResult
	attributedComparable.FirstErrorAttribution = nil
	if !reflect.DeepEqual(legacyComparable, attributedComparable) {
		t.Fatalf("attribution changed legacy aggregate/digest semantics:\nlegacy:     %#v\nattributed: %#v", legacyComparable, attributedComparable)
	}
}

func TestValidateFirstErrorAttributionAssociationChecksEveryCorrelationAxisWithoutMutation(t *testing.T) {
	attribution := validFirstErrorAttribution()
	attribution.Correlation.ExperimentID = "experiment-1"
	reference := mustFirstErrorAttributionReference(t, attribution)
	attributionBefore := cloneFirstErrorAttribution(t, attribution)
	referenceBefore := reference

	if err := ValidateFirstErrorAttributionAssociation(reference, attribution); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(attribution, attributionBefore) || !reflect.DeepEqual(reference, referenceBefore) {
		t.Fatalf("association validation mutated source records")
	}

	tests := []struct {
		name   string
		mutate func(*FirstErrorAttributionReference)
	}{
		{name: "attribution identity", mutate: func(ref *FirstErrorAttributionReference) { ref.ID = "sha256:other" }},
		{name: "corpus item", mutate: func(ref *FirstErrorAttributionReference) { ref.CorpusItemID = "item-2" }},
		{name: "badcase", mutate: func(ref *FirstErrorAttributionReference) { ref.BadcaseID = "badcase-2" }},
		{name: "run", mutate: func(ref *FirstErrorAttributionReference) { ref.RunID = "run-2" }},
		{name: "first-error step", mutate: func(ref *FirstErrorAttributionReference) { ref.StepID = "step-3" }},
		{name: "experiment", mutate: func(ref *FirstErrorAttributionReference) { ref.ExperimentID = "experiment-2" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			drifted := reference
			tt.mutate(&drifted)
			if err := ValidateFirstErrorAttributionAssociation(drifted, attribution); err == nil || err.Error() != ReasonFirstErrorCorrelationDrift {
				t.Fatalf("expected correlation drift, got %v", err)
			}
		})
	}
}

func TestValidateFeedbackKeepsAttributionRecommendationReviewOnly(t *testing.T) {
	attribution := validFirstErrorAttribution()
	attribution.Correlation.ExperimentID = "experiment-1"
	reference := mustFirstErrorAttributionReference(t, attribution)
	recommendation := FeedbackRecommendation{
		Version:               FeedbackVersionV1,
		ID:                    "feedback-1",
		ExperimentID:          "experiment-1",
		BadcaseID:             "badcase-1",
		ReviewerID:            "reviewer-1",
		DecisionContext:       "reviewed bounded evidence",
		Status:                "approved",
		Recommendation:        "prefer the evidence-backed action",
		ApplicationMode:       FeedbackApplicationReviewOnly,
		FirstErrorAttribution: &reference,
		Evidence: []AttributionReference{
			{Kind: "event", Owner: "runtime", ID: "event-1", Digest: "sha256:event"},
		},
	}
	before := recommendation
	before.Evidence = append([]AttributionReference(nil), recommendation.Evidence...)

	if err := ValidateFeedback(recommendation); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(recommendation, before) {
		t.Fatal("feedback validation mutated recommendation")
	}

	automatic := recommendation
	automatic.ApplicationMode = "automatic"
	if err := ValidateFeedback(automatic); err == nil || err.Error() != ReasonFeedbackAutoApplyForbidden {
		t.Fatalf("expected automatic application rejection, got %v", err)
	}

	missingEvidence := recommendation
	missingEvidence.Evidence = nil
	if err := ValidateFeedback(missingEvidence); err == nil || err.Error() != ReasonTrajectoryRequiredEvidenceMissing {
		t.Fatalf("expected evidence requirement, got %v", err)
	}

	historical := FeedbackRecommendation{ID: "legacy", ExperimentID: "experiment-1", ReviewerID: "reviewer-1", DecisionContext: "legacy review", Status: "approved"}
	if err := ValidateFeedback(historical); err != nil {
		t.Fatalf("historical recommendation behavior changed: %v", err)
	}
	assertJSONFieldAbsent(t, historical, "first_error_attribution")
}

func validAssociationExperiment() Experiment {
	return Experiment{
		ID:            "experiment-1",
		CorpusVersion: CorpusVersionV1,
		RunBatch:      "batch-1",
		Rubric:        Rubric{Name: "quality", Version: "1"},
		ExecutionMode: "local",
		Shards: []ShardMetric{
			{ShardID: "shard-1", ItemID: "item-1", Digest: "sha256:metric", Passed: 1, Total: 1},
		},
	}
}

func mustFirstErrorAttributionReference(t *testing.T, attribution FirstErrorAttribution) FirstErrorAttributionReference {
	t.Helper()
	reference, err := NewFirstErrorAttributionReference(attribution)
	if err != nil {
		t.Fatal(err)
	}
	return reference
}

func assertJSONFieldAbsent(t *testing.T, value any, field string) {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var object map[string]any
	if err := json.Unmarshal(raw, &object); err != nil {
		t.Fatal(err)
	}
	if _, exists := object[field]; exists {
		t.Fatalf("historical JSON unexpectedly contains %q: %s", field, raw)
	}
}
