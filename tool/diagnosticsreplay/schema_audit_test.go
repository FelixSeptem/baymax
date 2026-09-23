package diagnosticsreplay

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/FelixSeptem/baymax/tool/schemaaudit"
)

func TestReplayToolSchemaPressureSelectionAuditJSONIsDeterministic(t *testing.T) {
	facts, err := schemaaudit.CanonicalSchemaFacts(map[string]any{"type": "object"})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(schemaaudit.Input{Version: schemaaudit.AuditVersion, SnapshotID: "replay", Tools: []schemaaudit.AdmittedTool{{Identity: "search", Source: "local", Admitted: true, Schema: facts}}, Cases: []schemaaudit.TaskCase{{ID: "c", Expected: []string{"search"}, Allowed: []string{"search"}}}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := ReplayToolSchemaPressureSelectionAuditJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if result.Digest == "" || result.Version != ToolSchemaPressureSelectionAuditV1 {
		t.Fatalf("unexpected replay result: %#v", result)
	}
}

func TestReplayToolSchemaPressureSelectionAuditFixtureCompatibilityAndParity(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	fixture := filepath.Join(filepath.Dir(root), "schemaaudit", "testdata", "synthetic_pressure_quality.json")
	raw, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	run, err := ReplayToolSchemaPressureSelectionAuditJSON(append([]byte(`{"unknown_future_field":true,`), raw[1:]...))
	if err != nil {
		t.Fatal(err)
	}
	streamInput, err := schemaaudit.ParseFixtureJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	streamInput.Stream = true
	stream, err := schemaaudit.Audit(streamInput)
	if err != nil {
		t.Fatal(err)
	}
	if run.Digest == "" || run.Pressure.TokenEstimate == 0 || run.Strategies[0].Pressure.TokenEstimate == 0 || run.Conclusion == "" {
		t.Fatalf("incomplete replay evidence: %#v", run)
	}
	if err := CompareToolSchemaPressureSelectionAuditParity(run, stream); err != nil {
		t.Fatalf("run/stream parity: %v", err)
	}
	drift := stream
	drift.Strategies = append([]schemaaudit.StrategyResult(nil), stream.Strategies...)
	drift.Strategies[0].Selected = []string{"admin"}
	if code := schemaaudit.CodeOf(CompareToolSchemaPressureSelectionAuditParity(run, drift)); code != ReasonCodeSchemaAuditParityDrift {
		t.Fatalf("parity drift code = %q", code)
	}
	for _, test := range []struct {
		file string
		code string
	}{{"synthetic_gold_conflict.json", ReasonCodeSchemaAuditGoldSetConflict}, {"synthetic_overflow.json", ReasonCodeSchemaAuditOverflow}, {"synthetic_strategy_drift.json", ReasonCodeSchemaAuditStrategyDrift}} {
		bad, err := os.ReadFile(filepath.Join(filepath.Dir(fixture), test.file))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ReplayToolSchemaPressureSelectionAuditJSON(bad); schemaaudit.CodeOf(err) != test.code {
			t.Fatalf("%s replay code = %q, want %q", test.file, schemaaudit.CodeOf(err), test.code)
		}
	}
}
