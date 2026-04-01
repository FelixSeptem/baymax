package local

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
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

type fakeTool struct {
	name   string
	schema map[string]any
	invoke func(context.Context, map[string]any) (types.ToolResult, error)
	write  bool
}

func (t *fakeTool) Name() string               { return t.name }
func (t *fakeTool) Description() string        { return "fake" }
func (t *fakeTool) JSONSchema() map[string]any { return t.schema }
func (t *fakeTool) IsWrite() bool              { return t.write }
func (t *fakeTool) Invoke(ctx context.Context, args map[string]any) (types.ToolResult, error) {
	if t.invoke != nil {
		return t.invoke(ctx, args)
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

type fakeSandboxAdapterTool struct {
	*fakeTool
	build  func(ctx context.Context, args map[string]any) (types.SandboxExecSpec, error)
	handle func(ctx context.Context, result types.SandboxExecResult) (types.ToolResult, error)
}

func (t *fakeSandboxAdapterTool) BuildSandboxExecSpec(ctx context.Context, args map[string]any) (types.SandboxExecSpec, error) {
	if t != nil && t.build != nil {
		return t.build(ctx, args)
	}
	return types.SandboxExecSpec{}, nil
}

func (t *fakeSandboxAdapterTool) HandleSandboxExecResult(ctx context.Context, result types.SandboxExecResult) (types.ToolResult, error) {
	if t != nil && t.handle != nil {
		return t.handle(ctx, result)
	}
	return types.ToolResult{}, nil
}

func TestRegistryNamespacesLocalTools(t *testing.T) {
	reg := NewRegistry()
	name, err := reg.Register(&fakeTool{name: "search"})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if name != "local.search" {
		t.Fatalf("name = %q, want local.search", name)
	}
	if _, ok := reg.Get("search"); !ok {
		t.Fatal("Get(search) failed")
	}
	if _, ok := reg.Get("local.search"); !ok {
		t.Fatal("Get(local.search) failed")
	}
}

func TestDispatcherValidationErrorStructured(t *testing.T) {
	reg := NewRegistry()
	_, _ = reg.Register(&fakeTool{
		name: "search",
		schema: map[string]any{
			"type":     "object",
			"required": []any{"q"},
			"properties": map[string]any{
				"q": map[string]any{"type": "string"},
			},
		},
		invoke: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
			return types.ToolResult{Content: "should-not-run"}, nil
		},
	})
	dispatcher := NewDispatcher(reg)

	outcomes, err := dispatcher.Dispatch(context.Background(), []types.ToolCall{{CallID: "c1", Name: "local.search", Args: map[string]any{}}}, DispatchConfig{MaxCalls: 1, Concurrency: 1, FailFast: false})
	if err != nil {
		t.Fatalf("Dispatch failed: %v", err)
	}
	if len(outcomes) != 1 || outcomes[0].Result.Error == nil {
		t.Fatalf("unexpected outcomes: %#v", outcomes)
	}
	if outcomes[0].Result.Error.Class != types.ErrTool {
		t.Fatalf("error class = %q, want %q", outcomes[0].Result.Error.Class, types.ErrTool)
	}
	if outcomes[0].Result.Error.Details["validation"] == nil {
		t.Fatalf("missing validation details: %#v", outcomes[0].Result.Error)
	}
}

func TestDispatcherWriteToolsAreSerializedInOrder(t *testing.T) {
	reg := NewRegistry()
	var mu sync.Mutex
	order := make([]string, 0, 2)

	appendOrder := func(label string) func(context.Context, map[string]any) (types.ToolResult, error) {
		return func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
			time.Sleep(20 * time.Millisecond)
			mu.Lock()
			order = append(order, label)
			mu.Unlock()
			return types.ToolResult{Content: label}, nil
		}
	}

	_, _ = reg.Register(&fakeTool{name: "write_a", write: true, invoke: appendOrder("a")})
	_, _ = reg.Register(&fakeTool{name: "write_b", write: true, invoke: appendOrder("b")})
	dispatcher := NewDispatcher(reg)

	outcomes, err := dispatcher.Dispatch(context.Background(), []types.ToolCall{
		{CallID: "c1", Name: "local.write_a"},
		{CallID: "c2", Name: "local.write_b"},
	}, DispatchConfig{MaxCalls: 2, Concurrency: 2, FailFast: true})
	if err != nil {
		t.Fatalf("Dispatch failed: %v", err)
	}
	if len(outcomes) != 2 {
		t.Fatalf("outcomes len = %d, want 2", len(outcomes))
	}
	if order[0] != "a" || order[1] != "b" {
		t.Fatalf("order = %#v, want [a b]", order)
	}
}

