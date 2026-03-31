package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/context/assembler"
	"github.com/FelixSeptem/baymax/core/types"
	obsevent "github.com/FelixSeptem/baymax/observability/event"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
	runtimediag "github.com/FelixSeptem/baymax/runtime/diagnostics"
	"github.com/FelixSeptem/baymax/tool/local"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type fakeModel struct {
	generate func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error)
	stream   func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error
	count    func(ctx context.Context, req types.ModelRequest) (int, error)
	provider string
	caps     map[types.ModelCapability]types.CapabilitySupport
	discover error
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

func (f *fakeModel) CountTokens(ctx context.Context, req types.ModelRequest) (int, error) {
	if f.count == nil {
		return 0, errors.New("count tokens not configured")
	}
	return f.count(ctx, req)
}

func (f *fakeModel) ProviderName() string {
	if strings.TrimSpace(f.provider) == "" {
		return "fake"
	}
	return f.provider
}

func (f *fakeModel) DiscoverCapabilities(ctx context.Context, req types.ModelRequest) (types.ProviderCapabilities, error) {
	_ = ctx
	if f.discover != nil {
		return types.ProviderCapabilities{}, f.discover
	}
	support := f.caps
	if support == nil {
		support = map[types.ModelCapability]types.CapabilitySupport{
			types.ModelCapabilityStreaming: types.CapabilitySupportSupported,
			types.ModelCapabilityToolCall:  types.CapabilitySupportSupported,
		}
	}
	return types.ProviderCapabilities{
		Provider:  f.ProviderName(),
		Model:     req.Model,
		Support:   support,
		Source:    "test",
		CheckedAt: time.Now(),
	}, nil
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

type fakeSandboxAdapterRunnerTool struct {
	*fakeTool
	build  func(ctx context.Context, args map[string]any) (types.SandboxExecSpec, error)
	handle func(ctx context.Context, result types.SandboxExecResult) (types.ToolResult, error)
}

func (t *fakeSandboxAdapterRunnerTool) BuildSandboxExecSpec(ctx context.Context, args map[string]any) (types.SandboxExecSpec, error) {
	if t != nil && t.build != nil {
		return t.build(ctx, args)
	}
	return types.SandboxExecSpec{}, nil
}

func (t *fakeSandboxAdapterRunnerTool) HandleSandboxExecResult(ctx context.Context, result types.SandboxExecResult) (types.ToolResult, error) {
	if t != nil && t.handle != nil {
		return t.handle(ctx, result)
	}
	return types.ToolResult{}, nil
}

type fakeSandboxExecutor struct {
	probe   func(ctx context.Context) (types.SandboxCapabilityProbe, error)
	execute func(ctx context.Context, spec types.SandboxExecSpec) (types.SandboxExecResult, error)
}

func (f *fakeSandboxExecutor) Probe(ctx context.Context) (types.SandboxCapabilityProbe, error) {
	if f != nil && f.probe != nil {
		return f.probe(ctx)
	}
	return types.SandboxCapabilityProbe{}, nil
}

func (f *fakeSandboxExecutor) Execute(ctx context.Context, spec types.SandboxExecSpec) (types.SandboxExecResult, error) {
	if f != nil && f.execute != nil {
		return f.execute(ctx, spec)
	}
	return types.SandboxExecResult{}, nil
}

type fakeGateResolver struct {
	confirm func(ctx context.Context, req types.ActionGateConfirmRequest) (bool, error)
}

func (r *fakeGateResolver) Confirm(ctx context.Context, req types.ActionGateConfirmRequest) (bool, error) {
	if r.confirm == nil {
		return true, nil
	}
	return r.confirm(ctx, req)
}

type fakeClarificationResolver struct {
	resolve func(ctx context.Context, req types.ClarificationResolveRequest) (types.ClarificationResponse, error)
}

func (r *fakeClarificationResolver) Resolve(ctx context.Context, req types.ClarificationResolveRequest) (types.ClarificationResponse, error) {
	if r.resolve == nil {
		return types.ClarificationResponse{}, nil
	}
	return r.resolve(ctx, req)
}

type fakeGateMatcher struct {
	evaluate func(ctx context.Context, check types.ActionGateCheck) (types.ActionGateDecision, error)
}

func (m *fakeGateMatcher) Evaluate(ctx context.Context, check types.ActionGateCheck) (types.ActionGateDecision, error) {
	if m.evaluate == nil {
		return types.ActionGateDecisionAllow, nil
	}
	return m.evaluate(ctx, check)
}

type fakeModelInputFilter struct {
	filter func(ctx context.Context, req types.ModelRequest) (types.ModelRequest, types.SecurityFilterResult, error)
}

func (f *fakeModelInputFilter) FilterModelInput(ctx context.Context, req types.ModelRequest) (types.ModelRequest, types.SecurityFilterResult, error) {
	if f.filter == nil {
		return req, types.SecurityFilterResult{Decision: types.SecurityFilterDecisionAllow}, nil
	}
	return f.filter(ctx, req)
}

type fakeModelOutputFilter struct {
	filter func(ctx context.Context, output string) (string, types.SecurityFilterResult, error)
}

func (f *fakeModelOutputFilter) FilterModelOutput(ctx context.Context, output string) (string, types.SecurityFilterResult, error) {
	if f.filter == nil {
		return output, types.SecurityFilterResult{Decision: types.SecurityFilterDecisionAllow}, nil
	}
	return f.filter(ctx, output)
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

func (c *eventCollector) nonTimelineTypes() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]string, 0, len(c.types))
	for _, t := range c.types {
		if t == types.EventTypeActionTimeline {
			continue
		}
		out = append(out, t)
	}
	return out
}

func (c *eventCollector) timelineEvents() []types.Event {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]types.Event, 0, len(c.evs))
	for _, ev := range c.evs {
		if ev.Type == types.EventTypeActionTimeline {
			out = append(out, ev)
		}
	}
	return out
}

func (c *eventCollector) lastNonTimelineEvent() (types.Event, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := len(c.evs) - 1; i >= 0; i-- {
		if c.evs[i].Type == types.EventTypeActionTimeline {
			continue
		}
		return c.evs[i], true
	}
	return types.Event{}, false
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
	nonTimeline := collector.nonTimelineTypes()
	if len(nonTimeline) != 4 {
		t.Fatalf("event count = %d, want 4", len(nonTimeline))
	}
	want := []string{"run.started", "model.requested", "model.completed", "run.finished"}
	for i := range want {
		if nonTimeline[i] != want[i] {
			t.Fatalf("event[%d] = %q, want %q", i, nonTimeline[i], want[i])
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
	nonTimeline := collector.nonTimelineTypes()
	if nonTimeline[len(nonTimeline)-1] != "run.finished" {
		t.Fatalf("last event = %q, want run.finished", nonTimeline[len(nonTimeline)-1])
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
	nonTimeline := collector.nonTimelineTypes()
	if len(nonTimeline) < 5 {
		t.Fatalf("event count = %d, want at least 5", len(nonTimeline))
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
	nonTimeline := collector.nonTimelineTypes()
	if nonTimeline[len(nonTimeline)-1] != "run.finished" {
		t.Fatalf("last event = %q, want run.finished", nonTimeline[len(nonTimeline)-1])
	}
}

func TestStreamToolLoopSuccessAtStepBoundary(t *testing.T) {
	reg := local.NewRegistry()
	_, err := reg.Register(&fakeTool{
		name: "echo",
		invoke: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
			return types.ToolResult{Content: "hello"}, nil
		},
	})
	if err != nil {
		t.Fatalf("register tool: %v", err)
	}

	turn := 0
	model := &fakeModel{
		stream: func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
			turn++
			if turn == 1 {
				if len(req.ToolResult) != 0 {
					t.Fatalf("first step should not have tool results: %#v", req.ToolResult)
				}
				return onEvent(types.ModelEvent{
					Type:     types.ModelEventTypeToolCall,
					ToolCall: &types.ToolCall{CallID: "c1", Name: "local.echo"},
				})
			}
			if len(req.ToolResult) != 1 || req.ToolResult[0].Result.Content != "hello" {
				t.Fatalf("tool feedback not merged into next stream step: %#v", req.ToolResult)
			}
			return onEvent(types.ModelEvent{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "done"})
		},
	}
	engine := New(model, WithLocalRegistry(reg))
	collector := &eventCollector{}
	res, runErr := engine.Stream(context.Background(), types.RunRequest{Input: "x"}, collector)
	if runErr != nil {
		t.Fatalf("Stream failed: %v", runErr)
	}
	if res.FinalAnswer != "done" {
		t.Fatalf("FinalAnswer = %q, want done", res.FinalAnswer)
	}
	if len(res.ToolCalls) != 1 || res.ToolCalls[0].Name != "local.echo" {
		t.Fatalf("tool call summary mismatch: %#v", res.ToolCalls)
	}
	for _, ev := range collector.timelineEvents() {
		reason, _ := ev.Payload["reason"].(string)
		if reason == "stream_tool_dispatch_not_supported" {
			t.Fatalf("unexpected unsupported stream tool reason in timeline: %#v", ev)
		}
	}
}

func TestRunAndStreamToolCallLimitFailFast(t *testing.T) {
	reg := local.NewRegistry()
	_, err := reg.Register(&fakeTool{
		name: "echo",
		invoke: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
			return types.ToolResult{Content: "ok"}, nil
		},
	})
	if err != nil {
		t.Fatalf("register tool: %v", err)
	}

	runModelCalls := 0
	runModel := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			runModelCalls++
			return types.ModelResponse{
				ToolCalls: []types.ToolCall{
					{CallID: "c1", Name: "local.echo"},
					{CallID: "c2", Name: "local.echo"},
				},
			}, nil
		},
	}
	streamModelCalls := 0
	streamModel := &fakeModel{
		stream: func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
			streamModelCalls++
			if err := onEvent(types.ModelEvent{
				Type:     types.ModelEventTypeToolCall,
				ToolCall: &types.ToolCall{CallID: "c1", Name: "local.echo"},
			}); err != nil {
				return err
			}
			return onEvent(types.ModelEvent{
				Type:     types.ModelEventTypeToolCall,
				ToolCall: &types.ToolCall{CallID: "c2", Name: "local.echo"},
			})
		},
	}
	policy := types.DefaultLoopPolicy()
	policy.ToolCallLimit = 1
	policy.MaxIterations = 3

	runCollector := &eventCollector{}
	streamCollector := &eventCollector{}
	runEngine := New(runModel, WithLocalRegistry(reg))
	streamEngine := New(streamModel, WithLocalRegistry(reg))
	runRes, runErr := runEngine.Run(context.Background(), types.RunRequest{Input: "x", Policy: &policy}, runCollector)
	streamRes, streamErr := streamEngine.Stream(context.Background(), types.RunRequest{Input: "x", Policy: &policy}, streamCollector)
	if runErr == nil || streamErr == nil {
		t.Fatalf("expected tool-call-limit errors, got run=%v stream=%v", runErr, streamErr)
	}
	if runRes.Error == nil || streamRes.Error == nil {
		t.Fatalf("missing run/stream errors: run=%#v stream=%#v", runRes.Error, streamRes.Error)
	}
	if runRes.Error.Class != types.ErrIterationLimit || streamRes.Error.Class != types.ErrIterationLimit {
		t.Fatalf("run/stream error class mismatch: run=%#v stream=%#v", runRes.Error, streamRes.Error)
	}
	if runModelCalls != 1 || streamModelCalls != 1 {
		t.Fatalf("fail-fast should stop extra model steps, run_calls=%d stream_calls=%d", runModelCalls, streamModelCalls)
	}
	if len(runRes.ToolCalls) != 0 || len(streamRes.ToolCalls) != 0 {
		t.Fatalf("tool dispatch should not execute when budget is exceeded, run=%#v stream=%#v", runRes.ToolCalls, streamRes.ToolCalls)
	}

	runFinished, ok := runCollector.lastNonTimelineEvent()
	if !ok {
		t.Fatal("missing run run.finished")
	}
	streamFinished, ok := streamCollector.lastNonTimelineEvent()
	if !ok {
		t.Fatal("missing stream run.finished")
	}
	if runFinished.Payload["react_termination_reason"] != runtimeconfig.RuntimeReactTerminationToolCallLimitExceeded ||
		streamFinished.Payload["react_termination_reason"] != runtimeconfig.RuntimeReactTerminationToolCallLimitExceeded {
		t.Fatalf(
			"react termination mismatch run=%#v stream=%#v",
			runFinished.Payload["react_termination_reason"],
			streamFinished.Payload["react_termination_reason"],
		)
	}
	if runFinished.Payload["react_tool_call_budget_hit_total"] != 1 || streamFinished.Payload["react_tool_call_budget_hit_total"] != 1 {
		t.Fatalf(
			"react budget hit mismatch run=%#v stream=%#v",
			runFinished.Payload["react_tool_call_budget_hit_total"],
			streamFinished.Payload["react_tool_call_budget_hit_total"],
		)
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
	nonTimeline := c.nonTimelineTypes()
	if len(nonTimeline) != 4 {
		t.Fatalf("event count = %d, want 4", len(nonTimeline))
	}
	for _, ev := range c.evs {
		if ev.Type == types.EventTypeActionTimeline {
			continue
		}
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
		if nonTimeline[i] != order[i] {
			t.Fatalf("event order mismatch at %d: got %q want %q", i, nonTimeline[i], order[i])
		}
	}
}

func TestRunRecordsDiagnosticsWithRuntimeManager(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
mcp:
  active_profile: default
  profiles:
    default:
      call_timeout: 2s
      retry: 0
      backoff: 10ms
      queue_size: 16
      backpressure: block
      read_pool_size: 2
      write_pool_size: 1
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	model := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{FinalAnswer: "ok"}, nil
		},
	}
	r := New(model, WithRuntimeManager(mgr))
	rec := obsevent.NewRuntimeRecorder(mgr)
	_, err = r.Run(context.Background(), types.RunRequest{Input: "hi"}, rec)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	runs := mgr.RecentRuns(1)
	if len(runs) != 1 {
		t.Fatalf("run diagnostics len = %d, want 1", len(runs))
	}
	if runs[0].RunID == "" || runs[0].ErrorClass != "" || runs[0].Status != "success" {
		t.Fatalf("unexpected run diagnostics: %#v", runs[0])
	}
}

func TestWithSandboxExecutorBindsIntoRuntimeManagerRegardlessOfOptionOrder(t *testing.T) {
	model := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{FinalAnswer: "ok"}, nil
		},
	}
	t.Run("runtime-manager-then-sandbox", func(t *testing.T) {
		mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX_A51_RUNNER_TEST"})
		if err != nil {
			t.Fatalf("NewManager failed: %v", err)
		}
		t.Cleanup(func() { _ = mgr.Close() })
		executor := &fakeSandboxExecutor{}
		_ = New(model, WithRuntimeManager(mgr), WithSandboxExecutor(executor))
		if mgr.SandboxExecutor() != executor {
			t.Fatalf("runtime manager sandbox executor not wired, got=%T", mgr.SandboxExecutor())
		}
	})
	t.Run("sandbox-then-runtime-manager", func(t *testing.T) {
		mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX_A51_RUNNER_TEST"})
		if err != nil {
			t.Fatalf("NewManager failed: %v", err)
		}
		t.Cleanup(func() { _ = mgr.Close() })
		executor := &fakeSandboxExecutor{}
		_ = New(model, WithSandboxExecutor(executor), WithRuntimeManager(mgr))
		if mgr.SandboxExecutor() != executor {
			t.Fatalf("runtime manager sandbox executor not wired in reverse order, got=%T", mgr.SandboxExecutor())
		}
	})
}

func TestRunRecordsFailedDiagnosticsWithRuntimeRecorder(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
mcp:
  active_profile: default
  profiles:
    default:
      call_timeout: 2s
      retry: 0
      backoff: 10ms
      queue_size: 16
      backpressure: block
      read_pool_size: 2
      write_pool_size: 1
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	model := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{}, errors.New("model failed")
		},
	}
	r := New(model, WithRuntimeManager(mgr))
	rec := obsevent.NewRuntimeRecorder(mgr)
	_, err = r.Run(context.Background(), types.RunRequest{Input: "hi"}, rec)
	if err == nil {
		t.Fatal("expected model error")
	}

	runs := mgr.RecentRuns(1)
	if len(runs) != 1 {
		t.Fatalf("run diagnostics len = %d, want 1", len(runs))
	}
	if runs[0].Status != "failed" || runs[0].ErrorClass != string(types.ErrModel) {
		t.Fatalf("unexpected failed run diagnostics: %#v", runs[0])
	}
}

func TestRunSandboxDispatcherDenyMappedToSecurityReason(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime-sandbox-deny.yaml")
	cfg := `
security:
  sandbox:
    enabled: true
    mode: enforce
    required: true
    policy:
      default_action: deny
      profile: default
      fallback_action: deny
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX_A51_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	reg := local.NewRegistry()
	_, _ = reg.Register(&fakeTool{
		name: "search",
		schema: map[string]any{
			"type": "object",
		},
		invoke: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
			return types.ToolResult{Content: "host"}, nil
		},
	})
	model := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{
				ToolCalls: []types.ToolCall{
					{CallID: "call-sandbox-deny", Name: "local.search", Args: map[string]any{}},
				},
			}, nil
		},
	}
	collector := &eventCollector{}
	engine := New(model, WithLocalRegistry(reg), WithRuntimeManager(mgr))
	res, runErr := engine.Run(context.Background(), types.RunRequest{Input: "trigger sandbox deny"}, collector)
	if runErr == nil {
		t.Fatalf("expected sandbox deny run error, got nil result=%#v", res)
	}
	if res.Error == nil || res.Error.Class != types.ErrSecurity {
		t.Fatalf("run error class mismatch: %#v", res.Error)
	}
	if res.Error.Details["reason_code"] != "sandbox.policy_deny" {
		t.Fatalf("run reason_code=%#v, want sandbox.policy_deny", res.Error.Details["reason_code"])
	}

	var finished types.Event
	finishedFound := false
	for _, ev := range collector.evs {
		if ev.Type == "run.finished" {
			finished = ev
			finishedFound = true
		}
	}
	if !finishedFound {
		t.Fatal("missing run.finished event")
	}
	if finished.Payload["error_class"] != string(types.ErrSecurity) {
		t.Fatalf("run.finished error_class=%#v, want %q", finished.Payload["error_class"], types.ErrSecurity)
	}
	if finished.Payload["reason_code"] != "sandbox.policy_deny" {
		t.Fatalf("run.finished reason_code=%#v, want sandbox.policy_deny", finished.Payload["reason_code"])
	}
	if finished.Payload["policy_kind"] != "sandbox" {
		t.Fatalf("run.finished policy_kind=%#v, want sandbox", finished.Payload["policy_kind"])
	}
	if finished.Payload["decision"] != "deny" {
		t.Fatalf("run.finished decision=%#v, want deny", finished.Payload["decision"])
	}

	var sawSandboxTimelineReason bool
	for _, ev := range collector.timelineEvents() {
		if reason, ok := ev.Payload["reason"].(string); ok && reason == "sandbox.policy_deny" {
			sawSandboxTimelineReason = true
			break
		}
	}
	if !sawSandboxTimelineReason {
		t.Fatalf("expected timeline reason sandbox.policy_deny, got %#v", collector.timelineEvents())
	}
}

func TestRunSandboxFallbackAllowRecordsTimelineReason(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime-sandbox-fallback-allow.yaml")
	cfg := `
security:
  sandbox:
    enabled: true
    mode: enforce
    required: false
    policy:
      default_action: host
      by_tool:
        local+exec: sandbox
      profile: default
      fallback_action: allow_and_record
      fallback_action_by_tool:
        local+exec: allow_and_record
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX_A51_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	mgr.SetSandboxExecutor(&fakeSandboxExecutor{
		probe: func(ctx context.Context) (types.SandboxCapabilityProbe, error) {
			_ = ctx
			return types.SandboxCapabilityProbe{
				Backend:        runtimeconfig.SecuritySandboxBackendWindowsJob,
				Capabilities:   []string{runtimeconfig.SecuritySandboxCapabilityStdoutStderrCapture},
				SupportedModes: []string{runtimeconfig.SecuritySandboxSessionModePerCall},
			}, nil
		},
		execute: func(ctx context.Context, spec types.SandboxExecSpec) (types.SandboxExecResult, error) {
			_ = ctx
			_ = spec
			return types.SandboxExecResult{}, errors.New("sandbox launch failed")
		},
	})

	reg := local.NewRegistry()
	_, _ = reg.Register(&fakeSandboxAdapterRunnerTool{
		fakeTool: &fakeTool{
			name: "exec",
			invoke: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
				_ = ctx
				_ = args
				return types.ToolResult{Content: "host-fallback"}, nil
			},
		},
		build: func(ctx context.Context, args map[string]any) (types.SandboxExecSpec, error) {
			_ = ctx
			_ = args
			return types.SandboxExecSpec{Command: "cmd.exe", Args: []string{"/c", "echo test"}}, nil
		},
		handle: func(ctx context.Context, result types.SandboxExecResult) (types.ToolResult, error) {
			_ = ctx
			_ = result
			return types.ToolResult{Content: "sandbox"}, nil
		},
	})

	var modelCalls int32
	model := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			_ = ctx
			_ = req
			if atomic.AddInt32(&modelCalls, 1) == 1 {
				return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "call-1", Name: "local.exec"}}}, nil
			}
			return types.ModelResponse{FinalAnswer: "ok"}, nil
		},
	}
	collector := &eventCollector{}
	engine := New(model, WithLocalRegistry(reg), WithRuntimeManager(mgr))
	res, runErr := engine.Run(context.Background(), types.RunRequest{Input: "trigger sandbox fallback"}, collector)
	if runErr != nil {
		t.Fatalf("Run should succeed with fallback allow, err=%v result=%#v", runErr, res)
	}
	if res.FinalAnswer != "ok" {
		t.Fatalf("final answer=%q, want ok", res.FinalAnswer)
	}
	var finished types.Event
	var finishedFound bool
	for _, ev := range collector.evs {
		if ev.Type != "run.finished" {
			continue
		}
		finished = ev
		finishedFound = true
	}
	if !finishedFound {
		t.Fatal("missing run.finished event")
	}
	if finished.Payload["sandbox_decision"] != runtimeconfig.SecuritySandboxActionHost ||
		finished.Payload["sandbox_fallback_reason"] != "sandbox.fallback_allow_and_record" ||
		finished.Payload["sandbox_reason_code"] != "sandbox.fallback_allow_and_record" {
		t.Fatalf("sandbox fallback payload mismatch: %#v", finished.Payload)
	}
	if used, ok := finished.Payload["sandbox_fallback_used"].(bool); !ok || !used {
		t.Fatalf("sandbox_fallback_used=%#v, want true", finished.Payload["sandbox_fallback_used"])
	}
	if finished.Payload["sandbox_mode"] != runtimeconfig.SecuritySandboxModeEnforce {
		t.Fatalf("sandbox_mode=%#v, want %q", finished.Payload["sandbox_mode"], runtimeconfig.SecuritySandboxModeEnforce)
	}
	if finished.Payload["sandbox_backend"] != runtimeconfig.SecuritySandboxBackendWindowsJob {
		t.Fatalf("sandbox_backend=%#v, want %q", finished.Payload["sandbox_backend"], runtimeconfig.SecuritySandboxBackendWindowsJob)
	}

	var sawFallbackReason bool
	for _, ev := range collector.timelineEvents() {
		if reason, _ := ev.Payload["reason"].(string); reason == "sandbox.fallback_allow_and_record" {
			sawFallbackReason = true
			break
		}
	}
	if !sawFallbackReason {
		t.Fatalf("expected timeline reason sandbox.fallback_allow_and_record, got %#v", collector.timelineEvents())
	}
}

