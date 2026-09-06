package diagnosticsreplay

import (
	"encoding/json"
	"testing"

	"github.com/FelixSeptem/baymax/context/handoff"
)

func TestEvaluateHandoffFixtureSuccess(t *testing.T) {
	h := handoff.Handoff{Version: handoff.VersionV1, RunID: "run-1", Cut: handoff.CutCheckpoint, Objective: "x"}
	raw, err := json.Marshal(HandoffFixture{Version: ContextHandoffFixtureVersionV1, Handoff: h, Run: HandoffObservation{HandoffID: "h1", Restored: true}, Stream: HandoffObservation{HandoffID: "h1", Restored: true}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := EvaluateHandoffFixtureJSON(raw); err != nil {
		t.Fatalf("EvaluateHandoffFixtureJSON() error = %v", err)
	}
}

func TestEvaluateHandoffFixtureClassifiesRunStreamMismatch(t *testing.T) {
	h := handoff.Handoff{Version: handoff.VersionV1, RunID: "run-1", Cut: handoff.CutCheckpoint, Objective: "x"}
	raw, _ := json.Marshal(HandoffFixture{Version: ContextHandoffFixtureVersionV1, Handoff: h, Run: HandoffObservation{HandoffID: "h1", Restored: true}, Stream: HandoffObservation{HandoffID: "h2", Restored: true}})
	_, err := EvaluateHandoffFixtureJSON(raw)
	if err == nil || err.(*ValidationError).Code != ReasonCodeHandoffRunStreamMismatch {
		t.Fatalf("error = %v, want %s", err, ReasonCodeHandoffRunStreamMismatch)
	}
}

func TestEvaluateHandoffFixtureClassifiesQualityFallback(t *testing.T) {
	h := handoff.Handoff{Version: handoff.VersionV1, RunID: "run-1", Cut: handoff.CutCheckpoint, Objective: "x", Fallback: &handoff.Fallback{Reason: handoff.FallbackQualityBelowThreshold}}
	raw, _ := json.Marshal(HandoffFixture{Version: ContextHandoffFixtureVersionV1, Handoff: h})
	_, err := EvaluateHandoffFixtureJSON(raw)
	if err == nil || err.(*ValidationError).Code != ReasonCodeHandoffQualityBelow {
		t.Fatalf("error = %v, want %s", err, ReasonCodeHandoffQualityBelow)
	}
}

func TestEvaluateHandoffFixtureClassifiesSchemaDrift(t *testing.T) {
	raw := []byte(`{"version":"context_handoff.v0","handoff":{}}`)
	_, err := EvaluateHandoffFixtureJSON(raw)
	if err == nil || err.(*ValidationError).Code != ReasonCodeHandoffSchemaDrift {
		t.Fatalf("error = %v, want %s", err, ReasonCodeHandoffSchemaDrift)
	}
}

func TestEvaluateHandoffFixtureClassifiesFactAndReferenceLoss(t *testing.T) {
	h := handoff.Handoff{Version: handoff.VersionV1, RunID: "run-1", Cut: handoff.CutCheckpoint, Objective: "x"}
	raw, _ := json.Marshal(HandoffFixture{Version: ContextHandoffFixtureVersionV1, Handoff: h, ExpectedFacts: []string{"must survive"}})
	_, err := EvaluateHandoffFixtureJSON(raw)
	if err == nil || err.(*ValidationError).Code != ReasonCodeHandoffFactLoss {
		t.Fatalf("fact loss error = %v, want %s", err, ReasonCodeHandoffFactLoss)
	}
	h.Facts = []handoff.Evidence{{Kind: handoff.EvidenceFact, Value: "must survive", SourceID: "event-1"}}
	raw, _ = json.Marshal(HandoffFixture{Version: ContextHandoffFixtureVersionV1, Handoff: h, ExpectedReferences: []string{"checkpoint:cp-1"}})
	_, err = EvaluateHandoffFixtureJSON(raw)
	if err == nil || err.(*ValidationError).Code != ReasonCodeHandoffReferenceLoss {
		t.Fatalf("reference loss error = %v, want %s", err, ReasonCodeHandoffReferenceLoss)
	}
}