func TestDispatcherFailFastBehavior(t *testing.T) {
	reg := NewRegistry()
	_, _ = reg.Register(&fakeTool{
		name: "boom",
		invoke: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
			return types.ToolResult{}, errors.New("boom")
		},
	})
	dispatcher := NewDispatcher(reg)
	_, err := dispatcher.Dispatch(context.Background(), []types.ToolCall{{CallID: "c1", Name: "local.boom"}}, DispatchConfig{MaxCalls: 1, Concurrency: 1, FailFast: true})
	if err == nil {
		t.Fatal("expected fail-fast error, got nil")
	}
}

func TestDispatcherRecordsDiagnosticsWithRuntimeManager(t *testing.T) {
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

	reg := NewRegistry()
	_, _ = reg.Register(&fakeTool{name: "search", invoke: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
		return types.ToolResult{Content: "ok"}, nil
	}})
	dispatcher := NewDispatcherWithRuntimeManager(reg, mgr)
	_, err = dispatcher.Dispatch(context.Background(), []types.ToolCall{{CallID: "c1", Name: "local.search"}}, DispatchConfig{MaxCalls: 1, Concurrency: 1, FailFast: true})
	if err != nil {
		t.Fatalf("Dispatch failed: %v", err)
	}
	items := mgr.RecentCalls(1)
	if len(items) != 1 {
		t.Fatalf("diagnostic calls len = %d, want 1", len(items))
	}
	if items[0].Component != "tool" || items[0].Name != "local.search" {
		t.Fatalf("unexpected diagnostics call: %#v", items[0])
	}
}

func TestDispatcherDropLowPriorityDropsConfiguredLowCalls(t *testing.T) {
	reg := NewRegistry()
	_, _ = reg.Register(&fakeTool{
		name: "slow",
		invoke: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
			time.Sleep(8 * time.Millisecond)
			return types.ToolResult{Content: "ok"}, nil
		},
	})
	dispatcher := NewDispatcher(reg)
	calls := []types.ToolCall{
		{CallID: "c1", Name: "local.slow", Args: map[string]any{"q": "normal task"}},
		{CallID: "c2", Name: "local.slow", Args: map[string]any{"q": "cache warmup low"}},
	}
	outcomes, err := dispatcher.Dispatch(context.Background(), calls, DispatchConfig{
		MaxCalls:     2,
		Concurrency:  1,
		QueueSize:    1,
		Backpressure: types.BackpressureDropLowPriority,
		DropLowPriority: DropLowPriorityPolicy{
			PriorityByKeyword:   map[string]string{"cache": runtimeconfig.DropPriorityLow},
			DroppablePriorities: []string{runtimeconfig.DropPriorityLow},
		},
	})
	if err != nil {
		t.Fatalf("Dispatch failed: %v", err)
	}
	if outcomes[1].Result.Error == nil {
		t.Fatalf("expected second call dropped, got %#v", outcomes[1])
	}
	if outcomes[1].Result.Error.Details["drop_reason"] != "low_priority_dropped" {
		t.Fatalf("unexpected drop details: %#v", outcomes[1].Result.Error.Details)
	}
	if outcomes[1].Result.Error.Details["dispatch_phase"] != string(types.ActionPhaseTool) {
		t.Fatalf("dispatch_phase = %#v, want %q", outcomes[1].Result.Error.Details["dispatch_phase"], types.ActionPhaseTool)
	}
}