func TestRunSandboxTimeoutDenyEmitsCanonicalTimelineReason(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime-sandbox-timeout-deny.yaml")
	cfg := `
security:
  sandbox:
    enabled: true
    mode: enforce
    required: true
    policy:
      default_action: host
      by_tool:
        local+exec: sandbox
      profile: default
      fallback_action: deny
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX_A51_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	mgr.SetSandboxExecutor(&fakeSandboxExecutor{
		probe: func(ctx context.Context) (types.SandboxCapabilityProbe, error) {
			_ = ctx
			return types.SandboxCapabilityProbe{
				Backend:        runtimeconfig.SecuritySandboxBackendWindowsJob,
				Capabilities:   []string{runtimeconfig.SecuritySandboxCapabilityStdoutStderrCapture},
				SupportedModes: []string{runtimeconfig.SecuritySandboxSessionModePerCall},
			}, nil
		},
		execute: func(ctx context.Context, spec types.SandboxExecSpec) (types.SandboxExecResult, error) {
			_ = ctx
			_ = spec
			return types.SandboxExecResult{TimedOut: true}, nil
		},
	})

	reg := local.NewRegistry()
	_, _ = reg.Register(&fakeSandboxAdapterRunnerTool{
		fakeTool: &fakeTool{name: "exec"},
		build: func(ctx context.Context, args map[string]any) (types.SandboxExecSpec, error) {
			_ = ctx
			_ = args
			return types.SandboxExecSpec{Command: "cmd.exe", Args: []string{"/c", "timeout"}}, nil
		},
		handle: func(ctx context.Context, result types.SandboxExecResult) (types.ToolResult, error) {
			_ = ctx
			_ = result
			return types.ToolResult{Content: "unexpected"}, nil
		},
	})
	model := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			_ = ctx
			_ = req
			return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "call-timeout", Name: "local.exec"}}}, nil
		},
	}
	collector := &eventCollector{}
	engine := New(model, WithLocalRegistry(reg), WithRuntimeManager(mgr))
	res, runErr := engine.Run(context.Background(), types.RunRequest{Input: "timeout"}, collector)
	if runErr == nil {
		t.Fatalf("expected timeout deny run error, got result=%#v", res)
	}
	if res.Error == nil || res.Error.Class != types.ErrSecurity {
		t.Fatalf("run error class mismatch: %#v", res.Error)
	}
	if res.Error.Details["reason_code"] != types.SandboxViolationTimeout {
		t.Fatalf("run reason_code=%#v, want %q", res.Error.Details["reason_code"], types.SandboxViolationTimeout)
	}
	var finished types.Event
	var finishedFound bool
	for _, ev := range collector.evs {
		if ev.Type != "run.finished" {
			continue
		}
		finished = ev
		finishedFound = true
	}
	if !finishedFound {
		t.Fatal("missing run.finished event")
	}
	if finished.Payload["sandbox_reason_code"] != types.SandboxViolationTimeout ||
		finished.Payload["sandbox_decision"] != runtimeconfig.SecuritySandboxActionSandbox {
		t.Fatalf("sandbox timeout payload mismatch: %#v", finished.Payload)
	}
	if finished.Payload["sandbox_timeout_total"] != 1 {
		t.Fatalf("sandbox_timeout_total=%#v, want 1", finished.Payload["sandbox_timeout_total"])
	}
	if finished.Payload["sandbox_mode"] != runtimeconfig.SecuritySandboxModeEnforce ||
		finished.Payload["sandbox_backend"] != runtimeconfig.SecuritySandboxBackendWindowsJob {
		t.Fatalf("sandbox mode/backend mismatch: %#v", finished.Payload)
	}
	var sawTimeoutReason bool
	for _, ev := range collector.timelineEvents() {
		if reason, _ := ev.Payload["reason"].(string); reason == types.SandboxViolationTimeout {
			sawTimeoutReason = true
			break
		}
	}
	if !sawTimeoutReason {
		t.Fatalf("expected timeline reason %q, got %#v", types.SandboxViolationTimeout, collector.timelineEvents())
	}
}

func TestSecurityEventContractSandboxPolicyDenyRunAndStreamEquivalent(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime-sandbox-policy-deny-security-event.yaml")
	cfg := `
security:
  tool_governance:
    enabled: false
  sandbox:
    enabled: true
    mode: enforce
    required: true
    policy:
      default_action: deny
      profile: default
      fallback_action: deny
    executor:
      required_capabilities:
        - network_off
  security_event:
    enabled: true
    alert:
      trigger_policy: deny_only
      sink: callback
      callback:
        require_registered: true
    delivery:
      mode: sync
      timeout: 200ms
      retry:
        max_attempts: 1
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX_A51_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	reg := local.NewRegistry()
	_, _ = reg.Register(&fakeTool{name: "search"})

	runEvents := make([]types.SecurityEvent, 0, 1)
	streamEvents := make([]types.SecurityEvent, 0, 1)
	runEngine := New(
		&fakeModel{
			generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
				_ = ctx
				_ = req
				return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "c1", Name: "local.search"}}}, nil
			},
		},
		WithRuntimeManager(mgr),
		WithLocalRegistry(reg),
		WithSecurityAlertCallback(func(ctx context.Context, event types.SecurityEvent) error {
			_ = ctx
			runEvents = append(runEvents, event)
			return nil
		}),
	)
	streamEngine := New(
		&fakeModel{
			stream: func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
				_ = ctx
				_ = req
				return onEvent(types.ModelEvent{
					Type: types.ModelEventTypeToolCall,
					ToolCall: &types.ToolCall{
						CallID: "c1",
						Name:   "local.search",
					},
				})
			},
		},
		WithRuntimeManager(mgr),
		WithLocalRegistry(reg),
		WithSecurityAlertCallback(func(ctx context.Context, event types.SecurityEvent) error {
			_ = ctx
			streamEvents = append(streamEvents, event)
			return nil
		}),
	)

	runCollector := &eventCollector{}
	streamCollector := &eventCollector{}
	runRes, runErr := runEngine.Run(context.Background(), types.RunRequest{Input: "sandbox deny"}, runCollector)
	streamRes, streamErr := streamEngine.Stream(context.Background(), types.RunRequest{Input: "sandbox deny"}, streamCollector)
	if runErr == nil || runRes.Error == nil || runRes.Error.Class != types.ErrSecurity {
		t.Fatalf("expected run sandbox deny error, got err=%v result=%#v", runErr, runRes.Error)
	}
	if streamErr == nil || streamRes.Error == nil || streamRes.Error.Class != types.ErrSecurity {
		t.Fatalf("expected stream sandbox deny error, got err=%v result=%#v", streamErr, streamRes.Error)
	}
	if len(runEvents) != 1 || len(streamEvents) != 1 {
		t.Fatalf("sandbox callback count mismatch run=%d stream=%d", len(runEvents), len(streamEvents))
	}
	if runEvents[0].PolicyKind != "sandbox" ||
		runEvents[0].Decision != "deny" ||
		runEvents[0].ReasonCode != "sandbox.policy_deny" {
		t.Fatalf("run sandbox callback taxonomy mismatch: %#v", runEvents[0])
	}
	if streamEvents[0].PolicyKind != "sandbox" ||
		streamEvents[0].Decision != "deny" ||
		streamEvents[0].ReasonCode != "sandbox.policy_deny" {
		t.Fatalf("stream sandbox callback taxonomy mismatch: %#v", streamEvents[0])
	}

	runFinished, ok := runCollector.lastNonTimelineEvent()
	if !ok {
		t.Fatal("missing run run.finished")
	}
	streamFinished, ok := streamCollector.lastNonTimelineEvent()
	if !ok {
		t.Fatal("missing stream run.finished")
	}
	for _, key := range []string{
		"policy_kind",
		"decision",
		"reason_code",
		"alert_dispatch_status",
		"alert_delivery_mode",
		"alert_retry_count",
		"alert_circuit_state",
	} {
		if runFinished.Payload[key] != streamFinished.Payload[key] {
			t.Fatalf("sandbox run/stream payload mismatch key=%s run=%#v stream=%#v", key, runFinished.Payload[key], streamFinished.Payload[key])
		}
	}
}

func TestSecurityEventContractSandboxLaunchFailureDenyTriggersCallback(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime-sandbox-launch-failure-security-event.yaml")
	cfg := `
security:
  sandbox:
    enabled: true
    mode: enforce
    required: true
    policy:
      default_action: host
      by_tool:
        local+exec: sandbox
      profile: default
      fallback_action: deny
  security_event:
    enabled: true
    alert:
      trigger_policy: deny_only
      sink: callback
      callback:
        require_registered: true
    delivery:
      mode: sync
      timeout: 200ms
      retry:
        max_attempts: 1
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX_A51_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	mgr.SetSandboxExecutor(&fakeSandboxExecutor{
		probe: func(ctx context.Context) (types.SandboxCapabilityProbe, error) {
			_ = ctx
			return types.SandboxCapabilityProbe{
				Backend:        runtimeconfig.SecuritySandboxBackendWindowsJob,
				Capabilities:   []string{runtimeconfig.SecuritySandboxCapabilityStdoutStderrCapture},
				SupportedModes: []string{runtimeconfig.SecuritySandboxSessionModePerCall},
			}, nil
		},
		execute: func(ctx context.Context, spec types.SandboxExecSpec) (types.SandboxExecResult, error) {
			_ = ctx
			_ = spec
			return types.SandboxExecResult{}, errors.New("sandbox launch failed")
		},
	})
	reg := local.NewRegistry()
	_, _ = reg.Register(&fakeSandboxAdapterRunnerTool{
		fakeTool: &fakeTool{name: "exec"},
		build: func(ctx context.Context, args map[string]any) (types.SandboxExecSpec, error) {
			_ = ctx
			_ = args
			return types.SandboxExecSpec{Command: "cmd.exe", Args: []string{"/c", "echo a51"}}, nil
		},
		handle: func(ctx context.Context, result types.SandboxExecResult) (types.ToolResult, error) {
			_ = ctx
			_ = result
			return types.ToolResult{Content: "unexpected"}, nil
		},
	})
	events := make([]types.SecurityEvent, 0, 1)
	engine := New(
		&fakeModel{
			generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
				_ = ctx
				_ = req
				return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "c1", Name: "local.exec"}}}, nil
			},
		},
		WithRuntimeManager(mgr),
		WithLocalRegistry(reg),
		WithSecurityAlertCallback(func(ctx context.Context, event types.SecurityEvent) error {
			_ = ctx
			events = append(events, event)
			return nil
		}),
	)
	collector := &eventCollector{}
	res, runErr := engine.Run(context.Background(), types.RunRequest{Input: "sandbox launch fail"}, collector)
	if runErr == nil || res.Error == nil || res.Error.Class != types.ErrSecurity {
		t.Fatalf("expected security deny, got err=%v result=%#v", runErr, res.Error)
	}
	if res.Error.Details["reason_code"] != "sandbox.launch_failed" {
		t.Fatalf("run reason_code=%#v, want sandbox.launch_failed", res.Error.Details["reason_code"])
	}
	if res.Error.Details["alert_dispatch_status"] != "succeeded" {
		t.Fatalf("alert_dispatch_status=%#v, want succeeded", res.Error.Details["alert_dispatch_status"])
	}
	if len(events) != 1 {
		t.Fatalf("callback events len=%d, want 1", len(events))
	}
	if events[0].PolicyKind != "sandbox" || events[0].Decision != "deny" || events[0].ReasonCode != "sandbox.launch_failed" {
		t.Fatalf("sandbox callback taxonomy mismatch: %#v", events[0])
	}
	finished, ok := collector.lastNonTimelineEvent()
	if !ok {
		t.Fatal("missing run.finished")
	}
	if finished.Payload["alert_dispatch_status"] != "succeeded" ||
		finished.Payload["policy_kind"] != "sandbox" ||
		finished.Payload["reason_code"] != "sandbox.launch_failed" {
		t.Fatalf("run.finished sandbox alert payload mismatch: %#v", finished.Payload)
	}
}

func TestSecurityEventContractSandboxCapabilityMismatchDenyTriggersCallback(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime-sandbox-capability-mismatch-security-event.yaml")
	cfg := `
security:
  sandbox:
    enabled: true
    mode: enforce
    required: true
    policy:
      default_action: host
      by_tool:
        local+exec: sandbox
      profile: default
      fallback_action: deny
    executor:
      required_capabilities:
        - network_off
  security_event:
    enabled: true
    alert:
      trigger_policy: deny_only
      sink: callback
      callback:
        require_registered: true
    delivery:
      mode: sync
      timeout: 200ms
      retry:
        max_attempts: 1
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX_A51_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	mgr.SetSandboxExecutor(&fakeSandboxExecutor{
		probe: func(ctx context.Context) (types.SandboxCapabilityProbe, error) {
			_ = ctx
			return types.SandboxCapabilityProbe{
				Backend:        runtimeconfig.SecuritySandboxBackendWindowsJob,
				Capabilities:   []string{runtimeconfig.SecuritySandboxCapabilityStdoutStderrCapture},
				SupportedModes: []string{runtimeconfig.SecuritySandboxSessionModePerCall},
			}, nil
		},
		execute: func(ctx context.Context, spec types.SandboxExecSpec) (types.SandboxExecResult, error) {
			_ = ctx
			_ = spec
			return types.SandboxExecResult{ExitCode: 0}, nil
		},
	})
	reg := local.NewRegistry()
	_, _ = reg.Register(&fakeSandboxAdapterRunnerTool{
		fakeTool: &fakeTool{name: "exec"},
		build: func(ctx context.Context, args map[string]any) (types.SandboxExecSpec, error) {
			_ = ctx
			_ = args
			return types.SandboxExecSpec{Command: "cmd.exe", Args: []string{"/c", "echo a51"}}, nil
		},
		handle: func(ctx context.Context, result types.SandboxExecResult) (types.ToolResult, error) {
			_ = ctx
			_ = result
			return types.ToolResult{Content: "unexpected"}, nil
		},
	})
	events := make([]types.SecurityEvent, 0, 1)
	engine := New(
		&fakeModel{
			generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
				_ = ctx
				_ = req
				return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "c1", Name: "local.exec"}}}, nil
			},
		},
		WithRuntimeManager(mgr),
		WithLocalRegistry(reg),
		WithSecurityAlertCallback(func(ctx context.Context, event types.SecurityEvent) error {
			_ = ctx
			events = append(events, event)
			return nil
		}),
	)
	collector := &eventCollector{}
	res, runErr := engine.Run(context.Background(), types.RunRequest{Input: "sandbox capability mismatch"}, collector)
	if runErr == nil || res.Error == nil || res.Error.Class != types.ErrSecurity {
		t.Fatalf("expected security deny, got err=%v result=%#v", runErr, res.Error)
	}
	if res.Error.Details["reason_code"] != "sandbox.capability_mismatch" {
		t.Fatalf("run reason_code=%#v, want sandbox.capability_mismatch", res.Error.Details["reason_code"])
	}
	if len(events) != 1 {
		t.Fatalf("callback events len=%d, want 1", len(events))
	}
	if events[0].PolicyKind != "sandbox" || events[0].Decision != "deny" || events[0].ReasonCode != "sandbox.capability_mismatch" {
		t.Fatalf("sandbox callback taxonomy mismatch: %#v", events[0])
	}
	finished, ok := collector.lastNonTimelineEvent()
	if !ok {
		t.Fatal("missing run.finished")
	}
	if finished.Payload["alert_dispatch_status"] != "succeeded" ||
		finished.Payload["policy_kind"] != "sandbox" ||
		finished.Payload["reason_code"] != "sandbox.capability_mismatch" {
		t.Fatalf("run.finished sandbox capability payload mismatch: %#v", finished.Payload)
	}
}

func TestSecurityDeliveryContractSandboxDenyAsyncQueueAndDropSemantics(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime-sandbox-async-queue.yaml")
	cfg := `
security:
  tool_governance:
    enabled: false
  sandbox:
    enabled: true
    mode: enforce
    required: true
    policy:
      default_action: deny
      profile: default
      fallback_action: deny
  security_event:
    enabled: true
    alert:
      trigger_policy: deny_only
      sink: callback
      callback:
        require_registered: true
    delivery:
      mode: async
      queue:
        size: 1
        overflow_policy: drop_old
      timeout: 1s
      retry:
        max_attempts: 1
        backoff_initial: 1ms
        backoff_max: 1ms
      circuit_breaker:
        failure_threshold: 10
        open_window: 1s
        half_open_probes: 1
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX_A51_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	reg := local.NewRegistry()
	_, _ = reg.Register(&fakeTool{name: "search"})

	started := make(chan struct{}, 1)
	release := make(chan struct{})
	engine := New(
		&fakeModel{
			generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
				_ = ctx
				_ = req
				return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "c1", Name: "local.search"}}}, nil
			},
		},
		WithRuntimeManager(mgr),
		WithLocalRegistry(reg),
		WithSecurityAlertCallback(func(ctx context.Context, event types.SecurityEvent) error {
			_ = ctx
			_ = event
			select {
			case started <- struct{}{}:
				<-release
			default:
			}
			return nil
		}),
	)

	collector1 := &eventCollector{}
	res1, err1 := engine.Run(context.Background(), types.RunRequest{Input: "sandbox deny #1"}, collector1)
	if err1 == nil || res1.Error == nil || res1.Error.Class != types.ErrSecurity {
		t.Fatalf("expected first sandbox deny, got err=%v result=%#v", err1, res1.Error)
	}
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("first sandbox async callback did not start")
	}

	collector2 := &eventCollector{}
	res2, err2 := engine.Run(context.Background(), types.RunRequest{Input: "sandbox deny #2"}, collector2)
	if err2 == nil || res2.Error == nil || res2.Error.Class != types.ErrSecurity {
		t.Fatalf("expected second sandbox deny, got err=%v result=%#v", err2, res2.Error)
	}

	collector3 := &eventCollector{}
	res3, err3 := engine.Run(context.Background(), types.RunRequest{Input: "sandbox deny #3"}, collector3)
	close(release)
	if err3 == nil || res3.Error == nil || res3.Error.Class != types.ErrSecurity {
		t.Fatalf("expected third sandbox deny, got err=%v result=%#v", err3, res3.Error)
	}
	finished3, ok := collector3.lastNonTimelineEvent()
	if !ok {
		t.Fatal("missing third run.finished")
	}
	if finished3.Payload["alert_delivery_mode"] != runtimeconfig.SecurityEventDeliveryModeAsync ||
		finished3.Payload["alert_dispatch_status"] != "queued" ||
		finished3.Payload["alert_queue_dropped"] != true ||
		finished3.Payload["alert_queue_drop_count"] != 1 {
		t.Fatalf("sandbox async queue diagnostics mismatch: %#v", finished3.Payload)
	}
}

func TestSecurityDeliveryContractSandboxDenyCircuitOpenSemantics(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime-sandbox-circuit-open.yaml")
	cfg := `
security:
  tool_governance:
    enabled: false
  sandbox:
    enabled: true
    mode: enforce
    required: true
    policy:
      default_action: deny
      profile: default
      fallback_action: deny
  security_event:
    enabled: true
    alert:
      trigger_policy: deny_only
      sink: callback
      callback:
        require_registered: true
    delivery:
      mode: sync
      timeout: 40ms
      retry:
        max_attempts: 1
      circuit_breaker:
        failure_threshold: 1
        open_window: 1s
        half_open_probes: 1
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX_A51_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	reg := local.NewRegistry()
	_, _ = reg.Register(&fakeTool{name: "search"})

	engine := New(
		&fakeModel{
			generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
				_ = ctx
				_ = req
				return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "c1", Name: "local.search"}}}, nil
			},
		},
		WithRuntimeManager(mgr),
		WithLocalRegistry(reg),
		WithSecurityAlertCallback(func(ctx context.Context, event types.SecurityEvent) error {
			_ = ctx
			_ = event
			return errors.New("sandbox callback failed")
		}),
	)

	collector1 := &eventCollector{}
	res1, err1 := engine.Run(context.Background(), types.RunRequest{Input: "sandbox deny #1"}, collector1)
	if err1 == nil || res1.Error == nil || res1.Error.Class != types.ErrSecurity {
		t.Fatalf("expected first sandbox deny, got err=%v result=%#v", err1, res1.Error)
	}
	finished1, ok := collector1.lastNonTimelineEvent()
	if !ok {
		t.Fatal("missing first run.finished")
	}
	if finished1.Payload["alert_dispatch_status"] != "failed" ||
		finished1.Payload["alert_dispatch_failure_reason"] != "alert.retry_exhausted" {
		t.Fatalf("first sandbox circuit diagnostics mismatch: %#v", finished1.Payload)
	}

	collector2 := &eventCollector{}
	res2, err2 := engine.Run(context.Background(), types.RunRequest{Input: "sandbox deny #2"}, collector2)
	if err2 == nil || res2.Error == nil || res2.Error.Class != types.ErrSecurity {
		t.Fatalf("expected second sandbox deny, got err=%v result=%#v", err2, res2.Error)
	}
	finished2, ok := collector2.lastNonTimelineEvent()
	if !ok {
		t.Fatal("missing second run.finished")
	}
	if finished2.Payload["alert_dispatch_status"] != "failed" ||
		finished2.Payload["alert_dispatch_failure_reason"] != "alert.circuit_open" ||
		finished2.Payload["policy_kind"] != "sandbox" ||
		finished2.Payload["reason_code"] != "sandbox.policy_deny" {
		t.Fatalf("second sandbox circuit-open diagnostics mismatch: %#v", finished2.Payload)
	}
}

