package anthropic

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	providererror "github.com/FelixSeptem/baymax/model/providererror"
	anthropic "github.com/anthropics/anthropic-sdk-go"
)

type fakeAnthropicStream struct {
	events []anthropic.MessageStreamEventUnion
	err    error
	index  int
}

func (s *fakeAnthropicStream) Next() bool {
	if s.index >= len(s.events) {
		return false
	}
	s.index++
	return true
}

func (s *fakeAnthropicStream) Current() anthropic.MessageStreamEventUnion {
	return s.events[s.index-1]
}

func (s *fakeAnthropicStream) Err() error   { return s.err }
func (s *fakeAnthropicStream) Close() error { return nil }

func TestGenerateUsesConfiguredGenerateFn(t *testing.T) {
	c := NewClient(Config{
		GenerateFn: func(ctx context.Context, input string) (types.ModelResponse, error) {
			if input != "hello" {
				t.Fatalf("input = %q, want hello", input)
			}
			return types.ModelResponse{
				FinalAnswer: "ok",
				Usage:       types.TokenUsage{InputTokens: 1, OutputTokens: 1, TotalTokens: 2},
			}, nil
		},
	})
	got, err := c.Generate(context.Background(), types.ModelRequest{Input: "hello"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if got.FinalAnswer != "ok" || got.Usage.TotalTokens != 2 {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestGenerateClassifiesProviderErrors(t *testing.T) {
	c := NewClient(Config{
		GenerateFn: func(ctx context.Context, input string) (types.ModelResponse, error) {
			return types.ModelResponse{}, providererror.FromError(errors.New("429 rate limit"))
		},
	})
	_, err := c.Generate(context.Background(), types.ModelRequest{Input: "x"})
	if err == nil {
		t.Fatal("expected error")
	}
	var classified *providererror.Classified
	if !errors.As(err, &classified) {
		t.Fatalf("expected classified provider error, got %T", err)
	}
	if classified.Reason != "rate_limit" {
		t.Fatalf("reason = %q, want rate_limit", classified.Reason)
	}
}

func TestStreamEmitsTextAndCompleteToolCall(t *testing.T) {
	stream := &fakeAnthropicStream{events: []anthropic.MessageStreamEventUnion{
		{Type: "content_block_delta", Index: 0, Delta: anthropic.MessageStreamEventUnionDelta{Type: "text_delta", Text: "hello "}},
		{Type: "content_block_start", Index: 1, ContentBlock: anthropic.ContentBlockStartEventContentBlockUnion{Type: "tool_use", ID: "call-1", Name: "local.weather"}},
		{Type: "content_block_delta", Index: 1, Delta: anthropic.MessageStreamEventUnionDelta{Type: "input_json_delta", PartialJSON: `{"city":"shan`}},
		{Type: "content_block_delta", Index: 1, Delta: anthropic.MessageStreamEventUnionDelta{Type: "input_json_delta", PartialJSON: `ghai"}`}},
		{Type: "content_block_stop", Index: 1},
		{Type: "content_block_delta", Index: 0, Delta: anthropic.MessageStreamEventUnionDelta{Type: "text_delta", Text: "world"}},
		{Type: "message_stop"},
	}}
	c := NewClient(Config{StreamFn: func(ctx context.Context, input string) Stream { return stream }})

	var events []types.ModelEvent
	err := c.Stream(context.Background(), types.ModelRequest{Input: "x"}, func(ev types.ModelEvent) error {
		events = append(events, ev)
		return nil
	})
	if err != nil {
		t.Fatalf("Stream error: %v", err)
	}

	toolCount := 0
	text := ""
	for _, ev := range events {
		if ev.Type == types.ModelEventTypeOutputTextDelta {
			text += ev.TextDelta
		}
		if ev.Type == types.ModelEventTypeToolCall {
			toolCount++
			if ev.ToolCall == nil || ev.ToolCall.Name != "local.weather" {
				t.Fatalf("unexpected tool call: %#v", ev.ToolCall)
			}
			if ev.ToolCall.Args["city"] != "shanghai" {
				t.Fatalf("unexpected tool args: %#v", ev.ToolCall.Args)
			}
		}
	}
	if text != "hello world" {
		t.Fatalf("text = %q, want hello world", text)
	}
	if toolCount != 1 {
		t.Fatalf("tool count = %d, want 1", toolCount)
	}
	if events[len(events)-1].Type != types.ModelEventTypeResponseCompleted {
		t.Fatalf("last event = %q, want %q", events[len(events)-1].Type, types.ModelEventTypeResponseCompleted)
	}
}

func TestStreamFailFastAndClassified(t *testing.T) {
	c := NewClient(Config{StreamFn: func(ctx context.Context, input string) Stream {
		return &fakeAnthropicStream{err: errors.New("503 service unavailable")}
	}})
	var got []types.ModelEvent
	err := c.Stream(context.Background(), types.ModelRequest{Input: "x"}, func(ev types.ModelEvent) error {
		got = append(got, ev)
		return nil
	})
	if err == nil {
		t.Fatal("expected stream error")
	}
	var classified *providererror.Classified
	if !errors.As(err, &classified) {
		t.Fatalf("expected classified error, got %T", err)
	}
	if classified.Reason != "server" {
		t.Fatalf("reason = %q, want server", classified.Reason)
	}
	if len(got) == 0 || got[0].Type != types.ModelEventTypeResponseError {
		t.Fatalf("expected first event response.error, got %#v", got)
	}
}

func TestDiscoverCapabilitiesUsesConfiguredDiscoverFn(t *testing.T) {
	c := NewClient(Config{
		Model: "claude-3-5-sonnet-latest",
		DiscoverFn: func(ctx context.Context, model string) (types.ProviderCapabilities, error) {
			return types.ProviderCapabilities{
				Provider:  "anthropic",
				Model:     model,
				Source:    "test",
				CheckedAt: time.Now(),
				Support: map[types.ModelCapability]types.CapabilitySupport{
					types.ModelCapabilityStreaming: types.CapabilitySupportSupported,
					types.ModelCapabilityToolCall:  types.CapabilitySupportUnknown,
				},
			}, nil
		},
	})
	got, err := c.DiscoverCapabilities(context.Background(), types.ModelRequest{})
	if err != nil {
		t.Fatalf("DiscoverCapabilities failed: %v", err)
	}
	if got.Provider != "anthropic" || got.Model != "claude-3-5-sonnet-latest" {
		t.Fatalf("unexpected capabilities: %#v", got)
	}
}
