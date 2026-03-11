package runner

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/tool/local"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type fakeModel struct {
	generate func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error)
	stream   func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error
}

func (f *fakeModel) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
	if f.generate == nil {
		return types.ModelResponse{}, nil
	}
	return f.generate(ctx, req)
}

func (f *fakeModel) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	if f.stream == nil {
		return nil
	}
	return f.stream(ctx, req, onEvent)
}

type fakeTool struct {
	name   string
	schema map[string]any
	invoke func(ctx context.Context, args map[string]any) (types.ToolResult, error)
}

func (t *fakeTool) Name() string               { return t.name }
func (t *fakeTool) Description() string        { return "fake" }
func (t *fakeTool) JSONSchema() map[string]any { return t.schema }
func (t *fakeTool) Invoke(ctx context.Context, args map[string]any) (types.ToolResult, error) {
	if t.invoke != nil {
		return t.invoke(ctx, args)
	}
	return types.ToolResult{}, nil
}

type eventCollector struct {
	mu    sync.Mutex
	types []string
	evs   []types.Event
}

func (c *eventCollector) OnEvent(ctx context.Context, ev types.Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.types = append(c.types, ev.Type)
	c.evs = append(c.evs, ev)
}

func TestRunNormalCompletionAndEvents(t *testing.T) {
	model := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{
				FinalAnswer: "ok",
				Usage:       types.TokenUsage{InputTokens: 1, OutputTokens: 2, TotalTokens: 3},
			}, nil
		},
	}
	r := New(model)
	r.newRunID = func() string { return "run-fixed" }
	collector := &eventCollector{}

	res, err := r.Run(context.Background(), types.RunRequest{Input: "hi"}, collector)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if res.FinalAnswer != "ok" {
		t.Fatalf("FinalAnswer = %q, want ok", res.FinalAnswer)
	}
	if len(collector.types) != 4 {
		t.Fatalf("event count = %d, want 4", len(collector.types))
	}
	want := []string{"run.started", "model.requested", "model.completed", "run.finished"}
	for i := range want {
		if collector.types[i] != want[i] {
			t.Fatalf("event[%d] = %q, want %q", i, collector.types[i], want[i])
		}
	}
}

func TestRunTimeoutAbort(t *testing.T) {
	model := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			<-ctx.Done()
			return types.ModelResponse{}, ctx.Err()
		},
	}
	r := New(model)
	collector := &eventCollector{}
	policy := types.DefaultLoopPolicy()
	policy.StepTimeout = 10 * time.Millisecond

	res, err := r.Run(context.Background(), types.RunRequest{Input: "hi", Policy: &policy}, collector)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want deadline exceeded", err)
	}
	if res.Error == nil || res.Error.Class != types.ErrPolicyTimeout {
		t.Fatalf("error class = %#v, want ErrPolicyTimeout", res.Error)
	}
	if collector.types[len(collector.types)-1] != "run.finished" {
		t.Fatalf("last event = %q, want run.finished", collector.types[len(collector.types)-1])
	}
}

func TestRunIterationLimitAbort(t *testing.T) {
	model := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{
				ToolCalls: []types.ToolCall{{CallID: "c1", Name: "local.search"}},
			}, nil
		},
	}
	r := New(model)
	policy := types.DefaultLoopPolicy()
	policy.MaxIterations = 2

	res, err := r.Run(context.Background(), types.RunRequest{Input: "hi", Policy: &policy}, nil)
	if err == nil {
		t.Fatal("expected iteration-limit error, got nil")
	}
	if res.Error == nil || res.Error.Class != types.ErrIterationLimit {
		t.Fatalf("error class = %#v, want ErrIterationLimit", res.Error)
	}
	if res.Iterations != 2 {
		t.Fatalf("iterations = %d, want 2", res.Iterations)
	}
}

