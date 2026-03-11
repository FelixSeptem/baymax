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
