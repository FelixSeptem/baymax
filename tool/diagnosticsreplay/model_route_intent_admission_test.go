package diagnosticsreplay

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

const modelRouteIntentAdmissionFixturePath = "testdata/model_route_intent_admission.v1.json"

func TestReplayModelRouteIntentAdmissionFixtureIsVersionedBoundedAndIdempotent(t *testing.T) {
	raw, err := os.ReadFile(modelRouteIntentAdmissionFixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	lower := strings.ToLower(string(raw))
	for _, forbidden := range []string{"endpoint", "sk-", "token", "raw_response", "raw_payload", "password", "secret"} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf("fixture contains forbidden material %q", forbidden)
		}
	}
	first, err := ReplayModelRouteIntentAdmissionFixtureJSON(raw)
	if err != nil {
		t.Fatalf("first replay: %v", err)
	}
	second, err := ReplayModelRouteIntentAdmissionFixtureJSON(raw)
	if err != nil {
		t.Fatalf("second replay: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("replay is not idempotent: first=%#v second=%#v", first, second)
	}
	if len(first.Cases) < 5 {
		t.Fatalf("cases = %d, want at least 5", len(first.Cases))
	}
}

func TestReplayModelRouteIntentAdmissionFixtureDetectsVerdictDrift(t *testing.T) {
	raw, err := os.ReadFile(modelRouteIntentAdmissionFixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var fixture ModelRouteIntentAdmissionFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	fixture.Cases[0].Expected.Verdict = "route-gap-confirmed"
	mutated, err := json.Marshal(fixture)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	if _, err := ReplayModelRouteIntentAdmissionFixtureJSON(mutated); err == nil || !strings.Contains(err.Error(), ReasonCodeModelRouteIntentVerdictDrift) {
		t.Fatalf("error = %v, want verdict drift", err)
	}
}

func TestReplayModelRouteIntentAdmissionFixtureIgnoresUnknownFields(t *testing.T) {
	raw, err := os.ReadFile(modelRouteIntentAdmissionFixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	document["unknown_future_field"] = map[string]any{"value": "ignored"}
	mutated, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	if _, err := ReplayModelRouteIntentAdmissionFixtureJSON(mutated); err != nil {
		t.Fatalf("unknown fields should be ignored: %v", err)
	}
}
