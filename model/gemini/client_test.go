package gemini

import (
	"context"
	"errors"
	"iter"
	"strings"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	providererror "github.com/FelixSeptem/baymax/model/providererror"
	"google.golang.org/genai"
)

func TestGenerateUsesConfiguredGenerateFn(t *testing.T) {
	c := &Client{
		model: "gemini-2.5-flash",
		generate: func(ctx context.Context, input string) (types.ModelResponse, error) {
			if input != "hello" {
				t.Fatalf("input = %q, want hello", input)
			}
			return types.ModelResponse{
				FinalAnswer: "ok",
				Usage:       types.TokenUsage{InputTokens: 2, OutputTokens: 3, TotalTokens: 5},
			}, nil
		},
	}
	got, err := c.Generate(context.Background(), types.ModelRequest{Input: "hello"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if got.FinalAnswer != "ok" || got.Usage.TotalTokens != 5 {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestGenerateClassifiesTimeoutErrors(t *testing.T) {
	c := &Client{
		model: "gemini-2.5-flash",
		generate: func(ctx context.Context, input string) (types.ModelResponse, error) {
			return types.ModelResponse{}, providererror.FromError(errors.New("request timeout"))
		},
	}
	_, err := c.Generate(context.Background(), types.ModelRequest{Input: "x"})
	if err == nil {
		t.Fatal("expected error")
	}
	var classified *providererror.Classified
	if !errors.As(err, &classified) {
		t.Fatalf("expected classified provider error, got %T", err)
	}
	if classified.Class != types.ErrPolicyTimeout {
		t.Fatalf("class = %q, want %q", classified.Class, types.ErrPolicyTimeout)
	}
}

func TestGenerateInjectsCanonicalToolFeedback(t *testing.T) {
	captured := ""
	c := &Client{
		model: "gemini-2.5-flash",
		generate: func(ctx context.Context, input string) (types.ModelResponse, error) {
			captured = input
			return types.ModelResponse{FinalAnswer: "ok"}, nil
		},
	}
	_, err := c.Generate(context.Background(), types.ModelRequest{
		Input: "hello",
		ToolResult: []types.ToolCallOutcome{
			{
				CallID: "call-1",
				Name:   "local.echo",
				Result: types.ToolResult{Content: "done"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if !strings.Contains(captured, "[tool_result_feedback.v1]") ||
		!strings.Contains(captured, `"tool_call_id":"call-1"`) ||
		!strings.Contains(captured, `"tool_name":"local.echo"`) {
		t.Fatalf("captured canonical feedback missing expected fields: %q", captured)
	}
}

func TestGenerateRejectsInvalidToolFeedback(t *testing.T) {
	called := false
	c := &Client{
		model: "gemini-2.5-flash",
		generate: func(ctx context.Context, input string) (types.ModelResponse, error) {
			called = true
			return types.ModelResponse{FinalAnswer: "ok"}, nil
		},
	}
	_, err := c.Generate(context.Background(), types.ModelRequest{
		Input: "hello",
		ToolResult: []types.ToolCallOutcome{
			{
				CallID: "",
				Name:   "local.echo",
			},
		},
	})
	if err == nil {
		t.Fatal("expected feedback_invalid error")
	}
	var classified *providererror.Classified
	if !errors.As(err, &classified) {
		t.Fatalf("expected provider classified error, got %T", err)
	}
	if classified.Reason != "feedback_invalid" {
		t.Fatalf("reason=%q, want feedback_invalid", classified.Reason)
	}
	if called {
		t.Fatal("generate function should not be called for invalid feedback")
	}
}

func TestStreamEmitsTextAndToolCall(t *testing.T) {
	c := &Client{
		model: "gemini-2.5-flash",
		stream: func(ctx context.Context, input string) iter.Seq2[*genai.GenerateContentResponse, error] {
			return seqFromChunks([]*genai.GenerateContentResponse{
				{Candidates: []*genai.Candidate{{Content: &genai.Content{Parts: []*genai.Part{{Text: "he"}}}}}},
				{Candidates: []*genai.Candidate{{Content: &genai.Content{Parts: []*genai.Part{{
					FunctionCall: &genai.FunctionCall{ID: "call-1", Name: "local.weather", Args: map[string]any{"city": "shanghai"}},
				}}}}}},
				{Candidates: []*genai.Candidate{{Content: &genai.Content{Parts: []*genai.Part{{Text: "llo"}}}}}},
			}, nil)
		},
	}
	var events []types.ModelEvent
	err := c.Stream(context.Background(), types.ModelRequest{Input: "x"}, func(ev types.ModelEvent) error {
		events = append(events, ev)
		return nil
	})
	if err != nil {
		t.Fatalf("Stream error: %v", err)
	}
	text := ""
	toolCount := 0
	for _, ev := range events {
		if ev.Type == types.ModelEventTypeOutputTextDelta {
			text += ev.TextDelta
		}
		if ev.Type == types.ModelEventTypeToolCall {
			toolCount++
			if ev.ToolCall == nil || ev.ToolCall.CallID != "call-1" {
				t.Fatalf("unexpected tool call: %#v", ev.ToolCall)
			}
			if ev.Meta["tool_call_id"] != "call-1" || ev.Meta["tool_name"] != "local.weather" {
				t.Fatalf("unexpected tool call meta: %#v", ev.Meta)
			}
		}
	}
	if text != "hello" {
		t.Fatalf("text = %q, want hello", text)
	}
	if toolCount != 1 {
		t.Fatalf("tool count = %d, want 1", toolCount)
	}
	if events[len(events)-1].Type != types.ModelEventTypeResponseCompleted {
		t.Fatalf("last event = %q, want %q", events[len(events)-1].Type, types.ModelEventTypeResponseCompleted)
	}
}

func TestStreamFailFastAndClassified(t *testing.T) {
	c := &Client{
		model: "gemini-2.5-flash",
		stream: func(ctx context.Context, input string) iter.Seq2[*genai.GenerateContentResponse, error] {
			return seqFromChunks(nil, errors.New("500 internal server error"))
		},
	}
	var events []types.ModelEvent
	err := c.Stream(context.Background(), types.ModelRequest{Input: "x"}, func(ev types.ModelEvent) error {
		events = append(events, ev)
		return nil
	})
	if err == nil {
		t.Fatal("expected stream error")
	}
	var classified *providererror.Classified
	if !errors.As(err, &classified) {
		t.Fatalf("expected classified provider error, got %T", err)
	}
	if classified.Reason != "server" {
		t.Fatalf("reason = %q, want server", classified.Reason)
	}
	if len(events) == 0 || events[0].Type != types.ModelEventTypeResponseError {
		t.Fatalf("expected response.error event, got %#v", events)
	}
}

func seqFromChunks(chunks []*genai.GenerateContentResponse, tailErr error) iter.Seq2[*genai.GenerateContentResponse, error] {
	return func(yield func(*genai.GenerateContentResponse, error) bool) {
		for _, chunk := range chunks {
			if !yield(chunk, nil) {
				return
			}
		}
		if tailErr != nil {
			_ = yield(nil, tailErr)
		}
	}
}

func TestDiscoverCapabilitiesUsesConfiguredDiscoverFn(t *testing.T) {
	c := &Client{
		model: "gemini-2.5-flash",
		discover: func(ctx context.Context, model string) (types.ProviderCapabilities, error) {
			return types.ProviderCapabilities{
				Provider:  "gemini",
				Model:     model,
				Source:    "test",
				CheckedAt: time.Now(),
				Support: map[types.ModelCapability]types.CapabilitySupport{
					types.ModelCapabilityStreaming: types.CapabilitySupportSupported,
					types.ModelCapabilityToolCall:  types.CapabilitySupportUnknown,
				},
			}, nil
		},
	}
	got, err := c.DiscoverCapabilities(context.Background(), types.ModelRequest{})
	if err != nil {
		t.Fatalf("DiscoverCapabilities failed: %v", err)
	}
	if got.Provider != "gemini" || got.Model != "gemini-2.5-flash" {
		t.Fatalf("unexpected capabilities: %#v", got)
	}
}

func TestBuildTokenContentsNormalizesRolesAndKeepsInput(t *testing.T) {
	req := types.ModelRequest{
		Input: "latest question",
		Messages: []types.Message{
			{Role: "system", Content: "policy"},
			{Role: "assistant", Content: "intermediate answer"},
			{Role: "user", Content: "follow-up"},
		},
	}
	contents := buildTokenContents(req)
	if len(contents) != 4 {
		t.Fatalf("contents len = %d, want 4", len(contents))
	}
	if contents[0].Role != "user" {
		t.Fatalf("system role should be normalized to user, got %q", contents[0].Role)
	}
	if contents[1].Role != "model" {
		t.Fatalf("assistant role should be normalized to model, got %q", contents[1].Role)
	}
	if contents[2].Role != "user" {
		t.Fatalf("user role should remain user, got %q", contents[2].Role)
	}
	if text := contents[3].Parts[0].Text; text != "latest question" {
		t.Fatalf("input should be appended as additional content, got %q", text)
	}
}
