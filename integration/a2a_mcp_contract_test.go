package integration

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/a2a"
	"github.com/FelixSeptem/baymax/core/types"
	mcpretry "github.com/FelixSeptem/baymax/mcp/retry"
	stdiomcp "github.com/FelixSeptem/baymax/mcp/stdio"
)

type a2aTimelineCollector struct {
	mu     sync.Mutex
	events []types.Event
}

func (c *a2aTimelineCollector) OnEvent(_ context.Context, ev types.Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, ev)
}

func (c *a2aTimelineCollector) snapshot() []types.Event {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]types.Event, len(c.events))
	copy(out, c.events)
	return out
}

func TestA2AMCPContractHappyPath(t *testing.T) {
	timeline := &a2aTimelineCollector{}
	stdioClient := stdiomcp.NewClient(&contractSTDIOTransport{
		callFn: func(ctx context.Context, name string, args map[string]any) (stdiomcp.Response, error) {
			return stdiomcp.Response{Content: "mcp-ok"}, nil
		},
	}, stdiomcp.Config{
		CallTimeout: 300 * time.Millisecond,
		Retry:       1,
		Backoff:     time.Millisecond,
	})

	server := a2a.NewInMemoryServer(a2a.HandlerFunc(func(ctx context.Context, req a2a.TaskRequest) (map[string]any, error) {
		resp, err := stdioClient.CallTool(ctx, "tool", req.Payload)
		if err != nil {
			return nil, err
		}
		return map[string]any{"mcp_content": resp.Content}, nil
	}), timeline)
	client := a2a.NewClient(server, []a2a.AgentCard{
		{
			AgentID:      "agent-remote",
			PeerID:       "peer-remote",
			Capabilities: []string{"tool.call"},
			Priority:     1,
		},
	}, a2a.DeterministicRouter{RequireAll: true}, a2a.ClientPolicy{
		Timeout:            300 * time.Millisecond,
		RequestMaxAttempts: 1,
		CallbackRetry: a2a.RetryPolicy{
			MaxAttempts: 2,
			Backoff:     5 * time.Millisecond,
		},
	}, timeline)

	submitted, err := client.Submit(context.Background(), a2a.TaskRequest{
		AgentID:              "agent-main",
		Method:               "tool.proxy",
		RequiredCapabilities: []string{"tool.call"},
		Payload:              map[string]any{"query": "ping"},
	})
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	callbackAttempts := 0
	result, err := client.WaitResult(context.Background(), submitted.TaskID, 10*time.Millisecond, func(ctx context.Context, record a2a.TaskRecord) error {
		callbackAttempts++
		if callbackAttempts == 1 {
			return errors.New("callback temporary error")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WaitResult failed: %v", err)
	}
	if result.Status != a2a.StatusSucceeded {
		t.Fatalf("status = %q, want succeeded", result.Status)
	}
	if result.Result["mcp_content"] != "mcp-ok" {
		t.Fatalf("result mismatch: %#v", result.Result)
	}
	if callbackAttempts != 2 {
		t.Fatalf("callback attempts = %d, want 2", callbackAttempts)
	}

	summary := a2a.BuildRunSummary([]a2a.TaskRecord{result})
	if summary.A2ATaskTotal != 1 || summary.A2ATaskFailed != 0 {
		t.Fatalf("summary mismatch: %#v", summary)
	}
	if summary.PeerID != "peer-remote" {
		t.Fatalf("peer_id = %q, want peer-remote", summary.PeerID)
	}

	reasons := map[string]bool{}
	for _, ev := range timeline.snapshot() {
		if ev.Type != types.EventTypeActionTimeline {
			continue
		}
		if reason, _ := ev.Payload["reason"].(string); reason != "" {
			reasons[reason] = true
		}
	}
	for _, reason := range []string{
		a2a.ReasonSubmit,
		a2a.ReasonStatusPoll,
		a2a.ReasonCallbackRetry,
		a2a.ReasonResolve,
	} {
		if !reasons[reason] {
			t.Fatalf("missing timeline reason %q in %#v", reason, reasons)
		}
	}
}

func TestA2AMCPContractErrorClassification(t *testing.T) {
	stdioClient := stdiomcp.NewClient(&contractSTDIOTransport{
		callFn: func(ctx context.Context, name string, args map[string]any) (stdiomcp.Response, error) {
			return stdiomcp.Response{}, mcpretry.NonRetryable(errors.New("unsupported method"))
		},
	}, stdiomcp.Config{
		CallTimeout: 300 * time.Millisecond,
		Retry:       0,
		Backoff:     time.Millisecond,
	})

	server := a2a.NewInMemoryServer(a2a.HandlerFunc(func(ctx context.Context, req a2a.TaskRequest) (map[string]any, error) {
		resp, err := stdioClient.CallTool(ctx, "tool", req.Payload)
		if err != nil {
			return nil, err
		}
		return map[string]any{"mcp_content": resp.Content}, nil
	}), nil)
	client := a2a.NewClient(server, nil, nil, a2a.ClientPolicy{
		Timeout:            300 * time.Millisecond,
		RequestMaxAttempts: 1,
	}, nil)

	submitted, err := client.Submit(context.Background(), a2a.TaskRequest{
		AgentID: "agent-main",
		PeerID:  "peer-remote",
		Method:  "tool.unsupported",
	})
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	result, err := client.WaitResult(context.Background(), submitted.TaskID, 10*time.Millisecond, nil)
	if err != nil {
		t.Fatalf("WaitResult failed: %v", err)
	}
	if result.Status != a2a.StatusFailed {
		t.Fatalf("status = %q, want failed", result.Status)
	}
	if result.ErrorClass != types.ErrMCP {
		t.Fatalf("error_class = %q, want %q", result.ErrorClass, types.ErrMCP)
	}
	if result.A2AErrorLayer != string(a2a.ErrorLayerProtocol) {
		t.Fatalf("a2a_error_layer = %q, want %q", result.A2AErrorLayer, a2a.ErrorLayerProtocol)
	}
}
