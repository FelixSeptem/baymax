package integration

import (
	"context"
	"errors"
	"iter"
	"strings"
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/model/anthropic"
	"github.com/FelixSeptem/baymax/model/gemini"
	"github.com/FelixSeptem/baymax/model/openai"
	providererror "github.com/FelixSeptem/baymax/model/providererror"
	anthropicsdk "github.com/anthropics/anthropic-sdk-go"
	"google.golang.org/genai"
)

func TestProviderStreamPreservesEmptyAndUnicodeContent(t *testing.T) {
	const unicode = "line\u2028separator\u2029"
	cases := map[string]types.ModelClient{
		"openai": openai.NewClient(openai.Config{StreamFn: func(_ context.Context, _ types.ModelRequest, on func(types.ModelEvent) error) error {
			if err := on(types.ModelEvent{Type: types.ModelEventTypeOutputTextDelta, TextDelta: ""}); err != nil {
				return err
			}
			return on(types.ModelEvent{Type: types.ModelEventTypeOutputTextDelta, TextDelta: unicode})
		}}),
		"anthropic": anthropic.NewClient(anthropic.Config{StreamFn: func(context.Context, string) anthropic.Stream {
			return &edgeAnthropicStream{events: []anthropicsdk.MessageStreamEventUnion{
				{Type: "content_block_delta", Index: 0, Delta: anthropicsdk.MessageStreamEventUnionDelta{Type: "text_delta", Text: ""}},
				{Type: "content_block_delta", Index: 0, Delta: anthropicsdk.MessageStreamEventUnionDelta{Type: "text_delta", Text: unicode}},
				{Type: "message_stop"},
			}}
		}}),
		"gemini": mustGeminiClient(t, gemini.Config{StreamFn: func(context.Context, string) iter.Seq2[*genai.GenerateContentResponse, error] {
			return func(yield func(*genai.GenerateContentResponse, error) bool) {
				yield(&genai.GenerateContentResponse{Candidates: []*genai.Candidate{{Content: &genai.Content{Parts: []*genai.Part{{Text: ""}, {Text: unicode}}}}}}, nil)
			}
		}}),
	}
	for name, client := range cases {
		t.Run(name, func(t *testing.T) {
			var events []types.ModelEvent
			err := client.Stream(context.Background(), types.ModelRequest{Input: "x"}, func(ev types.ModelEvent) error {
				events = append(events, ev)
				return nil
			})
			if err != nil {
				t.Fatalf("stream: %v", err)
			}
			var got strings.Builder
			empty := false
			for _, ev := range events {
				if ev.Type == types.ModelEventTypeOutputTextDelta {
					if ev.TextDelta == "" {
						empty = true
					}
					got.WriteString(ev.TextDelta)
				}
			}
			if !empty || got.String() != unicode {
				t.Fatalf("content=%q empty=%v, want empty event and %q", got.String(), empty, unicode)
			}
		})
	}
}

func TestCanonicalFeedbackRejectsBoundedOverflowBeforeInvocation(t *testing.T) {
	called := false
	client := openai.NewClient(openai.Config{GenerateFn: func(context.Context, types.ModelRequest) (types.ModelResponse, error) {
		called = true
		return types.ModelResponse{}, nil
	}})
	_, err := client.Generate(context.Background(), types.ModelRequest{Input: "x", ToolResult: []types.ToolCallOutcome{{CallID: "call-1", Name: "tool", Result: types.ToolResult{Content: strings.Repeat("x", 70*1024)}}}})
	if err == nil {
		t.Fatal("expected overflow")
	}
	var classified *providererror.Classified
	if !errors.As(err, &classified) || classified.Reason != "overflow" {
		t.Fatalf("error=%v, want overflow classification", err)
	}
	if called {
		t.Fatal("provider invocation must not occur for oversized feedback")
	}
}

type edgeAnthropicStream struct {
	events []anthropicsdk.MessageStreamEventUnion
	index  int
}

func (s *edgeAnthropicStream) Next() bool { s.index++; return s.index <= len(s.events) }
func (s *edgeAnthropicStream) Current() anthropicsdk.MessageStreamEventUnion {
	return s.events[s.index-1]
}
func (s *edgeAnthropicStream) Err() error   { return nil }
func (s *edgeAnthropicStream) Close() error { return nil }
