package integration

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	httpmcp "github.com/FelixSeptem/baymax/mcp/http"
	mcpretry "github.com/FelixSeptem/baymax/mcp/retry"
	stdiomcp "github.com/FelixSeptem/baymax/mcp/stdio"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type contractHTTPSession struct {
	callFn func(ctx context.Context, params *mcp.CallToolParams) (*mcp.CallToolResult, error)
}

func (s *contractHTTPSession) ListTools(ctx context.Context, params *mcp.ListToolsParams) (*mcp.ListToolsResult, error) {
	return &mcp.ListToolsResult{}, nil
}

func (s *contractHTTPSession) CallTool(ctx context.Context, params *mcp.CallToolParams) (*mcp.CallToolResult, error) {
	return s.callFn(ctx, params)
}

func (s *contractHTTPSession) Close() error { return nil }

type contractSTDIOTransport struct {
	callFn func(ctx context.Context, name string, args map[string]any) (stdiomcp.Response, error)
}

func (t *contractSTDIOTransport) Initialize(ctx context.Context) error { return nil }
func (t *contractSTDIOTransport) ListTools(ctx context.Context) ([]types.MCPToolMeta, error) {
	return []types.MCPToolMeta{{Name: "tool"}}, nil
}
func (t *contractSTDIOTransport) CallTool(ctx context.Context, name string, args map[string]any) (stdiomcp.Response, error) {
	return t.callFn(ctx, name, args)
}
func (t *contractSTDIOTransport) Close() error { return nil }

func TestMCPTransportContractRetryAndSummary(t *testing.T) {
	var httpAttempts int32
	httpClient := httpmcp.NewClient(httpmcp.Config{
		Retry:   1,
		Backoff: time.Millisecond,
		Connect: func(ctx context.Context) (httpmcp.Session, error) {
			return &contractHTTPSession{
				callFn: func(ctx context.Context, params *mcp.CallToolParams) (*mcp.CallToolResult, error) {
					if atomic.AddInt32(&httpAttempts, 1) == 1 {
						return nil, errors.New("temporary")
					}
					return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil
				},
			}, nil
		},
	})

	var stdioAttempts int32
	stdioClient := stdiomcp.NewClient(&contractSTDIOTransport{
		callFn: func(ctx context.Context, name string, args map[string]any) (stdiomcp.Response, error) {
			if atomic.AddInt32(&stdioAttempts, 1) == 1 {
				return stdiomcp.Response{}, errors.New("temporary")
			}
			return stdiomcp.Response{Content: "ok"}, nil
		},
	}, stdiomcp.Config{
		Retry:   1,
		Backoff: time.Millisecond,
	})

	httpRes, httpErr := httpClient.CallTool(context.Background(), "tool", nil)
	if httpErr != nil {
		t.Fatalf("http call failed: %v", httpErr)
	}
	stdioRes, stdioErr := stdioClient.CallTool(context.Background(), "tool", nil)
	if stdioErr != nil {
		t.Fatalf("stdio call failed: %v", stdioErr)
	}
	if httpRes.Content != "ok" || stdioRes.Content != "ok" {
		t.Fatalf("unexpected content http=%q stdio=%q", httpRes.Content, stdioRes.Content)
	}

	httpSummary := httpClient.RecentCallSummary(1)
	stdioSummary := stdioClient.RecentCallSummary(1)
	if len(httpSummary) != 1 || len(stdioSummary) != 1 {
		t.Fatalf("unexpected summary len http=%d stdio=%d", len(httpSummary), len(stdioSummary))
	}
	if httpSummary[0].RetryCount != 1 || stdioSummary[0].RetryCount != 1 {
		t.Fatalf("retry_count mismatch http=%d stdio=%d", httpSummary[0].RetryCount, stdioSummary[0].RetryCount)
	}
}

func TestMCPTransportContractNonRetryableFailFast(t *testing.T) {
	var httpAttempts int32
	httpClient := httpmcp.NewClient(httpmcp.Config{
		Retry:   3,
		Backoff: time.Millisecond,
		Connect: func(ctx context.Context) (httpmcp.Session, error) {
			return &contractHTTPSession{
				callFn: func(ctx context.Context, params *mcp.CallToolParams) (*mcp.CallToolResult, error) {
					atomic.AddInt32(&httpAttempts, 1)
					return nil, mcpretry.NonRetryable(errors.New("hard fail"))
				},
			}, nil
		},
	})

	var stdioAttempts int32
	stdioClient := stdiomcp.NewClient(&contractSTDIOTransport{
		callFn: func(ctx context.Context, name string, args map[string]any) (stdiomcp.Response, error) {
			atomic.AddInt32(&stdioAttempts, 1)
			return stdiomcp.Response{}, mcpretry.NonRetryable(errors.New("hard fail"))
		},
	}, stdiomcp.Config{
		Retry:   3,
		Backoff: time.Millisecond,
	})

	if _, err := httpClient.CallTool(context.Background(), "tool", nil); err == nil {
		t.Fatal("expected http error")
	}
	if _, err := stdioClient.CallTool(context.Background(), "tool", nil); err == nil {
		t.Fatal("expected stdio error")
	}
	if httpAttempts != 1 || stdioAttempts != 1 {
		t.Fatalf("fail-fast mismatch http=%d stdio=%d", httpAttempts, stdioAttempts)
	}
}

