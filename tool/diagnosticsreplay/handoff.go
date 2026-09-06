package diagnosticsreplay

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/FelixSeptem/baymax/context/handoff"
)

const ContextHandoffFixtureVersionV1 = "context_handoff.v1"

const (
	ReasonCodeHandoffFactLoss             = "handoff_fact_loss"
	ReasonCodeHandoffReferenceLoss        = "handoff_reference_loss"
	ReasonCodeHandoffCutInvalid           = "handoff_cut_invalid"
	ReasonCodeHandoffQualityBelow         = "handoff_quality_below_threshold"
	ReasonCodeHandoffSchemaDrift          = "handoff_schema_drift"
	ReasonCodeHandoffRestoreNonIdempotent = "handoff_restore_non_idempotent"
	ReasonCodeHandoffRunStreamMismatch    = "handoff_run_stream_mismatch"
)

type HandoffFixture struct {
	Version            string             `json:"version"`
	Handoff            handoff.Handoff    `json:"handoff"`
	ExpectedFacts      []string           `json:"expected_facts,omitempty"`
	ExpectedReferences []string           `json:"expected_references,omitempty"`
	Run                HandoffObservation `json:"run"`
	Stream             HandoffObservation `json:"stream"`
}

type HandoffObservation struct {
	NextActions []string `json:"next_actions,omitempty"`
	Restored    bool     `json:"restored"`
	HandoffID   string   `json:"handoff_id"`
}

func EvaluateHandoffFixtureJSON(raw []byte) (HandoffFixture, error) {
	var fixture HandoffFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		return HandoffFixture{}, &ValidationError{Code: ReasonCodeHandoffSchemaDrift, Message: err.Error()}
	}
	if fixture.Version != ContextHandoffFixtureVersionV1 {
		return HandoffFixture{}, &ValidationError{Code: ReasonCodeHandoffSchemaDrift, Message: fmt.Sprintf("unsupported fixture version %q", fixture.Version)}
	}
	if err := fixture.Handoff.Validate(handoff.DefaultLimits()); err != nil {
		return HandoffFixture{}, &ValidationError{Code: ReasonCodeHandoffSchemaDrift, Message: err.Error()}
	}
	actualFacts := make(map[string]struct{}, len(fixture.Handoff.Facts))
	for _, fact := range fixture.Handoff.Facts {
		actualFacts[fact.Value] = struct{}{}
	}
	for _, expected := range fixture.ExpectedFacts {
		if _, ok := actualFacts[expected]; !ok {
			return fixture, &ValidationError{Code: ReasonCodeHandoffFactLoss, Message: fmt.Sprintf("missing protected fact %q", expected)}
		}
	}
	actualRefs := make(map[string]struct{}, len(fixture.Handoff.References))
	for _, ref := range fixture.Handoff.References {
		actualRefs[string(ref.Kind)+":"+ref.ID] = struct{}{}
	}
	for _, expected := range fixture.ExpectedReferences {
		if _, ok := actualRefs[expected]; !ok {
			return fixture, &ValidationError{Code: ReasonCodeHandoffReferenceLoss, Message: fmt.Sprintf("missing handoff reference %q", expected)}
		}
	}
	if fixture.Handoff.Fallback != nil && strings.TrimSpace(fixture.Handoff.Fallback.Reason) == handoff.FallbackQualityBelowThreshold {
		return fixture, &ValidationError{Code: ReasonCodeHandoffQualityBelow, Message: "handoff quality below threshold"}
	}
	if fixture.Run.HandoffID != fixture.Stream.HandoffID || fixture.Run.Restored != fixture.Stream.Restored {
		return fixture, &ValidationError{Code: ReasonCodeHandoffRunStreamMismatch, Message: "run and stream handoff observations differ"}
	}
	return fixture, nil
}