func TestSecurityPolicyContractPermissionAllowAndDeny(t *testing.T) {
	t.Run("allow", func(t *testing.T) {
		cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
		cfg := `
security:
  tool_governance:
    enabled: true
    mode: enforce
    permission:
      default: allow
      deny_action: deny
      by_tool:
        local+echo: allow
    rate_limit:
      enabled: false
`
		if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
			t.Fatalf("write config: %v", err)
		}
		mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
		if err != nil {
			t.Fatalf("NewManager failed: %v", err)
		}
		t.Cleanup(func() { _ = mgr.Close() })

		reg := local.NewRegistry()
		invoked := 0
		_, _ = reg.Register(&fakeTool{
			name: "echo",
			invoke: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
				invoked++
				return types.ToolResult{Content: "ok"}, nil
			},
		})
		turn := 0
		model := &fakeModel{
			generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
				turn++
				if turn == 1 {
					return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "c1", Name: "local.echo"}}}, nil
				}
				return types.ModelResponse{FinalAnswer: "done"}, nil
			},
		}
		engine := New(model, WithLocalRegistry(reg), WithRuntimeManager(mgr))
		res, runErr := engine.Run(context.Background(), types.RunRequest{Input: "allow"}, nil)
		if runErr != nil {
			t.Fatalf("run should allow tool call, got %v", runErr)
		}
		if res.FinalAnswer != "done" {
			t.Fatalf("final answer = %q, want done", res.FinalAnswer)
		}
		if invoked != 1 {
			t.Fatalf("tool invoke count = %d, want 1", invoked)
		}
	})

	t.Run("deny", func(t *testing.T) {
		cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
		cfg := `
security:
  tool_governance:
    enabled: true
    mode: enforce
    permission:
      default: allow
      deny_action: deny
      by_tool:
        local+echo: deny
    rate_limit:
      enabled: false
`
		if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
			t.Fatalf("write config: %v", err)
		}
		mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
		if err != nil {
			t.Fatalf("NewManager failed: %v", err)
		}
		t.Cleanup(func() { _ = mgr.Close() })

		reg := local.NewRegistry()
		invoked := 0
		_, _ = reg.Register(&fakeTool{
			name: "echo",
			invoke: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
				invoked++
				return types.ToolResult{Content: "ok"}, nil
			},
		})
		model := &fakeModel{
			generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
				return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "c1", Name: "local.echo"}}}, nil
			},
		}
		collector := &eventCollector{}
		engine := New(model, WithLocalRegistry(reg), WithRuntimeManager(mgr))
		res, runErr := engine.Run(context.Background(), types.RunRequest{Input: "deny"}, collector)
		if runErr == nil {
			t.Fatal("expected permission deny error")
		}
		if res.Error == nil || res.Error.Class != types.ErrSecurity {
			t.Fatalf("error class = %#v, want ErrSecurity", res.Error)
		}
		if invoked != 0 {
			t.Fatalf("tool invoke count = %d, want 0", invoked)
		}
		finished, ok := collector.lastNonTimelineEvent()
		if !ok {
			t.Fatal("missing run.finished")
		}
		if finished.Payload["policy_kind"] != "permission" || finished.Payload["namespace_tool"] != "local+echo" {
			t.Fatalf("unexpected policy diagnostics payload: %#v", finished.Payload)
		}
		if finished.Payload["decision"] != "deny" || finished.Payload["reason_code"] != "security.permission_denied" {
			t.Fatalf("unexpected decision diagnostics payload: %#v", finished.Payload)
		}
	})
}

func TestSecurityPolicyContractRateLimitDeny(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
security:
  tool_governance:
    enabled: true
    mode: enforce
    permission:
      default: allow
      deny_action: deny
    rate_limit:
      enabled: true
      scope: process
      window: 1m
      limit: 1
      by_tool_limit:
        local+echo: 1
      exceed_action: deny
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	reg := local.NewRegistry()
	invoked := 0
	_, _ = reg.Register(&fakeTool{
		name: "echo",
		invoke: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
			invoked++
			return types.ToolResult{Content: "ok"}, nil
		},
	})
	turn := 0
	model := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			turn++
			switch turn {
			case 1:
				return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "c1", Name: "local.echo"}}}, nil
			case 2:
				return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "c2", Name: "local.echo"}}}, nil
			default:
				return types.ModelResponse{FinalAnswer: "done"}, nil
			}
		},
	}
	collector := &eventCollector{}
	engine := New(model, WithLocalRegistry(reg), WithRuntimeManager(mgr))
	res, runErr := engine.Run(context.Background(), types.RunRequest{Input: "rate-limit"}, collector)
	if runErr == nil {
		t.Fatal("expected rate-limit deny error")
	}
	if res.Error == nil || res.Error.Class != types.ErrSecurity {
		t.Fatalf("error class = %#v, want ErrSecurity", res.Error)
	}
	if invoked != 1 {
		t.Fatalf("tool invoke count = %d, want 1", invoked)
	}
	finished, ok := collector.lastNonTimelineEvent()
	if !ok {
		t.Fatal("missing run.finished")
	}
	if finished.Payload["policy_kind"] != "rate_limit" || finished.Payload["namespace_tool"] != "local+echo" {
		t.Fatalf("unexpected rate-limit diagnostics payload: %#v", finished.Payload)
	}
	if finished.Payload["decision"] != "deny" || finished.Payload["reason_code"] != "security.rate_limit_exceeded" {
		t.Fatalf("unexpected rate-limit decision payload: %#v", finished.Payload)
	}
}

func TestSecurityPolicyContractRequireRegisteredInputFilter(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
security:
  model_io_filtering:
    enabled: true
    require_registered_filter: true
    input:
      enabled: true
      block_action: deny
    output:
      enabled: false
      block_action: deny
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	called := 0
	model := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			called++
			return types.ModelResponse{FinalAnswer: "ok"}, nil
		},
	}
	collector := &eventCollector{}
	engine := New(model, WithRuntimeManager(mgr))
	res, runErr := engine.Run(context.Background(), types.RunRequest{Input: "blocked"}, collector)
	if runErr == nil {
		t.Fatal("expected filter registration deny error")
	}
	if res.Error == nil || res.Error.Class != types.ErrSecurity {
		t.Fatalf("error class = %#v, want ErrSecurity", res.Error)
	}
	if called != 0 {
		t.Fatalf("model invoke count = %d, want 0", called)
	}
	finished, ok := collector.lastNonTimelineEvent()
	if !ok {
		t.Fatal("missing run.finished")
	}
	if finished.Payload["policy_kind"] != "io_filter" || finished.Payload["filter_stage"] != "input" {
		t.Fatalf("unexpected io-filter diagnostics payload: %#v", finished.Payload)
	}
	if finished.Payload["decision"] != "deny" || finished.Payload["reason_code"] != "security.io_filter_missing" {
		t.Fatalf("unexpected io-filter decision payload: %#v", finished.Payload)
	}
}

func TestSecurityPolicyContractInputFilterDenyRunAndStreamEquivalent(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
security:
  model_io_filtering:
    enabled: true
    require_registered_filter: false
    input:
      enabled: true
      block_action: deny
    output:
      enabled: false
      block_action: deny
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	runCalled := 0
	streamCalled := 0
	inputDeny := &fakeModelInputFilter{
		filter: func(ctx context.Context, req types.ModelRequest) (types.ModelRequest, types.SecurityFilterResult, error) {
			return req, types.SecurityFilterResult{
				Decision:   types.SecurityFilterDecisionDeny,
				ReasonCode: "contract.input.blocked",
			}, nil
		},
	}
	runModel := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			runCalled++
			return types.ModelResponse{FinalAnswer: "ok"}, nil
		},
	}
	streamModel := &fakeModel{
		stream: func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
			streamCalled++
			return nil
		},
	}
	runCollector := &eventCollector{}
	streamCollector := &eventCollector{}
	runEngine := New(runModel, WithRuntimeManager(mgr), WithModelInputFilters(inputDeny))
	streamEngine := New(streamModel, WithRuntimeManager(mgr), WithModelInputFilters(inputDeny))
	runRes, runErr := runEngine.Run(context.Background(), types.RunRequest{Input: "x"}, runCollector)
	streamRes, streamErr := streamEngine.Stream(context.Background(), types.RunRequest{Input: "x"}, streamCollector)
	if runErr == nil || streamErr == nil {
		t.Fatalf("expected input filter deny for run/stream, got run=%v stream=%v", runErr, streamErr)
	}
	if runRes.Error == nil || streamRes.Error == nil {
		t.Fatalf("missing run/stream errors: run=%#v stream=%#v", runRes.Error, streamRes.Error)
	}
	if runRes.Error.Class != types.ErrSecurity || streamRes.Error.Class != types.ErrSecurity {
		t.Fatalf("run/stream error class mismatch: run=%#v stream=%#v", runRes.Error, streamRes.Error)
	}
	if runCalled != 0 || streamCalled != 0 {
		t.Fatalf("model invocation should be blocked before provider call, run=%d stream=%d", runCalled, streamCalled)
	}

	runFinished, ok := runCollector.lastNonTimelineEvent()
	if !ok {
		t.Fatal("missing run run.finished")
	}
	streamFinished, ok := streamCollector.lastNonTimelineEvent()
	if !ok {
		t.Fatal("missing stream run.finished")
	}
	if runFinished.Payload["filter_stage"] != "input" || streamFinished.Payload["filter_stage"] != "input" {
		t.Fatalf("filter stage mismatch run=%#v stream=%#v", runFinished.Payload["filter_stage"], streamFinished.Payload["filter_stage"])
	}
	if runFinished.Payload["reason_code"] != "contract.input.blocked" || streamFinished.Payload["reason_code"] != "contract.input.blocked" {
		t.Fatalf("reason code mismatch run=%#v stream=%#v", runFinished.Payload["reason_code"], streamFinished.Payload["reason_code"])
	}
}

func TestSecurityPolicyContractRunAndStreamPermissionSemanticsEquivalent(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
security:
  tool_governance:
    enabled: true
    mode: enforce
    permission:
      default: allow
      deny_action: deny
      by_tool:
        local+echo: deny
    rate_limit:
      enabled: false
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	reg := local.NewRegistry()
	_, _ = reg.Register(&fakeTool{name: "echo"})
	runModel := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "c1", Name: "local.echo"}}}, nil
		},
	}
	streamModel := &fakeModel{
		stream: func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
			return onEvent(types.ModelEvent{
				Type: types.ModelEventTypeToolCall,
				ToolCall: &types.ToolCall{
					CallID: "c1",
					Name:   "local.echo",
				},
			})
		},
	}
	runCollector := &eventCollector{}
	streamCollector := &eventCollector{}
	runEngine := New(runModel, WithRuntimeManager(mgr), WithLocalRegistry(reg))
	streamEngine := New(streamModel, WithRuntimeManager(mgr))

	runRes, runErr := runEngine.Run(context.Background(), types.RunRequest{Input: "x"}, runCollector)
	streamRes, streamErr := streamEngine.Stream(context.Background(), types.RunRequest{Input: "x"}, streamCollector)
	if runErr == nil || streamErr == nil {
		t.Fatalf("expected deny errors for run/stream, got run=%v stream=%v", runErr, streamErr)
	}
	if runRes.Error == nil || streamRes.Error == nil {
		t.Fatalf("missing run/stream errors: run=%#v stream=%#v", runRes.Error, streamRes.Error)
	}
	if runRes.Error.Class != types.ErrSecurity || streamRes.Error.Class != types.ErrSecurity {
		t.Fatalf("run/stream error class mismatch: run=%#v stream=%#v", runRes.Error, streamRes.Error)
	}

	runFinished, ok := runCollector.lastNonTimelineEvent()
	if !ok {
		t.Fatal("missing run run.finished")
	}
	streamFinished, ok := streamCollector.lastNonTimelineEvent()
	if !ok {
		t.Fatal("missing stream run.finished")
	}
	if runFinished.Payload["policy_kind"] != "permission" || streamFinished.Payload["policy_kind"] != "permission" {
		t.Fatalf("policy kind mismatch run=%#v stream=%#v", runFinished.Payload["policy_kind"], streamFinished.Payload["policy_kind"])
	}
	if runFinished.Payload["reason_code"] != "security.permission_denied" || streamFinished.Payload["reason_code"] != "security.permission_denied" {
		t.Fatalf("reason code mismatch run=%#v stream=%#v", runFinished.Payload["reason_code"], streamFinished.Payload["reason_code"])
	}
}

func TestSecurityPolicyContractOutputFilterDenyRunAndStreamEquivalent(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
security:
  model_io_filtering:
    enabled: true
    require_registered_filter: false
    input:
      enabled: false
      block_action: deny
    output:
      enabled: true
      block_action: deny
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	denyFilter := &fakeModelOutputFilter{
		filter: func(ctx context.Context, output string) (string, types.SecurityFilterResult, error) {
			return output, types.SecurityFilterResult{
				Decision:   types.SecurityFilterDecisionDeny,
				ReasonCode: "contract.output.blocked",
			}, nil
		},
	}
	runModel := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{FinalAnswer: "secret"}, nil
		},
	}
	streamModel := &fakeModel{
		stream: func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
			return onEvent(types.ModelEvent{
				Type:      types.ModelEventTypeOutputTextDelta,
				TextDelta: "secret",
			})
		},
	}
	runCollector := &eventCollector{}
	streamCollector := &eventCollector{}
	runEngine := New(runModel, WithRuntimeManager(mgr), WithModelOutputFilters(denyFilter))
	streamEngine := New(streamModel, WithRuntimeManager(mgr), WithModelOutputFilters(denyFilter))
	runRes, runErr := runEngine.Run(context.Background(), types.RunRequest{Input: "x"}, runCollector)
	streamRes, streamErr := streamEngine.Stream(context.Background(), types.RunRequest{Input: "x"}, streamCollector)
	if runErr == nil || streamErr == nil {
		t.Fatalf("expected output filter deny for run/stream, got run=%v stream=%v", runErr, streamErr)
	}
	if runRes.Error == nil || streamRes.Error == nil {
		t.Fatalf("missing run/stream errors: run=%#v stream=%#v", runRes.Error, streamRes.Error)
	}
	if runRes.Error.Class != types.ErrSecurity || streamRes.Error.Class != types.ErrSecurity {
		t.Fatalf("run/stream error class mismatch: run=%#v stream=%#v", runRes.Error, streamRes.Error)
	}

	runFinished, ok := runCollector.lastNonTimelineEvent()
	if !ok {
		t.Fatal("missing run run.finished")
	}
	streamFinished, ok := streamCollector.lastNonTimelineEvent()
	if !ok {
		t.Fatal("missing stream run.finished")
	}
	if runFinished.Payload["policy_kind"] != "io_filter" || streamFinished.Payload["policy_kind"] != "io_filter" {
		t.Fatalf("policy kind mismatch run=%#v stream=%#v", runFinished.Payload["policy_kind"], streamFinished.Payload["policy_kind"])
	}
	if runFinished.Payload["filter_stage"] != "output" || streamFinished.Payload["filter_stage"] != "output" {
		t.Fatalf("filter stage mismatch run=%#v stream=%#v", runFinished.Payload["filter_stage"], streamFinished.Payload["filter_stage"])
	}
	if runFinished.Payload["decision"] != "deny" || streamFinished.Payload["decision"] != "deny" {
		t.Fatalf("decision mismatch run=%#v stream=%#v", runFinished.Payload["decision"], streamFinished.Payload["decision"])
	}
	if runFinished.Payload["reason_code"] != "contract.output.blocked" || streamFinished.Payload["reason_code"] != "contract.output.blocked" {
		t.Fatalf("reason code mismatch run=%#v stream=%#v", runFinished.Payload["reason_code"], streamFinished.Payload["reason_code"])
	}
}

func TestSecurityPolicyContractFilterMatchDiagnostics(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
security:
  model_io_filtering:
    enabled: true
    require_registered_filter: false
    input:
      enabled: false
      block_action: deny
    output:
      enabled: true
      block_action: deny
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	matchFilter := &fakeModelOutputFilter{
		filter: func(ctx context.Context, output string) (string, types.SecurityFilterResult, error) {
			return strings.ToUpper(output), types.SecurityFilterResult{
				Decision:   types.SecurityFilterDecisionMatch,
				ReasonCode: "contract.output.matched",
			}, nil
		},
	}
	model := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{FinalAnswer: "hello"}, nil
		},
	}
	collector := &eventCollector{}
	engine := New(model, WithRuntimeManager(mgr), WithModelOutputFilters(matchFilter))
	res, runErr := engine.Run(context.Background(), types.RunRequest{Input: "x"}, collector)
	if runErr != nil {
		t.Fatalf("run failed: %v", runErr)
	}
	if res.FinalAnswer != "HELLO" {
		t.Fatalf("final answer = %q, want HELLO", res.FinalAnswer)
	}
	finished, ok := collector.lastNonTimelineEvent()
	if !ok {
		t.Fatal("missing run.finished")
	}
	if finished.Payload["policy_kind"] != "io_filter" || finished.Payload["decision"] != "match" {
		t.Fatalf("expected io_filter match diagnostics, got %#v", finished.Payload)
	}
	if finished.Payload["reason_code"] != "contract.output.matched" {
		t.Fatalf("reason_code = %#v, want contract.output.matched", finished.Payload["reason_code"])
	}
}