func TestDispatcherDropLowPriorityKeepsNonDroppableCalls(t *testing.T) {
	reg := NewRegistry()
	_, _ = reg.Register(&fakeTool{
		name: "slow",
		invoke: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
			time.Sleep(5 * time.Millisecond)
			return types.ToolResult{Content: "ok"}, nil
		},
	})
	dispatcher := NewDispatcher(reg)
	calls := []types.ToolCall{
		{CallID: "c1", Name: "local.slow", Args: map[string]any{"q": "high"}},
		{CallID: "c2", Name: "local.slow", Args: map[string]any{"q": "high"}},
	}
	outcomes, err := dispatcher.Dispatch(context.Background(), calls, DispatchConfig{
		MaxCalls:     2,
		Concurrency:  1,
		QueueSize:    1,
		Backpressure: types.BackpressureDropLowPriority,
		DropLowPriority: DropLowPriorityPolicy{
			PriorityByKeyword:   map[string]string{"high": runtimeconfig.DropPriorityHigh},
			DroppablePriorities: []string{runtimeconfig.DropPriorityLow},
		},
	})
	if err != nil {
		t.Fatalf("Dispatch failed: %v", err)
	}
	if outcomes[0].Result.Error != nil || outcomes[1].Result.Error != nil {
		t.Fatalf("expected non-droppable calls to execute, got %#v", outcomes)
	}
}

func TestDispatcherDropLowPriorityMarksMCPAndSkillPhase(t *testing.T) {
	reg := NewRegistry()
	_, _ = reg.Register(&fakeTool{
		name: "mcp_proxy",
		invoke: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
			time.Sleep(5 * time.Millisecond)
			return types.ToolResult{Content: "ok"}, nil
		},
	})
	_, _ = reg.Register(&fakeTool{
		name: "skill_router",
		invoke: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
			time.Sleep(5 * time.Millisecond)
			return types.ToolResult{Content: "ok"}, nil
		},
	})
	dispatcher := NewDispatcher(reg)
	calls := []types.ToolCall{
		{CallID: "m1", Name: "local.mcp_proxy", Args: map[string]any{"q": "cache"}},
		{CallID: "s1", Name: "local.skill_router", Args: map[string]any{"q": "cache"}},
	}
	outcomes, err := dispatcher.Dispatch(context.Background(), calls, DispatchConfig{
		MaxCalls:     2,
		Concurrency:  1,
		QueueSize:    1,
		Backpressure: types.BackpressureDropLowPriority,
		DropLowPriority: DropLowPriorityPolicy{
			PriorityByKeyword:   map[string]string{"cache": runtimeconfig.DropPriorityLow},
			DroppablePriorities: []string{runtimeconfig.DropPriorityLow},
		},
	})
	if err != nil {
		t.Fatalf("Dispatch failed: %v", err)
	}
	if outcomes[0].Result.Error == nil || outcomes[1].Result.Error == nil {
		t.Fatalf("expected both calls dropped, got %#v", outcomes)
	}
	if outcomes[0].Result.Error.Details["dispatch_phase"] != string(types.ActionPhaseMCP) {
		t.Fatalf("first dispatch_phase = %#v, want %q", outcomes[0].Result.Error.Details["dispatch_phase"], types.ActionPhaseMCP)
	}
	if outcomes[1].Result.Error.Details["dispatch_phase"] != string(types.ActionPhaseSkill) {
		t.Fatalf("second dispatch_phase = %#v, want %q", outcomes[1].Result.Error.Details["dispatch_phase"], types.ActionPhaseSkill)
	}
}

func TestDispatcherSandboxPolicyDenyFailFast(t *testing.T) {
	mgr := newSandboxTestManager(t, `
security:
  sandbox:
    enabled: true
    mode: enforce
    required: true
    policy:
      default_action: deny
      profile: default
      fallback_action: deny
`)
	defer func() { _ = mgr.Close() }()

	reg := NewRegistry()
	_, _ = reg.Register(&fakeTool{name: "search", invoke: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
		return types.ToolResult{Content: "host"}, nil
	}})
	dispatcher := NewDispatcherWithRuntimeManager(reg, mgr)

	outcomes, err := dispatcher.Dispatch(context.Background(), []types.ToolCall{
		{CallID: "c1", Name: "local.search"},
	}, DispatchConfig{MaxCalls: 1, Concurrency: 1, FailFast: true})
	if err == nil {
		t.Fatal("expected sandbox policy deny error, got nil")
	}
	if len(outcomes) != 1 || outcomes[0].Result.Error == nil {
		t.Fatalf("unexpected outcomes: %#v", outcomes)
	}
	if outcomes[0].Result.Error.Class != types.ErrSecurity {
		t.Fatalf("error class = %q, want %q", outcomes[0].Result.Error.Class, types.ErrSecurity)
	}
	if outcomes[0].Result.Error.Details["reason_code"] != sandboxReasonPolicyDeny {
		t.Fatalf("reason_code = %#v, want %q", outcomes[0].Result.Error.Details["reason_code"], sandboxReasonPolicyDeny)
	}
}

