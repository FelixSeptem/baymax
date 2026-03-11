package stdio

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	mcpprofile "github.com/FelixSeptem/baymax/mcp/profile"
	mcpretry "github.com/FelixSeptem/baymax/mcp/retry"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

type fakeTransport struct {
	initCalls int32
	listCalls int32
	callCalls int32

	callFn func(ctx context.Context, name string, args map[string]any) (Response, error)
}

func (f *fakeTransport) Initialize(ctx context.Context) error {
	atomic.AddInt32(&f.initCalls, 1)
	return nil
}

func (f *fakeTransport) ListTools(ctx context.Context) ([]types.MCPToolMeta, error) {
	atomic.AddInt32(&f.listCalls, 1)
	return []types.MCPToolMeta{{Name: "tool"}}, nil
}

func (f *fakeTransport) CallTool(ctx context.Context, name string, args map[string]any) (Response, error) {
	atomic.AddInt32(&f.callCalls, 1)
	if f.callFn != nil {
		return f.callFn(ctx, name, args)
	}
	return Response{Content: "ok"}, nil
}

func (f *fakeTransport) Close() error { return nil }

type collector struct {
	mu     sync.Mutex
	events []types.Event
}

func (c *collector) OnEvent(ctx context.Context, ev types.Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, ev)
}

func TestWarmupInitializeAndListOnce(t *testing.T) {
	ft := &fakeTransport{}
	client := NewClient(ft, Config{})

	if err := client.Warmup(context.Background()); err != nil {
		t.Fatalf("Warmup failed: %v", err)
	}
	if _, err := client.CallTool(context.Background(), "a", nil); err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if got := atomic.LoadInt32(&ft.initCalls); got != 1 {
		t.Fatalf("init calls = %d, want 1", got)
	}
	if got := atomic.LoadInt32(&ft.listCalls); got != 1 {
		t.Fatalf("list calls = %d, want 1", got)
	}
}

func TestCallToolTimeoutClassification(t *testing.T) {
	ft := &fakeTransport{
		callFn: func(ctx context.Context, name string, args map[string]any) (Response, error) {
			<-ctx.Done()
			return Response{}, ctx.Err()
		},
	}
	client := NewClient(ft, Config{CallTimeout: 20 * time.Millisecond})

	result, err := client.CallTool(context.Background(), "slow", nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want deadline exceeded", err)
	}
	if result.Error == nil || result.Error.Class != types.ErrPolicyTimeout {
		t.Fatalf("result error = %#v, want ErrPolicyTimeout", result.Error)
	}
}

func TestCallToolRetryThenSuccess(t *testing.T) {
	var attempts int32
	ft := &fakeTransport{
		callFn: func(ctx context.Context, name string, args map[string]any) (Response, error) {
			current := atomic.AddInt32(&attempts, 1)
			if current == 1 {
				return Response{}, errors.New("temporary")
			}
			return Response{Content: "ok"}, nil
		},
	}
	client := NewClient(ft, Config{Retry: 1, Backoff: 1 * time.Millisecond})

	result, err := client.CallTool(context.Background(), "retry", nil)
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if result.Content != "ok" {
		t.Fatalf("content = %q, want ok", result.Content)
	}
	if got := atomic.LoadInt32(&attempts); got != 2 {
		t.Fatalf("attempts = %d, want 2", got)
	}
}

func TestReadPoolLimitsConcurrency(t *testing.T) {
	var current int32
	var maxSeen int32
	ft := &fakeTransport{
		callFn: func(ctx context.Context, name string, args map[string]any) (Response, error) {
			inFlight := atomic.AddInt32(&current, 1)
			for {
				old := atomic.LoadInt32(&maxSeen)
				if inFlight <= old || atomic.CompareAndSwapInt32(&maxSeen, old, inFlight) {
					break
				}
			}
			time.Sleep(30 * time.Millisecond)
			atomic.AddInt32(&current, -1)
			return Response{Content: "ok"}, nil
		},
	}
	client := NewClient(ft, Config{ReadPoolSize: 1})

	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = client.CallTool(context.Background(), "read", nil)
		}()
	}
	wg.Wait()
	if got := atomic.LoadInt32(&maxSeen); got != 1 {
		t.Fatalf("max concurrency = %d, want 1", got)
	}
}

func TestEmitRequestedCompletedFailedEvents(t *testing.T) {
	ft := &fakeTransport{
		callFn: func(ctx context.Context, name string, args map[string]any) (Response, error) {
			if name == "bad" {
				return Response{Error: "failed"}, nil
			}
			return Response{Content: "ok"}, nil
		},
	}
	col := &collector{}
	client := NewClient(ft, Config{EventHandler: col})
	_, _ = client.CallTool(context.Background(), "ok", nil)
	_, _ = client.CallTool(context.Background(), "bad", nil)

	if len(col.events) < 4 {
		t.Fatalf("event count = %d, want >=4", len(col.events))
	}
	if col.events[0].Type != "mcp.requested" || col.events[1].Type != "mcp.completed" {
		t.Fatalf("first sequence unexpected: %s, %s", col.events[0].Type, col.events[1].Type)
	}
	if col.events[2].Type != "mcp.requested" || col.events[3].Type != "mcp.failed" {
		t.Fatalf("second sequence unexpected: %s, %s", col.events[2].Type, col.events[3].Type)
	}
}

func TestStdioProfileDefaultsApplied(t *testing.T) {
	client := NewClient(&fakeTransport{}, Config{Profile: mcpprofile.HighThroughput})
	if client.cfg.ReadPoolSize < 8 {
		t.Fatalf("read pool = %d, want >= 8 for high-throughput", client.cfg.ReadPoolSize)
	}
	if client.cfg.Backpressure != types.BackpressureReject {
		t.Fatalf("backpressure = %q, want reject", client.cfg.Backpressure)
	}
}

func TestStdioNonRetryableFailFast(t *testing.T) {
	var attempts int32
	ft := &fakeTransport{
		callFn: func(ctx context.Context, name string, args map[string]any) (Response, error) {
			atomic.AddInt32(&attempts, 1)
			return Response{}, mcpretry.NonRetryable(errors.New("hard fail"))
		},
	}
	client := NewClient(ft, Config{Retry: 3, Backoff: time.Millisecond})
	_, err := client.CallTool(context.Background(), "x", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if atomic.LoadInt32(&attempts) != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
}

func TestStdioRecentCallSummary(t *testing.T) {
	ft := &fakeTransport{}
	client := NewClient(ft, Config{})
	_, err := client.CallTool(context.Background(), "x", nil)
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	summary := client.RecentCallSummary(1)
	if len(summary) != 1 {
		t.Fatalf("summary len = %d, want 1", len(summary))
	}
	if summary[0].Transport != "stdio" {
		t.Fatalf("transport = %q, want stdio", summary[0].Transport)
	}
}

func TestStdioRuntimeManagerConfigAndDiagnostics(t *testing.T) {
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
      read_pool_size: 2
      write_pool_size: 1
`
	if err := os.WriteFile(file, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	ft := &fakeTransport{}
	client := NewClient(ft, Config{RuntimeManager: mgr})
	_, err = client.CallTool(context.Background(), "x", nil)
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	summary := mgr.RecentCalls(1)
	if len(summary) != 1 || summary[0].Transport != "stdio" {
		t.Fatalf("unexpected manager summary: %#v", summary)
	}
}