func TestSecurityEventContractPermissionDenyTriggersCallback(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
security:
  tool_governance:
    enabled: true
    mode: enforce
    permission:
      default: allow
      deny_action: deny
      by_tool:
        local+echo: deny
    rate_limit:
      enabled: false
  security_event:
    enabled: true
    alert:
      trigger_policy: deny_only
      sink: callback
      callback:
        require_registered: true
    delivery:
      mode: sync
      timeout: 200ms
      retry:
        max_attempts: 1
    severity:
      default: high
      by_reason_code:
        security.permission_denied: medium
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	reg := local.NewRegistry()
	_, _ = reg.Register(&fakeTool{name: "echo"})

	events := make([]types.SecurityEvent, 0, 1)
	engine := New(
		&fakeModel{
			generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
				return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "c1", Name: "local.echo"}}}, nil
			},
		},
		WithRuntimeManager(mgr),
		WithLocalRegistry(reg),
		WithSecurityAlertCallback(func(ctx context.Context, event types.SecurityEvent) error {
			events = append(events, event)
			return nil
		}),
	)
	collector := &eventCollector{}
	res, runErr := engine.Run(context.Background(), types.RunRequest{Input: "x"}, collector)
	if runErr == nil || res.Error == nil || res.Error.Class != types.ErrSecurity {
		t.Fatalf("expected security deny, got err=%v result=%#v", runErr, res.Error)
	}
	if len(events) != 1 {
		t.Fatalf("callback events len = %d, want 1", len(events))
	}
	if events[0].PolicyKind != "permission" || events[0].Decision != "deny" || events[0].ReasonCode != "security.permission_denied" {
		t.Fatalf("callback taxonomy mismatch: %#v", events[0])
	}
	if events[0].Severity != "medium" {
		t.Fatalf("callback severity = %q, want medium", events[0].Severity)
	}
	finished, ok := collector.lastNonTimelineEvent()
	if !ok {
		t.Fatal("missing run.finished")
	}
	if finished.Payload["severity"] != "medium" {
		t.Fatalf("run.finished severity = %#v, want medium", finished.Payload["severity"])
	}
	if finished.Payload["alert_dispatch_status"] != "succeeded" {
		t.Fatalf("run.finished alert_dispatch_status = %#v, want succeeded", finished.Payload["alert_dispatch_status"])
	}
}

func TestSecurityEventContractCallbackFailureDoesNotChangeDenyOutcome(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
security:
  tool_governance:
    enabled: true
    mode: enforce
    permission:
      default: allow
      deny_action: deny
      by_tool:
        local+echo: deny
    rate_limit:
      enabled: false
  security_event:
    enabled: true
    alert:
      trigger_policy: deny_only
      sink: callback
      callback:
        require_registered: true
    delivery:
      mode: sync
      timeout: 100ms
      retry:
        max_attempts: 1
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	reg := local.NewRegistry()
	_, _ = reg.Register(&fakeTool{name: "echo"})

	calls := 0
	engine := New(
		&fakeModel{
			generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
				return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "c1", Name: "local.echo"}}}, nil
			},
		},
		WithRuntimeManager(mgr),
		WithLocalRegistry(reg),
		WithSecurityAlertCallback(func(ctx context.Context, event types.SecurityEvent) error {
			calls++
			return errors.New("boom")
		}),
	)
	collector := &eventCollector{}
	res, runErr := engine.Run(context.Background(), types.RunRequest{Input: "x"}, collector)
	if runErr == nil || res.Error == nil || res.Error.Class != types.ErrSecurity {
		t.Fatalf("expected security deny, got err=%v result=%#v", runErr, res.Error)
	}
	if calls != 1 {
		t.Fatalf("callback calls = %d, want 1", calls)
	}
	finished, ok := collector.lastNonTimelineEvent()
	if !ok {
		t.Fatal("missing run.finished")
	}
	if finished.Payload["alert_dispatch_status"] != "failed" {
		t.Fatalf("alert_dispatch_status = %#v, want failed", finished.Payload["alert_dispatch_status"])
	}
	if finished.Payload["alert_dispatch_failure_reason"] != "alert.retry_exhausted" {
		t.Fatalf("alert_dispatch_failure_reason = %#v, want alert.retry_exhausted", finished.Payload["alert_dispatch_failure_reason"])
	}
}

func TestSecurityEventContractMatchDoesNotTriggerCallback(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
security:
  model_io_filtering:
    enabled: true
    require_registered_filter: false
    input:
      enabled: false
      block_action: deny
    output:
      enabled: true
      block_action: deny
  security_event:
    enabled: true
    alert:
      trigger_policy: deny_only
      sink: callback
      callback:
        require_registered: true
    delivery:
      mode: sync
      timeout: 200ms
      retry:
        max_attempts: 1
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	calls := 0
	matchFilter := &fakeModelOutputFilter{
		filter: func(ctx context.Context, output string) (string, types.SecurityFilterResult, error) {
			return strings.ToUpper(output), types.SecurityFilterResult{
				Decision:   types.SecurityFilterDecisionMatch,
				ReasonCode: "security.io_filter_match",
			}, nil
		},
	}
	engine := New(
		&fakeModel{
			generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
				return types.ModelResponse{FinalAnswer: "ok"}, nil
			},
		},
		WithRuntimeManager(mgr),
		WithModelOutputFilters(matchFilter),
		WithSecurityAlertCallback(func(ctx context.Context, event types.SecurityEvent) error {
			calls++
			return nil
		}),
	)
	collector := &eventCollector{}
	res, runErr := engine.Run(context.Background(), types.RunRequest{Input: "x"}, collector)
	if runErr != nil || res.FinalAnswer != "OK" {
		t.Fatalf("unexpected run result, err=%v answer=%q", runErr, res.FinalAnswer)
	}
	if calls != 0 {
		t.Fatalf("callback calls = %d, want 0 for match", calls)
	}
	finished, ok := collector.lastNonTimelineEvent()
	if !ok {
		t.Fatal("missing run.finished")
	}
	if finished.Payload["alert_dispatch_status"] != "not_triggered" {
		t.Fatalf("alert_dispatch_status = %#v, want not_triggered", finished.Payload["alert_dispatch_status"])
	}
}

func TestSecurityEventContractRunAndStreamSeverityAndAlertEquivalent(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
security:
  tool_governance:
    enabled: true
    mode: enforce
    permission:
      default: allow
      deny_action: deny
      by_tool:
        local+echo: deny
    rate_limit:
      enabled: false
  security_event:
    enabled: true
    alert:
      trigger_policy: deny_only
      sink: callback
      callback:
        require_registered: true
    delivery:
      mode: sync
      timeout: 200ms
      retry:
        max_attempts: 1
    severity:
      default: high
      by_reason_code:
        security.permission_denied: medium
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	reg := local.NewRegistry()
	_, _ = reg.Register(&fakeTool{name: "echo"})

	runAlerts := 0
	streamAlerts := 0
	runEngine := New(
		&fakeModel{
			generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
				return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "c1", Name: "local.echo"}}}, nil
			},
		},
		WithRuntimeManager(mgr),
		WithLocalRegistry(reg),
		WithSecurityAlertCallback(func(ctx context.Context, event types.SecurityEvent) error {
			runAlerts++
			return nil
		}),
	)
	streamEngine := New(
		&fakeModel{
			stream: func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
				return onEvent(types.ModelEvent{
					Type: types.ModelEventTypeToolCall,
					ToolCall: &types.ToolCall{
						CallID: "c1",
						Name:   "local.echo",
					},
				})
			},
		},
		WithRuntimeManager(mgr),
		WithSecurityAlertCallback(func(ctx context.Context, event types.SecurityEvent) error {
			streamAlerts++
			return nil
		}),
	)
	runCollector := &eventCollector{}
	streamCollector := &eventCollector{}
	_, runErr := runEngine.Run(context.Background(), types.RunRequest{Input: "x"}, runCollector)
	_, streamErr := streamEngine.Stream(context.Background(), types.RunRequest{Input: "x"}, streamCollector)
	if runErr == nil || streamErr == nil {
		t.Fatalf("expected deny errors for run/stream, got run=%v stream=%v", runErr, streamErr)
	}
	if runAlerts != 1 || streamAlerts != 1 {
		t.Fatalf("callback alert count mismatch run=%d stream=%d", runAlerts, streamAlerts)
	}

	runFinished, ok := runCollector.lastNonTimelineEvent()
	if !ok {
		t.Fatal("missing run run.finished")
	}
	streamFinished, ok := streamCollector.lastNonTimelineEvent()
	if !ok {
		t.Fatal("missing stream run.finished")
	}
	for _, key := range []string{
		"policy_kind",
		"decision",
		"reason_code",
		"severity",
		"alert_dispatch_status",
		"alert_delivery_mode",
		"alert_retry_count",
		"alert_circuit_state",
	} {
		if runFinished.Payload[key] != streamFinished.Payload[key] {
			t.Fatalf("%s mismatch run=%#v stream=%#v", key, runFinished.Payload[key], streamFinished.Payload[key])
		}
	}
}

func TestSecurityDeliveryContractAsyncQueueOverflowDropsOldest(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
security:
  security_event:
    enabled: true
    alert:
      trigger_policy: deny_only
      sink: callback
      callback:
        require_registered: true
    delivery:
      mode: async
      queue:
        size: 1
        overflow_policy: drop_old
      timeout: 1s
      retry:
        max_attempts: 1
        backoff_initial: 1ms
        backoff_max: 1ms
      circuit_breaker:
        failure_threshold: 10
        open_window: 1s
        half_open_probes: 1
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	started := make(chan struct{}, 1)
	release := make(chan struct{})
	engine := New(
		&fakeModel{},
		WithRuntimeManager(mgr),
		WithSecurityAlertCallback(func(ctx context.Context, event types.SecurityEvent) error {
			select {
			case started <- struct{}{}:
				<-release
			default:
			}
			return nil
		}),
	)
	decision := securityDecision{
		PolicyKind: "permission",
		Decision:   string(types.SecurityFilterDecisionDeny),
		ReasonCode: "security.permission_denied",
		Severity:   "high",
	}

	first := engine.dispatchSecurityAlert(context.Background(), "run-1", 1, decision)
	if first.Status != securityAlertDispatchQueued {
		t.Fatalf("first status = %q, want queued", first.Status)
	}
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("first async callback did not start")
	}
	_ = engine.dispatchSecurityAlert(context.Background(), "run-1", 2, decision)
	third := engine.dispatchSecurityAlert(context.Background(), "run-1", 3, decision)
	if third.Status != securityAlertDispatchQueued {
		t.Fatalf("third status = %q, want queued", third.Status)
	}
	if !third.QueueDropped || third.QueueDropCount != 1 {
		t.Fatalf("queue drop diagnostics mismatch: %#v", third)
	}
	close(release)
}

func TestSecurityDeliveryContractRetryBudget(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
security:
  security_event:
    enabled: true
    alert:
      trigger_policy: deny_only
      sink: callback
      callback:
        require_registered: true
    delivery:
      mode: sync
      timeout: 40ms
      retry:
        max_attempts: 3
        backoff_initial: 1ms
        backoff_max: 2ms
      circuit_breaker:
        failure_threshold: 10
        open_window: 40ms
        half_open_probes: 1
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	failuresLeft := 2
	callbackCalls := 0
	engine := New(
		&fakeModel{},
		WithRuntimeManager(mgr),
		WithSecurityAlertCallback(func(ctx context.Context, event types.SecurityEvent) error {
			callbackCalls++
			if failuresLeft > 0 {
				failuresLeft--
				return errors.New("callback failed")
			}
			return nil
		}),
	)
	decision := securityDecision{
		PolicyKind: "permission",
		Decision:   string(types.SecurityFilterDecisionDeny),
		ReasonCode: "security.permission_denied",
		Severity:   "high",
	}

	outcome := engine.dispatchSecurityAlert(context.Background(), "run-sync", 1, decision)
	if outcome.Status != securityAlertDispatchSucceeded || outcome.RetryCount != 2 {
		t.Fatalf("retry outcome mismatch: %#v", outcome)
	}
	if callbackCalls != 3 {
		t.Fatalf("callback calls = %d, want 3", callbackCalls)
	}
}

func TestSecurityDeliveryContractCircuitTransitions(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
security:
  security_event:
    enabled: true
    alert:
      trigger_policy: deny_only
      sink: callback
      callback:
        require_registered: true
    delivery:
      mode: sync
      timeout: 40ms
      retry:
        max_attempts: 1
        backoff_initial: 1ms
        backoff_max: 1ms
      circuit_breaker:
        failure_threshold: 2
        open_window: 40ms
        half_open_probes: 1
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	fail := true
	callbackCalls := 0
	engine := New(
		&fakeModel{},
		WithRuntimeManager(mgr),
		WithSecurityAlertCallback(func(ctx context.Context, event types.SecurityEvent) error {
			callbackCalls++
			if fail {
				return errors.New("callback failed")
			}
			return nil
		}),
	)
	decision := securityDecision{
		PolicyKind: "permission",
		Decision:   string(types.SecurityFilterDecisionDeny),
		ReasonCode: "security.permission_denied",
		Severity:   "high",
	}

	_ = engine.dispatchSecurityAlert(context.Background(), "run-sync", 1, decision)
	opening := engine.dispatchSecurityAlert(context.Background(), "run-sync", 2, decision)
	if opening.CircuitState != runtimeconfig.SecurityEventCircuitStateOpen {
		t.Fatalf("circuit should open after threshold failures, got %#v", opening)
	}
	beforeFastFailCalls := callbackCalls
	fastFail := engine.dispatchSecurityAlert(context.Background(), "run-sync", 3, decision)
	if fastFail.FailureReason != securityAlertFailureCircuitOpen {
		t.Fatalf("fast-fail reason = %q, want %q", fastFail.FailureReason, securityAlertFailureCircuitOpen)
	}
	if callbackCalls != beforeFastFailCalls {
		t.Fatalf("circuit open fast-fail should skip callback, calls before=%d after=%d", beforeFastFailCalls, callbackCalls)
	}

	time.Sleep(60 * time.Millisecond)
	fail = false
	afterRecovery := engine.dispatchSecurityAlert(context.Background(), "run-sync", 4, decision)
	if afterRecovery.Status != securityAlertDispatchSucceeded {
		t.Fatalf("half-open recovery should succeed, got %#v", afterRecovery)
	}
	if afterRecovery.CircuitState != runtimeconfig.SecurityEventCircuitStateClosed {
		t.Fatalf("circuit state after recovery = %q, want closed", afterRecovery.CircuitState)
	}
}

func TestRunProviderFallbackByCapability(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
provider_fallback:
  enabled: true
  providers: [openai, anthropic]
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	primary := &fakeModel{
		provider: "openai",
		caps: map[types.ModelCapability]types.CapabilitySupport{
			types.ModelCapabilityToolCall:  types.CapabilitySupportUnsupported,
			types.ModelCapabilityStreaming: types.CapabilitySupportSupported,
		},
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			t.Fatalf("primary provider should not be invoked")
			return types.ModelResponse{}, nil
		},
	}
	secondary := &fakeModel{
		provider: "anthropic",
		caps: map[types.ModelCapability]types.CapabilitySupport{
			types.ModelCapabilityToolCall:  types.CapabilitySupportSupported,
			types.ModelCapabilityStreaming: types.CapabilitySupportSupported,
		},
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{FinalAnswer: "fallback-ok"}, nil
		},
	}

	collector := &eventCollector{}
	r := New(primary,
		WithProviderModels("openai", map[string]types.ModelClient{
			"openai":    primary,
			"anthropic": secondary,
		}),
		WithRuntimeManager(mgr),
	)
	res, err := r.Run(context.Background(), types.RunRequest{
		Input: "hello",
		Capabilities: types.CapabilityRequirements{
			Required: []types.ModelCapability{types.ModelCapabilityToolCall},
		},
	}, collector)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if res.FinalAnswer != "fallback-ok" {
		t.Fatalf("FinalAnswer = %q, want fallback-ok", res.FinalAnswer)
	}
	finished, ok := collector.lastNonTimelineEvent()
	if !ok {
		t.Fatal("missing non-timeline finished event")
	}
	if finished.Type != "run.finished" {
		t.Fatalf("last event = %q, want run.finished", finished.Type)
	}
	if finished.Payload["model_provider"] != "anthropic" {
		t.Fatalf("model_provider = %#v, want anthropic", finished.Payload["model_provider"])
	}
	if finished.Payload["fallback_used"] != true {
		t.Fatalf("fallback_used = %#v, want true", finished.Payload["fallback_used"])
	}
}

func TestRunProviderFallbackUsesSelectedTokenCounterForCA3(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
provider_fallback:
  enabled: true
  providers: [openai, anthropic]
context_assembler:
  enabled: true
  ca3:
    enabled: true
    max_context_tokens: 200
    absolute_thresholds:
      safe: 10
      comfort: 20
      warning: 30
      danger: 40
      emergency: 50
    tokenizer:
      mode: sdk_preferred
      small_delta_tokens: 0
      sdk_refresh_interval: 1ns
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	primaryCountCalls := 0
	secondaryCountCalls := 0
	primary := &fakeModel{
		provider: "openai",
		caps: map[types.ModelCapability]types.CapabilitySupport{
			types.ModelCapabilityToolCall:  types.CapabilitySupportUnsupported,
			types.ModelCapabilityStreaming: types.CapabilitySupportSupported,
		},
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			t.Fatalf("primary provider should not be invoked")
			return types.ModelResponse{}, nil
		},
		count: func(ctx context.Context, req types.ModelRequest) (int, error) {
			primaryCountCalls++
			return 10, nil
		},
	}
	secondary := &fakeModel{
		provider: "anthropic",
		caps: map[types.ModelCapability]types.CapabilitySupport{
			types.ModelCapabilityToolCall:  types.CapabilitySupportSupported,
			types.ModelCapabilityStreaming: types.CapabilitySupportSupported,
		},
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{FinalAnswer: "ok"}, nil
		},
		count: func(ctx context.Context, req types.ModelRequest) (int, error) {
			secondaryCountCalls++
			return 18, nil
		},
	}

	eng := New(primary,
		WithProviderModels("openai", map[string]types.ModelClient{
			"openai":    primary,
			"anthropic": secondary,
		}),
		WithRuntimeManager(mgr),
	)
	res, runErr := eng.Run(context.Background(), types.RunRequest{
		Input: "need tool-call capability so fallback selects anthropic",
		Capabilities: types.CapabilityRequirements{
			Required: []types.ModelCapability{types.ModelCapabilityToolCall},
		},
	}, nil)
	if runErr != nil {
		t.Fatalf("Run failed: %v", runErr)
	}
	if res.FinalAnswer != "ok" {
		t.Fatalf("final answer = %q, want ok", res.FinalAnswer)
	}
	if primaryCountCalls != 0 {
		t.Fatalf("primary count tokens should not be used, got %d calls", primaryCountCalls)
	}
	if secondaryCountCalls == 0 {
		t.Fatal("selected provider token counter should be used at least once")
	}
}

func TestRunProviderFallbackFailFastWhenExhausted(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
provider_fallback:
  enabled: true
  providers: [openai, anthropic]
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	unsupported := map[types.ModelCapability]types.CapabilitySupport{
		types.ModelCapabilityToolCall:  types.CapabilitySupportUnsupported,
		types.ModelCapabilityStreaming: types.CapabilitySupportSupported,
	}
	primary := &fakeModel{provider: "openai", caps: unsupported}
	secondary := &fakeModel{provider: "anthropic", caps: unsupported}
	r := New(primary,
		WithProviderModels("openai", map[string]types.ModelClient{
			"openai":    primary,
			"anthropic": secondary,
		}),
		WithRuntimeManager(mgr),
	)
	res, err := r.Run(context.Background(), types.RunRequest{
		Input: "hello",
		Capabilities: types.CapabilityRequirements{
			Required: []types.ModelCapability{types.ModelCapabilityToolCall},
		},
	}, nil)
	if err == nil {
		t.Fatal("expected fail-fast error")
	}
	if res.Error == nil || res.Error.Class != types.ErrModel {
		t.Fatalf("error class = %#v, want ErrModel", res.Error)
	}
	if res.Error.Details["provider_reason"] != "capability_unsupported" {
		t.Fatalf("provider_reason = %#v, want capability_unsupported", res.Error.Details["provider_reason"])
	}
}

func TestStreamProviderFallbackKeepsStreamSemantics(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
provider_fallback:
  enabled: true
  providers: [openai, anthropic]
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	primary := &fakeModel{
		provider: "openai",
		caps: map[types.ModelCapability]types.CapabilitySupport{
			types.ModelCapabilityStreaming: types.CapabilitySupportUnsupported,
		},
	}
	secondary := &fakeModel{
		provider: "anthropic",
		caps: map[types.ModelCapability]types.CapabilitySupport{
			types.ModelCapabilityStreaming: types.CapabilitySupportSupported,
		},
		stream: func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
			if err := onEvent(types.ModelEvent{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "he"}); err != nil {
				return err
			}
			return onEvent(types.ModelEvent{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "llo"})
		},
	}

	collector := &eventCollector{}
	r := New(primary,
		WithProviderModels("openai", map[string]types.ModelClient{
			"openai":    primary,
			"anthropic": secondary,
		}),
		WithRuntimeManager(mgr),
	)
	res, err := r.Stream(context.Background(), types.RunRequest{Input: "hello"}, collector)
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}
	if res.FinalAnswer != "hello" {
		t.Fatalf("FinalAnswer = %q, want hello", res.FinalAnswer)
	}
	finished, ok := collector.lastNonTimelineEvent()
	if !ok {
		t.Fatal("missing non-timeline finished event")
	}
	if finished.Payload["model_provider"] != "anthropic" {
		t.Fatalf("model_provider = %#v, want anthropic", finished.Payload["model_provider"])
	}
}