func TestDispatcherSandboxEgressDenyFailFast(t *testing.T) {
	mgr := newSandboxTestManager(t, `
security:
  sandbox:
    enabled: true
    mode: enforce
    required: true
    policy:
      default_action: host
      profile: default
      fallback_action: deny
    egress:
      enabled: true
      default_action: deny
      on_violation: deny
`)
	defer func() { _ = mgr.Close() }()

	reg := NewRegistry()
	_, _ = reg.Register(&fakeTool{name: "search", invoke: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
		return types.ToolResult{Content: "host"}, nil
	}})
	dispatcher := NewDispatcherWithRuntimeManager(reg, mgr)

	outcomes, err := dispatcher.Dispatch(context.Background(), []types.ToolCall{
		{CallID: "c1", Name: "local.search", Args: map[string]any{"url": "https://blocked.example/v1"}},
	}, DispatchConfig{MaxCalls: 1, Concurrency: 1, FailFast: true})
	if err == nil {
		t.Fatal("expected sandbox egress deny error, got nil")
	}
	if len(outcomes) != 1 || outcomes[0].Result.Error == nil {
		t.Fatalf("unexpected outcomes: %#v", outcomes)
	}
	if outcomes[0].Result.Error.Class != types.ErrSecurity {
		t.Fatalf("error class = %q, want %q", outcomes[0].Result.Error.Class, types.ErrSecurity)
	}
	if outcomes[0].Result.Error.Details["reason_code"] != sandboxReasonEgressDeny {
		t.Fatalf("reason_code = %#v, want %q", outcomes[0].Result.Error.Details["reason_code"], sandboxReasonEgressDeny)
	}
	if outcomes[0].Result.Error.Details["sandbox_egress_policy_source"] != "default_action" {
		t.Fatalf(
			"sandbox_egress_policy_source = %#v, want default_action",
			outcomes[0].Result.Error.Details["sandbox_egress_policy_source"],
		)
	}
}

func TestDispatcherSandboxEgressAllowByAllowlist(t *testing.T) {
	mgr := newSandboxTestManager(t, `
security:
  sandbox:
    enabled: true
    mode: enforce
    required: false
    policy:
      default_action: host
      profile: default
      fallback_action: deny
    egress:
      enabled: true
      default_action: deny
      allowlist:
        - api.example.com
      on_violation: deny
`)
	defer func() { _ = mgr.Close() }()

	reg := NewRegistry()
	_, _ = reg.Register(&fakeTool{name: "search", invoke: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
		return types.ToolResult{Content: "host-ok"}, nil
	}})
	dispatcher := NewDispatcherWithRuntimeManager(reg, mgr)

	outcomes, err := dispatcher.Dispatch(context.Background(), []types.ToolCall{
		{CallID: "c1", Name: "local.search", Args: map[string]any{"url": "https://api.example.com/v1"}},
	}, DispatchConfig{MaxCalls: 1, Concurrency: 1, FailFast: true})
	if err != nil {
		t.Fatalf("dispatch should allow allowlisted target, err=%v outcomes=%#v", err, outcomes)
	}
	if outcomes[0].Result.Error != nil || outcomes[0].Result.Content != "host-ok" {
		t.Fatalf("unexpected allowlist result: %#v", outcomes[0].Result)
	}
}

