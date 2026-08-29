package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelixSeptem/baymax/tool/diagnosticsreplay"
)

func TestAgentRuntimeProtocolReplayFixtureSuccess(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "diagnostics-replay", "agent-runtime-protocol", "success.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	out, err := diagnosticsreplay.EvaluateProtocolFixtureJSON(raw)
	if err != nil || len(out.Cases) != 1 {
		t.Fatalf("out=%#v err=%v", out, err)
	}
}

func TestAgentRuntimeProtocolReplayFixtureEventStreamBindingSuite(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "diagnostics-replay", "agent-runtime-protocol", "stream-binding.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	out, err := diagnosticsreplay.EvaluateProtocolFixtureJSON(raw)
	if err != nil {
		t.Fatalf("evaluate binding fixture: %v", err)
	}
	if len(out.Cases) != 10 {
		t.Fatalf("binding fixture cases = %d, want 10", len(out.Cases))
	}
}

func TestAgentRuntimeProtocolReplayFixtureRejectsMappingDrift(t *testing.T) {
	raw := []byte(`{"version":"agent_runtime_protocol.v1","cases":[{"name":"drift","run":{"run_id":"run-1","state":"completed"},"stream":{"run_id":"run-1","state":"failed"},"expected":{"run_id":"run-1","state":"completed"},"idempotency":{"first_logical_ingest_total":1,"replay_logical_ingest_total":1}}]}`)
	_, err := diagnosticsreplay.EvaluateProtocolFixtureJSON(raw)
	if err == nil || !strings.Contains(err.Error(), diagnosticsreplay.ReasonCodeProtocolDrift) {
		t.Fatalf("error=%v, want protocol drift", err)
	}
}

func TestAgentRuntimeProtocolReplayFixtureDriftTaxonomy(t *testing.T) {
	cases := map[string]string{
		"profile-drift.json":       diagnosticsreplay.ReasonCodeProtocolProfileDrift,
		"context-limit-drift.json": diagnosticsreplay.ReasonCodeProtocolContextDrift,
		"admission-drift.json":     diagnosticsreplay.ReasonCodeProtocolAdmissionDrift,
		"missing-correlation.json": diagnosticsreplay.ReasonCodeProtocolCorrelationDrift,
	}
	for name, reason := range cases {
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("testdata", "diagnostics-replay", "agent-runtime-protocol", name))
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			_, err = diagnosticsreplay.EvaluateProtocolFixtureJSON(raw)
			if err == nil || !strings.Contains(err.Error(), reason) {
				t.Fatalf("error=%v, want %s", err, reason)
			}
		})
	}
}

func TestAgentRuntimeProtocolCheckpointProvenanceFixtures(t *testing.T) {
	tests := []struct {
		name       string
		file       string
		wantReason string
		wantCases  int
	}{
		{name: "success", file: "agent_runtime_protocol_checkpoint_provenance_success_input.json", wantCases: 1},
		{name: "workspace-integrity-drift", file: "agent_runtime_protocol_checkpoint_provenance_workspace_drift_input.json", wantReason: diagnosticsreplay.ReasonCodeWorkspaceIntegrityDrift},
		{name: "malformed", file: "agent_runtime_protocol_checkpoint_provenance_malformed_input.json", wantReason: diagnosticsreplay.ReasonCodeProtocolSchema},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("..", "tool", "diagnosticsreplay", "testdata", tc.file))
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			out, err := diagnosticsreplay.EvaluateProtocolFixtureJSON(raw)
			if tc.wantReason == "" {
				if err != nil || len(out.Cases) != tc.wantCases {
					t.Fatalf("out=%#v err=%v", out, err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantReason) {
				t.Fatalf("error=%v, want %s", err, tc.wantReason)
			}
		})
	}
}
