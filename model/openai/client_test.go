package openai

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
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
}
