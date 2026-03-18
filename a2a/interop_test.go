package a2a

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

type timelineCollector struct {
	mu     sync.Mutex
	events []types.Event
}

func (c *timelineCollector) OnEvent(_ context.Context, ev types.Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, ev)
}

func (c *timelineCollector) Snapshot() []types.Event {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]types.Event, len(c.events))
	copy(out, c.events)
	return out
}

func waitForTerminal(t *testing.T, server Server, taskID string) TaskRecord {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		rec, err := server.Status(context.Background(), taskID)
		if err != nil {
			t.Fatalf("status(%s) failed: %v", taskID, err)
		}
		if isTerminal(rec.Status) {
			return rec
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("task %s did not reach terminal status", taskID)
	return TaskRecord{}
}

func TestA2ASubmitStatusResultAndTimelineNormalization(t *testing.T) {
	collector := &timelineCollector{}
	server := NewInMemoryServer(HandlerFunc(func(ctx context.Context, req TaskRequest) (map[string]any, error) {
		return map[string]any{"echo": req.Method}, nil
	}), collector)
	client := NewClient(server, nil, nil, ClientPolicy{
		Timeout:            300 * time.Millisecond,
		RequestMaxAttempts: 1,
	}, collector)

	submitted, err := client.Submit(context.Background(), TaskRequest{
		AgentID: "agent-a",
		PeerID:  "peer-1",
		Method:  "echo",
	})
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}
	if submitted.Status != StatusSubmitted {
		t.Fatalf("submitted status = %q, want submitted", submitted.Status)
	}
	if strings.TrimSpace(submitted.TaskID) == "" {
		t.Fatal("task_id should not be empty")
	}

	result, err := client.WaitResult(context.Background(), submitted.TaskID, 10*time.Millisecond, nil)
	if err != nil {
		t.Fatalf("WaitResult failed: %v", err)
	}
	if result.Status != StatusSucceeded {
		t.Fatalf("terminal status = %q, want succeeded", result.Status)
	}
	if result.Result["echo"] != "echo" {
		t.Fatalf("result payload mismatch: %#v", result.Result)
	}

	events := collector.Snapshot()
	if len(events) == 0 {
		t.Fatal("expected timeline events")
	}
	reasons := map[string]bool{}
	foundPendingSubmit := false
	for _, ev := range events {
		if ev.Type != types.EventTypeActionTimeline {
			continue
		}
		reason, _ := ev.Payload["reason"].(string)
		reasons[reason] = true
		if reason == ReasonSubmit {
			status, _ := ev.Payload["status"].(string)
			if status == string(types.ActionStatusPending) {
				foundPendingSubmit = true
			}
			if ev.Payload["task_id"] == "" || ev.Payload["agent_id"] == "" || ev.Payload["peer_id"] == "" {
				t.Fatalf("submit payload missing correlation fields: %#v", ev.Payload)
			}
		}
	}
	for _, reason := range []string{ReasonSubmit, ReasonStatusPoll, ReasonResolve} {
		if !reasons[reason] {
			t.Fatalf("missing timeline reason %q", reason)
		}
	}
	if !foundPendingSubmit {
		t.Fatal("submitted state should be normalized to pending in timeline payload")
	}
}

func TestDeterministicRouterCapabilitySelection(t *testing.T) {
	router := DeterministicRouter{MaxCandidates: 8, RequireAll: true}
	cards := []AgentCard{
		{AgentID: "agent-z", PeerID: "peer-z", Capabilities: []string{"search", "math"}, Priority: 1},
		{AgentID: "agent-a", PeerID: "peer-a", Capabilities: []string{"search", "math"}, Priority: 1},
		{AgentID: "agent-b", PeerID: "peer-b", Capabilities: []string{"search"}, Priority: 9},
	}
	selected, err := router.SelectPeer(cards, []string{"math", "search"})
	if err != nil {
		t.Fatalf("SelectPeer failed: %v", err)
	}
	if selected.PeerID != "peer-a" {
		t.Fatalf("selected peer = %q, want peer-a (deterministic lexical tie-break)", selected.PeerID)
	}

	relaxed := DeterministicRouter{MaxCandidates: 8, RequireAll: false}
	selected, err = relaxed.SelectPeer(cards, []string{"search", "math"})
	if err != nil {
		t.Fatalf("SelectPeer (relaxed) failed: %v", err)
	}
	if selected.PeerID != "peer-a" {
		t.Fatalf("selected peer (relaxed) = %q, want peer-a (higher capability score)", selected.PeerID)
	}

	_, err = router.SelectPeer(cards, []string{"vision"})
	if err == nil {
		t.Fatal("expected routing error for unmatched capability")
	}
}

