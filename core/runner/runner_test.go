package runner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

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
