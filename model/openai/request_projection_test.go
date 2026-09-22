package openai

import (
	"context"
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

func TestRequestProjectionAuditPreservesMessagesWhenInputIsEmpty(t *testing.T) {
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

	items := captured.Input.OfInputItemList
	if len(items) != len(req.Messages)+len(req.ToolResult) {
		t.Fatalf("input item count = %d, want %d", len(items), len(req.Messages)+len(req.ToolResult))
	}
	if items[0].OfMessage == nil || items[0].OfMessage.Role != responses.EasyInputMessageRoleSystem {
		t.Fatalf("system message was not preserved: %#v", items[0])
	}
}

func assertOpenAIProjection(t *testing.T, captured responses.ResponseNewParams, req types.ModelRequest) {
	t.Helper()

	if captured.Input.OfString.Valid() {
		t.Fatal("request must not collapse canonical facts into a single text input")
	}
	items := captured.Input.OfInputItemList
	if len(items) != len(req.Messages)+1+len(req.ToolResult) {
		t.Fatalf("input item count = %d, want %d", len(items), len(req.Messages)+1+len(req.ToolResult))
	}
	wantRoles := []responses.EasyInputMessageRole{
		responses.EasyInputMessageRoleSystem,
		responses.EasyInputMessageRoleUser,
		responses.EasyInputMessageRoleAssistant,
		responses.EasyInputMessageRoleUser,
	}
	wantContent := []string{
		"skill fragment: always cite files",
		"summarize the repository",
		"working on it",
		"summarize the repository",
	}
	for i := range wantRoles {
		message := items[i].OfMessage
		if message == nil || message.Role != wantRoles[i] || message.Content.OfString.Value != wantContent[i] {
			t.Fatalf("input item %d = %#v, want role=%q content=%q", i, items[i], wantRoles[i], wantContent[i])
		}
	}
	output := items[len(wantRoles)].OfFunctionCallOutput
	if output == nil || output.CallID != "call-1" {
		t.Fatalf("tool result is not an associated native function output: %#v", items[len(wantRoles)])
	}
	if strings.Contains(output.Output, toolcontract.FeedbackHeader) ||
		!strings.Contains(output.Output, `"tool_name":"read_file"`) ||
		!strings.Contains(output.Output, `"content":"file body"`) {
		t.Fatalf("native function output did not preserve tool identity/result: %q", output.Output)
	}
}
