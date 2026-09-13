package diagnosticsreplay

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

const ProviderHandoffStreamEdgeFixtureV1 = "provider_handoff_stream_edge.v1"

const (
	ReasonCodeProviderHandoffSchemaDrift       = "provider_handoff_schema_drift"
	ReasonCodeProviderToolCallCorrelationDrift = "provider_tool_call_correlation_drift"
	ReasonCodeProviderToolResultFeedbackDrift  = "provider_tool_result_feedback_drift"
	ReasonCodeProviderThinkingProjectionDrift  = "provider_thinking_projection_drift"
	ReasonCodeProviderStreamBoundaryDrift      = "provider_stream_boundary_drift"
	ReasonCodeProviderAbortUsageDrift          = "provider_abort_usage_drift"
	ReasonCodeProviderOverflowDrift            = "provider_overflow_drift"
	ReasonCodeProviderUnicodeEmptyContentDrift = "provider_unicode_empty_content_drift"
	ReasonCodeProviderFallbackFenceDrift       = "provider_fallback_fence_drift"
	ReasonCodeProviderRunStreamParityDrift     = "provider_run_stream_parity_drift"
	ReasonCodeProviderErrorTaxonomyDrift       = "provider_error_taxonomy_drift"
)

type ProviderHandoffStreamEdgeFixture struct {
	Version string                          `json:"version"`
	Cases   []ProviderHandoffStreamEdgeCase `json:"cases"`
}
type ProviderHandoffStreamEdgeCase struct {
	Name        string                    `json:"name"`
	Provider    string                    `json:"provider"`
	Mode        string                    `json:"mode,omitempty"`
	Expected    ProviderHandoffProjection `json:"expected"`
	Observed    ProviderHandoffProjection `json:"observed"`
	Run         ProviderHandoffProjection `json:"run"`
	Stream      ProviderHandoffProjection `json:"stream"`
	Idempotency ProviderReplayIdempotency `json:"idempotency"`
}
type ProviderReplayIdempotency struct {
	FirstDigest  string `json:"first_digest"`
	ReplayDigest string `json:"replay_digest"`
}
type ProviderCorrelation struct {
	RunID      string `json:"run_id"`
	StepID     string `json:"step_id"`
	ToolCallID string `json:"tool_call_id,omitempty"`
}
type ProviderToolCallProjection struct {
	Name            string `json:"name,omitempty"`
	ArgumentsDigest string `json:"arguments_digest,omitempty"`
}
type ProviderToolResultProjection struct {
	ToolCallID   string `json:"tool_call_id,omitempty"`
	Status       string `json:"status,omitempty"`
	ResultDigest string `json:"result_digest,omitempty"`
}
type ProviderThinkingProjection struct {
	Present bool   `json:"present"`
	Digest  string `json:"digest,omitempty"`
}
type ProviderStreamProjection struct {
	Boundaries       []string `json:"boundaries,omitempty"`
	Outcome          string   `json:"outcome,omitempty"`
	ContentDigest    string   `json:"content_digest,omitempty"`
	UnicodePreserved bool     `json:"unicode_preserved"`
	EmptyContent     bool     `json:"empty_content"`
}
type ProviderUsageProjection struct {
	Available    bool  `json:"available"`
	InputTokens  int64 `json:"input_tokens,omitempty"`
	OutputTokens int64 `json:"output_tokens,omitempty"`
}
type ProviderOverflowProjection struct {
	Bounded        bool   `json:"bounded"`
	Classification string `json:"classification,omitempty"`
}
type ProviderFallbackProjection struct {
	Phase            string `json:"phase,omitempty"`
	SelectedProvider string `json:"selected_provider,omitempty"`
	ProviderSwitched bool   `json:"provider_switched"`
}
type ProviderTerminalProjection struct {
	Outcome    string `json:"outcome,omitempty"`
	ErrorClass string `json:"error_class,omitempty"`
}
type ProviderHandoffProjection struct {
	Correlation ProviderCorrelation          `json:"correlation"`
	ToolCall    ProviderToolCallProjection   `json:"tool_call"`
	ToolResult  ProviderToolResultProjection `json:"tool_result"`
	Thinking    ProviderThinkingProjection   `json:"thinking"`
	Stream      ProviderStreamProjection     `json:"stream"`
	Usage       ProviderUsageProjection      `json:"usage"`
	Overflow    ProviderOverflowProjection   `json:"overflow"`
	Fallback    ProviderFallbackProjection   `json:"fallback"`
	Terminal    ProviderTerminalProjection   `json:"terminal"`
}

