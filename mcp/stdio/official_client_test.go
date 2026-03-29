package stdio

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type fakeOfficialSession struct {
	listTools func(ctx context.Context) ([]types.MCPToolMeta, error)
	callTool  func(ctx context.Context, name string, args map[string]any) (types.ToolResult, error)
	closeFn   func() error
}

func (f *fakeOfficialSession) ListTools(ctx context.Context) ([]types.MCPToolMeta, error) {
	if f != nil && f.listTools != nil {
		return f.listTools(ctx)
	}
	return []types.MCPToolMeta{{Name: "tool"}}, nil
}

func (f *fakeOfficialSession) CallTool(ctx context.Context, name string, args map[string]any) (types.ToolResult, error) {
	if f != nil && f.callTool != nil {
		return f.callTool(ctx, name, args)
	}
	return types.ToolResult{Content: "ok"}, nil
}

func (f *fakeOfficialSession) Close() error {
	if f != nil && f.closeFn != nil {
		return f.closeFn()
	}
	return nil
}

func TestOfficialClientSandboxPolicyDenyBlocksStartup(t *testing.T) {
	mgr := newOfficialClientTestManager(t, `
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

	client := NewOfficialClient(OfficialConfig{
		Command:        exec.Command("pwsh", "-NoProfile", "-Command", "echo test"),
		RuntimeManager: mgr,
	})
	var connectCalls int32
	client.connect = func(ctx context.Context, command *exec.Cmd, opts *mcp.ClientSessionOptions) (officialSession, error) {
		_ = ctx
		_ = command
		_ = opts
		atomic.AddInt32(&connectCalls, 1)
		return &fakeOfficialSession{}, nil
	}

	_, err := client.ListTools(context.Background())
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "sandbox.policy_deny") {
		t.Fatalf("expected sandbox.policy_deny, got err=%v", err)
	}
	if atomic.LoadInt32(&connectCalls) != 0 {
		t.Fatalf("connect_calls=%d, want 0", connectCalls)
	}
}

func TestOfficialClientSandboxFallbackAllowPerCallLifecycle(t *testing.T) {
	mgr := newOfficialClientTestManager(t, `
security:
  sandbox:
    enabled: true
    mode: enforce
    required: false
    policy:
      default_action: host
      by_tool:
        mcp+stdio_command: sandbox
      profile: default
      fallback_action: allow_and_record
      fallback_action_by_tool:
        mcp+stdio_command: allow_and_record
    executor:
      session_mode: per_call
`)
	defer func() { _ = mgr.Close() }()

	client := NewOfficialClient(OfficialConfig{
		Command:        exec.Command("pwsh", "-NoProfile", "-Command", "echo test"),
		RuntimeManager: mgr,
	})

	var connectCalls int32
	var closedCalls int32
	client.connect = func(ctx context.Context, command *exec.Cmd, opts *mcp.ClientSessionOptions) (officialSession, error) {
		_ = ctx
		_ = command
		_ = opts
		atomic.AddInt32(&connectCalls, 1)
		return &fakeOfficialSession{
			listTools: func(ctx context.Context) ([]types.MCPToolMeta, error) {
				_ = ctx
				return []types.MCPToolMeta{{Name: "ok"}}, nil
			},
			closeFn: func() error {
				atomic.AddInt32(&closedCalls, 1)
				return nil
			},
		}, nil
	}

	tools, err := client.ListTools(context.Background())
	if err != nil || len(tools) != 1 {
		t.Fatalf("first ListTools failed tools=%#v err=%v", tools, err)
	}
	tools, err = client.ListTools(context.Background())
	if err != nil || len(tools) != 1 {
		t.Fatalf("second ListTools failed tools=%#v err=%v", tools, err)
	}
	if atomic.LoadInt32(&connectCalls) != 2 {
		t.Fatalf("connect_calls=%d, want 2", connectCalls)
	}
	if atomic.LoadInt32(&closedCalls) != 2 {
		t.Fatalf("closed_calls=%d, want 2", closedCalls)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("first Close failed: %v", err)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("second Close should be idempotent: %v", err)
	}
}

func TestOfficialClientPerSessionReconnectOnCrash(t *testing.T) {
	mgr := newOfficialClientTestManager(t, `
security:
  sandbox:
    enabled: true
    mode: enforce
    policy:
      default_action: host
      profile: default
      fallback_action: allow_and_record
    executor:
      session_mode: per_session
`)
	defer func() { _ = mgr.Close() }()

	client := NewOfficialClient(OfficialConfig{
		Command:        exec.Command("pwsh", "-NoProfile", "-Command", "echo test"),
		RuntimeManager: mgr,
	})

	var connectCalls int32
	var firstSessionCalls int32
	client.connect = func(ctx context.Context, command *exec.Cmd, opts *mcp.ClientSessionOptions) (officialSession, error) {
		_ = ctx
		_ = command
		_ = opts
		n := atomic.AddInt32(&connectCalls, 1)
		if n == 1 {
			return &fakeOfficialSession{
				callTool: func(ctx context.Context, name string, args map[string]any) (types.ToolResult, error) {
					_ = ctx
					_ = name
					_ = args
					atomic.AddInt32(&firstSessionCalls, 1)
					return types.ToolResult{}, errors.New("broken pipe")
				},
			}, nil
		}
		return &fakeOfficialSession{
			callTool: func(ctx context.Context, name string, args map[string]any) (types.ToolResult, error) {
				_ = ctx
				_ = name
				_ = args
				return types.ToolResult{Content: "ok"}, nil
			},
		}, nil
	}

	result, err := client.CallTool(context.Background(), "tool", nil)
	if err != nil || result.Content != "ok" {
		t.Fatalf("CallTool should reconnect and succeed result=%#v err=%v", result, err)
	}
	if atomic.LoadInt32(&connectCalls) != 2 {
		t.Fatalf("connect_calls=%d, want 2", connectCalls)
	}
	if atomic.LoadInt32(&firstSessionCalls) != 1 {
		t.Fatalf("first_session_calls=%d, want 1", firstSessionCalls)
	}
	result, err = client.CallTool(context.Background(), "tool", nil)
	if err != nil || result.Content != "ok" {
		t.Fatalf("second CallTool should reuse recovered session result=%#v err=%v", result, err)
	}
	if atomic.LoadInt32(&connectCalls) != 2 {
		t.Fatalf("connect_calls after reuse=%d, want 2", connectCalls)
	}
}

func TestOfficialClientPerCallCancelClosesSession(t *testing.T) {
	mgr := newOfficialClientTestManager(t, `
security:
  sandbox:
    enabled: true
    mode: enforce
    required: false
    policy:
      default_action: host
      by_tool:
        mcp+stdio_command: sandbox
      profile: default
      fallback_action: allow_and_record
      fallback_action_by_tool:
        mcp+stdio_command: allow_and_record
    executor:
      session_mode: per_call
`)
	defer func() { _ = mgr.Close() }()

	client := NewOfficialClient(OfficialConfig{
		Command:        exec.Command("pwsh", "-NoProfile", "-Command", "echo test"),
		RuntimeManager: mgr,
	})
	var closedCalls int32
	client.connect = func(ctx context.Context, command *exec.Cmd, opts *mcp.ClientSessionOptions) (officialSession, error) {
		_ = ctx
		_ = command
		_ = opts
		return &fakeOfficialSession{
			callTool: func(ctx context.Context, name string, args map[string]any) (types.ToolResult, error) {
				_ = name
				_ = args
				<-ctx.Done()
				return types.ToolResult{}, ctx.Err()
			},
			closeFn: func() error {
				atomic.AddInt32(&closedCalls, 1)
				return nil
			},
		}, nil
	}

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := client.CallTool(timeoutCtx, "tool", nil)
	if err == nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline exceeded, got %v", err)
	}
	if atomic.LoadInt32(&closedCalls) != 1 {
		t.Fatalf("closed_calls=%d, want 1", closedCalls)
	}
}

func newOfficialClientTestManager(t *testing.T, content string) *runtimeconfig.Manager {
	t.Helper()
	cfgPath := filepath.Join(t.TempDir(), "runtime-official-client.yaml")
	if err := os.WriteFile(cfgPath, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write config failed: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
		FilePath:  cfgPath,
		EnvPrefix: "BAYMAX_A51_STDIO_TEST",
	})
	if err != nil {
		t.Fatalf("new runtime manager failed: %v", err)
	}
	return mgr
}
