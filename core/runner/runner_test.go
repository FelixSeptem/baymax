package runner

import (
	"context"
	"errors"
	"fmt"
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
	if finished.Payload["alert_dispatch_failure_reason"] != "alert.callback_error" {
		t.Fatalf("alert_dispatch_failure_reason = %#v, want alert.callback_error", finished.Payload["alert_dispatch_failure_reason"])
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
	for _, key := range []string{"policy_kind", "decision", "reason_code", "severity", "alert_dispatch_status"} {
		if runFinished.Payload[key] != streamFinished.Payload[key] {
			t.Fatalf("%s mismatch run=%#v stream=%#v", key, runFinished.Payload[key], streamFinished.Payload[key])
		}
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