func TestStreamForwardsDelta(t *testing.T) {
	model := &fakeModel{
		stream: func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
			if err := onEvent(types.ModelEvent{Type: "final_answer", TextDelta: "he"}); err != nil {
				return err
			}
			return onEvent(types.ModelEvent{Type: "final_answer", TextDelta: "llo"})
		},
	}
	r := New(model)
	collector := &eventCollector{}

	res, err := r.Stream(context.Background(), types.RunRequest{Input: "hello"}, collector)
	if err != nil {
		t.Fatalf("Stream returned error: %v", err)
	}
	if res.FinalAnswer != "hello" {
		t.Fatalf("FinalAnswer = %q, want hello", res.FinalAnswer)
	}
	if len(collector.types) < 5 {
		t.Fatalf("event count = %d, want at least 5", len(collector.types))
	}
}

func TestStreamAggregatesNativeOutputTextDelta(t *testing.T) {
	model := &fakeModel{
		stream: func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
			if err := onEvent(types.ModelEvent{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "he"}); err != nil {
				return err
			}
			return onEvent(types.ModelEvent{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "llo"})
		},
	}
	r := New(model)
	collector := &eventCollector{}

	res, err := r.Stream(context.Background(), types.RunRequest{Input: "hello"}, collector)
	if err != nil {
		t.Fatalf("Stream returned error: %v", err)
	}
	if res.FinalAnswer != "hello" {
		t.Fatalf("FinalAnswer = %q, want hello", res.FinalAnswer)
	}
}

func TestStreamFailFastWithErrModel(t *testing.T) {
	model := &fakeModel{
		stream: func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
			_ = onEvent(types.ModelEvent{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "partial"})
			return errors.New("stream failure")
		},
	}
	r := New(model)
	collector := &eventCollector{}

	res, err := r.Stream(context.Background(), types.RunRequest{Input: "hello"}, collector)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if res.Error == nil || res.Error.Class != types.ErrModel {
		t.Fatalf("error class = %#v, want ErrModel", res.Error)
	}
	if collector.types[len(collector.types)-1] != "run.finished" {
		t.Fatalf("last event = %q, want run.finished", collector.types[len(collector.types)-1])
	}
}

func TestRunToolLoopSuccess(t *testing.T) {
	reg := local.NewRegistry()
	_, err := reg.Register(&fakeTool{
		name: "echo",
		schema: map[string]any{
			"type":     "object",
			"required": []any{"q"},
			"properties": map[string]any{
				"q": map[string]any{"type": "string"},
			},
		},
		invoke: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
			return types.ToolResult{Content: args["q"].(string)}, nil
		},
	})
	if err != nil {
		t.Fatalf("register tool: %v", err)
	}

	calls := 0
	model := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			calls++
			if calls == 1 {
				return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "c1", Name: "local.echo", Args: map[string]any{"q": "hello"}}}}, nil
			}
			if len(req.ToolResult) != 1 || req.ToolResult[0].Result.Content != "hello" {
				t.Fatalf("tool feedback not merged: %#v", req.ToolResult)
			}
			return types.ModelResponse{FinalAnswer: "done"}, nil
		},
	}
	r := New(model, WithLocalRegistry(reg))
	res, err := r.Run(context.Background(), types.RunRequest{Input: "do it"}, nil)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if res.FinalAnswer != "done" {
		t.Fatalf("FinalAnswer = %q, want done", res.FinalAnswer)
	}
	if len(res.ToolCalls) != 1 || res.ToolCalls[0].Name != "local.echo" {
		t.Fatalf("tool calls summary mismatch: %#v", res.ToolCalls)
	}
}

