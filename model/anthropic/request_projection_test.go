package anthropic

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/model/conformance"
	"github.com/FelixSeptem/baymax/model/toolcontract"
)

// auditModelRequest mirrors the provider-neutral request shape the runtime
// admits once Skill bundle mapping has appended system-role prompt fragments
// and the ReAct loop has produced canonical tool-result feedback.
func auditModelRequest() types.ModelRequest {
	return types.ModelRequest{
		RunID: "run-audit",
		Model: "claude-3-5-sonnet-latest",
		Input: "summarize the repository",
		Messages: []types.Message{
			{Role: "system", Content: "skill fragment: always cite files"},
			{Role: "user", Content: "summarize the repository"},
			{Role: "assistant", Content: "working on it"},
		},
		ToolResult: []types.ToolCallOutcome{
			{
				CallID: "call-1",
				Name:   "read_file",
				Result: types.ToolResult{Content: "file body"},
			},
		},
		Capabilities: types.CapabilityRequirements{
			Required: []types.ModelCapability{types.ModelCapabilityToolCall, types.ModelCapabilityStreaming},
		},
	}
}

// auditDeclaredGaps is the pinned request projection gap set for the Anthropic
// adapter. It is evidence, not a tolerance: fixing any gap requires an explicit
// contract change.
var auditDeclaredGaps = []string{
	conformance.ReasonRequestCapabilityProjectionDrift,
	conformance.ReasonRequestRoleProjectionDrift,
	conformance.ReasonRequestToolResultNativeDrift,
}

func TestRequestProjectionAuditStreamBoundaryProjection(t *testing.T) {
	req := auditModelRequest()
	var captured string
	client := NewClient(Config{
		StreamFn: func(ctx context.Context, input string) Stream {
			captured = input
			return &fakeAnthropicStream{}
		},
	})

	if err := client.Stream(context.Background(), req, func(types.ModelEvent) error { return nil }); err != nil {
		t.Fatalf("Stream returned error: %v", err)
	}

	assertAnthropicProjection(t, captured, req)
}

func TestRequestProjectionAuditRunBoundaryProjection(t *testing.T) {
	req := auditModelRequest()
	var captured string
	client := NewClient(Config{
		GenerateFn: func(ctx context.Context, input string) (types.ModelResponse, error) {
			captured = input
			return types.ModelResponse{FinalAnswer: "ok"}, nil
		},
	})

	if _, err := client.Generate(context.Background(), req); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	assertAnthropicProjection(t, captured, req)
}

func TestRequestProjectionAuditRunStreamParity(t *testing.T) {
	req := auditModelRequest()

	var streamed string
	streamClient := NewClient(Config{
		StreamFn: func(ctx context.Context, input string) Stream {
			streamed = input
			return &fakeAnthropicStream{}
		},
	})
	if err := streamClient.Stream(context.Background(), req, func(types.ModelEvent) error { return nil }); err != nil {
		t.Fatalf("Stream returned error: %v", err)
	}

	var generated string
	runClient := NewClient(Config{
		GenerateFn: func(ctx context.Context, input string) (types.ModelResponse, error) {
			generated = input
			return types.ModelResponse{FinalAnswer: "ok"}, nil
		},
	})
	if _, err := runClient.Generate(context.Background(), req); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	source := conformance.RequestFactsFromModelRequest(req)
	runProjection := conformance.ProjectRequestTextEnvelope(generated, source.ToolResults)
	streamProjection := conformance.ProjectRequestTextEnvelope(streamed, source.ToolResults)
	if code := conformance.ClassifyRunStreamParity(runProjection, streamProjection); code != "" {
		t.Fatalf("run/stream request projection parity violated: %s", code)
	}
}

func assertAnthropicProjection(t *testing.T, projected string, req types.ModelRequest) {
	t.Helper()

	canonical, err := toolcontract.CanonicalInput(req)
	if err != nil {
		t.Fatalf("canonical input: %v", err)
	}
	if projected != canonical {
		t.Fatalf("audit expectation: the adapter must forward canonical input verbatim")
	}
	if strings.Contains(projected, "skill fragment") {
		t.Fatal("audit expectation changed: system-role skill fragment reached the SDK boundary")
	}
	if strings.Contains(projected, "working on it") {
		t.Fatal("audit expectation changed: assistant history reached the SDK boundary")
	}
	if !strings.Contains(projected, toolcontract.FeedbackHeader) {
		t.Fatal("canonical tool-result feedback envelope is missing from the projected input")
	}

	source := conformance.RequestFactsFromModelRequest(req)
	observed := conformance.ProjectRequestTextEnvelope(projected, source.ToolResults)
	if !observed.ToolResultCorrelated {
		t.Fatal("tool-result correlation must be preserved inside the text envelope")
	}
	if len(observed.Capabilities) != 0 {
		t.Fatal("audit expectation changed: capabilities are now projected")
	}
	if err := conformance.ValidateCacheUsageBaselineUnavailable(observed); err != nil {
		t.Fatalf("cache usage baseline violated: %v", err)
	}

	gaps := conformance.ClassifyRequestGaps(source, observed)
	if strings.Join(gaps, ",") != strings.Join(auditDeclaredGaps, ",") {
		t.Fatalf("request projection gaps changed: got %v want %v", gaps, auditDeclaredGaps)
	}

	first, err := conformance.RequestProjectionDigest(observed)
	if err != nil {
		t.Fatalf("digest: %v", err)
	}
	raw, err := conformance.CanonicalRequestProjection(observed)
	if err != nil {
		t.Fatalf("canonical: %v", err)
	}
	var reparsed conformance.RequestProjection
	if err := json.Unmarshal(raw, &reparsed); err != nil {
		t.Fatalf("round-trip decode: %v", err)
	}
	replay, err := conformance.RequestProjectionDigest(reparsed)
	if err != nil {
		t.Fatalf("digest: %v", err)
	}
	if first != replay {
		t.Fatalf("request projection digest is not idempotent: %q != %q", first, replay)
	}
}