func TestDispatcherSandboxEgressAllowAndRecordOnViolation(t *testing.T) {
	mgr := newSandboxTestManager(t, `
security:
  sandbox:
    enabled: true
    mode: enforce
    required: false
    policy:
      default_action: host
      profile: default
      fallback_action: deny
    egress:
      enabled: true
      default_action: deny
      on_violation: allow_and_record
`)
	defer func() { _ = mgr.Close() }()

	reg := NewRegistry()
	_, _ = reg.Register(&fakeTool{name: "search", invoke: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
		return types.ToolResult{Content: "host-ok"}, nil
	}})
	dispatcher := NewDispatcherWithRuntimeManager(reg, mgr)

	outcomes, err := dispatcher.Dispatch(context.Background(), []types.ToolCall{
		{CallID: "c1", Name: "local.search", Args: map[string]any{"url": "https://blocked.example/v1"}},
	}, DispatchConfig{MaxCalls: 1, Concurrency: 1, FailFast: true})
	if err != nil {
		t.Fatalf("dispatch should allow_and_record violation, err=%v outcomes=%#v", err, outcomes)
	}
	if outcomes[0].Result.Error != nil || outcomes[0].Result.Content != "host-ok" {
		t.Fatalf("unexpected allow_and_record result: %#v", outcomes[0].Result)
	}
	if outcomes[0].Result.Structured["sandbox_reason_code"] != sandboxReasonEgressAllowAndRecord {
		t.Fatalf("sandbox_reason_code=%#v, want %q", outcomes[0].Result.Structured["sandbox_reason_code"], sandboxReasonEgressAllowAndRecord)
	}
	if outcomes[0].Result.Structured["sandbox_egress_action"] != runtimeconfig.SecuritySandboxEgressActionAllowAndRecord {
		t.Fatalf(
			"sandbox_egress_action=%#v, want %q",
			outcomes[0].Result.Structured["sandbox_egress_action"],
			runtimeconfig.SecuritySandboxEgressActionAllowAndRecord,
		)
	}
	if outcomes[0].Result.Structured["sandbox_egress_policy_source"] != "on_violation" {
		t.Fatalf(
			"sandbox_egress_policy_source=%#v, want on_violation",
			outcomes[0].Result.Structured["sandbox_egress_policy_source"],
		)
	}
}

func TestDispatcherSandboxEgressByToolOverridePrecedence(t *testing.T) {
	mgr := newSandboxTestManager(t, `
security:
  sandbox:
    enabled: true
    mode: enforce
    required: true
    policy:
      default_action: host
      profile: default
      fallback_action: deny
    egress:
      enabled: true
      default_action: deny
      by_tool:
        local+search: deny
      allowlist:
        - api.example.com
      on_violation: deny
`)
	defer func() { _ = mgr.Close() }()

	reg := NewRegistry()
	_, _ = reg.Register(&fakeTool{name: "search", invoke: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
		return types.ToolResult{Content: "host"}, nil
	}})
	dispatcher := NewDispatcherWithRuntimeManager(reg, mgr)

	outcomes, err := dispatcher.Dispatch(context.Background(), []types.ToolCall{
		{CallID: "c1", Name: "local.search", Args: map[string]any{"url": "https://api.example.com/v1"}},
	}, DispatchConfig{MaxCalls: 1, Concurrency: 1, FailFast: true})
	if err == nil {
		t.Fatal("expected by_tool egress deny error, got nil")
	}
	if len(outcomes) != 1 || outcomes[0].Result.Error == nil {
		t.Fatalf("unexpected outcomes: %#v", outcomes)
	}
	if outcomes[0].Result.Error.Details["reason_code"] != sandboxReasonEgressDeny {
		t.Fatalf("reason_code=%#v, want %q", outcomes[0].Result.Error.Details["reason_code"], sandboxReasonEgressDeny)
	}
	if outcomes[0].Result.Error.Details["sandbox_egress_policy_source"] != "by_tool" {
		t.Fatalf(
			"sandbox_egress_policy_source=%#v, want by_tool",
			outcomes[0].Result.Error.Details["sandbox_egress_policy_source"],
		)
	}
}

func TestDispatcherSandboxToolNotAdaptedDenyByDefaultForHighRisk(t *testing.T) {
	mgr := newSandboxTestManager(t, `
security:
  sandbox:
    enabled: true
    mode: enforce
    required: false
    policy:
      default_action: host
      by_tool:
        local+shell: sandbox
      profile: default
      fallback_action: allow_and_record
`)
	defer func() { _ = mgr.Close() }()

	reg := NewRegistry()
	_, _ = reg.Register(&fakeTool{name: "shell", invoke: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
		return types.ToolResult{Content: "host"}, nil
	}})
	dispatcher := NewDispatcherWithRuntimeManager(reg, mgr)
	outcomes, err := dispatcher.Dispatch(context.Background(), []types.ToolCall{
		{CallID: "c1", Name: "local.shell"},
	}, DispatchConfig{MaxCalls: 1, Concurrency: 1, FailFast: true})
	if err == nil {
		t.Fatal("expected sandbox adapter-missing deny error, got nil")
	}
	if outcomes[0].Result.Error == nil {
		t.Fatalf("expected error outcome, got %#v", outcomes[0])
	}
	if outcomes[0].Result.Error.Class != types.ErrSecurity {
		t.Fatalf("error class=%q, want %q", outcomes[0].Result.Error.Class, types.ErrSecurity)
	}
	if outcomes[0].Result.Error.Details["reason_code"] != sandboxReasonToolNotAdapted {
		t.Fatalf("reason_code=%#v, want %q", outcomes[0].Result.Error.Details["reason_code"], sandboxReasonToolNotAdapted)
	}
}