func TestRunToolValidationFailureContinue(t *testing.T) {
	reg := local.NewRegistry()
	_, _ = reg.Register(&fakeTool{
		name: "search",
		schema: map[string]any{
			"type":     "object",
			"required": []any{"q"},
			"properties": map[string]any{
				"q": map[string]any{"type": "string"},
			},
		},
	})

	calls := 0
	model := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			calls++
			if calls == 1 {
				return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "c1", Name: "local.search", Args: map[string]any{}}}}, nil
			}
			if len(req.ToolResult) != 1 || req.ToolResult[0].Result.Error == nil {
				t.Fatalf("expected validation error feedback, got %#v", req.ToolResult)
			}
			return types.ModelResponse{FinalAnswer: "fallback"}, nil
		},
	}

	policy := types.DefaultLoopPolicy()
	policy.ContinueOnToolError = true
	r := New(model, WithLocalRegistry(reg))
	res, err := r.Run(context.Background(), types.RunRequest{Input: "x", Policy: &policy}, nil)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if res.FinalAnswer != "fallback" {
		t.Fatalf("FinalAnswer = %q, want fallback", res.FinalAnswer)
	}
}

func TestRunToolFailurePolicy(t *testing.T) {
	newRunner := func(continueOnError bool) (*Engine, *int) {
		reg := local.NewRegistry()
		_, _ = reg.Register(&fakeTool{
			name: "boom",
			invoke: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
				return types.ToolResult{}, errors.New("boom")
			},
		})
		calls := 0
		model := &fakeModel{
			generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
				calls++
				if calls == 1 {
					return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "c1", Name: "local.boom"}}}, nil
				}
				return types.ModelResponse{FinalAnswer: "after-error"}, nil
			},
		}
		policy := types.DefaultLoopPolicy()
		policy.ContinueOnToolError = continueOnError
		engine := New(model, WithLocalRegistry(reg))
		_ = policy
		return engine, &calls
	}

	t.Run("fail-fast", func(t *testing.T) {
		engine, _ := newRunner(false)
		policy := types.DefaultLoopPolicy()
		policy.ContinueOnToolError = false
		res, err := engine.Run(context.Background(), types.RunRequest{Input: "x", Policy: &policy}, nil)
		if err == nil {
			t.Fatal("expected error in fail-fast mode")
		}
		if res.Error == nil || res.Error.Class != types.ErrTool {
			t.Fatalf("error class = %#v, want ErrTool", res.Error)
		}
	})

	t.Run("continue", func(t *testing.T) {
		engine, calls := newRunner(true)
		policy := types.DefaultLoopPolicy()
		policy.ContinueOnToolError = true
		res, err := engine.Run(context.Background(), types.RunRequest{Input: "x", Policy: &policy}, nil)
		if err != nil {
			t.Fatalf("unexpected error in continue mode: %v", err)
		}
		if res.FinalAnswer != "after-error" {
			t.Fatalf("FinalAnswer = %q, want after-error", res.FinalAnswer)
		}
		if *calls < 2 {
			t.Fatalf("model calls = %d, want >= 2", *calls)
		}
	})
}

func TestRunEventCorrelationFieldsComplete(t *testing.T) {
	provider := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(provider)
	t.Cleanup(func() {
		_ = provider.Shutdown(context.Background())
	})

	model := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{FinalAnswer: "ok"}, nil
		},
	}
	c := &eventCollector{}
	r := New(model)
	r.newRunID = func() string { return "run-corr" }

	_, err := r.Run(context.Background(), types.RunRequest{Input: "hello"}, c)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if len(c.evs) != 4 {
		t.Fatalf("event count = %d, want 4", len(c.evs))
	}
	for _, ev := range c.evs {
		if ev.Version != types.EventSchemaVersionV1 {
			t.Fatalf("event version = %q, want %q", ev.Version, types.EventSchemaVersionV1)
		}
		if ev.RunID != "run-corr" {
			t.Fatalf("run_id = %q, want run-corr", ev.RunID)
		}
		if ev.TraceID == "" || ev.SpanID == "" {
			t.Fatalf("missing trace/span correlation in event: %#v", ev)
		}
	}
	order := []string{"run.started", "model.requested", "model.completed", "run.finished"}
	for i := range order {
		if c.evs[i].Type != order[i] {
			t.Fatalf("event order mismatch at %d: got %q want %q", i, c.evs[i].Type, order[i])
		}
	}
}