func EvaluateProviderHandoffStreamEdgeFixtureJSON(raw []byte) (ProviderHandoffStreamEdgeFixture, error) {
	if len(raw) > 2<<20 {
		return ProviderHandoffStreamEdgeFixture{}, schemaProviderError("fixture exceeds 2 MiB")
	}
	var f ProviderHandoffStreamEdgeFixture
	if err := json.Unmarshal(raw, &f); err != nil {
		return f, schemaProviderError(err.Error())
	}
	if strings.TrimSpace(f.Version) != ProviderHandoffStreamEdgeFixtureV1 {
		return f, schemaProviderError(fmt.Sprintf("unsupported fixture version %q", f.Version))
	}
	if len(f.Cases) == 0 || len(f.Cases) > 256 {
		return f, schemaProviderError("cases must contain 1..256 items")
	}
	for i := range f.Cases {
		c := &f.Cases[i]
		c.Name = strings.TrimSpace(c.Name)
		c.Provider = strings.ToLower(strings.TrimSpace(c.Provider))
		if c.Name == "" || len(c.Name) > 128 || !isSupportedProvider(c.Provider) {
			return f, schemaProviderError(fmt.Sprintf("cases[%d] name/provider invalid", i))
		}
		for _, p := range []ProviderHandoffProjection{c.Expected, c.Observed, c.Run, c.Stream} {
			if err := validateProviderProjection(p); err != nil {
				return f, schemaProviderError(fmt.Sprintf("cases[%d]: %v", i, err))
			}
		}
		if c.Idempotency.FirstDigest == "" || c.Idempotency.ReplayDigest == "" || c.Idempotency.FirstDigest != c.Idempotency.ReplayDigest {
			return f, schemaProviderError(fmt.Sprintf("cases[%d].idempotency digest mismatch", i))
		}
		if code := classifyProviderProjectionDrift(c.Expected, c.Observed); code != "" {
			return f, &ValidationError{Code: code, Message: c.Name}
		}
		if code := classifyProviderProjectionDrift(c.Run, c.Stream); code != "" {
			return f, &ValidationError{Code: ReasonCodeProviderRunStreamParityDrift, Message: c.Name + ": " + code}
		}
	}
	return f, nil
}

// ParseProviderHandoffStreamEdgeFixtureJSON is the compatibility parser entrypoint.
// Parsing performs the same bounded, side-effect-free validation as evaluation.
func ParseProviderHandoffStreamEdgeFixtureJSON(raw []byte) (ProviderHandoffStreamEdgeFixture, error) {
	return EvaluateProviderHandoffStreamEdgeFixtureJSON(raw)
}
func schemaProviderError(msg string) *ValidationError {
	return &ValidationError{Code: ReasonCodeProviderHandoffSchemaDrift, Message: msg}
}
func isSupportedProvider(p string) bool { return p == "openai" || p == "anthropic" || p == "gemini" }
func validateProviderProjection(p ProviderHandoffProjection) error {
	if strings.TrimSpace(p.Correlation.RunID) == "" || strings.TrimSpace(p.Correlation.StepID) == "" {
		return fmt.Errorf("run_id and step_id are required")
	}
	if len(p.Stream.Boundaries) > 32 {
		return fmt.Errorf("stream boundaries exceed bound")
	}
	for _, b := range p.Stream.Boundaries {
		if len(b) > 32 {
			return fmt.Errorf("stream boundary exceeds bound")
		}
	}
	if len(p.ToolCall.ArgumentsDigest) > 256 || len(p.Stream.ContentDigest) > 256 {
		return fmt.Errorf("digest exceeds bound")
	}
	return nil
}
func classifyProviderProjectionDrift(expected, observed ProviderHandoffProjection) string {
	if !reflect.DeepEqual(expected.Correlation, observed.Correlation) || !reflect.DeepEqual(expected.ToolCall, observed.ToolCall) {
		return ReasonCodeProviderToolCallCorrelationDrift
	}
	if !reflect.DeepEqual(expected.ToolResult, observed.ToolResult) {
		return ReasonCodeProviderToolResultFeedbackDrift
	}
	if !reflect.DeepEqual(expected.Thinking, observed.Thinking) {
		return ReasonCodeProviderThinkingProjectionDrift
	}
	if !reflect.DeepEqual(expected.Stream.Boundaries, observed.Stream.Boundaries) || expected.Stream.Outcome != observed.Stream.Outcome {
		return ReasonCodeProviderStreamBoundaryDrift
	}
	if !reflect.DeepEqual(expected.Usage, observed.Usage) {
		return ReasonCodeProviderAbortUsageDrift
	}
	if !reflect.DeepEqual(expected.Overflow, observed.Overflow) {
		return ReasonCodeProviderOverflowDrift
	}
	if expected.Stream.ContentDigest != observed.Stream.ContentDigest || expected.Stream.UnicodePreserved != observed.Stream.UnicodePreserved || expected.Stream.EmptyContent != observed.Stream.EmptyContent {
		return ReasonCodeProviderUnicodeEmptyContentDrift
	}
	if !reflect.DeepEqual(expected.Fallback, observed.Fallback) {
		return ReasonCodeProviderFallbackFenceDrift
	}
	if !reflect.DeepEqual(expected.Terminal, observed.Terminal) {
		return ReasonCodeProviderErrorTaxonomyDrift
	}
	return ""
}
