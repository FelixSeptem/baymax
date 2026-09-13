package conformance

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseProviderHandoffStreamEdgeFixtureNormalizesBounds(t *testing.T) {
	raw := []byte(`{"version":"provider_handoff_stream_edge.v1","cases":[{"name":"openai-tool","provider":"openai","mode":"stream","correlation":{"run_id":"run-1","step_id":"step-1","tool_call_id":"call-1"},"events":[{"kind":"start"},{"kind":"partial","text":"hi\u2028there"},{"kind":"complete"}],"expected":{"tool_call":{"id":"call-1","name":"lookup","arguments":{"city":"shanghai"}},"usage":{"input_tokens":3},"reasoning":null,"outcome":"completed","drift_class":""}}]}`)
	fixture, err := ParseFixtureJSON(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if fixture.Version != FixtureVersionV1 || len(fixture.Cases) != 1 {
		t.Fatalf("fixture=%#v", fixture)
	}
	got := fixture.Cases[0]
	if got.Provider != "openai" || got.Correlation.ToolCallID != "call-1" || got.Events[1].Text != "hi\u2028there" {
		t.Fatalf("normalized=%#v", got)
	}
	if got.Expected.Usage == nil || got.Expected.Usage.InputTokens != 3 {
		t.Fatalf("usage=%#v", got.Expected.Usage)
	}
}

func TestParseProviderHandoffStreamEdgeFixtureRejectsMalformedAndUnknownVersion(t *testing.T) {
	cases := []struct{ name, raw, code string }{
		{"unknown", `{"version":"provider_handoff_stream_edge.v2","cases":[]}`, ReasonSchemaDrift},
		{"missing-cases", `{"version":"provider_handoff_stream_edge.v1"}`, ReasonSchemaDrift},
		{"missing-correlation", `{"version":"provider_handoff_stream_edge.v1","cases":[{"name":"x","provider":"openai","mode":"run","events":[],"expected":{"outcome":"completed"}}]}`, ReasonSchemaDrift},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseFixtureJSON([]byte(tc.raw))
			if err == nil || !strings.Contains(err.Error(), tc.code) {
				t.Fatalf("err=%v, want %s", err, tc.code)
			}
		})
	}
}

func TestCanonicalProjectionDigestIsDeterministic(t *testing.T) {
	in := Projection{Outcome: "completed", Events: []Event{{Kind: "partial", Text: "x"}, {Kind: "complete"}}}
	a, err := CanonicalProjection(in)
	if err != nil {
		t.Fatal(err)
	}
	b, err := CanonicalProjection(in)
	if err != nil {
		t.Fatal(err)
	}
	if string(a) != string(b) {
		t.Fatalf("non-deterministic projection: %s != %s", a, b)
	}
	var decoded map[string]any
	if err := json.Unmarshal(a, &decoded); err != nil {
		t.Fatal(err)
	}
}