func TestStreamProviderFallbackUsesSelectedTokenCounterForCA3(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
provider_fallback:
  enabled: true
  providers: [openai, anthropic]
context_assembler:
  enabled: true
  ca3:
    enabled: true
    max_context_tokens: 200
    absolute_thresholds:
      safe: 10
      comfort: 20
      warning: 30
      danger: 40
      emergency: 50
    tokenizer:
      mode: sdk_preferred
      small_delta_tokens: 0
      sdk_refresh_interval: 1ns
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	primaryCountCalls := 0
	secondaryCountCalls := 0
	primary := &fakeModel{
		provider: "openai",
		caps: map[types.ModelCapability]types.CapabilitySupport{
			types.ModelCapabilityStreaming: types.CapabilitySupportUnsupported,
		},
		count: func(ctx context.Context, req types.ModelRequest) (int, error) {
			primaryCountCalls++
			return 11, nil
		},
	}
	secondary := &fakeModel{
		provider: "anthropic",
		caps: map[types.ModelCapability]types.CapabilitySupport{
			types.ModelCapabilityStreaming: types.CapabilitySupportSupported,
		},
		stream: func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
			return onEvent(types.ModelEvent{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "ok"})
		},
		count: func(ctx context.Context, req types.ModelRequest) (int, error) {
			secondaryCountCalls++
			return 22, nil
		},
	}

	eng := New(primary,
		WithProviderModels("openai", map[string]types.ModelClient{
			"openai":    primary,
			"anthropic": secondary,
		}),
		WithRuntimeManager(mgr),
	)
	res, runErr := eng.Stream(context.Background(), types.RunRequest{Input: "stream path with fallback"}, nil)
	if runErr != nil {
		t.Fatalf("Stream failed: %v", runErr)
	}
	if res.FinalAnswer != "ok" {
		t.Fatalf("final answer = %q, want ok", res.FinalAnswer)
	}
	if primaryCountCalls != 0 {
		t.Fatalf("primary count tokens should not be used, got %d calls", primaryCountCalls)
	}
	if secondaryCountCalls == 0 {
		t.Fatal("selected provider token counter should be used at least once")
	}
}

func TestRunFailsFastWhenContextAssemblerDBBackendConfigured(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
context_assembler:
  enabled: true
  storage:
    backend: db
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	called := false
	model := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			called = true
			return types.ModelResponse{FinalAnswer: "should-not-happen"}, nil
		},
	}
	eng := New(model, WithRuntimeManager(mgr))
	res, runErr := eng.Run(context.Background(), types.RunRequest{Input: "x", SessionID: "s-1"}, nil)
	if runErr == nil {
		t.Fatal("expected context assembler backend error")
	}
	if res.Error == nil || res.Error.Class != types.ErrContext {
		t.Fatalf("error class = %#v, want ErrContext", res.Error)
	}
	if called {
		t.Fatal("model should not be called when context assembler fails")
	}
}

func TestRunCA2BestEffortKeepsModelPath(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
context_assembler:
  enabled: true
  ca2:
    enabled: true
    routing_mode: rules
    stage_policy:
      stage1: fail_fast
      stage2: best_effort
    stage2:
      provider: rag
      external:
        endpoint: http://127.0.0.1:1/retrieve
    routing:
      min_input_chars: 1
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	model := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{FinalAnswer: "ok"}, nil
		},
	}
	rec := obsevent.NewRuntimeRecorder(mgr)
	eng := New(model, WithRuntimeManager(mgr))
	res, runErr := eng.Run(context.Background(), types.RunRequest{Input: "need retrieval", SessionID: "s-1"}, rec)
	if runErr != nil {
		t.Fatalf("Run failed: %v", runErr)
	}
	if res.FinalAnswer != "ok" {
		t.Fatalf("final answer = %q, want ok", res.FinalAnswer)
	}
	runs := mgr.RecentRuns(1)
	if len(runs) != 1 {
		t.Fatalf("run diagnostics len = %d, want 1", len(runs))
	}
	if runs[0].AssembleStageStatus == "" {
		t.Fatalf("assemble stage status should be populated: %#v", runs[0])
	}
}

func TestStreamCA2FailFastStopsBeforeModel(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
context_assembler:
  enabled: true
  ca2:
    enabled: true
    routing_mode: rules
    stage_policy:
      stage1: fail_fast
      stage2: fail_fast
    stage2:
      provider: db
      external:
        endpoint: http://127.0.0.1:1/retrieve
    routing:
      min_input_chars: 1
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	called := false
	model := &fakeModel{
		stream: func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
			called = true
			return onEvent(types.ModelEvent{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "x"})
		},
	}
	collector := &eventCollector{}
	eng := New(model, WithRuntimeManager(mgr))
	res, runErr := eng.Stream(context.Background(), types.RunRequest{Input: "need retrieval", SessionID: "s-1"}, collector)
	if runErr == nil {
		t.Fatal("expected fail-fast error from CA2 stage2 provider")
	}
	if res.Error == nil || res.Error.Class != types.ErrContext {
		t.Fatalf("error class = %#v, want ErrContext", res.Error)
	}
	if called {
		t.Fatal("model stream should not be called when CA2 fail-fast triggers")
	}
	want := []string{"run.started", "run.finished"}
	nonTimeline := collector.nonTimelineTypes()
	if len(nonTimeline) != len(want) {
		t.Fatalf("event count = %d, want %d", len(nonTimeline), len(want))
	}
	for i := range want {
		if nonTimeline[i] != want[i] {
			t.Fatalf("event[%d]=%s, want %s", i, nonTimeline[i], want[i])
		}
	}
}

func TestRunAndStreamTimelineSemanticEquivalence(t *testing.T) {
	runModel := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{FinalAnswer: "ok"}, nil
		},
	}
	streamModel := &fakeModel{
		stream: func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
			return onEvent(types.ModelEvent{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "ok"})
		},
	}
	runCollector := &eventCollector{}
	streamCollector := &eventCollector{}

	runEngine := New(runModel)
	streamEngine := New(streamModel)
	runEngine.newRunID = func() string { return "run-eq" }
	streamEngine.newRunID = func() string { return "stream-eq" }

	if _, err := runEngine.Run(context.Background(), types.RunRequest{Input: "hello"}, runCollector); err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if _, err := streamEngine.Stream(context.Background(), types.RunRequest{Input: "hello"}, streamCollector); err != nil {
		t.Fatalf("Stream failed: %v", err)
	}

	runSeq := timelinePhaseStatus(runCollector.timelineEvents())
	streamSeq := timelinePhaseStatus(streamCollector.timelineEvents())
	if len(runSeq) == 0 || len(streamSeq) == 0 {
		t.Fatalf("timeline events should be emitted by default: run=%d stream=%d", len(runSeq), len(streamSeq))
	}
	if !containsTimelineStep(runSeq, "run:succeeded") || !containsTimelineStep(streamSeq, "run:succeeded") {
		t.Fatalf("run and stream timeline should both finish as succeeded: run=%v stream=%v", runSeq, streamSeq)
	}
	if !containsTimelineStep(runSeq, "model:succeeded") || !containsTimelineStep(streamSeq, "model:succeeded") {
		t.Fatalf("run and stream timeline should both contain model:succeeded: run=%v stream=%v", runSeq, streamSeq)
	}
}

func TestTimelineSequenceIsMonotonic(t *testing.T) {
	model := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{FinalAnswer: "ok"}, nil
		},
	}
	collector := &eventCollector{}
	engine := New(model)
	if _, err := engine.Run(context.Background(), types.RunRequest{Input: "hello"}, collector); err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	events := collector.timelineEvents()
	if len(events) == 0 {
		t.Fatal("timeline events should not be empty")
	}
	prev := int64(0)
	for _, ev := range events {
		seq, _ := ev.Payload["sequence"].(int64)
		if seq <= prev {
			t.Fatalf("timeline sequence should be monotonic increasing: prev=%d current=%d", prev, seq)
		}
		prev = seq
	}
}

func TestTimelineContextAssemblerPhaseOnlyWhenEnabled(t *testing.T) {
	model := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{FinalAnswer: "ok"}, nil
		},
	}
	collectorDisabled := &eventCollector{}
	engineDisabled := New(model)
	if _, err := engineDisabled.Run(context.Background(), types.RunRequest{Input: "hello"}, collectorDisabled); err != nil {
		t.Fatalf("Run failed with default config: %v", err)
	}
	for _, step := range timelinePhaseStatus(collectorDisabled.timelineEvents()) {
		if strings.HasPrefix(step, "context_assembler:") {
			t.Fatalf("context_assembler phase should not be emitted when assembler disabled: %v", step)
		}
	}

	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
context_assembler:
  enabled: true
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	collectorEnabled := &eventCollector{}
	engineEnabled := New(model, WithRuntimeManager(mgr))
	if _, err := engineEnabled.Run(context.Background(), types.RunRequest{Input: "hello", SessionID: "s-1"}, collectorEnabled); err != nil {
		t.Fatalf("Run failed with assembler enabled: %v", err)
	}
	steps := timelinePhaseStatus(collectorEnabled.timelineEvents())
	if !containsTimelineStep(steps, "context_assembler:succeeded") {
		t.Fatalf("context_assembler phase should be emitted when enabled: %v", steps)
	}
}

func timelinePhaseStatus(events []types.Event) []string {
	out := make([]string, 0, len(events))
	for _, ev := range events {
		phase, _ := ev.Payload["phase"].(string)
		status, _ := ev.Payload["status"].(string)
		if strings.TrimSpace(phase) == "" || strings.TrimSpace(status) == "" {
			continue
		}
		out = append(out, phase+":"+status)
	}
	return out
}

func containsTimelineStep(steps []string, want string) bool {
	for _, step := range steps {
		if step == want {
			return true
		}
	}
	return false
}

func TestRunAndStreamTimelineAggregatesEquivalent(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
mcp:
  active_profile: default
  profiles:
    default:
      call_timeout: 2s
      retry: 0
      backoff: 10ms
      queue_size: 16
      backpressure: block
      read_pool_size: 2
      write_pool_size: 1
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	runModel := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{FinalAnswer: "ok"}, nil
		},
	}
	streamModel := &fakeModel{
		stream: func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
			return onEvent(types.ModelEvent{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "ok"})
		},
	}
	rec := obsevent.NewRuntimeRecorder(mgr)

	runEngine := New(runModel, WithRuntimeManager(mgr))
	streamEngine := New(streamModel, WithRuntimeManager(mgr))
	if _, err := runEngine.Run(context.Background(), types.RunRequest{RunID: "run-agg", Input: "hello"}, rec); err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if _, err := streamEngine.Stream(context.Background(), types.RunRequest{RunID: "stream-agg", Input: "hello"}, rec); err != nil {
		t.Fatalf("Stream failed: %v", err)
	}

	runs := mgr.RecentRuns(10)
	var runRec, streamRec *runtimediag.RunRecord
	for i := range runs {
		switch runs[i].RunID {
		case "run-agg":
			rec := runs[i]
			runRec = &rec
		case "stream-agg":
			rec := runs[i]
			streamRec = &rec
		}
	}
	if runRec == nil || streamRec == nil {
		t.Fatalf("missing diagnostics records: %#v", runs)
	}
	assertPhaseDistEqual(t, runRec.TimelinePhases, streamRec.TimelinePhases, "run")
	assertPhaseDistEqual(t, runRec.TimelinePhases, streamRec.TimelinePhases, "model")

	trends := mgr.TimelineTrends(runtimediag.TimelineTrendQuery{
		Mode:      runtimediag.TimelineTrendModeLastNRuns,
		LastNRuns: 2,
	})
	if len(trends) == 0 {
		t.Fatal("timeline trends should not be empty")
	}
	byKey := map[string]runtimediag.TimelineTrendRecord{}
	for _, item := range trends {
		byKey[item.Phase+"|"+item.Status] = item
	}
	for _, key := range []string{"run|succeeded", "model|succeeded"} {
		item, ok := byKey[key]
		if !ok {
			t.Fatalf("missing trend bucket %q in %#v", key, trends)
		}
		if item.CountTotal != 2 {
			t.Fatalf("trend %q count_total = %d, want 2", key, item.CountTotal)
		}
		if item.LatencyP95Ms < 0 {
			t.Fatalf("trend %q latency_p95_ms = %d, want >= 0", key, item.LatencyP95Ms)
		}
	}
}

func TestRunAndStreamCA3PressureSemanticsEquivalent(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
context_assembler:
  enabled: true
  ca3:
    enabled: true
    max_context_tokens: 60
    percent_thresholds:
      safe: 10
      comfort: 20
      warning: 30
      danger: 40
      emergency: 50
    absolute_thresholds:
      safe: 6
      comfort: 12
      warning: 18
      danger: 24
      emergency: 30
    compaction:
      mode: semantic
      quality:
        threshold: 0.4
      evidence:
        keywords: [mustkeep]
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	runModel := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			if strings.Contains(req.Input, "Source:") {
				return types.ModelResponse{FinalAnswer: "mustkeep semantic summary"}, nil
			}
			return types.ModelResponse{FinalAnswer: "ok"}, nil
		},
	}
	streamModel := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			if strings.Contains(req.Input, "Source:") {
				return types.ModelResponse{FinalAnswer: "mustkeep semantic summary"}, nil
			}
			return types.ModelResponse{FinalAnswer: "ok"}, nil
		},
		stream: func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
			return onEvent(types.ModelEvent{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "ok"})
		},
	}
	runCollector := &eventCollector{}
	streamCollector := &eventCollector{}

	runEngine := New(runModel, WithRuntimeManager(mgr))
	streamEngine := New(streamModel, WithRuntimeManager(mgr))
	_, err = runEngine.Run(context.Background(), types.RunRequest{
		RunID:     "run-ca3",
		SessionID: "s-1",
		Input:     strings.Repeat("payload ", 50),
		Messages:  []types.Message{{Role: "system", Content: "base"}},
	}, runCollector)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	_, err = streamEngine.Stream(context.Background(), types.RunRequest{
		RunID:     "stream-ca3",
		SessionID: "s-1",
		Input:     strings.Repeat("payload ", 50),
		Messages:  []types.Message{{Role: "system", Content: "base"}},
	}, streamCollector)
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}

	runFinished, ok := runCollector.lastNonTimelineEvent()
	if !ok {
		t.Fatal("missing run finished event")
	}
	streamFinished, ok := streamCollector.lastNonTimelineEvent()
	if !ok {
		t.Fatal("missing stream finished event")
	}
	if runFinished.Payload["ca3_pressure_zone"] != streamFinished.Payload["ca3_pressure_zone"] {
		t.Fatalf("run/stream ca3 pressure zone mismatch: run=%v stream=%v", runFinished.Payload["ca3_pressure_zone"], streamFinished.Payload["ca3_pressure_zone"])
	}
	if runFinished.Payload["ca3_pressure_reason"] != streamFinished.Payload["ca3_pressure_reason"] {
		t.Fatalf("run/stream ca3 pressure reason mismatch: run=%v stream=%v", runFinished.Payload["ca3_pressure_reason"], streamFinished.Payload["ca3_pressure_reason"])
	}
	if runFinished.Payload["ca3_pressure_trigger"] != streamFinished.Payload["ca3_pressure_trigger"] {
		t.Fatalf("run/stream ca3 pressure trigger mismatch: run=%v stream=%v", runFinished.Payload["ca3_pressure_trigger"], streamFinished.Payload["ca3_pressure_trigger"])
	}
	if runFinished.Payload["ca3_compaction_mode"] != streamFinished.Payload["ca3_compaction_mode"] {
		t.Fatalf("run/stream ca3 compaction mode mismatch: run=%v stream=%v", runFinished.Payload["ca3_compaction_mode"], streamFinished.Payload["ca3_compaction_mode"])
	}
	if runFinished.Payload["ca3_compaction_fallback"] != streamFinished.Payload["ca3_compaction_fallback"] {
		t.Fatalf("run/stream ca3 compaction fallback mismatch: run=%v stream=%v", runFinished.Payload["ca3_compaction_fallback"], streamFinished.Payload["ca3_compaction_fallback"])
	}
	if runFinished.Payload["ca3_compaction_quality_reason"] != streamFinished.Payload["ca3_compaction_quality_reason"] {
		t.Fatalf("run/stream ca3 compaction quality reason mismatch: run=%v stream=%v", runFinished.Payload["ca3_compaction_quality_reason"], streamFinished.Payload["ca3_compaction_quality_reason"])
	}
	if runFinished.Payload["ca3_compaction_fallback_reason"] != streamFinished.Payload["ca3_compaction_fallback_reason"] {
		t.Fatalf("run/stream ca3 compaction fallback reason mismatch: run=%v stream=%v", runFinished.Payload["ca3_compaction_fallback_reason"], streamFinished.Payload["ca3_compaction_fallback_reason"])
	}
	if runFinished.Payload["ca3_compaction_embedding_status"] != streamFinished.Payload["ca3_compaction_embedding_status"] {
		t.Fatalf("run/stream ca3 compaction embedding status mismatch: run=%v stream=%v", runFinished.Payload["ca3_compaction_embedding_status"], streamFinished.Payload["ca3_compaction_embedding_status"])
	}
	if runFinished.Payload["ca3_compaction_embedding_provider"] != streamFinished.Payload["ca3_compaction_embedding_provider"] {
		t.Fatalf("run/stream ca3 compaction embedding provider mismatch: run=%v stream=%v", runFinished.Payload["ca3_compaction_embedding_provider"], streamFinished.Payload["ca3_compaction_embedding_provider"])
	}
}

func TestRunAndStreamCA3GovernanceSemanticsEquivalent(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
context_assembler:
  enabled: true
  ca3:
    enabled: true
    max_context_tokens: 80
    percent_thresholds:
      safe: 10
      comfort: 20
      warning: 30
      danger: 40
      emergency: 50
    absolute_thresholds:
      safe: 8
      comfort: 16
      warning: 24
      danger: 32
      emergency: 40
    compaction:
      mode: semantic
      quality:
        threshold: 0.25
      embedding:
        enabled: true
        selector: default
        provider: anthropic
        model: claude-3-haiku
        timeout: 300ms
      reranker:
        enabled: true
        timeout: 200ms
        max_retries: 0
        governance:
          mode: enforce
          profile_version: e5-canary-v1
          rollout_provider_models:
            - anthropic:claude-3-haiku
        threshold_profiles:
          anthropic:claude-3-haiku: 0.75
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	runModel := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			if strings.Contains(req.Input, "Source:") {
				return types.ModelResponse{FinalAnswer: "semantic governance summary"}, nil
			}
			return types.ModelResponse{FinalAnswer: "ok"}, nil
		},
	}
	streamModel := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			if strings.Contains(req.Input, "Source:") {
				return types.ModelResponse{FinalAnswer: "semantic governance summary"}, nil
			}
			return types.ModelResponse{FinalAnswer: "ok"}, nil
		},
		stream: func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
			return onEvent(types.ModelEvent{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "ok"})
		},
	}
	runCollector := &eventCollector{}
	streamCollector := &eventCollector{}

	runEngine := New(runModel, WithRuntimeManager(mgr))
	streamEngine := New(streamModel, WithRuntimeManager(mgr))
	_, err = runEngine.Run(context.Background(), types.RunRequest{
		RunID:     "run-ca3-governance",
		SessionID: "s-1",
		Input:     strings.Repeat("payload ", 40),
		Messages:  []types.Message{{Role: "system", Content: "base"}},
	}, runCollector)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	_, err = streamEngine.Stream(context.Background(), types.RunRequest{
		RunID:     "stream-ca3-governance",
		SessionID: "s-1",
		Input:     strings.Repeat("payload ", 40),
		Messages:  []types.Message{{Role: "system", Content: "base"}},
	}, streamCollector)
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}

	runFinished, ok := runCollector.lastNonTimelineEvent()
	if !ok {
		t.Fatal("missing run finished event")
	}
	streamFinished, ok := streamCollector.lastNonTimelineEvent()
	if !ok {
		t.Fatal("missing stream finished event")
	}
	for _, key := range []string{
		"ca3_compaction_reranker_threshold_source",
		"ca3_compaction_reranker_threshold_hit",
		"ca3_compaction_reranker_profile_version",
		"ca3_compaction_reranker_rollout_hit",
	} {
		if runFinished.Payload[key] != streamFinished.Payload[key] {
			t.Fatalf("run/stream governance payload mismatch for %s: run=%v stream=%v", key, runFinished.Payload[key], streamFinished.Payload[key])
		}
	}
}

func TestActionGateRequireConfirmWithoutResolverFailsFast(t *testing.T) {
	reg := local.NewRegistry()
	_, err := reg.Register(&fakeTool{
		name: "shell",
		invoke: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
			return types.ToolResult{Content: "ok"}, nil
		},
	})
	if err != nil {
		t.Fatalf("register tool: %v", err)
	}
	model := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "c1", Name: "local.shell"}}}, nil
		},
	}
	collector := &eventCollector{}
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
action_gate:
  enabled: true
  policy: require_confirm
  timeout: 100ms
  tool_names: [shell]
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	engine := New(model, WithLocalRegistry(reg), WithRuntimeManager(mgr))
	res, runErr := engine.Run(context.Background(), types.RunRequest{Input: "please run shell"}, collector)
	if runErr == nil {
		t.Fatal("expected action gate fail-fast error")
	}
	if res.Error == nil || res.Error.Class != types.ErrTool {
		t.Fatalf("error class = %#v, want ErrTool", res.Error)
	}
	finished, ok := collector.lastNonTimelineEvent()
	if !ok {
		t.Fatal("missing run.finished")
	}
	if finished.Payload["gate_checks"] != 1 || finished.Payload["gate_denied_count"] != 1 || finished.Payload["gate_timeout_count"] != 0 {
		t.Fatalf("unexpected gate counters: %#v", finished.Payload)
	}
	steps := timelinePhaseStatus(collector.timelineEvents())
	if !containsTimelineStep(steps, "tool:failed") {
		t.Fatalf("expected tool:failed timeline, got %v", steps)
	}
}

