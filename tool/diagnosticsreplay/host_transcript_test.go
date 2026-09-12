package diagnosticsreplay

import (
	"strings"
	"testing"
)

func TestHostTranscriptFixtureCoversFirstProfileScenarios(t *testing.T) {
	raw := mustReadFixture(t, "embedded_host_protocol.v1.json")
	out, err := EvaluateHostTranscriptFixtureJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if out.Version != HostTranscriptFixtureV1 || len(out.Cases) != 9 {
		t.Fatalf("output=%#v, want version %q and 9 cases", out, HostTranscriptFixtureV1)
	}
	want := map[string]bool{
		"valid": true, "rejection": true, "duplicate": true,
		"hitl": true, "disconnect": true, "terminal-conflict": true,
		"framing": true, "output-failure": true, "backpressure": true,
	}
	for _, tc := range out.Cases {
		if !want[tc.Name] {
			t.Fatalf("unexpected case %q", tc.Name)
		}
		delete(want, tc.Name)
		if tc.RunStreamParity != "equivalent" || tc.Idempotency.FirstLogicalIngestTotal != tc.Idempotency.ReplayLogicalIngestTotal {
			t.Fatalf("case %q parity/idempotency=%#v", tc.Name, tc)
		}
	}
	if len(want) != 0 {
		t.Fatalf("missing scenarios=%v", want)
	}
}

func TestHostTranscriptFixtureRejectsParityDrift(t *testing.T) {
	raw := []byte(`{"version":"embedded_host_protocol.v1","cases":[{"name":"drift","run":{"admission":"accepted","terminal":"completed"},"stream":{"admission":"accepted","terminal":"failed"},"expected":{"admission":"accepted","terminal":"completed"},"run_stream_parity":"equivalent","idempotency":{"first_logical_ingest_total":1,"replay_logical_ingest_total":1}}]}`)
	_, err := EvaluateHostTranscriptFixtureJSON(raw)
	if err == nil || !strings.Contains(err.Error(), ReasonCodeHostTranscriptParityDrift) {
		t.Fatalf("error=%v, want %s", err, ReasonCodeHostTranscriptParityDrift)
	}
}
