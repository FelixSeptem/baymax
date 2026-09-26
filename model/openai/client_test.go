package openai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/model/conformance"
	providererror "github.com/FelixSeptem/baymax/model/providererror"
	"github.com/FelixSeptem/baymax/model/toolcontract"
	"github.com/openai/openai-go/responses"
)

func TestProjectResponseCacheUsageMapsExplicitCachedTokens(t *testing.T) {
	var usage responses.ResponseUsage
	if err := json.Unmarshal([]byte(`{"input_tokens":30,"input_tokens_details":{"cached_tokens":12},"output_tokens":2,"output_tokens_details":{"reasoning_tokens":0},"total_tokens":32}`), &usage); err != nil {
		t.Fatalf("decode usage: %v", err)
	}
	got, err := projectResponseCacheUsage(usage)
	if err != nil {
		t.Fatalf("project cache usage: %v", err)
	}
	want := conformance.CacheUsageProjection{Available: true, ReadTokens: 12, TotalTokens: 12, SourceKind: conformance.CacheUsageSourceOpenAIResponses, SourceVersion: "v1"}
	if got != want {
		t.Fatalf("projection = %+v, want %+v", got, want)
	}
}

func TestProjectResponseCacheUsageRejectsMissingAndNegativeCachedTokens(t *testing.T) {
	for _, raw := range []string{
		`{"input_tokens":30,"input_tokens_details":{},"output_tokens":2,"output_tokens_details":{},"total_tokens":32}`,
		`{"input_tokens":30,"input_tokens_details":{"cached_tokens":-1},"output_tokens":2,"output_tokens_details":{},"total_tokens":32}`,
	} {
		var usage responses.ResponseUsage
		if err := json.Unmarshal([]byte(raw), &usage); err != nil {
			t.Fatalf("decode usage: %v", err)
		}
		got, err := projectResponseCacheUsage(usage)
		if err == nil || got.Available || got.ReadTokens != 0 || got.TotalTokens != 0 {
			t.Fatalf("invalid usage %s produced projection %+v and error %v", raw, got, err)
		}
	}
}

func TestGenerateProjectsCacheUsageOnModelResponse(t *testing.T) {
	var response responses.Response
	if err := json.Unmarshal([]byte(`{"output":[],"usage":{"input_tokens":30,"input_tokens_details":{"cached_tokens":12},"output_tokens":2,"output_tokens_details":{"reasoning_tokens":0},"total_tokens":32}}`), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	c := NewClient(Config{})
	c.newResponse = func(context.Context, responses.ResponseNewParams) (*responses.Response, error) { return &response, nil }
	got, err := c.Generate(context.Background(), types.ModelRequest{Input: "hello"})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	want := types.CacheUsageProjection{Available: true, ReadTokens: 12, TotalTokens: 12, SourceKind: conformance.CacheUsageSourceOpenAIResponses, SourceVersion: "v1"}
	if got.CacheUsage != want {
		t.Fatalf("cache usage = %+v, want %+v", got.CacheUsage, want)
	}
}

func TestStreamProjectsCacheUsageOnlyOnCompletedEvent(t *testing.T) {
	var completed responses.ResponseStreamEventUnion
	if err := json.Unmarshal([]byte(`{"type":"response.completed","sequence_number":2,"response":{"output":[],"usage":{"input_tokens":30,"input_tokens_details":{"cached_tokens":12},"output_tokens":2,"output_tokens_details":{"reasoning_tokens":0},"total_tokens":32}}}`), &completed); err != nil {
		t.Fatalf("decode completed event: %v", err)
	}
	c := NewClient(Config{})
	c.newStream = func(context.Context, responses.ResponseNewParams) responseStream {
		return &fakeResponseStream{events: []responses.ResponseStreamEventUnion{{Type: "response.in_progress"}, completed}}
	}
	var events []types.ModelEvent
	if err := c.Stream(context.Background(), types.ModelRequest{Input: "hello"}, func(ev types.ModelEvent) error {
		events = append(events, ev)
		return nil
	}); err != nil {
		t.Fatalf("Stream error: %v", err)
	}
	if _, ok := events[0].Meta["cache_usage"]; ok {
		t.Fatalf("interim event must not carry cache usage: %+v", events[0])
	}
	got, ok := events[len(events)-1].Meta["cache_usage"].(types.CacheUsageProjection)
	if !ok || !got.Available || got.ReadTokens != 12 {
		t.Fatalf("completed cache usage = %#v", events[len(events)-1].Meta["cache_usage"])
	}
}

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
	var classified *providererror.Classified
	if !errors.As(err, &classified) || classified.StreamPhase != "pre_execution" {
		t.Fatalf("decoder error = %T %#v, want classified pre_execution", err, err)
	}
}

func TestStreamClassifiesPreAndPostStartDecoderErrors(t *testing.T) {
	preClient := NewClient(Config{Model: "gpt-4.1-mini"})
	preClient.newStream = func(ctx context.Context, body responses.ResponseNewParams) responseStream {
		return &fakeResponseStream{err: io.ErrUnexpectedEOF}
	}
	preErr := preClient.Stream(context.Background(), types.ModelRequest{Input: "x"}, nil)
	var preClassified *providererror.Classified
	if !errors.As(preErr, &preClassified) || preClassified.StreamPhase != "pre_execution" {
		t.Fatalf("pre-start error = %T %#v, want classified pre_execution", preErr, preErr)
	}

	postClient := NewClient(Config{Model: "gpt-4.1-mini"})
	postClient.newStream = func(ctx context.Context, body responses.ResponseNewParams) responseStream {
		return &fakeResponseStream{
			events: []responses.ResponseStreamEventUnion{{
				Type:  "response.output_text.delta",
				Delta: responses.ResponseStreamEventUnionDelta{OfString: "partial"},
			}},
			err: io.ErrUnexpectedEOF,
		}
	}
	postErr := postClient.Stream(context.Background(), types.ModelRequest{Input: "x"}, func(types.ModelEvent) error { return nil })
	var postClassified *providererror.Classified
	if !errors.As(postErr, &postClassified) || postClassified.StreamPhase != "post_start" {
		t.Fatalf("post-start error = %T %#v, want classified post_start", postErr, postErr)
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

func TestGenerateCallbackReceivesCanonicalToolFeedbackWithoutTextEnvelope(t *testing.T) {
	var captured types.ModelRequest
	client := NewClient(Config{
		Model: "gpt-4.1-mini",
		GenerateFn: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			captured = req
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
	if captured.Input != "hello" || len(captured.ToolResult) != 1 ||
		captured.ToolResult[0].CallID != "call-1" || captured.ToolResult[0].Name != "local.echo" {
		t.Fatalf("captured request lost canonical feedback: %#v", captured)
	}
	if strings.Contains(captured.Input, toolcontract.FeedbackHeader) {
		t.Fatalf("callback request must not contain a text feedback envelope: %q", captured.Input)
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
