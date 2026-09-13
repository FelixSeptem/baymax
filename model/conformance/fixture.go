// Package conformance contains the provider-neutral, bounded stream-edge
// projection used by adapter tests and replay tooling. Provider SDK payloads
// must be normalized before entering this package.
package conformance

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

const FixtureVersionV1 = "provider_handoff_stream_edge.v1"

const (
	ReasonSchemaDrift              = "provider_handoff_schema_drift"
	ReasonToolCallCorrelationDrift = "provider_tool_call_correlation_drift"
	ReasonToolResultFeedbackDrift  = "provider_tool_result_feedback_drift"
	ReasonThinkingProjectionDrift  = "provider_thinking_projection_drift"
	ReasonStreamBoundaryDrift      = "provider_stream_boundary_drift"
	ReasonAbortUsageDrift          = "provider_abort_usage_drift"
	ReasonOverflowDrift            = "provider_overflow_drift"
	ReasonUnicodeEmptyContentDrift = "provider_unicode_empty_content_drift"
	ReasonFallbackFenceDrift       = "provider_fallback_fence_drift"
	ReasonRunStreamParityDrift     = "provider_run_stream_parity_drift"
)

const (
	maxCases      = 256
	maxEvents     = 512
	maxTextBytes  = 64 * 1024
	maxIdentifier = 256
)

type Fixture struct {
	Version string `json:"version"`
	Cases   []Case `json:"cases"`
}

type Case struct {
	Name        string      `json:"name"`
	Provider    string      `json:"provider"`
	Mode        string      `json:"mode"`
	Correlation Correlation `json:"correlation"`
	Events      []Event     `json:"events"`
	Expected    Projection  `json:"expected"`
}

type Correlation struct {
	RunID      string `json:"run_id"`
	StepID     string `json:"step_id"`
	ToolCallID string `json:"tool_call_id,omitempty"`
}

type Event struct {
	Kind string `json:"kind"`
	Text string `json:"text,omitempty"`
}

type Projection struct {
	ToolCall   *ToolCall `json:"tool_call,omitempty"`
	Usage      *Usage    `json:"usage,omitempty"`
	Reasoning  *string   `json:"reasoning,omitempty"`
	Outcome    string    `json:"outcome"`
	DriftClass string    `json:"drift_class,omitempty"`
	Events     []Event   `json:"events,omitempty"`
}

type ToolCall struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

type Usage struct {
	InputTokens  int `json:"input_tokens,omitempty"`
	OutputTokens int `json:"output_tokens,omitempty"`
	TotalTokens  int `json:"total_tokens,omitempty"`
}

func ParseFixtureJSON(raw []byte) (Fixture, error) {
	var fixture Fixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		return Fixture{}, fmt.Errorf("%s: %w", ReasonSchemaDrift, err)
	}
	if fixture.Version != FixtureVersionV1 {
		return Fixture{}, fmt.Errorf("%s: unsupported fixture version %q", ReasonSchemaDrift, fixture.Version)
	}
	if len(fixture.Cases) == 0 || len(fixture.Cases) > maxCases {
		return Fixture{}, fmt.Errorf("%s: cases must contain 1..%d items", ReasonSchemaDrift, maxCases)
	}
	for i := range fixture.Cases {
		if err := validateCase(&fixture.Cases[i]); err != nil {
			return Fixture{}, fmt.Errorf("%s: cases[%d]: %w", ReasonSchemaDrift, i, err)
		}
	}
	return fixture, nil
}

func validateCase(c *Case) error {
	if strings.TrimSpace(c.Name) == "" || strings.TrimSpace(c.Provider) == "" || strings.TrimSpace(c.Mode) == "" {
		return fmt.Errorf("name, provider, and mode are required")
	}
	if c.Provider != "openai" && c.Provider != "anthropic" && c.Provider != "gemini" {
		return fmt.Errorf("unsupported provider %q", c.Provider)
	}
	if c.Mode != "run" && c.Mode != "stream" {
		return fmt.Errorf("unsupported mode %q", c.Mode)
	}
	if strings.TrimSpace(c.Correlation.RunID) == "" || strings.TrimSpace(c.Correlation.StepID) == "" {
		return fmt.Errorf("correlation.run_id and correlation.step_id are required")
	}
	if len(c.Correlation.RunID) > maxIdentifier || len(c.Correlation.StepID) > maxIdentifier || len(c.Correlation.ToolCallID) > maxIdentifier {
		return fmt.Errorf("correlation identifier exceeds %d bytes", maxIdentifier)
	}
	if len(c.Events) > maxEvents {
		return fmt.Errorf("events exceeds %d items", maxEvents)
	}
	for i := range c.Events {
		if c.Events[i].Kind != "start" && c.Events[i].Kind != "partial" && c.Events[i].Kind != "complete" && c.Events[i].Kind != "abort" {
			return fmt.Errorf("events[%d].kind %q is invalid", i, c.Events[i].Kind)
		}
		if len(c.Events[i].Text) > maxTextBytes {
			return fmt.Errorf("events[%d].text exceeds %d bytes", i, maxTextBytes)
		}
	}
	if strings.TrimSpace(c.Expected.Outcome) == "" {
		return fmt.Errorf("expected.outcome is required")
	}
	if c.Expected.ToolCall != nil {
		if strings.TrimSpace(c.Expected.ToolCall.ID) == "" || strings.TrimSpace(c.Expected.ToolCall.Name) == "" {
			return fmt.Errorf("expected.tool_call id and name are required")
		}
		if len(c.Expected.ToolCall.Arguments) == 0 {
			c.Expected.ToolCall.Arguments = map[string]any{}
		}
	}
	return nil
}

func CanonicalProjection(in Projection) ([]byte, error) {
	if len(in.Events) > maxEvents {
		return nil, fmt.Errorf("%s: events exceeds %d items", ReasonOverflowDrift, maxEvents)
	}
	for _, ev := range in.Events {
		if len(ev.Text) > maxTextBytes {
			return nil, fmt.Errorf("%s: event text exceeds %d bytes", ReasonOverflowDrift, maxTextBytes)
		}
	}
	return json.Marshal(in)
}

func ProjectionDigest(in Projection) (string, error) {
	raw, err := CanonicalProjection(in)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:]), nil
}
