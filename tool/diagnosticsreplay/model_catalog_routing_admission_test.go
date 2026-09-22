package diagnosticsreplay

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

const modelCatalogRoutingAdmissionFixturePath = "testdata/model_catalog_routing_admission.v1.json"

func TestReplayModelCatalogRoutingAdmissionFixtureIsVersionedRedactedAndIdempotent(t *testing.T) {
	raw, err := os.ReadFile(modelCatalogRoutingAdmissionFixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if strings.Contains(strings.ToLower(string(raw)), "endpoint") || strings.Contains(strings.ToLower(string(raw)), "sk-") || strings.Contains(strings.ToLower(string(raw)), "token") {
		t.Fatalf("fixture contains sensitive material")
	}
	first, err := ReplayModelCatalogRoutingAdmissionFixtureJSON(raw)
	if err != nil {
		t.Fatalf("first replay: %v", err)
	}
	second, err := ReplayModelCatalogRoutingAdmissionFixtureJSON(raw)
	if err != nil {
		t.Fatalf("second replay: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("replay is not idempotent: first=%#v second=%#v", first, second)
	}
	if len(first.Cases) < 7 {
		t.Fatalf("fixture cases = %d, want coverage cases", len(first.Cases))
	}
	for _, item := range first.Cases {
		if !item.Idempotent || item.Digest != item.ReplayDigest {
			t.Fatalf("case %q is not idempotent: %#v", item.CaseID, item)
		}
	}
}

func TestReplayModelCatalogRoutingAdmissionFixtureDetectsSelectionDrift(t *testing.T) {
	raw, err := os.ReadFile(modelCatalogRoutingAdmissionFixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var fixture ModelCatalogRoutingAdmissionFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	fixture.Cases[0].Expected.Status = "blocked"
	mutated, err := json.Marshal(fixture)
	if err != nil {
		t.Fatalf("marshal mutated fixture: %v", err)
	}
	if _, err := ReplayModelCatalogRoutingAdmissionFixtureJSON(mutated); err == nil {
		t.Fatal("expected selection drift")
	} else if !strings.Contains(err.Error(), ReasonCodeModelCatalogSelectionDrift) {
		t.Fatalf("error = %v, want %q", err, ReasonCodeModelCatalogSelectionDrift)
	}
}

func TestReplayModelCatalogRoutingAdmissionFixtureIgnoresUnknownFields(t *testing.T) {
	raw, err := os.ReadFile(modelCatalogRoutingAdmissionFixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	document["unknown_future_field"] = "ignored"
	mutated, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	if _, err := ReplayModelCatalogRoutingAdmissionFixtureJSON(mutated); err != nil {
		t.Fatalf("unknown fields should be ignored: %v", err)
	}
}

func TestReplayModelCatalogRoutingAdmissionRunStreamParity(t *testing.T) {
	raw, err := os.ReadFile(modelCatalogRoutingAdmissionFixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var fixture ModelCatalogRoutingAdmissionFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	var runCase, streamCase *ModelCatalogRoutingAdmissionFixtureCase
	for index := range fixture.Cases {
		switch fixture.Cases[index].CaseID {
		case "run-parity":
			runCase = &fixture.Cases[index]
		case "stream-parity":
			streamCase = &fixture.Cases[index]
		}
	}
	if runCase == nil || streamCase == nil {
		t.Fatal("fixture must include run-parity and stream-parity cases")
	}
	run, err := replayModelCatalogRoutingAdmissionCase(*runCase)
	if err != nil {
		t.Fatalf("run replay: %v", err)
	}
	stream, err := replayModelCatalogRoutingAdmissionCase(*streamCase)
	if err != nil {
		t.Fatalf("stream replay: %v", err)
	}
	if run.CatalogGeneration != stream.CatalogGeneration || run.Status != stream.Status || !reflect.DeepEqual(run.Selected, stream.Selected) || !reflect.DeepEqual(run.Fallback, stream.Fallback) || !reflect.DeepEqual(run.Reasons, stream.Reasons) {
		t.Fatalf("run/stream parity drift: run=%#v stream=%#v", run, stream)
	}
}
