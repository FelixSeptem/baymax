package gemini

import (
	"context"
	"errors"
	"iter"
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