func TestActionGateResolverTimeoutDeny(t *testing.T) {
	reg := local.NewRegistry()
	_, err := reg.Register(&fakeTool{
		name: "shell",
		invoke: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
			return types.ToolResult{Content: "ok"}, nil
		},
	})
	if err != nil {
		t.Fatalf("register tool: %v", err)
	}
	model := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "c1", Name: "local.shell"}}}, nil
		},
	}
	matcher := &fakeGateMatcher{
		evaluate: func(ctx context.Context, check types.ActionGateCheck) (types.ActionGateDecision, error) {
			return types.ActionGateDecisionRequireConfirm, nil
		},
	}
	resolver := &fakeGateResolver{
		confirm: func(ctx context.Context, req types.ActionGateConfirmRequest) (bool, error) {
			<-ctx.Done()
			return false, ctx.Err()
		},
	}
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
action_gate:
  timeout: 1ms
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	engine := New(model,
		WithLocalRegistry(reg),
		WithRuntimeManager(mgr),
		WithActionGateMatcher(matcher),
		WithActionGateResolver(resolver),
	)
	collector := &eventCollector{}
	res, runErr := engine.Run(context.Background(), types.RunRequest{Input: "do shell"}, collector)
	if !errors.Is(runErr, context.DeadlineExceeded) {
		t.Fatalf("runErr = %v, want deadline exceeded", runErr)
	}
	if res.Error == nil || res.Error.Class != types.ErrPolicyTimeout {
		t.Fatalf("error class = %#v, want ErrPolicyTimeout", res.Error)
	}
	finished, ok := collector.lastNonTimelineEvent()
	if !ok {
		t.Fatal("missing run.finished")
	}
	if finished.Payload["gate_checks"] != 1 || finished.Payload["gate_denied_count"] != 1 || finished.Payload["gate_timeout_count"] != 1 {
		t.Fatalf("unexpected gate counters: %#v", finished.Payload)
	}
	steps := timelinePhaseStatus(collector.timelineEvents())
	if !containsTimelineStep(steps, "tool:canceled") {
		t.Fatalf("expected tool:canceled timeline for timeout, got %v", steps)
	}
}

func TestActionGateAllowPathKeepsToolExecution(t *testing.T) {
	reg := local.NewRegistry()
	_, err := reg.Register(&fakeTool{
		name: "echo",
		invoke: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
			return types.ToolResult{Content: "ok"}, nil
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
				return types.ModelResponse{
					ToolCalls: []types.ToolCall{{CallID: "c1", Name: "local.echo", Args: map[string]any{"cmd": "list"}}},
				}, nil
			}
			return types.ModelResponse{FinalAnswer: "done"}, nil
		},
	}
	matcher := &fakeGateMatcher{
		evaluate: func(ctx context.Context, check types.ActionGateCheck) (types.ActionGateDecision, error) {
			return types.ActionGateDecisionAllow, nil
		},
	}
	engine := New(model, WithLocalRegistry(reg), WithActionGateMatcher(matcher))
	res, runErr := engine.Run(context.Background(), types.RunRequest{Input: "run safely"}, nil)
	if runErr != nil {
		t.Fatalf("Run failed: %v", runErr)
	}
	if res.FinalAnswer != "done" {
		t.Fatalf("final answer = %q, want done", res.FinalAnswer)
	}
}

func TestActionGateKeywordRuleHitAndMiss(t *testing.T) {
	reg := local.NewRegistry()
	_, err := reg.Register(&fakeTool{name: "echo"})
	if err != nil {
		t.Fatalf("register tool: %v", err)
	}
	calls := 0
	model := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			calls++
			if calls == 1 {
				return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "c1", Name: "local.echo"}}}, nil
			}
			return types.ModelResponse{FinalAnswer: "ok"}, nil
		},
	}
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
action_gate:
  enabled: true
  policy: require_confirm
  timeout: 100ms
  keywords: [danger]
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	engine := New(model, WithLocalRegistry(reg), WithRuntimeManager(mgr), WithActionGateResolver(&fakeGateResolver{}))
	if _, runErr := engine.Run(context.Background(), types.RunRequest{Input: "safe input"}, nil); runErr != nil {
		t.Fatalf("safe input should pass with resolver: %v", runErr)
	}

	calls = 0
	engineNoResolver := New(model, WithLocalRegistry(reg), WithRuntimeManager(mgr))
	if _, runErr := engineNoResolver.Run(context.Background(), types.RunRequest{Input: "danger delete request"}, nil); runErr == nil {
		t.Fatal("expected deny when keyword rule hits and resolver missing")
	}
}

func TestActionGateRunAndStreamDenySemanticsEquivalent(t *testing.T) {
	runModel := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "c1", Name: "local.shell"}}}, nil
		},
	}
	streamModel := &fakeModel{
		stream: func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
			return onEvent(types.ModelEvent{
				Type:     types.ModelEventTypeToolCall,
				ToolCall: &types.ToolCall{CallID: "c1", Name: "local.shell"},
			})
		},
	}
	denyMatcher := &fakeGateMatcher{
		evaluate: func(ctx context.Context, check types.ActionGateCheck) (types.ActionGateDecision, error) {
			return types.ActionGateDecisionDeny, nil
		},
	}

	reg := local.NewRegistry()
	_, _ = reg.Register(&fakeTool{name: "shell"})
	runCollector := &eventCollector{}
	streamCollector := &eventCollector{}
	runEngine := New(runModel, WithLocalRegistry(reg), WithActionGateMatcher(denyMatcher))
	streamEngine := New(streamModel, WithActionGateMatcher(denyMatcher))

	runRes, runErr := runEngine.Run(context.Background(), types.RunRequest{Input: "danger"}, runCollector)
	if runErr == nil {
		t.Fatal("run should be denied by action gate")
	}
	streamRes, streamErr := streamEngine.Stream(context.Background(), types.RunRequest{Input: "danger"}, streamCollector)
	if streamErr == nil {
		t.Fatal("stream should be denied by action gate")
	}
	if runRes.Error == nil || streamRes.Error == nil {
		t.Fatalf("expected classified errors for run/stream, got run=%#v stream=%#v", runRes.Error, streamRes.Error)
	}
	if runRes.Error.Class != streamRes.Error.Class {
		t.Fatalf("run/stream error class mismatch: run=%s stream=%s", runRes.Error.Class, streamRes.Error.Class)
	}
}

func TestActionGateRunAndStreamTimeoutSemanticsEquivalent(t *testing.T) {
	runModel := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{ToolCalls: []types.ToolCall{{CallID: "c1", Name: "local.shell"}}}, nil
		},
	}
	streamModel := &fakeModel{
		stream: func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
			return onEvent(types.ModelEvent{
				Type:     types.ModelEventTypeToolCall,
				ToolCall: &types.ToolCall{CallID: "c1", Name: "local.shell"},
			})
		},
	}
	requireMatcher := &fakeGateMatcher{
		evaluate: func(ctx context.Context, check types.ActionGateCheck) (types.ActionGateDecision, error) {
			return types.ActionGateDecisionRequireConfirm, nil
		},
	}
	timeoutResolver := &fakeGateResolver{
		confirm: func(ctx context.Context, req types.ActionGateConfirmRequest) (bool, error) {
			<-ctx.Done()
			return false, ctx.Err()
		},
	}

	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
action_gate:
  timeout: 1ms
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	reg := local.NewRegistry()
	_, _ = reg.Register(&fakeTool{name: "shell"})

	runEngine := New(runModel,
		WithLocalRegistry(reg),
		WithRuntimeManager(mgr),
		WithActionGateMatcher(requireMatcher),
		WithActionGateResolver(timeoutResolver),
	)
	streamEngine := New(streamModel,
		WithRuntimeManager(mgr),
		WithActionGateMatcher(requireMatcher),
		WithActionGateResolver(timeoutResolver),
	)

	runRes, runErr := runEngine.Run(context.Background(), types.RunRequest{Input: "danger"}, nil)
	streamRes, streamErr := streamEngine.Stream(context.Background(), types.RunRequest{Input: "danger"}, nil)
	if !errors.Is(runErr, context.DeadlineExceeded) || !errors.Is(streamErr, context.DeadlineExceeded) {
		t.Fatalf("expected timeout for run/stream, got run=%v stream=%v", runErr, streamErr)
	}
	if runRes.Error == nil || streamRes.Error == nil {
		t.Fatalf("missing run/stream classified errors: run=%#v stream=%#v", runRes.Error, streamRes.Error)
	}
	if runRes.Error.Class != types.ErrPolicyTimeout || streamRes.Error.Class != types.ErrPolicyTimeout {
		t.Fatalf("run/stream error class mismatch: run=%#v stream=%#v", runRes.Error, streamRes.Error)
	}
}

func TestClarificationRunLifecycleResume(t *testing.T) {
	turn := 0
	model := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			turn++
			if turn == 1 {
				return types.ModelResponse{
					ClarificationRequest: &types.ClarificationRequest{
						RequestID:      "clarify-1",
						Questions:      []string{"which env?"},
						ContextSummary: "missing env",
					},
				}, nil
			}
			if len(req.Messages) == 0 || !strings.Contains(req.Messages[len(req.Messages)-1].Content, "prod") {
				t.Fatalf("clarification answer should be injected into request messages: %#v", req.Messages)
			}
			return types.ModelResponse{FinalAnswer: "done"}, nil
		},
	}
	resolver := &fakeClarificationResolver{
		resolve: func(ctx context.Context, req types.ClarificationResolveRequest) (types.ClarificationResponse, error) {
			return types.ClarificationResponse{
				RequestID: req.Request.RequestID,
				Answers:   []string{"prod"},
			}, nil
		},
	}
	collector := &eventCollector{}
	engine := New(model, WithClarificationResolver(resolver))
	res, err := engine.Run(context.Background(), types.RunRequest{Input: "deploy?"}, collector)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if res.FinalAnswer != "done" {
		t.Fatalf("final answer = %q, want done", res.FinalAnswer)
	}
	finished, ok := collector.lastNonTimelineEvent()
	if !ok {
		t.Fatal("missing run.finished")
	}
	if finished.Payload["await_count"] != 1 || finished.Payload["resume_count"] != 1 || finished.Payload["cancel_by_user_count"] != 0 {
		t.Fatalf("unexpected clarification counters: %#v", finished.Payload)
	}
	timeline := timelinePhaseStatus(collector.timelineEvents())
	if !containsTimelineStep(timeline, "hitl:pending") || !containsTimelineStep(timeline, "hitl:succeeded") {
		t.Fatalf("missing hitl lifecycle timeline events: %v", timeline)
	}
}

func TestClarificationRunTimeoutCancelsByUser(t *testing.T) {
	model := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{
				ClarificationRequest: &types.ClarificationRequest{
					RequestID: "clarify-timeout",
				},
			}, nil
		},
	}
	resolver := &fakeClarificationResolver{
		resolve: func(ctx context.Context, req types.ClarificationResolveRequest) (types.ClarificationResponse, error) {
			<-ctx.Done()
			return types.ClarificationResponse{}, ctx.Err()
		},
	}
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
clarification:
  enabled: true
  timeout: 1ms
  timeout_policy: cancel_by_user
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	collector := &eventCollector{}
	engine := New(model, WithRuntimeManager(mgr), WithClarificationResolver(resolver))
	res, runErr := engine.Run(context.Background(), types.RunRequest{Input: "need info"}, collector)
	if !errors.Is(runErr, context.DeadlineExceeded) {
		t.Fatalf("runErr = %v, want deadline exceeded", runErr)
	}
	if res.Error == nil || res.Error.Class != types.ErrPolicyTimeout {
		t.Fatalf("error class = %#v, want ErrPolicyTimeout", res.Error)
	}
	finished, ok := collector.lastNonTimelineEvent()
	if !ok {
		t.Fatal("missing run.finished")
	}
	if finished.Payload["await_count"] != 1 || finished.Payload["resume_count"] != 0 || finished.Payload["cancel_by_user_count"] != 1 {
		t.Fatalf("unexpected clarification counters: %#v", finished.Payload)
	}
	timeline := collector.timelineEvents()
	reasons := make([]string, 0, len(timeline))
	for _, ev := range timeline {
		if reason, _ := ev.Payload["reason"].(string); reason != "" {
			reasons = append(reasons, reason)
		}
	}
	if !containsString(reasons, "hitl.await_user") || !containsString(reasons, "hitl.canceled_by_user") {
		t.Fatalf("missing hitl reason codes: %v", reasons)
	}
}

func TestClarificationRunAndStreamTimeoutSemanticsEquivalent(t *testing.T) {
	runModel := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{
				ClarificationRequest: &types.ClarificationRequest{RequestID: "c-1"},
			}, nil
		},
	}
	streamModel := &fakeModel{
		stream: func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
			return onEvent(types.ModelEvent{
				Type: types.ModelEventTypeClarificationRequest,
				ClarificationRequest: &types.ClarificationRequest{
					RequestID: "c-1",
				},
			})
		},
	}
	resolver := &fakeClarificationResolver{
		resolve: func(ctx context.Context, req types.ClarificationResolveRequest) (types.ClarificationResponse, error) {
			<-ctx.Done()
			return types.ClarificationResponse{}, ctx.Err()
		},
	}
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
clarification:
  enabled: true
  timeout: 1ms
  timeout_policy: cancel_by_user
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	runCollector := &eventCollector{}
	streamCollector := &eventCollector{}
	runEngine := New(runModel, WithRuntimeManager(mgr), WithClarificationResolver(resolver))
	streamEngine := New(streamModel, WithRuntimeManager(mgr), WithClarificationResolver(resolver))

	runRes, runErr := runEngine.Run(context.Background(), types.RunRequest{Input: "x"}, runCollector)
	streamRes, streamErr := streamEngine.Stream(context.Background(), types.RunRequest{Input: "x"}, streamCollector)
	if !errors.Is(runErr, context.DeadlineExceeded) || !errors.Is(streamErr, context.DeadlineExceeded) {
		t.Fatalf("expected deadline for run/stream, got run=%v stream=%v", runErr, streamErr)
	}
	if runRes.Error == nil || streamRes.Error == nil {
		t.Fatalf("missing classified errors: run=%#v stream=%#v", runRes.Error, streamRes.Error)
	}
	if runRes.Error.Class != streamRes.Error.Class || runRes.Error.Class != types.ErrPolicyTimeout {
		t.Fatalf("run/stream class mismatch: run=%#v stream=%#v", runRes.Error, streamRes.Error)
	}
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func TestActionGateParameterRuleOperators(t *testing.T) {
	cases := []struct {
		name      string
		condition types.ActionGateRuleCondition
		args      map[string]any
		want      bool
	}{
		{
			name: "eq",
			condition: types.ActionGateRuleCondition{
				Path:     "q",
				Operator: types.ActionGateRuleOperatorEQ,
				Expected: "ok",
			},
			args: map[string]any{"q": "ok"},
			want: true,
		},
		{
			name: "ne",
			condition: types.ActionGateRuleCondition{
				Path:     "q",
				Operator: types.ActionGateRuleOperatorNE,
				Expected: "bad",
			},
			args: map[string]any{"q": "ok"},
			want: true,
		},
		{
			name: "contains",
			condition: types.ActionGateRuleCondition{
				Path:     "q",
				Operator: types.ActionGateRuleOperatorContains,
				Expected: "loop",
			},
			args: map[string]any{"q": "tool-loop"},
			want: true,
		},
		{
			name: "regex",
			condition: types.ActionGateRuleCondition{
				Path:     "q",
				Operator: types.ActionGateRuleOperatorRegex,
				Expected: "^tool-",
			},
			args: map[string]any{"q": "tool-loop"},
			want: true,
		},
		{
			name: "in",
			condition: types.ActionGateRuleCondition{
				Path:     "priority",
				Operator: types.ActionGateRuleOperatorIn,
				Expected: []any{"high", "critical"},
			},
			args: map[string]any{"priority": "high"},
			want: true,
		},
		{
			name: "not_in",
			condition: types.ActionGateRuleCondition{
				Path:     "priority",
				Operator: types.ActionGateRuleOperatorNotIn,
				Expected: []any{"low", "medium"},
			},
			args: map[string]any{"priority": "high"},
			want: true,
		},
		{
			name: "gt",
			condition: types.ActionGateRuleCondition{
				Path:     "size",
				Operator: types.ActionGateRuleOperatorGT,
				Expected: 10,
			},
			args: map[string]any{"size": 12},
			want: true,
		},
		{
			name: "gte",
			condition: types.ActionGateRuleCondition{
				Path:     "size",
				Operator: types.ActionGateRuleOperatorGTE,
				Expected: 12,
			},
			args: map[string]any{"size": 12},
			want: true,
		},
		{
			name: "lt",
			condition: types.ActionGateRuleCondition{
				Path:     "size",
				Operator: types.ActionGateRuleOperatorLT,
				Expected: 20,
			},
			args: map[string]any{"size": 12},
			want: true,
		},
		{
			name: "lte",
			condition: types.ActionGateRuleCondition{
				Path:     "size",
				Operator: types.ActionGateRuleOperatorLTE,
				Expected: 12,
			},
			args: map[string]any{"size": 12},
			want: true,
		},
		{
			name: "exists",
			condition: types.ActionGateRuleCondition{
				Path:     "meta.token",
				Operator: types.ActionGateRuleOperatorExists,
			},
			args: map[string]any{"meta": map[string]any{"token": "x"}},
			want: true,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := evaluateRuleCondition(tc.condition, tc.args)
			if err != nil {
				t.Fatalf("evaluateRuleCondition failed: %v", err)
			}
			if got != tc.want {
				t.Fatalf("evaluateRuleCondition = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestActionGateParameterRuleCompositeShortCircuit(t *testing.T) {
	andCondition := types.ActionGateRuleCondition{
		All: []types.ActionGateRuleCondition{
			{
				Path:     "q",
				Operator: types.ActionGateRuleOperatorEQ,
				Expected: "miss",
			},
			{
				Path:     "q",
				Operator: types.ActionGateRuleOperatorRegex,
				Expected: 42, // invalid regex expected type; should not execute because first condition is false.
			},
		},
	}
	matched, err := evaluateRuleCondition(andCondition, map[string]any{"q": "tool-loop"})
	if err != nil {
		t.Fatalf("AND short-circuit should not error, got %v", err)
	}
	if matched {
		t.Fatal("AND condition should be false")
	}

	orCondition := types.ActionGateRuleCondition{
		Any: []types.ActionGateRuleCondition{
			{
				Path:     "q",
				Operator: types.ActionGateRuleOperatorContains,
				Expected: "tool",
			},
			{
				Path:     "q",
				Operator: types.ActionGateRuleOperatorRegex,
				Expected: 42, // invalid regex expected type; should not execute because first condition is true.
			},
		},
	}
	matched, err = evaluateRuleCondition(orCondition, map[string]any{"q": "tool-loop"})
	if err != nil {
		t.Fatalf("OR short-circuit should not error, got %v", err)
	}
	if !matched {
		t.Fatal("OR condition should be true")
	}
}

func TestActionGateParameterRulePriorityOverKeyword(t *testing.T) {
	reg := local.NewRegistry()
	_, err := reg.Register(&fakeTool{name: "echo"})
	if err != nil {
		t.Fatalf("register tool: %v", err)
	}
	calls := 0
	model := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			calls++
			if calls == 1 {
				return types.ModelResponse{
					ToolCalls: []types.ToolCall{{CallID: "c1", Name: "local.echo", Args: map[string]any{"q": "tool-loop"}}},
				}, nil
			}
			return types.ModelResponse{FinalAnswer: "done"}, nil
		},
	}
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
action_gate:
  enabled: true
  policy: require_confirm
  decision_by_keyword:
    "tool-loop": deny
  parameter_rules:
    - id: allow-echoloop
      tool_names: [echo]
      action: allow
      condition:
        path: q
        operator: contains
        expected: tool-loop
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	collector := &eventCollector{}
	engine := New(model, WithLocalRegistry(reg), WithRuntimeManager(mgr))
	res, runErr := engine.Run(context.Background(), types.RunRequest{Input: "tool-loop"}, collector)
	if runErr != nil {
		t.Fatalf("run should allow by parameter rule priority: %v", runErr)
	}
	if res.FinalAnswer != "done" {
		t.Fatalf("final answer = %q, want done", res.FinalAnswer)
	}
	reasons := make([]string, 0)
	for _, ev := range collector.timelineEvents() {
		if reason, _ := ev.Payload["reason"].(string); reason != "" {
			reasons = append(reasons, reason)
		}
	}
	if !containsString(reasons, "gate.rule_match") {
		t.Fatalf("expected gate.rule_match timeline reason, got %v", reasons)
	}
	finished, ok := collector.lastNonTimelineEvent()
	if !ok {
		t.Fatal("missing run.finished")
	}
	if finished.Payload["gate_rule_hit_count"] != 1 || finished.Payload["gate_rule_last_id"] != "allow-echoloop" {
		t.Fatalf("unexpected rule diagnostics payload: %#v", finished.Payload)
	}
}

func TestActionGateParameterRuleActionInheritsGlobalPolicy(t *testing.T) {
	reg := local.NewRegistry()
	_, err := reg.Register(&fakeTool{name: "echo"})
	if err != nil {
		t.Fatalf("register tool: %v", err)
	}
	model := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{
				ToolCalls: []types.ToolCall{{CallID: "c1", Name: "local.echo", Args: map[string]any{"dangerous": true}}},
			}, nil
		},
	}
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
action_gate:
  enabled: true
  policy: deny
  parameter_rules:
    - id: inherit-deny
      tool_names: [echo]
      condition:
        path: dangerous
        operator: eq
        expected: true
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	engine := New(model, WithLocalRegistry(reg), WithRuntimeManager(mgr))
	res, runErr := engine.Run(context.Background(), types.RunRequest{Input: "x"}, nil)
	if runErr == nil {
		t.Fatal("expected deny from inherited global policy")
	}
	if res.Error == nil || res.Error.Class != types.ErrTool {
		t.Fatalf("error class = %#v, want ErrTool", res.Error)
	}
}

