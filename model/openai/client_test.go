package openai

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	providererror "github.com/FelixSeptem/baymax/model/providererror"
	"github.com/openai/openai-go/responses"
)

type fakeResponseStream struct {
	events []responses.ResponseStreamEventUnion
	err    error
	index  int
}

func (s *fakeResponseStream) Next() bool {
	if s.index >= len(s.events) {
		return false
	}
	s.index++
	return true
}

func (s *fakeResponseStream) Current() responses.ResponseStreamEventUnion {
	return s.events[s.index-1]
}

func (s *fakeResponseStream) Err() error {
	return s.err
}

func (s *fakeResponseStream) Close() error {
	return nil
}

func TestStreamMapsNativeEventsAndEmitsCompleteToolCall(t *testing.T) {
	stream := &fakeResponseStream{
		events: []responses.ResponseStreamEventUnion{
			{
				Type:   "response.output_item.done",
				ItemID: "item-1",
				Item: responses.ResponseOutputItemUnion{
					Type:   "function_call",
					CallID: "call-1",
					Name:   "local.weather",
				},
			},
			{
				Type:   "response.function_call_arguments.delta",
				ItemID: "item-1",
				Delta: responses.ResponseStreamEventUnionDelta{
					OfString: `{"city":"`,
				},
			},
			{
				Type:      "response.function_call_arguments.done",
				ItemID:    "item-1",
				Arguments: `{"city":"shanghai"}`,
			},
			{
				Type: "response.output_text.delta",
				Delta: responses.ResponseStreamEventUnionDelta{
					OfString: "hello ",
				},
			},
			{
				Type: "response.output_text.delta",
				Delta: responses.ResponseStreamEventUnionDelta{
					OfString: "world",
				},
			},
		},
	}
	client := NewClient(Config{Model: "gpt-4.1-mini"})
	client.newStream = func(ctx context.Context, body responses.ResponseNewParams) responseStream {
		return stream
	}

	var out []types.ModelEvent
	err := client.Stream(context.Background(), types.ModelRequest{Input: "x"}, func(ev types.ModelEvent) error {
		out = append(out, ev)
		return nil
	})
	if err != nil {
		t.Fatalf("Stream returned error: %v", err)
	}

	toolEvents := 0
	for _, ev := range out {
		if ev.Type != types.ModelEventTypeToolCall {
			continue
		}
		toolEvents++
		if ev.ToolCall == nil {
			t.Fatalf("tool event missing tool call payload: %#v", ev)
		}
		if ev.ToolCall.CallID != "call-1" || ev.ToolCall.Name != "local.weather" {
			t.Fatalf("unexpected tool call identity: %#v", ev.ToolCall)
		}
		if ev.ToolCall.Args["city"] != "shanghai" {
			t.Fatalf("unexpected tool args: %#v", ev.ToolCall.Args)
		}
		if ev.Meta["tool_call_id"] != "call-1" || ev.Meta["tool_name"] != "local.weather" {
			t.Fatalf("unexpected tool call meta: %#v", ev.Meta)
		}
	}
	if toolEvents != 1 {
		t.Fatalf("tool events = %d, want 1", toolEvents)
	}
}

func TestStreamFailsFastOnErrorEvent(t *testing.T) {
	client := NewClient(Config{Model: "gpt-4.1-mini"})
	client.newStream = func(ctx context.Context, body responses.ResponseNewParams) responseStream {
		return &fakeResponseStream{
			events: []responses.ResponseStreamEventUnion{
				{
					Type:    "error",
					Message: "stream exploded",
				},
			},
		}
	}

	err := client.Stream(context.Background(), types.ModelRequest{Input: "x"}, nil)
	if err == nil || err.Error() != "stream exploded" {
		t.Fatalf("err = %v, want stream exploded", err)
	}
}

func TestStreamReturnsDecoderError(t *testing.T) {
	client := NewClient(Config{Model: "gpt-4.1-mini"})
	client.newStream = func(ctx context.Context, body responses.ResponseNewParams) responseStream {
		return &fakeResponseStream{err: io.ErrUnexpectedEOF}
	}

	err := client.Stream(context.Background(), types.ModelRequest{Input: "x"}, nil)
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("err = %v, want unexpected EOF", err)
	}
}

func TestStreamFailsOnInvalidToolArguments(t *testing.T) {
	client := NewClient(Config{Model: "gpt-4.1-mini"})
	client.newStream = func(ctx context.Context, body responses.ResponseNewParams) responseStream {
		return &fakeResponseStream{
			events: []responses.ResponseStreamEventUnion{
				{
					Type:   "response.output_item.done",
					ItemID: "item-1",
					Item: responses.ResponseOutputItemUnion{
						Type:   "function_call",
						CallID: "call-1",
						Name:   "local.invalid",
					},
				},
				{
					Type:      "response.function_call_arguments.done",
					ItemID:    "item-1",
					Arguments: `{"broken":`,
				},
			},
		}
	}

	err := client.Stream(context.Background(), types.ModelRequest{Input: "x"}, nil)
	if err == nil {
		t.Fatal("expected parsing error, got nil")
	}
	var classified *providererror.Classified
	if !errors.As(err, &classified) {
		t.Fatalf("expected provider classified error, got %T", err)
	}
	if classified.Reason != "request_invalid" {
		t.Fatalf("reason=%q, want request_invalid", classified.Reason)
	}
}

func TestGenerateInjectsCanonicalToolFeedbackIntoRequest(t *testing.T) {
	var captured string
	client := NewClient(Config{
		Model: "gpt-4.1-mini",
		GenerateFn: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			captured = req.Input
			return types.ModelResponse{FinalAnswer: "ok"}, nil
		},
	})
	_, err := client.Generate(context.Background(), types.ModelRequest{
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
	client := NewClient(Config{
		Model: "gpt-4.1-mini",
		GenerateFn: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			called = true
			return types.ModelResponse{FinalAnswer: "ok"}, nil
		},
	})
	_, err := client.Generate(context.Background(), types.ModelRequest{
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
		t.Fatal("GenerateFn should not be called for invalid feedback")
	}
}

func TestDiscoverCapabilitiesUsesConfiguredDiscoverFn(t *testing.T) {
	client := NewClient(Config{
		Model: "gpt-4.1-mini",
		DiscoverFn: func(ctx context.Context, model string) (types.ProviderCapabilities, error) {
			return types.ProviderCapabilities{
				Provider:  "openai",
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
	got, err := client.DiscoverCapabilities(context.Background(), types.ModelRequest{})
	if err != nil {
		t.Fatalf("DiscoverCapabilities failed: %v", err)
	}
	if got.Provider != "openai" || got.Model != "gpt-4.1-mini" {
		t.Fatalf("unexpected capability report: %#v", got)
	}
}

func TestCountTokensReturnsUnsupportedError(t *testing.T) {
	client := NewClient(Config{Model: "gpt-4.1-mini"})
	_, err := client.CountTokens(context.Background(), types.ModelRequest{Input: "hello"})
	if err == nil {
		t.Fatal("expected unsupported token count error")
	}
	if err.Error() != "openai official sdk does not provide token count api in this adapter" {
		t.Fatalf("unexpected error: %v", err)
	}
}
