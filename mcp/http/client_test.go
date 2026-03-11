package http

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
	mcpprofile "github.com/FelixSeptem/baymax/mcp/profile"
	mcpretry "github.com/FelixSeptem/baymax/mcp/retry"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type fakeSession struct {
	listFn func(ctx context.Context, params *mcp.ListToolsParams) (*mcp.ListToolsResult, error)
	callFn func(ctx context.Context, params *mcp.CallToolParams) (*mcp.CallToolResult, error)
}

func (f *fakeSession) ListTools(ctx context.Context, params *mcp.ListToolsParams) (*mcp.ListToolsResult, error) {
	if f.listFn != nil {
		return f.listFn(ctx, params)
	}
	return &mcp.ListToolsResult{}, nil
}

func (f *fakeSession) CallTool(ctx context.Context, params *mcp.CallToolParams) (*mcp.CallToolResult, error) {
	if f.callFn != nil {
		return f.callFn(ctx, params)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil
}

func (f *fakeSession) Close() error { return nil }

type eventCollector struct {
	mu     sync.Mutex
	events []types.Event
}

func (c *eventCollector) OnEvent(ctx context.Context, ev types.Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, ev)
}

func TestHTTPReconnectFlowAndStableCallID(t *testing.T) {
	attempt := 0
	connector := func(ctx context.Context) (Session, error) {
		attempt++
		if attempt == 1 {
			return &fakeSession{callFn: func(ctx context.Context, params *mcp.CallToolParams) (*mcp.CallToolResult, error) {
				return nil, errors.New("transport down")
			}}, nil
		}
		return &fakeSession{callFn: func(ctx context.Context, params *mcp.CallToolParams) (*mcp.CallToolResult, error) {
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil
		}}, nil
	}
	col := &eventCollector{}
	c := NewClient(Config{Connect: connector, Retry: 1, Backoff: 1 * time.Millisecond, EventHandler: col})

	res, err := c.CallTool(context.Background(), "tool", nil)
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if res.Content != "ok" {
		t.Fatalf("result content = %q, want ok", res.Content)
	}
	if attempt < 2 {
		t.Fatalf("connect attempts = %d, want >=2", attempt)
	}

	if len(col.events) < 3 {
		t.Fatalf("events too short: %#v", col.events)
	}
	if col.events[0].Type != "mcp.requested" {
		t.Fatalf("first event = %q, want mcp.requested", col.events[0].Type)
	}
	if col.events[1].Type != "mcp.reconnected" {
		t.Fatalf("second event = %q, want mcp.reconnected", col.events[1].Type)
	}
	if col.events[len(col.events)-1].Type != "mcp.completed" {
		t.Fatalf("last event = %q, want mcp.completed", col.events[len(col.events)-1].Type)
	}
	if col.events[0].CallID != col.events[len(col.events)-1].CallID {
		t.Fatalf("call id changed across reconnect: %q -> %q", col.events[0].CallID, col.events[len(col.events)-1].CallID)
	}
}

func TestHTTPHeartbeatReconnect(t *testing.T) {
	connected := 0
	connector := func(ctx context.Context) (Session, error) {
		connected++
		if connected == 1 {
			return &fakeSession{
				listFn: func(ctx context.Context, params *mcp.ListToolsParams) (*mcp.ListToolsResult, error) {
					return nil, errors.New("heartbeat failed")
				},
				callFn: func(ctx context.Context, params *mcp.CallToolParams) (*mcp.CallToolResult, error) {
					return nil, errors.New("should not run")
				},
			}, nil
		}
		return &fakeSession{
			listFn: func(ctx context.Context, params *mcp.ListToolsParams) (*mcp.ListToolsResult, error) {
				return &mcp.ListToolsResult{}, nil
			},
			callFn: func(ctx context.Context, params *mcp.CallToolParams) (*mcp.CallToolResult, error) {
				return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil
			},
		}, nil
	}
	col := &eventCollector{}
	c := NewClient(Config{
		Connect:           connector,
		Retry:             1,
		Backoff:           time.Millisecond,
		HeartbeatInterval: time.Nanosecond,
		HeartbeatTimeout:  10 * time.Millisecond,
		EventHandler:      col,
	})
	c.lastActivity.Store(time.Now().Add(-time.Minute).UnixNano())

	res, err := c.CallTool(context.Background(), "tool", nil)
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if res.Content != "ok" {
		t.Fatalf("result content = %q, want ok", res.Content)
	}
	if connected < 2 {
		t.Fatalf("expected reconnect, got connects=%d", connected)
	}
}