func TestActionGateParameterRuleRunAndStreamTimeoutSemanticsEquivalent(t *testing.T) {
	runModel := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{
				ToolCalls: []types.ToolCall{{CallID: "c1", Name: "local.shell", Args: map[string]any{"cmd": "danger"}}},
			}, nil
		},
	}
	streamModel := &fakeModel{
		stream: func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
			return onEvent(types.ModelEvent{
				Type: types.ModelEventTypeToolCall,
				ToolCall: &types.ToolCall{
					CallID: "c1",
					Name:   "local.shell",
					Args:   map[string]any{"cmd": "danger"},
				},
			})
		},
	}
	timeoutResolver := &fakeGateResolver{
		confirm: func(ctx context.Context, req types.ActionGateConfirmRequest) (bool, error) {
			<-ctx.Done()
			return false, ctx.Err()
		},
	}

	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
action_gate:
  enabled: true
  timeout: 1ms
  policy: allow
  parameter_rules:
    - id: require-confirm
      tool_names: [shell]
      action: require_confirm
      condition:
        path: cmd
        operator: contains
        expected: danger
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	reg := local.NewRegistry()
	_, _ = reg.Register(&fakeTool{name: "shell"})
	runEngine := New(runModel, WithLocalRegistry(reg), WithRuntimeManager(mgr), WithActionGateResolver(timeoutResolver))
	streamEngine := New(streamModel, WithRuntimeManager(mgr), WithActionGateResolver(timeoutResolver))

	runRes, runErr := runEngine.Run(context.Background(), types.RunRequest{Input: "danger"}, nil)
	streamRes, streamErr := streamEngine.Stream(context.Background(), types.RunRequest{Input: "danger"}, nil)
	if !errors.Is(runErr, context.DeadlineExceeded) || !errors.Is(streamErr, context.DeadlineExceeded) {
		t.Fatalf("expected timeout for run/stream, got run=%v stream=%v", runErr, streamErr)
	}
	if runRes.Error == nil || streamRes.Error == nil {
		t.Fatalf("missing run/stream classified errors: run=%#v stream=%#v", runRes.Error, streamRes.Error)
	}
	if runRes.Error.Class != types.ErrPolicyTimeout || streamRes.Error.Class != types.ErrPolicyTimeout {
		t.Fatalf("run/stream error class mismatch: run=%#v stream=%#v", runRes.Error, streamRes.Error)
	}
}

func TestCA2AgenticRoutingRunAndStreamSemanticEquivalent(t *testing.T) {
	dir := t.TempDir()
	stage2File := filepath.Join(dir, "stage2.jsonl")
	if err := os.WriteFile(stage2File, []byte(`{"session_id":"session-1","content":"agentic-ctx"}`), 0o600); err != nil {
		t.Fatalf("write stage2 file: %v", err)
	}
	cfgPath := filepath.Join(dir, "runtime.yaml")
	cfg := fmt.Sprintf(`
context_assembler:
  enabled: true
  journal_path: '%s'
  ca2:
    enabled: true
    routing_mode: agentic
    agentic:
      decision_timeout: 80ms
      failure_policy: best_effort_rules
    stage2:
      provider: file
      file_path: '%s'
    routing:
      min_input_chars: 9999
      trigger_keywords: []
  ca3:
    enabled: false
`, filepath.ToSlash(filepath.Join(dir, "journal.jsonl")), filepath.ToSlash(stage2File))
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	runModel := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{FinalAnswer: "ok"}, nil
		},
	}
	streamModel := &fakeModel{
		stream: func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
			return onEvent(types.ModelEvent{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "ok"})
		},
	}
	router := assembler.AgenticRouterFunc(func(ctx context.Context, req assembler.AgenticRoutingRequest) (assembler.AgenticRoutingDecision, error) {
		return assembler.AgenticRoutingDecision{RunStage2: true, Reason: "agentic.force_stage2"}, nil
	})
	runEngine := New(runModel, WithRuntimeManager(mgr), WithContextAssemblerAgenticRouter(router))
	streamEngine := New(streamModel, WithRuntimeManager(mgr), WithContextAssemblerAgenticRouter(router))
	runCollector := &eventCollector{}
	streamCollector := &eventCollector{}
	baseReq := types.RunRequest{
		SessionID: "session-1",
		Input:     "short",
		Messages:  []types.Message{{Role: "system", Content: "s"}},
	}

	runRes, runErr := runEngine.Run(context.Background(), baseReq, runCollector)
	streamRes, streamErr := streamEngine.Stream(context.Background(), baseReq, streamCollector)
	if runErr != nil || streamErr != nil {
		t.Fatalf("run/stream failed: run=%v stream=%v", runErr, streamErr)
	}
	if runRes.FinalAnswer != "ok" || streamRes.FinalAnswer != "ok" {
		t.Fatalf("final answer mismatch run=%q stream=%q", runRes.FinalAnswer, streamRes.FinalAnswer)
	}

	runFinished, ok := runCollector.lastNonTimelineEvent()
	if !ok {
		t.Fatal("missing run run.finished")
	}
	streamFinished, ok := streamCollector.lastNonTimelineEvent()
	if !ok {
		t.Fatal("missing stream run.finished")
	}
	keys := []string{
		"assemble_stage_status",
		"stage2_router_mode",
		"stage2_router_decision",
		"stage2_router_reason",
		"stage2_router_error",
	}
	for _, key := range keys {
		if runFinished.Payload[key] != streamFinished.Payload[key] {
			t.Fatalf("run/stream payload mismatch key=%s run=%#v stream=%#v", key, runFinished.Payload[key], streamFinished.Payload[key])
		}
	}
	if runFinished.Payload["stage2_router_mode"] != "agentic" {
		t.Fatalf("stage2_router_mode = %#v, want agentic", runFinished.Payload["stage2_router_mode"])
	}
	if runFinished.Payload["stage2_router_decision"] != "run_stage2" {
		t.Fatalf("stage2_router_decision = %#v, want run_stage2", runFinished.Payload["stage2_router_decision"])
	}
	if runFinished.Payload["stage2_router_reason"] != "agentic.force_stage2" {
		t.Fatalf("stage2_router_reason = %#v, want agentic.force_stage2", runFinished.Payload["stage2_router_reason"])
	}
}

func TestCA2AgenticRoutingFallbackRunAndStreamSemanticEquivalent(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "runtime.yaml")
	cfg := fmt.Sprintf(`
context_assembler:
  enabled: true
  journal_path: '%s'
  ca2:
    enabled: true
    routing_mode: agentic
    agentic:
      decision_timeout: 80ms
      failure_policy: best_effort_rules
    routing:
      min_input_chars: 9999
      trigger_keywords: []
  ca3:
    enabled: false
`, filepath.ToSlash(filepath.Join(dir, "journal.jsonl")))
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	runModel := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{FinalAnswer: "ok"}, nil
		},
	}
	streamModel := &fakeModel{
		stream: func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
			return onEvent(types.ModelEvent{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "ok"})
		},
	}
	runEngine := New(runModel, WithRuntimeManager(mgr))
	streamEngine := New(streamModel, WithRuntimeManager(mgr))
	runCollector := &eventCollector{}
	streamCollector := &eventCollector{}
	baseReq := types.RunRequest{
		SessionID: "session-1",
		Input:     "short",
		Messages:  []types.Message{{Role: "system", Content: "s"}},
	}

	runRes, runErr := runEngine.Run(context.Background(), baseReq, runCollector)
	streamRes, streamErr := streamEngine.Stream(context.Background(), baseReq, streamCollector)
	if runErr != nil || streamErr != nil {
		t.Fatalf("run/stream failed: run=%v stream=%v", runErr, streamErr)
	}
	if runRes.FinalAnswer != "ok" || streamRes.FinalAnswer != "ok" {
		t.Fatalf("final answer mismatch run=%q stream=%q", runRes.FinalAnswer, streamRes.FinalAnswer)
	}

	runFinished, ok := runCollector.lastNonTimelineEvent()
	if !ok {
		t.Fatal("missing run run.finished")
	}
	streamFinished, ok := streamCollector.lastNonTimelineEvent()
	if !ok {
		t.Fatal("missing stream run.finished")
	}
	keys := []string{
		"assemble_stage_status",
		"stage2_skip_reason",
		"stage2_router_mode",
		"stage2_router_decision",
		"stage2_router_reason",
		"stage2_router_error",
	}
	for _, key := range keys {
		if runFinished.Payload[key] != streamFinished.Payload[key] {
			t.Fatalf("run/stream payload mismatch key=%s run=%#v stream=%#v", key, runFinished.Payload[key], streamFinished.Payload[key])
		}
	}
	if runFinished.Payload["stage2_router_mode"] != "agentic" {
		t.Fatalf("stage2_router_mode = %#v, want agentic", runFinished.Payload["stage2_router_mode"])
	}
	if runFinished.Payload["stage2_router_decision"] != "skip_stage2" {
		t.Fatalf("stage2_router_decision = %#v, want skip_stage2", runFinished.Payload["stage2_router_decision"])
	}
	if runFinished.Payload["stage2_router_error"] != "agentic.callback_missing" {
		t.Fatalf("stage2_router_error = %#v, want agentic.callback_missing", runFinished.Payload["stage2_router_error"])
	}
	reason, _ := runFinished.Payload["stage2_router_reason"].(string)
	if !strings.Contains(reason, "agentic.fallback.agentic.callback_missing") {
		t.Fatalf("stage2_router_reason = %q, want fallback marker", reason)
	}
}

func TestRunBackpressureBlockDiagnosticsAndTimeline(t *testing.T) {
	reg := local.NewRegistry()
	_, err := reg.Register(&fakeTool{
		name: "slow",
		invoke: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
			time.Sleep(2 * time.Millisecond)
			return types.ToolResult{Content: "ok"}, nil
		},
	})
	if err != nil {
		t.Fatalf("register tool: %v", err)
	}
	turn := 0
	model := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			turn++
			if turn == 1 {
				calls := make([]types.ToolCall, 0, 6)
				for i := 0; i < 6; i++ {
					calls = append(calls, types.ToolCall{
						CallID: fmt.Sprintf("c-%d", i),
						Name:   "local.slow",
					})
				}
				return types.ModelResponse{ToolCalls: calls}, nil
			}
			return types.ModelResponse{FinalAnswer: "done"}, nil
		},
	}
	policy := types.DefaultLoopPolicy()
	policy.LocalDispatch.MaxWorkers = 2
	policy.LocalDispatch.QueueSize = 1
	policy.LocalDispatch.Backpressure = types.BackpressureBlock
	engine := New(model, WithLocalRegistry(reg))
	collector := &eventCollector{}
	res, runErr := engine.Run(context.Background(), types.RunRequest{Input: "x", Policy: &policy}, collector)
	if runErr != nil {
		t.Fatalf("run failed: %v", runErr)
	}
	if res.FinalAnswer != "done" {
		t.Fatalf("final answer = %q, want done", res.FinalAnswer)
	}
	finished, ok := collector.lastNonTimelineEvent()
	if !ok {
		t.Fatal("missing run.finished")
	}
	if finished.Payload["backpressure_drop_count"] != 0 {
		t.Fatalf("backpressure_drop_count = %#v, want 0", finished.Payload["backpressure_drop_count"])
	}
	inflight, _ := finished.Payload["inflight_peak"].(int)
	if inflight <= 0 {
		t.Fatalf("inflight_peak = %#v, want > 0", finished.Payload["inflight_peak"])
	}
	reasons := make([]string, 0)
	for _, ev := range collector.timelineEvents() {
		if reason, _ := ev.Payload["reason"].(string); reason != "" {
			reasons = append(reasons, reason)
		}
	}
	if !containsString(reasons, "backpressure.block") {
		t.Fatalf("expected backpressure.block reason, got %v", reasons)
	}
}

func TestRunBackpressureDropLowPriorityAllDroppedFailsFast(t *testing.T) {
	reg := local.NewRegistry()
	_, err := reg.Register(&fakeTool{
		name: "slow",
		invoke: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
			time.Sleep(2 * time.Millisecond)
			return types.ToolResult{Content: "ok"}, nil
		},
	})
	if err != nil {
		t.Fatalf("register tool: %v", err)
	}
	model := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{
				ToolCalls: []types.ToolCall{
					{CallID: "c-1", Name: "local.slow", Args: map[string]any{"q": "cache warmup"}},
					{CallID: "c-2", Name: "local.slow", Args: map[string]any{"q": "cache prefetch"}},
				},
			}, nil
		},
	}
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
concurrency:
  backpressure: drop_low_priority
  local_max_workers: 1
  local_queue_size: 1
  drop_low_priority:
    priority_by_keyword:
      cache: low
    droppable_priorities: [low]
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	engine := New(model, WithLocalRegistry(reg), WithRuntimeManager(mgr))
	collector := &eventCollector{}
	res, runErr := engine.Run(context.Background(), types.RunRequest{Input: "x"}, collector)
	if runErr == nil {
		t.Fatal("expected fail-fast error when all calls are dropped")
	}
	if res.Error == nil || res.Error.Class != types.ErrTool {
		t.Fatalf("unexpected run error: %#v", res.Error)
	}
	finished, ok := collector.lastNonTimelineEvent()
	if !ok {
		t.Fatal("missing run.finished")
	}
	if finished.Payload["backpressure_drop_count"] != 2 {
		t.Fatalf("backpressure_drop_count = %#v, want 2", finished.Payload["backpressure_drop_count"])
	}
	byPhase, ok := finished.Payload["backpressure_drop_count_by_phase"].(map[string]int)
	if !ok {
		t.Fatalf("backpressure_drop_count_by_phase type = %T, want map[string]int", finished.Payload["backpressure_drop_count_by_phase"])
	}
	if byPhase["local"] != 2 {
		t.Fatalf("backpressure_drop_count_by_phase[local] = %d, want 2", byPhase["local"])
	}
	reasons := make([]string, 0)
	for _, ev := range collector.timelineEvents() {
		if reason, _ := ev.Payload["reason"].(string); reason != "" {
			reasons = append(reasons, reason)
		}
	}
	if !containsString(reasons, "backpressure.drop_low_priority") {
		t.Fatalf("expected backpressure.drop_low_priority reason, got %v", reasons)
	}
}

func TestRunBackpressureDropLowPriorityMCPAndSkillAllDroppedFailsFast(t *testing.T) {
	reg := local.NewRegistry()
	_, err := reg.Register(&fakeTool{
		name: "mcp_proxy",
		invoke: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
			time.Sleep(2 * time.Millisecond)
			return types.ToolResult{Content: "ok"}, nil
		},
	})
	if err != nil {
		t.Fatalf("register mcp_proxy: %v", err)
	}
	_, err = reg.Register(&fakeTool{
		name: "skill_router",
		invoke: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
			time.Sleep(2 * time.Millisecond)
			return types.ToolResult{Content: "ok"}, nil
		},
	})
	if err != nil {
		t.Fatalf("register skill_router: %v", err)
	}
	model := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			return types.ModelResponse{
				ToolCalls: []types.ToolCall{
					{CallID: "m-1", Name: "local.mcp_proxy", Args: map[string]any{"q": "cache route"}},
					{CallID: "s-1", Name: "local.skill_router", Args: map[string]any{"q": "cache route"}},
				},
			}, nil
		},
	}
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	cfg := `
concurrency:
  backpressure: drop_low_priority
  local_max_workers: 1
  local_queue_size: 1
  drop_low_priority:
    priority_by_keyword:
      cache: low
    droppable_priorities: [low]
`
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	engine := New(model, WithLocalRegistry(reg), WithRuntimeManager(mgr))
	collector := &eventCollector{}
	res, runErr := engine.Run(context.Background(), types.RunRequest{Input: "x"}, collector)
	if runErr == nil {
		t.Fatal("expected fail-fast error when mcp/skill calls are dropped")
	}
	if res.Error == nil || res.Error.Class != types.ErrTool {
		t.Fatalf("unexpected run error: %#v", res.Error)
	}
	finished, ok := collector.lastNonTimelineEvent()
	if !ok {
		t.Fatal("missing run.finished")
	}
	if finished.Payload["backpressure_drop_count"] != 2 {
		t.Fatalf("backpressure_drop_count = %#v, want 2", finished.Payload["backpressure_drop_count"])
	}
	byPhase, ok := finished.Payload["backpressure_drop_count_by_phase"].(map[string]int)
	if !ok {
		t.Fatalf("backpressure_drop_count_by_phase type = %T, want map[string]int", finished.Payload["backpressure_drop_count_by_phase"])
	}
	if byPhase["mcp"] != 1 || byPhase["skill"] != 1 {
		t.Fatalf("backpressure_drop_count_by_phase = %#v, want mcp=1 skill=1", byPhase)
	}
	var hasMCPReason bool
	var hasSkillReason bool
	for _, ev := range collector.timelineEvents() {
		reason, _ := ev.Payload["reason"].(string)
		phase, _ := ev.Payload["phase"].(string)
		if reason != "backpressure.drop_low_priority" {
			continue
		}
		if phase == string(types.ActionPhaseMCP) {
			hasMCPReason = true
		}
		if phase == string(types.ActionPhaseSkill) {
			hasSkillReason = true
		}
	}
	if !hasMCPReason || !hasSkillReason {
		t.Fatalf("expected mcp/skill backpressure.drop_low_priority timeline reasons, got mcp=%v skill=%v", hasMCPReason, hasSkillReason)
	}
}

func TestRunAndStreamCancelPropagationSemanticsEquivalent(t *testing.T) {
	runModel := &fakeModel{
		generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
			<-ctx.Done()
			return types.ModelResponse{}, ctx.Err()
		},
	}
	streamModel := &fakeModel{
		stream: func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
			<-ctx.Done()
			return ctx.Err()
		},
	}
	runEngine := New(runModel)
	streamEngine := New(streamModel)
	runCollector := &eventCollector{}
	streamCollector := &eventCollector{}

	runCtx, runCancel := context.WithCancel(context.Background())
	streamCtx, streamCancel := context.WithCancel(context.Background())
	runCancel()
	streamCancel()

	runRes, runErr := runEngine.Run(runCtx, types.RunRequest{Input: "x"}, runCollector)
	streamRes, streamErr := streamEngine.Stream(streamCtx, types.RunRequest{Input: "x"}, streamCollector)
	if !errors.Is(runErr, context.Canceled) || !errors.Is(streamErr, context.Canceled) {
		t.Fatalf("expected context canceled for run/stream, got run=%v stream=%v", runErr, streamErr)
	}
	if runRes.Error == nil || streamRes.Error == nil {
		t.Fatalf("missing run/stream errors: run=%#v stream=%#v", runRes.Error, streamRes.Error)
	}
	if runRes.Error.Class != types.ErrPolicyTimeout || streamRes.Error.Class != types.ErrPolicyTimeout {
		t.Fatalf("run/stream class mismatch: run=%#v stream=%#v", runRes.Error, streamRes.Error)
	}

	runFinished, ok := runCollector.lastNonTimelineEvent()
	if !ok {
		t.Fatal("missing run finished event")
	}
	streamFinished, ok := streamCollector.lastNonTimelineEvent()
	if !ok {
		t.Fatal("missing stream finished event")
	}
	if runFinished.Payload["cancel_propagated_count"] != 1 || streamFinished.Payload["cancel_propagated_count"] != 1 {
		t.Fatalf("cancel_propagated_count mismatch run=%#v stream=%#v", runFinished.Payload["cancel_propagated_count"], streamFinished.Payload["cancel_propagated_count"])
	}
}