func TestClientUsesCapabilityRoutingWhenPeerMissing(t *testing.T) {
	server := NewInMemoryServer(HandlerFunc(func(ctx context.Context, req TaskRequest) (map[string]any, error) {
		return map[string]any{"ok": true}, nil
	}), nil)
	client := NewClient(server, []AgentCard{
		{AgentID: "agent-1", PeerID: "peer-1", Capabilities: []string{"search"}},
		{AgentID: "agent-2", PeerID: "peer-2", Capabilities: []string{"search", "math"}},
	}, DeterministicRouter{RequireAll: true}, ClientPolicy{Timeout: 300 * time.Millisecond}, nil)

	record, err := client.Submit(context.Background(), TaskRequest{
		AgentID:              "agent-origin",
		Method:               "query",
		RequiredCapabilities: []string{"math", "search"},
	})
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}
	if record.PeerID != "peer-2" {
		t.Fatalf("peer_id = %q, want peer-2", record.PeerID)
	}
}

func TestCallbackRetryBoundedAndReasonCode(t *testing.T) {
	collector := &timelineCollector{}
	server := NewInMemoryServer(HandlerFunc(func(ctx context.Context, req TaskRequest) (map[string]any, error) {
		return map[string]any{"ok": true}, nil
	}), nil)
	client := NewClient(server, nil, nil, ClientPolicy{
		Timeout:            300 * time.Millisecond,
		RequestMaxAttempts: 1,
		CallbackRetry: RetryPolicy{
			MaxAttempts: 3,
			Backoff:     5 * time.Millisecond,
		},
	}, collector)

	submitted, err := client.Submit(context.Background(), TaskRequest{
		AgentID: "agent-callback",
		PeerID:  "peer-callback",
		Method:  "notify",
	})
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	attempts := 0
	_, err = client.WaitResult(context.Background(), submitted.TaskID, 10*time.Millisecond, func(ctx context.Context, record TaskRecord) error {
		attempts++
		if attempts < 3 {
			return errors.New("callback temporary error")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WaitResult should succeed after retries: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("callback attempts = %d, want 3", attempts)
	}

	retryEvents := 0
	for _, ev := range collector.Snapshot() {
		if ev.Type != types.EventTypeActionTimeline {
			continue
		}
		reason, _ := ev.Payload["reason"].(string)
		if reason == ReasonCallbackRetry {
			retryEvents++
		}
	}
	if retryEvents != 2 {
		t.Fatalf("callback_retry events = %d, want 2", retryEvents)
	}
}

func TestClassifyErrorMapping(t *testing.T) {
	testCases := []struct {
		name      string
		err       error
		wantClass types.ErrorClass
		wantLayer ErrorLayer
		wantCode  string
	}{
		{
			name:      "timeout",
			err:       context.DeadlineExceeded,
			wantClass: types.ErrPolicyTimeout,
			wantLayer: ErrorLayerTransport,
			wantCode:  "timeout",
		},
		{
			name:      "unsupported_method",
			err:       errors.New("unsupported method"),
			wantClass: types.ErrMCP,
			wantLayer: ErrorLayerProtocol,
			wantCode:  "unsupported_method",
		},
		{
			name:      "invalid_payload",
			err:       errors.New("invalid schema"),
			wantClass: types.ErrContext,
			wantLayer: ErrorLayerSemantic,
			wantCode:  "invalid_payload",
		},
		{
			name:      "transport_failure",
			err:       errors.New("connection refused"),
			wantClass: types.ErrMCP,
			wantLayer: ErrorLayerTransport,
			wantCode:  "transport_failure",
		},
		{
			name:      "unknown",
			err:       errors.New("mystery"),
			wantClass: types.ErrMCP,
			wantLayer: ErrorLayerProtocol,
			wantCode:  "unknown",
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			gotClass, gotLayer, gotCode := ClassifyError(tc.err)
			if gotClass != tc.wantClass || gotLayer != tc.wantLayer || gotCode != tc.wantCode {
				t.Fatalf(
					"ClassifyError mismatch: got (%q,%q,%q), want (%q,%q,%q)",
					gotClass,
					gotLayer,
					gotCode,
					tc.wantClass,
					tc.wantLayer,
					tc.wantCode,
				)
			}
		})
	}
}

func TestRunAndStreamSemanticEquivalence(t *testing.T) {
	server := NewInMemoryServer(HandlerFunc(func(ctx context.Context, req TaskRequest) (map[string]any, error) {
		if strings.EqualFold(req.Method, "fail") {
			return nil, errors.New("unsupported method")
		}
		return map[string]any{"peer": req.PeerID, "method": req.Method}, nil
	}), nil)
	client := NewClient(server, nil, nil, ClientPolicy{
		Timeout:            300 * time.Millisecond,
		RequestMaxAttempts: 1,
	}, nil)

	runSubmitted, err := server.Submit(context.Background(), TaskRequest{
		TaskID:  "task-run",
		AgentID: "agent-run",
		PeerID:  "peer-1",
		Method:  "ok",
	})
	if err != nil {
		t.Fatalf("run-path submit failed: %v", err)
	}
	waitForTerminal(t, server, runSubmitted.TaskID)
	runResult, err := server.Result(context.Background(), runSubmitted.TaskID)
	if err != nil {
		t.Fatalf("run-path result failed: %v", err)
	}

	streamSubmitted, err := client.Submit(context.Background(), TaskRequest{
		TaskID:  "task-stream",
		AgentID: "agent-stream",
		PeerID:  "peer-1",
		Method:  "ok",
	})
	if err != nil {
		t.Fatalf("stream-path submit failed: %v", err)
	}
	streamResult, err := client.WaitResult(context.Background(), streamSubmitted.TaskID, 10*time.Millisecond, nil)
	if err != nil {
		t.Fatalf("stream-path result failed: %v", err)
	}

	if runResult.Status != streamResult.Status {
		t.Fatalf("status mismatch run=%q stream=%q", runResult.Status, streamResult.Status)
	}
	if runResult.ErrorClass != streamResult.ErrorClass {
		t.Fatalf("error_class mismatch run=%q stream=%q", runResult.ErrorClass, streamResult.ErrorClass)
	}
	if runResult.Result["method"] != streamResult.Result["method"] {
		t.Fatalf("result mismatch run=%#v stream=%#v", runResult.Result, streamResult.Result)
	}

	runSummary := BuildRunSummary([]TaskRecord{runResult})
	streamSummary := BuildRunSummary([]TaskRecord{streamResult})
	if runSummary.A2ATaskTotal != streamSummary.A2ATaskTotal ||
		runSummary.A2ATaskFailed != streamSummary.A2ATaskFailed ||
		runSummary.PeerID != streamSummary.PeerID {
		t.Fatalf("summary mismatch run=%#v stream=%#v", runSummary, streamSummary)
	}
}

func TestBuildRunSummaryReplayIdempotency(t *testing.T) {
	base := time.Now()
	tasks := []TaskRecord{
		{
			TaskID:        "task-1",
			AgentID:       "agent-a",
			PeerID:        "peer-z",
			Status:        StatusFailed,
			A2AErrorLayer: "transport",
			UpdatedAt:     base.Add(20 * time.Millisecond),
		},
		{
			TaskID:        "task-1",
			AgentID:       "agent-a",
			PeerID:        "peer-z",
			Status:        StatusFailed,
			A2AErrorLayer: "transport",
			UpdatedAt:     base.Add(20 * time.Millisecond),
		},
		{
			TaskID:    "task-2",
			AgentID:   "agent-b",
			PeerID:    "peer-z",
			Status:    StatusSucceeded,
			UpdatedAt: base.Add(30 * time.Millisecond),
		},
		{
			TaskID:    "task-2",
			AgentID:   "agent-b",
			PeerID:    "peer-z",
			Status:    StatusRunning,
			UpdatedAt: base.Add(10 * time.Millisecond),
		},
	}
	summary := BuildRunSummary(tasks)
	if summary.A2ATaskTotal != 2 {
		t.Fatalf("a2a_task_total = %d, want 2", summary.A2ATaskTotal)
	}
	if summary.A2ATaskFailed != 1 {
		t.Fatalf("a2a_task_failed = %d, want 1", summary.A2ATaskFailed)
	}
	if summary.PeerID != "peer-z" {
		t.Fatalf("peer_id = %q, want peer-z", summary.PeerID)
	}
	if summary.A2AErrorLayer != "transport" {
		t.Fatalf("a2a_error_layer = %q, want transport", summary.A2AErrorLayer)
	}
}
