package diagnosticsreplay

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

const readinessFixturePath = "testdata/tool_schema_projection_readiness.v1.json"

func TestReplayToolSchemaProjectionReadinessFixtureIsRedactedAndIdempotent(t *testing.T) {
	raw, err := os.ReadFile(readinessFixturePath)
	if err != nil {
		t.Fatal(err)
	}
	lower := strings.ToLower(string(raw))
	for _, forbidden := range []string{"endpoint", "credential", "password", "prompt", "tool_result", "sk-"} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf("fixture contains forbidden material %q", forbidden)
		}
	}
	first, err := ReplayToolSchemaProjectionReadinessFixtureJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ReplayToolSchemaProjectionReadinessFixtureJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) || len(first.Cases) != 1 || !first.Cases[0].Idempotent {
		t.Fatalf("replay is not idempotent: %#v / %#v", first, second)
	}
	if first.Cases[0].Conclusion != "ready_for_runtime_design" {
		t.Fatalf("conclusion = %q", first.Cases[0].Conclusion)
	}
}

func TestReplayToolSchemaProjectionReadinessFixtureIgnoresUnknownFields(t *testing.T) {
	raw, err := os.ReadFile(readinessFixturePath)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	document["unknown_future_field"] = true
	mutated, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ReplayToolSchemaProjectionReadinessFixtureJSON(mutated); err != nil {
		t.Fatalf("unknown fields should be tolerated: %v", err)
	}
}

func TestReplayToolSchemaProjectionReadinessRunStreamParity(t *testing.T) {
	raw, err := os.ReadFile(readinessFixturePath)
	if err != nil {
		t.Fatal(err)
	}
	var fixture ToolSchemaProjectionReadinessFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	run := fixture.Cases[0].Input
	stream := fixture.Cases[0].Input
	stream.Stream = true
	runResult, err := EvaluateToolSchemaProjectionReadiness(run)
	if err != nil {
		t.Fatal(err)
	}
	streamResult, err := EvaluateToolSchemaProjectionReadiness(stream)
	if err != nil {
		t.Fatal(err)
	}
	if err := CompareToolSchemaProjectionReadinessParity(runResult, streamResult); err != nil {
		t.Fatal(err)
	}
	streamResult.Conclusion = "not_ready"
	if err := CompareToolSchemaProjectionReadinessParity(runResult, streamResult); err == nil || !strings.Contains(err.Error(), ReasonCodeToolSchemaProjectionReadinessParityDrift) {
		t.Fatalf("expected parity drift, got %v", err)
	}
}

func TestReplayToolSchemaProjectionReadinessRejectsNegativeFixtures(t *testing.T) {
	for _, name := range []string{"tool_schema_projection_readiness_overflow.json", "tool_schema_projection_readiness_privacy.json"} {
		raw, err := os.ReadFile("testdata/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ReplayToolSchemaProjectionReadinessFixtureJSON(raw); err == nil {
			t.Fatalf("%s unexpectedly succeeded", name)
		}
	}
}
