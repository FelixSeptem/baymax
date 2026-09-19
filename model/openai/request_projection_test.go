package openai

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/model/conformance"
	"github.com/FelixSeptem/baymax/model/toolcontract"
	"github.com/openai/openai-go/responses"
)

// auditModelRequest is the provider-neutral request shape the runtime admits
// once Skill bundle mapping has appended system-role prompt fragments and the
// ReAct loop has produced canonical tool-result feedback.
func auditModelRequest() types.ModelRequest {
	return types.ModelRequest{
		RunID: "run-audit",
		Model: "gpt-4.1-mini",
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

// auditDeclaredGaps is the pinned request projection gap set for the OpenAI
// adapter. It is evidence, not a tolerance: fixing any of these requires an
// explicit contract change (see openspec change
// establish-provider-request-projection-and-cache-observability-contract).
var auditDeclaredGaps = []string{
	conformance.ReasonRequestCapabilityProjectionDrift,
	conformance.ReasonRequestRoleProjectionDrift,
	conformance.ReasonRequestToolResultNativeDrift,
}

func TestRequestProjectionAuditStreamShape(t *testing.T) {
	var captured responses.ResponseNewParams
	client := NewClient(Config{Model: "gpt-4.1-mini"})
	client.newStream = func(ctx context.Context, body responses.ResponseNewParams) responseStream {
		captured = body
		return &fakeResponseStream{}
	}

	req := auditModelRequest()
	if err := client.Stream(context.Background(), req, func(types.ModelEvent) error { return nil }); err != nil {
		t.Fatalf("Stream returned error: %v", err)
	}

	assertOpenAIProjection(t, captured, req)
}

func TestRequestProjectionAuditRunShape(t *testing.T) {
	var captured responses.ResponseNewParams
	client := NewClient(Config{Model: "gpt-4.1-mini"})
	client.newResponse = func(ctx context.Context, body responses.ResponseNewParams) (*responses.Response, error) {
		captured = body
		return &responses.Response{}, nil
	}

	req := auditModelRequest()
	if _, err := client.Generate(context.Background(), req); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	assertOpenAIProjection(t, captured, req)
}

func TestRequestProjectionAuditRunStreamParity(t *testing.T) {
	req := auditModelRequest()

	var streamParams responses.ResponseNewParams
	streamClient := NewClient(Config{Model: "gpt-4.1-mini"})
	streamClient.newStream = func(ctx context.Context, body responses.ResponseNewParams) responseStream {
		streamParams = body
		return &fakeResponseStream{}
	}
	if err := streamClient.Stream(context.Background(), req, func(types.ModelEvent) error { return nil }); err != nil {
		t.Fatalf("Stream returned error: %v", err)
	}

	var runParams responses.ResponseNewParams
	runClient := NewClient(Config{Model: "gpt-4.1-mini"})
	runClient.newResponse = func(ctx context.Context, body responses.ResponseNewParams) (*responses.Response, error) {
		runParams = body
		return &responses.Response{}, nil
	}
	if _, err := runClient.Generate(context.Background(), req); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	source := conformance.RequestFactsFromModelRequest(req)
	runProjection := conformance.ProjectRequestTextEnvelope(runParams.Input.OfString.Value, source.ToolResults)
	streamProjection := conformance.ProjectRequestTextEnvelope(streamParams.Input.OfString.Value, source.ToolResults)
	if code := conformance.ClassifyRunStreamParity(runProjection, streamProjection); code != "" {
		t.Fatalf("run/stream request projection parity violated: %s", code)
	}
}

func TestRequestProjectionAuditFallsBackToLastMessageOnly(t *testing.T) {
	// With no Input, the adapter falls back to the last message content only,
	// so earlier roles are still lost and the system fragment still never
	// reaches the SDK.
	req := auditModelRequest()
	req.Input = ""

	var captured responses.ResponseNewParams
	client := NewClient(Config{Model: "gpt-4.1-mini"})
	client.newStream = func(ctx context.Context, body responses.ResponseNewParams) responseStream {
		captured = body
		return &fakeResponseStream{}
	}
	if err := client.Stream(context.Background(), req, func(types.ModelEvent) error { return nil }); err != nil {
		t.Fatalf("Stream returned error: %v", err)
	}

	projected := captured.Input.OfString.Value
	if !strings.Contains(projected, "working on it") {
		t.Fatalf("expected last message content to be used as fallback, got %q", projected)
	}
	if strings.Contains(projected, "skill fragment") {
		t.Fatalf("system-role skill fragment reached the SDK: %q", projected)
	}
}

func assertOpenAIProjection(t *testing.T, captured responses.ResponseNewParams, req types.ModelRequest) {
	t.Helper()

	if !captured.Input.OfString.Valid() {
		t.Fatal("audit expectation: the request input must be projected as a single string")
	}
	projected := captured.Input.OfString.Value
	if strings.TrimSpace(projected) == "" {
		t.Fatal("projected input is empty")
	}
	if len(captured.Input.OfInputItemList) != 0 {
		t.Fatalf("audit expectation changed: input items are now projected (%d)", len(captured.Input.OfInputItemList))
	}
	if captured.Instructions.Valid() {
		t.Fatal("audit expectation changed: system instructions are now projected")
	}
	if strings.Contains(projected, "skill fragment") {
		t.Fatal("audit expectation changed: system-role skill fragment reached the SDK")
	}
	if strings.Contains(projected, "working on it") {
		t.Fatal("audit expectation changed: assistant history reached the SDK")
	}
	if !strings.Contains(projected, toolcontract.FeedbackHeader) {
		t.Fatal("canonical tool-result feedback envelope is missing from the projected input")
	}

	source := conformance.RequestFactsFromModelRequest(req)
	observed := conformance.ProjectRequestTextEnvelope(projected, source.ToolResults)
	if !observed.ToolResultCorrelated {
		t.Fatal("tool-result correlation must be preserved inside the text envelope")
	}

	gaps := conformance.ClassifyRequestGaps(source, observed)
	if strings.Join(gaps, ",") != strings.Join(auditDeclaredGaps, ",") {
		t.Fatalf("request projection gaps changed: got %v want %v", gaps, auditDeclaredGaps)
	}

	if err := conformance.ValidateCacheUsageBaselineUnavailable(observed); err != nil {
		t.Fatalf("cache usage baseline violated: %v", err)
	}

	first, err := conformance.RequestProjectionDigest(observed)
	if err != nil {
		t.Fatalf("digest: %v", err)
	}
	canonical, err := conformance.CanonicalRequestProjection(observed)
	if err != nil {
		t.Fatalf("canonical: %v", err)
	}
	var reparsed conformance.RequestProjection
	if err := json.Unmarshal(canonical, &reparsed); err != nil {
		t.Fatalf("round-trip decode: %v", err)
	}
	replay, err := conformance.RequestProjectionDigest(reparsed)
	if err != nil {
		t.Fatalf("digest: %v", err)
	}
	if first != replay {
		t.Fatalf("request projection digest is not idempotent: %q != %q", first, replay)
	}

	if len(observed.Capabilities) != 0 {
		t.Fatal("audit expectation changed: capabilities are now projected")
	}
}