func TestDispatcherSandboxToolNotAdaptedAllowFallbackWithExplicitOverride(t *testing.T) {
	mgr := newSandboxTestManager(t, `
security:
  sandbox:
    enabled: true
    mode: enforce
    required: false
    policy:
      default_action: host
      by_tool:
        local+shell: sandbox
      profile: default
      fallback_action: allow_and_record
      fallback_action_by_tool:
        local+shell: allow_and_record
`)
	defer func() { _ = mgr.Close() }()

	reg := NewRegistry()
	_, _ = reg.Register(&fakeTool{name: "shell", invoke: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
		return types.ToolResult{Content: "host-ok"}, nil
	}})
	dispatcher := NewDispatcherWithRuntimeManager(reg, mgr)
	outcomes, err := dispatcher.Dispatch(context.Background(), []types.ToolCall{
		{CallID: "c1", Name: "local.shell"},
	}, DispatchConfig{MaxCalls: 1, Concurrency: 1, FailFast: true})
	if err != nil {
		t.Fatalf("dispatch should fallback host and succeed, err=%v outcomes=%#v", err, outcomes)
	}
	if outcomes[0].Result.Error != nil || outcomes[0].Result.Content != "host-ok" {
		t.Fatalf("unexpected fallback result: %#v", outcomes[0].Result)
	}
	if outcomes[0].Result.Structured["sandbox_fallback_reason"] != sandboxReasonFallbackAllowRecord {
		t.Fatalf("sandbox_fallback_reason=%#v, want %q", outcomes[0].Result.Structured["sandbox_fallback_reason"], sandboxReasonFallbackAllowRecord)
	}
}

func TestDispatcherSandboxAdapterExecutesThroughSandboxExecutor(t *testing.T) {
	mgr := newSandboxTestManager(t, `
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
`)
	defer func() { _ = mgr.Close() }()
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
			if spec.NamespaceTool != "local+exec" {
				t.Fatalf("namespace_tool=%q, want local+exec", spec.NamespaceTool)
			}
			return types.SandboxExecResult{ExitCode: 0, Stdout: "sandbox-ok"}, nil
		},
	})

	reg := NewRegistry()
	adapterTool := &fakeSandboxAdapterTool{
		fakeTool: &fakeTool{name: "exec", invoke: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
			return types.ToolResult{}, errors.New("host path should not be used")
		}},
		build: func(ctx context.Context, args map[string]any) (types.SandboxExecSpec, error) {
			_ = ctx
			return types.SandboxExecSpec{
				Command: "cmd.exe",
				Args:    []string{"/c", "echo sandbox"},
			}, nil
		},
		handle: func(ctx context.Context, result types.SandboxExecResult) (types.ToolResult, error) {
			_ = ctx
			return types.ToolResult{Content: strings.TrimSpace(result.Stdout)}, nil
		},
	}
	_, _ = reg.Register(adapterTool)
	dispatcher := NewDispatcherWithRuntimeManager(reg, mgr)
	outcomes, err := dispatcher.Dispatch(context.Background(), []types.ToolCall{
		{CallID: "c1", Name: "local.exec"},
	}, DispatchConfig{MaxCalls: 1, Concurrency: 1, FailFast: true})
	if err != nil {
		t.Fatalf("dispatch via sandbox adapter failed: %v", err)
	}
	if outcomes[0].Result.Error != nil || outcomes[0].Result.Content != "sandbox-ok" {
		t.Fatalf("unexpected sandbox result: %#v", outcomes[0].Result)
	}
	if outcomes[0].Result.Structured["sandbox_decision"] != runtimeconfig.SecuritySandboxActionSandbox {
		t.Fatalf("sandbox_decision=%#v, want %q", outcomes[0].Result.Structured["sandbox_decision"], runtimeconfig.SecuritySandboxActionSandbox)
	}
	if outcomes[0].Result.Structured["sandbox_backend"] != runtimeconfig.SecuritySandboxBackendWindowsJob {
		t.Fatalf("sandbox_backend=%#v, want %q", outcomes[0].Result.Structured["sandbox_backend"], runtimeconfig.SecuritySandboxBackendWindowsJob)
	}
	if outcomes[0].Result.Structured["sandbox_exit_code"] != 0 {
		t.Fatalf("sandbox_exit_code=%#v, want 0", outcomes[0].Result.Structured["sandbox_exit_code"])
	}
}