func TestHTTPEventOrderingOnFailure(t *testing.T) {
	connector := func(ctx context.Context) (Session, error) {
		return &fakeSession{callFn: func(ctx context.Context, params *mcp.CallToolParams) (*mcp.CallToolResult, error) {
			return nil, errors.New("hard failure")
		}}, nil
	}
	col := &eventCollector{}
	c := NewClient(Config{Connect: connector, Retry: 0, EventHandler: col})

	_, err := c.CallTool(context.Background(), "tool", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if len(col.events) < 2 {
		t.Fatalf("events too short: %#v", col.events)
	}
	if col.events[0].Type != "mcp.requested" {
		t.Fatalf("first event = %q, want mcp.requested", col.events[0].Type)
	}
	if col.events[len(col.events)-1].Type != "mcp.failed" {
		t.Fatalf("last event = %q, want mcp.failed", col.events[len(col.events)-1].Type)
	}
}

func TestHTTPProfileDefaultsApplied(t *testing.T) {
	c := NewClient(Config{Profile: mcpprofile.HighReliab, Connect: func(ctx context.Context) (Session, error) {
		return &fakeSession{}, nil
	}})
	if c.cfg.Retry != 3 {
		t.Fatalf("retry = %d, want 3", c.cfg.Retry)
	}
	if c.cfg.CallTimeout <= 10*time.Second {
		t.Fatalf("call timeout = %v, want > 10s", c.cfg.CallTimeout)
	}
}

func TestHTTPNonRetryableFailFast(t *testing.T) {
	var calls int
	connector := func(ctx context.Context) (Session, error) {
		return &fakeSession{callFn: func(ctx context.Context, params *mcp.CallToolParams) (*mcp.CallToolResult, error) {
			calls++
			return nil, mcpretry.NonRetryable(errors.New("non-retryable"))
		}}, nil
	}
	c := NewClient(Config{Connect: connector, Retry: 3, Backoff: time.Millisecond})
	_, err := c.CallTool(context.Background(), "tool", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}

func TestHTTPRecentCallSummary(t *testing.T) {
	connector := func(ctx context.Context) (Session, error) {
		return &fakeSession{}, nil
	}
	c := NewClient(Config{Connect: connector, Profile: mcpprofile.Default})
	_, err := c.CallTool(context.Background(), "tool", nil)
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	summary := c.RecentCallSummary(1)
	if len(summary) != 1 {
		t.Fatalf("summary len = %d, want 1", len(summary))
	}
	if summary[0].Transport != "http" || summary[0].Tool != "tool" {
		t.Fatalf("unexpected summary: %#v", summary[0])
	}
}

func TestHTTPRuntimeManagerConfigAndDiagnostics(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	content := `
mcp:
  active_profile: default
  profiles:
    default:
      retry: 0
      call_timeout: 2s
      backoff: 5ms
      queue_size: 32
      backpressure: block
      read_pool_size: 4
      write_pool_size: 1
diagnostics:
  max_call_records: 10
`
	if err := os.WriteFile(file, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	connector := func(ctx context.Context) (Session, error) {
		return &fakeSession{}, nil
	}
	c := NewClient(Config{Connect: connector, RuntimeManager: mgr})
	_, err = c.CallTool(context.Background(), "tool", nil)
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	summary := mgr.RecentCalls(1)
	if len(summary) != 1 || summary[0].Transport != "http" {
		t.Fatalf("unexpected manager summary: %#v", summary)
	}
}