func TestRunAndStreamMemorySemanticsEquivalentExternalAndBuiltin(t *testing.T) {
	type tc struct {
		name              string
		memoryMode        string
		memoryProvider    string
		memoryProfile     string
		fallbackPolicy    string
		stage2Endpoint    string
		wantFallbackTotal any
	}
	cases := []tc{
		{
			name:              "builtin_filesystem",
			memoryMode:        runtimeconfig.RuntimeMemoryModeBuiltinFilesystem,
			memoryProvider:    runtimeconfig.RuntimeMemoryModeBuiltinFilesystem,
			memoryProfile:     runtimeconfig.RuntimeMemoryProfileGeneric,
			fallbackPolicy:    runtimeconfig.RuntimeMemoryFallbackPolicyFailFast,
			wantFallbackTotal: 0,
		},
		{
			name:              "external_spi_degrade_without_memory",
			memoryMode:        runtimeconfig.RuntimeMemoryModeExternalSPI,
			memoryProvider:    "mem0",
			memoryProfile:     "mem0",
			fallbackPolicy:    runtimeconfig.RuntimeMemoryFallbackPolicyDegradeWithoutMemory,
			stage2Endpoint:    "http://127.0.0.1:1",
			wantFallbackTotal: 1,
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			cfgPath := filepath.Join(t.TempDir(), "runtime-memory.yaml")
			journalPath := filepath.ToSlash(filepath.Join(t.TempDir(), "journal.jsonl"))
			memoryRootDir, mkErr := os.MkdirTemp("", "baymax-a54-memory-store-*")
			if mkErr != nil {
				t.Fatalf("create memory temp dir failed: %v", mkErr)
			}
			t.Cleanup(func() { _ = os.RemoveAll(memoryRootDir) })
			memoryRoot := filepath.ToSlash(memoryRootDir)
			stage2EndpointLine := ""
			if strings.TrimSpace(c.stage2Endpoint) != "" {
				stage2EndpointLine = fmt.Sprintf("\n        endpoint: %q", strings.TrimSpace(c.stage2Endpoint))
			}
			providerLine := "      provider: \"\""
			profileLine := "      profile: generic"
			if c.memoryMode == runtimeconfig.RuntimeMemoryModeExternalSPI {
				providerLine = fmt.Sprintf("      provider: %s", c.memoryProvider)
				profileLine = fmt.Sprintf("      profile: %s", c.memoryProfile)
			}
			cfg := fmt.Sprintf(`
context_assembler:
  enabled: true
  journal_path: %q
  ca2:
    enabled: true
    routing:
      min_input_chars: 1
    stage_policy:
      stage2: fail_fast
    stage2:
      provider: memory
      external:%s
runtime:
  memory:
    mode: %s
    external:
%s
%s
      contract_version: memory.v1
    builtin:
      root_dir: %q
      compaction:
        enabled: true
        min_ops: 8
        max_wal_bytes: 1024
    fallback:
      policy: %s
`, journalPath, stage2EndpointLine, c.memoryMode, providerLine, profileLine, memoryRoot, c.fallbackPolicy)
			if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(cfg)), 0o600); err != nil {
				t.Fatalf("write config failed: %v", err)
			}
			mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX"})
			if err != nil {
				t.Fatalf("new manager failed: %v", err)
			}
			t.Cleanup(func() { _ = mgr.Close() })

			runEngine := New(&fakeModel{
				generate: func(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
					_ = ctx
					_ = req
					return types.ModelResponse{FinalAnswer: "ok"}, nil
				},
			}, WithRuntimeManager(mgr))
			streamEngine := New(&fakeModel{
				stream: func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
					_ = ctx
					_ = req
					return onEvent(types.ModelEvent{
						Type:      types.ModelEventTypeOutputTextDelta,
						TextDelta: "ok",
					})
				},
			}, WithRuntimeManager(mgr))

			runCollector := &eventCollector{}
			streamCollector := &eventCollector{}
			runResult, runErr := runEngine.Run(context.Background(), types.RunRequest{
				RunID:     "run-memory-equivalent-" + c.name,
				SessionID: "session-memory-equivalent",
				Input:     "lookup memory",
				Messages:  []types.Message{{Role: "system", Content: "s"}},
			}, runCollector)
			streamResult, streamErr := streamEngine.Stream(context.Background(), types.RunRequest{
				RunID:     "stream-memory-equivalent-" + c.name,
				SessionID: "session-memory-equivalent",
				Input:     "lookup memory",
				Messages:  []types.Message{{Role: "system", Content: "s"}},
			}, streamCollector)
			if runErr != nil || streamErr != nil {
				t.Fatalf("run/stream should both succeed, runErr=%v streamErr=%v runResult=%#v streamResult=%#v", runErr, streamErr, runResult, streamResult)
			}

			runFinished, ok := runCollector.lastNonTimelineEvent()
			if !ok {
				t.Fatal("missing run.finished in run lane")
			}
			streamFinished, ok := streamCollector.lastNonTimelineEvent()
			if !ok {
				t.Fatal("missing run.finished in stream lane")
			}
			keys := []string{
				"memory_mode",
				"memory_provider",
				"memory_profile",
				"memory_contract_version",
				"memory_query_total",
				"memory_upsert_total",
				"memory_delete_total",
				"memory_error_total",
				"memory_fallback_total",
				"memory_fallback_reason_code",
			}
			for _, key := range keys {
				if runFinished.Payload[key] != streamFinished.Payload[key] {
					t.Fatalf("run/stream memory payload mismatch key=%s run=%#v stream=%#v", key, runFinished.Payload[key], streamFinished.Payload[key])
				}
			}
			runLatency, runLatencyOK := runFinished.Payload["memory_latency_ms_p95"].(int64)
			streamLatency, streamLatencyOK := streamFinished.Payload["memory_latency_ms_p95"].(int64)
			if !runLatencyOK || !streamLatencyOK || runLatency < 0 || streamLatency < 0 {
				t.Fatalf(
					"memory latency payload should be non-negative int64, run=%#v stream=%#v",
					runFinished.Payload["memory_latency_ms_p95"],
					streamFinished.Payload["memory_latency_ms_p95"],
				)
			}
			if runFinished.Payload["memory_mode"] != c.memoryMode {
				t.Fatalf("memory_mode = %#v, want %q", runFinished.Payload["memory_mode"], c.memoryMode)
			}
			if runFinished.Payload["memory_provider"] != c.memoryProvider {
				t.Fatalf("memory_provider = %#v, want %q", runFinished.Payload["memory_provider"], c.memoryProvider)
			}
			if runFinished.Payload["memory_profile"] != c.memoryProfile {
				t.Fatalf("memory_profile = %#v, want %q", runFinished.Payload["memory_profile"], c.memoryProfile)
			}
			if runFinished.Payload["memory_contract_version"] != runtimeconfig.RuntimeMemoryContractVersionV1 {
				t.Fatalf("memory_contract_version = %#v, want %q", runFinished.Payload["memory_contract_version"], runtimeconfig.RuntimeMemoryContractVersionV1)
			}
			if runFinished.Payload["memory_fallback_total"] != c.wantFallbackTotal {
				t.Fatalf("memory_fallback_total = %#v, want %#v", runFinished.Payload["memory_fallback_total"], c.wantFallbackTotal)
			}
		})
	}
}

func assertPhaseDistEqual(
	t *testing.T,
	a, b map[string]runtimediag.TimelinePhaseAggregate,
	phase string,
) {
	t.Helper()
	pa, oka := a[phase]
	pb, okb := b[phase]
	if !oka || !okb {
		t.Fatalf("missing phase aggregate %q: a=%v b=%v", phase, a, b)
	}
	if pa.CountTotal != pb.CountTotal || pa.FailedTotal != pb.FailedTotal || pa.CanceledTotal != pb.CanceledTotal || pa.SkippedTotal != pb.SkippedTotal {
		t.Fatalf("phase distribution mismatch for %q: a=%#v b=%#v", phase, pa, pb)
	}
}

func TestMemoryRunDiagnosticsAccumulatorSnapshot(t *testing.T) {
	acc := memoryRunDiagnosticsAccumulator{}
	acc.observeAssemble(types.ContextAssembleResult{
		Stage: types.AssembleStage{
			Stage2Provider:   runtimeconfig.ContextStage2ProviderMemory,
			Stage2Source:     runtimeconfig.ContextStage2ProviderMemory,
			Stage2ReasonCode: "memory.ok",
			Stage2LatencyMs:  12,
		},
	})
	acc.observeAssemble(types.ContextAssembleResult{
		Stage: types.AssembleStage{
			Stage2Provider:   runtimeconfig.ContextStage2ProviderMemory,
			Stage2Source:     runtimeconfig.ContextStage2ProviderMemory,
			Stage2ReasonCode: "memory.fallback.used",
			Stage2LatencyMs:  25,
		},
	})
	acc.observeAssemble(types.ContextAssembleResult{
		Stage: types.AssembleStage{
			Stage2Provider:   runtimeconfig.ContextStage2ProviderMemory,
			Stage2Source:     runtimeconfig.ContextStage2ProviderMemory,
			Stage2ReasonCode: "memory.provider_unavailable",
			Stage2LatencyMs:  30,
		},
	})

	got := acc.snapshot(memoryRuntimeSnapshot{
		Mode:            runtimeconfig.RuntimeMemoryModeExternalSPI,
		Provider:        "mem0",
		Profile:         "mem0",
		ContractVersion: runtimeconfig.RuntimeMemoryContractVersionV1,
	})
	if !got.Observed {
		t.Fatal("memory diagnostics should be observed")
	}
	if got.Mode != runtimeconfig.RuntimeMemoryModeExternalSPI ||
		got.Provider != "mem0" ||
		got.Profile != "mem0" ||
		got.ContractVersion != runtimeconfig.RuntimeMemoryContractVersionV1 {
		t.Fatalf("memory runtime snapshot mismatch: %#v", got)
	}
	if got.QueryTotal != 3 || got.UpsertTotal != 0 || got.DeleteTotal != 0 {
		t.Fatalf("memory operation totals mismatch: %#v", got)
	}
	if got.ErrorTotal != 1 {
		t.Fatalf("memory_error_total = %d, want 1", got.ErrorTotal)
	}
	if got.FallbackTotal != 1 || got.FallbackReasonCode != "memory.fallback.used" {
		t.Fatalf("memory fallback fields mismatch: %#v", got)
	}
	if got.LatencyMsP95 != 30 {
		t.Fatalf("memory_latency_ms_p95 = %d, want 30", got.LatencyMsP95)
	}
}

func TestRunFinishedPayloadIncludesMemoryAdditiveFields(t *testing.T) {
	payload := runFinishedPayload(types.RunResult{
		RunID:      "run-memory-finished",
		Iterations: 2,
		LatencyMs:  120,
	}, "success", "", runFinishMeta{
		Memory: memoryRunDiagnostics{
			Observed:           true,
			Mode:               runtimeconfig.RuntimeMemoryModeBuiltinFilesystem,
			Provider:           runtimeconfig.RuntimeMemoryModeBuiltinFilesystem,
			Profile:            runtimeconfig.RuntimeMemoryProfileGeneric,
			ContractVersion:    runtimeconfig.RuntimeMemoryContractVersionV1,
			QueryTotal:         2,
			UpsertTotal:        1,
			DeleteTotal:        0,
			ErrorTotal:         0,
			FallbackTotal:      1,
			FallbackReasonCode: "memory.fallback.used",
			LatencyMsP95:       18,
		},
	})
	if payload["memory_mode"] != runtimeconfig.RuntimeMemoryModeBuiltinFilesystem ||
		payload["memory_provider"] != runtimeconfig.RuntimeMemoryModeBuiltinFilesystem ||
		payload["memory_profile"] != runtimeconfig.RuntimeMemoryProfileGeneric ||
		payload["memory_contract_version"] != runtimeconfig.RuntimeMemoryContractVersionV1 ||
		payload["memory_query_total"] != 2 ||
		payload["memory_upsert_total"] != 1 ||
		payload["memory_delete_total"] != 0 ||
		payload["memory_error_total"] != 0 ||
		payload["memory_fallback_total"] != 1 ||
		payload["memory_fallback_reason_code"] != "memory.fallback.used" ||
		payload["memory_latency_ms_p95"] != int64(18) {
		t.Fatalf("memory payload mismatch: %#v", payload)
	}

	withoutObserved := runFinishedPayload(types.RunResult{
		RunID:      "run-memory-empty",
		Iterations: 1,
		LatencyMs:  10,
	}, "success", "", runFinishMeta{})
	if _, ok := withoutObserved["memory_mode"]; ok {
		t.Fatalf("memory fields should be omitted when not observed: %#v", withoutObserved)
	}
}

func TestRunFinishedPayloadIncludesReactAdditiveFields(t *testing.T) {
	payload := runFinishedPayload(types.RunResult{
		RunID:      "run-react-finished",
		Iterations: 3,
		LatencyMs:  88,
	}, "failed", string(types.ErrIterationLimit), runFinishMeta{
		ReactEnabled:        true,
		ReactIterationTotal: 3,
		ReactToolCallTotal:  4,
		ReactToolBudgetHit:  1,
		ReactIterBudgetHit:  0,
		ReactTermination:    runtimeconfig.RuntimeReactTerminationToolCallLimitExceeded,
		ReactStreamDispatch: true,
	})
	if payload["react_enabled"] != true ||
		payload["react_iteration_total"] != 3 ||
		payload["react_tool_call_total"] != 4 ||
		payload["react_tool_call_budget_hit_total"] != 1 ||
		payload["react_iteration_budget_hit_total"] != 0 ||
		payload["react_termination_reason"] != runtimeconfig.RuntimeReactTerminationToolCallLimitExceeded ||
		payload["react_stream_dispatch_enabled"] != true {
		t.Fatalf("react payload mismatch: %#v", payload)
	}
}

func TestResolveReactTerminationReasonDeterministicMapping(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name                 string
		terminal             *types.ClassifiedError
		runErr               error
		toolBudgetHitTotal   int
		iterationBudgetTotal int
		want                 string
	}{
		{
			name:               "tool budget hit has top priority",
			terminal:           classified(types.ErrModel, "provider failed", false),
			toolBudgetHitTotal: 1,
			want:               runtimeconfig.RuntimeReactTerminationToolCallLimitExceeded,
		},
		{
			name:                 "iteration budget hit when tool budget not hit",
			terminal:             classified(types.ErrModel, "provider failed", false),
			iterationBudgetTotal: 1,
			want:                 runtimeconfig.RuntimeReactTerminationMaxIterationsExceeded,
		},
		{
			name:     "context canceled by run error",
			terminal: classified(types.ErrModel, "provider failed", false),
			runErr:   context.Canceled,
			want:     runtimeconfig.RuntimeReactTerminationContextCanceled,
		},
		{
			name:     "tool class maps to tool dispatch failed",
			terminal: classified(types.ErrTool, "tool failed", false),
			want:     runtimeconfig.RuntimeReactTerminationToolDispatchFailed,
		},
		{
			name:     "policy timeout maps to context canceled",
			terminal: classified(types.ErrPolicyTimeout, "timed out", true),
			want:     runtimeconfig.RuntimeReactTerminationContextCanceled,
		},
		{
			name:     "model class maps to provider error",
			terminal: classified(types.ErrModel, "provider failed", false),
			want:     runtimeconfig.RuntimeReactTerminationProviderError,
		},
		{
			name: "nil terminal defaults to provider error",
			want: runtimeconfig.RuntimeReactTerminationProviderError,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := resolveReactTerminationReason(tc.terminal, tc.runErr, tc.toolBudgetHitTotal, tc.iterationBudgetTotal)
			if got != tc.want {
				t.Fatalf("termination=%q, want %q", got, tc.want)
			}
			gotAgain := resolveReactTerminationReason(tc.terminal, tc.runErr, tc.toolBudgetHitTotal, tc.iterationBudgetTotal)
			if gotAgain != got {
				t.Fatalf("termination should be deterministic, first=%q second=%q", got, gotAgain)
			}
		})
	}
}

func TestStreamReactDuplicateToolCallEventsAreIdempotent(t *testing.T) {
	reg := local.NewRegistry()
	invokeCount := 0
	lastArgs := map[string]any{}
	_, err := reg.Register(&fakeTool{
		name: "echo",
		invoke: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
			invokeCount++
			lastArgs = args
			return types.ToolResult{Content: "ok"}, nil
		},
	})
	if err != nil {
		t.Fatalf("register tool: %v", err)
	}

	streamStep := 0
	model := &fakeModel{
		stream: func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
			streamStep++
			switch streamStep {
			case 1:
				if err := onEvent(types.ModelEvent{
					Type: types.ModelEventTypeToolCall,
					ToolCall: &types.ToolCall{
						CallID: "call-1",
						Name:   "local.echo",
						Args:   map[string]any{"v": "first"},
					},
				}); err != nil {
					return err
				}
				return onEvent(types.ModelEvent{
					Type: types.ModelEventTypeToolCall,
					ToolCall: &types.ToolCall{
						CallID: "call-1",
						Name:   "local.echo",
						Args:   map[string]any{"v": "final"},
					},
				})
			case 2:
				return onEvent(types.ModelEvent{
					Type:      types.ModelEventTypeOutputTextDelta,
					TextDelta: "done",
				})
			default:
				return errors.New("unexpected extra stream step")
			}
		},
	}

	engine := New(model, WithLocalRegistry(reg))
	collector := &eventCollector{}
	res, runErr := engine.Stream(context.Background(), types.RunRequest{
		RunID: "run-react-stream-idempotent",
		Input: "x",
	}, collector)
	if runErr != nil {
		t.Fatalf("Stream error: %v", runErr)
	}
	if res.Error != nil {
		t.Fatalf("unexpected classified error: %#v", res.Error)
	}
	if res.FinalAnswer != "done" {
		t.Fatalf("final answer=%q, want done", res.FinalAnswer)
	}
	if invokeCount != 1 {
		t.Fatalf("tool invoke count=%d, want 1", invokeCount)
	}
	if got, _ := lastArgs["v"].(string); got != "final" {
		t.Fatalf("tool args should use latest duplicate event payload, got=%#v", lastArgs)
	}
	if len(res.ToolCalls) != 1 || res.ToolCalls[0].CallID != "call-1" {
		t.Fatalf("tool summary mismatch: %#v", res.ToolCalls)
	}
	finished, ok := collector.lastNonTimelineEvent()
	if !ok || finished.Type != "run.finished" {
		t.Fatalf("missing run.finished event: %#v", finished)
	}
	if finished.Payload["react_termination_reason"] != runtimeconfig.RuntimeReactTerminationCompleted {
		t.Fatalf(
			"react_termination_reason=%#v, want %q",
			finished.Payload["react_termination_reason"],
			runtimeconfig.RuntimeReactTerminationCompleted,
		)
	}
	if finished.Payload["react_tool_call_total"] != 1 {
		t.Fatalf("react_tool_call_total=%#v, want 1", finished.Payload["react_tool_call_total"])
	}
}

func TestStreamReactCancellationUsesCanonicalTerminationReason(t *testing.T) {
	model := &fakeModel{
		stream: func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
			<-ctx.Done()
			return ctx.Err()
		},
	}
	engine := New(model)
	collector := &eventCollector{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	res, runErr := engine.Stream(ctx, types.RunRequest{
		RunID: "run-react-cancel",
		Input: "cancel",
	}, collector)
	if !errors.Is(runErr, context.Canceled) {
		t.Fatalf("runErr=%v, want context canceled", runErr)
	}
	if res.Error == nil || res.Error.Class != types.ErrPolicyTimeout {
		t.Fatalf("result error=%#v, want ErrPolicyTimeout", res.Error)
	}
	finished, ok := collector.lastNonTimelineEvent()
	if !ok || finished.Type != "run.finished" {
		t.Fatalf("missing run.finished event: %#v", finished)
	}
	if finished.Payload["react_termination_reason"] != runtimeconfig.RuntimeReactTerminationContextCanceled {
		t.Fatalf(
			"react_termination_reason=%#v, want %q",
			finished.Payload["react_termination_reason"],
			runtimeconfig.RuntimeReactTerminationContextCanceled,
		)
	}
}

func TestStreamReactToolDispatchFailureUsesCanonicalTerminationReason(t *testing.T) {
	reg := local.NewRegistry()
	_, err := reg.Register(&fakeTool{
		name: "explode",
		invoke: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
			return types.ToolResult{}, errors.New("tool boom")
		},
	})
	if err != nil {
		t.Fatalf("register tool: %v", err)
	}

	streamStep := 0
	model := &fakeModel{
		stream: func(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
			streamStep++
			switch streamStep {
			case 1:
				return onEvent(types.ModelEvent{
					Type: types.ModelEventTypeToolCall,
					ToolCall: &types.ToolCall{
						CallID: "call-explode",
						Name:   "local.explode",
					},
				})
			default:
				return onEvent(types.ModelEvent{
					Type:      types.ModelEventTypeOutputTextDelta,
					TextDelta: "should-not-reach",
				})
			}
		},
	}

	engine := New(model, WithLocalRegistry(reg))
	collector := &eventCollector{}
	res, runErr := engine.Stream(context.Background(), types.RunRequest{
		RunID: "run-react-tool-fail",
		Input: "x",
	}, collector)
	if runErr == nil {
		t.Fatal("expected stream error")
	}
	if res.Error == nil || res.Error.Class != types.ErrTool {
		t.Fatalf("result error=%#v, want ErrTool", res.Error)
	}
	finished, ok := collector.lastNonTimelineEvent()
	if !ok || finished.Type != "run.finished" {
		t.Fatalf("missing run.finished event: %#v", finished)
	}
	if finished.Payload["react_termination_reason"] != runtimeconfig.RuntimeReactTerminationToolDispatchFailed {
		t.Fatalf(
			"react_termination_reason=%#v, want %q",
			finished.Payload["react_termination_reason"],
			runtimeconfig.RuntimeReactTerminationToolDispatchFailed,
		)
	}
}
