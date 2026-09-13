package diagnosticsreplay

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestEvaluateProviderHandoffStreamEdgeFixtureSuccessIsIdempotent(t *testing.T) {
	raw := []byte(`{"version":"provider_handoff_stream_edge.v1","cases":[{"name":"unicode-empty","provider":"openai","expected":{"correlation":{"run_id":"run-1","step_id":"step-1","tool_call_id":"call-1"},"tool_call":{"name":"lookup","arguments_digest":"sha256:args"},"tool_result":{"tool_call_id":"call-1","status":"success","result_digest":"sha256:result"},"thinking":{"present":false},"stream":{"boundaries":["start","partial","completed"],"outcome":"completed","content_digest":"sha256:unicode","unicode_preserved":true,"empty_content":true},"usage":{"available":false},"overflow":{"bounded":true,"classification":"none"},"fallback":{"phase":"pre_step","selected_provider":"openai","provider_switched":false},"terminal":{"outcome":"completed","error_class":"none"}},"observed":{"correlation":{"run_id":"run-1","step_id":"step-1","tool_call_id":"call-1"},"tool_call":{"name":"lookup","arguments_digest":"sha256:args"},"tool_result":{"tool_call_id":"call-1","status":"success","result_digest":"sha256:result"},"thinking":{"present":false},"stream":{"boundaries":["start","partial","completed"],"outcome":"completed","content_digest":"sha256:unicode","unicode_preserved":true,"empty_content":true},"usage":{"available":false},"overflow":{"bounded":true,"classification":"none"},"fallback":{"phase":"pre_step","selected_provider":"openai","provider_switched":false},"terminal":{"outcome":"completed","error_class":"none"}},"run":{"correlation":{"run_id":"run-1","step_id":"step-1","tool_call_id":"call-1"},"terminal":{"outcome":"completed","error_class":"none"}},"stream":{"correlation":{"run_id":"run-1","step_id":"step-1","tool_call_id":"call-1"},"terminal":{"outcome":"completed","error_class":"none"}},"idempotency":{"first_digest":"sha256:stable","replay_digest":"sha256:stable"}}]}`)
	first, err := EvaluateProviderHandoffStreamEdgeFixtureJSON(raw)
	if err != nil {
		t.Fatalf("first replay: %v", err)
	}
	second, err := EvaluateProviderHandoffStreamEdgeFixtureJSON(raw)
	if err != nil {
		t.Fatalf("second replay: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("replay is not idempotent")
	}
}

func TestProviderHandoffStreamEdgeReplayKeepsHistoricalFixtureCompatible(t *testing.T) {
	raw := mustReadFixture(t, "provider_model_admission.v1.json")
	if _, err := ParseMinimalReplayJSON(raw); err != nil {
		t.Fatalf("historical replay: %v", err)
	}
}

func TestProviderHandoffStreamEdgeReplayRejectsSchemaAndBounds(t *testing.T) {
	for _, tc := range []struct{ name, raw string }{
		{"unknown version", `{"version":"provider_handoff_stream_edge.v2","cases":[]}`},
		{"missing provider", `{"version":"provider_handoff_stream_edge.v1","cases":[{"name":"bad"}]}`},
		{"unbounded name", `{"version":"provider_handoff_stream_edge.v1","cases":[{"name":"` + strings.Repeat("x", 300) + `","provider":"openai"}]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := EvaluateProviderHandoffStreamEdgeFixtureJSON([]byte(tc.raw))
			if err == nil || err.(*ValidationError).Code != ReasonCodeProviderHandoffSchemaDrift {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func TestProviderHandoffStreamEdgeReplayClassifiesDrift(t *testing.T) {
	base := ProviderHandoffProjection{
		Correlation: ProviderCorrelation{RunID: "run-1", StepID: "step-1", ToolCallID: "call-1"},
		ToolCall:    ProviderToolCallProjection{Name: "lookup", ArgumentsDigest: "args"},
		ToolResult:  ProviderToolResultProjection{ToolCallID: "call-1", Status: "success", ResultDigest: "result"},
		Thinking:    ProviderThinkingProjection{Present: true, Digest: "thinking"},
		Stream:      ProviderStreamProjection{Boundaries: []string{"start", "completed"}, Outcome: "completed", ContentDigest: "content", UnicodePreserved: true},
		Usage:       ProviderUsageProjection{Available: true, InputTokens: 1, OutputTokens: 2},
		Overflow:    ProviderOverflowProjection{Bounded: true, Classification: "none"},
		Fallback:    ProviderFallbackProjection{Phase: "pre_step", SelectedProvider: "openai"},
		Terminal:    ProviderTerminalProjection{Outcome: "completed", ErrorClass: "none"},
	}
	cases := []struct {
		name, want string
		mutate     func(*ProviderHandoffProjection)
	}{
		{"correlation", ReasonCodeProviderToolCallCorrelationDrift, func(p *ProviderHandoffProjection) { p.Correlation.ToolCallID = "other" }},
		{"feedback", ReasonCodeProviderToolResultFeedbackDrift, func(p *ProviderHandoffProjection) { p.ToolResult.Status = "error" }},
		{"thinking", ReasonCodeProviderThinkingProjectionDrift, func(p *ProviderHandoffProjection) { p.Thinking.Digest = "other" }},
		{"stream", ReasonCodeProviderStreamBoundaryDrift, func(p *ProviderHandoffProjection) { p.Stream.Boundaries = []string{"start", "abort"} }},
		{"usage", ReasonCodeProviderAbortUsageDrift, func(p *ProviderHandoffProjection) { p.Usage.OutputTokens = 3 }},
		{"overflow", ReasonCodeProviderOverflowDrift, func(p *ProviderHandoffProjection) { p.Overflow.Bounded = false }},
		{"unicode", ReasonCodeProviderUnicodeEmptyContentDrift, func(p *ProviderHandoffProjection) { p.Stream.UnicodePreserved = false }},
		{"fallback", ReasonCodeProviderFallbackFenceDrift, func(p *ProviderHandoffProjection) {
			p.Fallback.Phase = "post_start"
			p.Fallback.ProviderSwitched = true
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			observed := cloneProviderProjection(t, base)
			tc.mutate(&observed)
			fixture := ProviderHandoffStreamEdgeFixture{Version: ProviderHandoffStreamEdgeFixtureV1, Cases: []ProviderHandoffStreamEdgeCase{{Name: tc.name, Provider: "openai", Expected: base, Observed: observed, Run: base, Stream: base, Idempotency: ProviderReplayIdempotency{FirstDigest: "stable", ReplayDigest: "stable"}}}}
			raw, _ := json.Marshal(fixture)
			_, err := EvaluateProviderHandoffStreamEdgeFixtureJSON(raw)
			if err == nil || err.(*ValidationError).Code != tc.want {
				t.Fatalf("error=%v want=%s", err, tc.want)
			}
		})
	}
}

func TestProviderHandoffStreamEdgeReplayClassifiesParityAndIdempotency(t *testing.T) {
	base := ProviderHandoffProjection{Correlation: ProviderCorrelation{RunID: "run", StepID: "step", ToolCallID: "call"}, Terminal: ProviderTerminalProjection{Outcome: "completed", ErrorClass: "none"}}
	stream := base
	stream.Terminal.Outcome = "aborted"
	fixture := ProviderHandoffStreamEdgeFixture{Version: ProviderHandoffStreamEdgeFixtureV1, Cases: []ProviderHandoffStreamEdgeCase{{Name: "parity", Provider: "gemini", Expected: base, Observed: base, Run: base, Stream: stream, Idempotency: ProviderReplayIdempotency{FirstDigest: "stable", ReplayDigest: "stable"}}}}
	raw, _ := json.Marshal(fixture)
	_, err := EvaluateProviderHandoffStreamEdgeFixtureJSON(raw)
	if err == nil || err.(*ValidationError).Code != ReasonCodeProviderRunStreamParityDrift {
		t.Fatalf("error=%v", err)
	}
	fixture.Cases[0].Stream = base
	fixture.Cases[0].Idempotency.ReplayDigest = "changed"
	raw, _ = json.Marshal(fixture)
	_, err = EvaluateProviderHandoffStreamEdgeFixtureJSON(raw)
	if err == nil || err.(*ValidationError).Code != ReasonCodeProviderHandoffSchemaDrift {
		t.Fatalf("error=%v", err)
	}
}

func cloneProviderProjection(t *testing.T, in ProviderHandoffProjection) ProviderHandoffProjection {
	t.Helper()
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out ProviderHandoffProjection
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return out
}