func TestMCPTransportContractTimeoutClassification(t *testing.T) {
	httpClient := httpmcp.NewClient(httpmcp.Config{
		CallTimeout: 10 * time.Millisecond,
		Retry:       0,
		Backoff:     time.Millisecond,
		Connect: func(ctx context.Context) (httpmcp.Session, error) {
			return &contractHTTPSession{
				callFn: func(ctx context.Context, params *mcp.CallToolParams) (*mcp.CallToolResult, error) {
					time.Sleep(50 * time.Millisecond)
					return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "late"}}}, nil
				},
			}, nil
		},
	})
	stdioClient := stdiomcp.NewClient(&contractSTDIOTransport{
		callFn: func(ctx context.Context, name string, args map[string]any) (stdiomcp.Response, error) {
			time.Sleep(50 * time.Millisecond)
			return stdiomcp.Response{Content: "late"}, nil
		},
	}, stdiomcp.Config{
		CallTimeout: 10 * time.Millisecond,
		Retry:       0,
		Backoff:     time.Millisecond,
	})

	httpRes, httpErr := httpClient.CallTool(context.Background(), "tool", nil)
	stdioRes, stdioErr := stdioClient.CallTool(context.Background(), "tool", nil)
	if httpErr == nil || stdioErr == nil {
		t.Fatalf("expected timeout errors, got http=%v stdio=%v", httpErr, stdioErr)
	}
	if httpRes.Error == nil || httpRes.Error.Class != types.ErrPolicyTimeout {
		t.Fatalf("http timeout class = %+v, want %q", httpRes.Error, types.ErrPolicyTimeout)
	}
	if stdioRes.Error == nil || stdioRes.Error.Class != types.ErrPolicyTimeout {
		t.Fatalf("stdio timeout class = %+v, want %q", stdioRes.Error, types.ErrPolicyTimeout)
	}
}

func TestMCPTransportContractReconnectAndBackpressure(t *testing.T) {
	var connectCount int32
	var firstSessionCall int32
	httpClient := httpmcp.NewClient(httpmcp.Config{
		Retry:         1,
		Backoff:       time.Millisecond,
		MaxReconnects: 1,
		Connect: func(ctx context.Context) (httpmcp.Session, error) {
			idx := atomic.AddInt32(&connectCount, 1)
			if idx == 1 {
				return &contractHTTPSession{
					callFn: func(ctx context.Context, params *mcp.CallToolParams) (*mcp.CallToolResult, error) {
						atomic.AddInt32(&firstSessionCall, 1)
						return nil, errors.New("broken stream")
					},
				}, nil
			}
			return &contractHTTPSession{
				callFn: func(ctx context.Context, params *mcp.CallToolParams) (*mcp.CallToolResult, error) {
					return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil
				},
			}, nil
		},
	})

	httpRes, httpErr := httpClient.CallTool(context.Background(), "tool", nil)
	if httpErr != nil {
		t.Fatalf("http reconnect call failed: %v", httpErr)
	}
	if httpRes.Content != "ok" {
		t.Fatalf("http reconnect content = %q, want ok", httpRes.Content)
	}
	httpSummary := httpClient.RecentCallSummary(1)
	if len(httpSummary) != 1 || httpSummary[0].ReconnectCount < 1 {
		t.Fatalf("http reconnect summary = %+v, want reconnect_count >= 1", httpSummary)
	}

	started := make(chan struct{}, 1)
	release := make(chan struct{})
	stdioClient := stdiomcp.NewClient(&contractSTDIOTransport{
		callFn: func(ctx context.Context, name string, args map[string]any) (stdiomcp.Response, error) {
			select {
			case started <- struct{}{}:
			default:
			}
			<-release
			return stdiomcp.Response{Content: "ok"}, nil
		},
	}, stdiomcp.Config{
		ReadPoolSize:  1,
		WritePoolSize: 1,
		Backpressure:  types.BackpressureReject,
		CallTimeout:   200 * time.Millisecond,
		Retry:         0,
	})
	defer close(release)

	firstDone := make(chan struct{})
	go func() {
		defer close(firstDone)
		_, _ = stdioClient.CallTool(context.Background(), "tool", nil)
	}()
	<-started

	stdioRes, stdioErr := stdioClient.CallTool(context.Background(), "tool", nil)
	if stdioErr == nil {
		t.Fatal("expected stdio backpressure reject error")
	}
	if stdioRes.Error == nil || stdioRes.Error.Class != types.ErrPolicyTimeout {
		t.Fatalf("stdio backpressure class = %+v, want %q", stdioRes.Error, types.ErrPolicyTimeout)
	}
}