func TestDispatcherSandboxLaunchFailureAllowFallbackToHost(t *testing.T) {
	mgr := newSandboxTestManager(t, `
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
`)
	defer func() { _ = mgr.Close() }()
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
			return types.SandboxExecResult{}, errors.New("sandbox launch failed")
		},
	})

	reg := NewRegistry()
	adapterTool := &fakeSandboxAdapterTool{
		fakeTool: &fakeTool{name: "exec", invoke: func(ctx context.Context, args map[string]any) (types.ToolResult, error) {
			return types.ToolResult{Content: "host-ok"}, nil
		}},
		build: func(ctx context.Context, args map[string]any) (types.SandboxExecSpec, error) {
			_ = ctx
			return types.SandboxExecSpec{Command: "cmd.exe", Args: []string{"/c", "echo fallback"}}, nil
		},
		handle: func(ctx context.Context, result types.SandboxExecResult) (types.ToolResult, error) {
			_ = ctx
			return types.ToolResult{Content: strings.TrimSpace(result.Stdout)}, nil
		},
	}
	_, _ = reg.Register(adapterTool)
	dispatcher := NewDispatcherWithRuntimeManager(reg, mgr)
	outcomes, err := dispatcher.Dispatch(context.Background(), []types.ToolCall{
		{CallID: "c1", Name: "local.exec"},
	}, DispatchConfig{MaxCalls: 1, Concurrency: 1, FailFast: true})
	if err != nil {
		t.Fatalf("dispatch should fallback host and succeed, err=%v", err)
	}
	if outcomes[0].Result.Error != nil || outcomes[0].Result.Content != "host-ok" {
		t.Fatalf("unexpected fallback result: %#v", outcomes[0].Result)
	}
	if outcomes[0].Result.Structured["sandbox_fallback_reason"] != sandboxReasonFallbackAllowRecord {
		t.Fatalf("sandbox_fallback_reason=%#v, want %q", outcomes[0].Result.Structured["sandbox_fallback_reason"], sandboxReasonFallbackAllowRecord)
	}
}

func TestDispatcherSandboxTimeoutDenyWithCanonicalReason(t *testing.T) {
	mgr := newSandboxTestManager(t, `
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
`)
	defer func() { _ = mgr.Close() }()
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

	reg := NewRegistry()
	adapterTool := &fakeSandboxAdapterTool{
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
	}
	_, _ = reg.Register(adapterTool)
	dispatcher := NewDispatcherWithRuntimeManager(reg, mgr)
	outcomes, err := dispatcher.Dispatch(context.Background(), []types.ToolCall{
		{CallID: "c1", Name: "local.exec"},
	}, DispatchConfig{MaxCalls: 1, Concurrency: 1, FailFast: true})
	if err == nil {
		t.Fatal("expected timeout deny error, got nil")
	}
	if outcomes[0].Result.Error == nil {
		t.Fatalf("expected deny outcome, got %#v", outcomes[0])
	}
	if outcomes[0].Result.Error.Details["reason_code"] != types.SandboxViolationTimeout {
		t.Fatalf("reason_code=%#v, want %q", outcomes[0].Result.Error.Details["reason_code"], types.SandboxViolationTimeout)
	}
}

func newSandboxTestManager(t *testing.T, content string) *runtimeconfig.Manager {
	t.Helper()
	cfgPath := filepath.Join(t.TempDir(), "runtime-sandbox.yaml")
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write sandbox config failed: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
		FilePath:  cfgPath,
		EnvPrefix: "BAYMAX_A51_LOCAL_TEST",
	})
	if err != nil {
		t.Fatalf("new runtime manager failed: %v", err)
	}
	return mgr
}
