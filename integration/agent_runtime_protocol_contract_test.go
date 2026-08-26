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
