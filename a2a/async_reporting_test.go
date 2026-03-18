package a2a

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestSubmitAsyncDeliversReportViaChannelSink(t *testing.T) {
	collector := &timelineCollector{}
	server := NewInMemoryServer(HandlerFunc(func(ctx context.Context, req TaskRequest) (map[string]any, error) {
		return map[string]any{"ok": true, "attempt_id": "attempt-1"}, nil
	}), collector)
	client := NewClient(server, nil, nil, ClientPolicy{
		Timeout:            300 * time.Millisecond,
		RequestMaxAttempts: 1,
		AsyncReporting: AsyncReportingPolicy{
			Enabled:  true,
			Sink:     AsyncReportSinkChannel,
			SinkImpl: NewChannelReportSink(8),
			Retry: AsyncReportingRetryPolicy{
				MaxAttempts:    2,
				BackoffInitial: time.Millisecond,
				BackoffMax:     2 * time.Millisecond,
			},
			JitterRatio: 0,
		},
	}, collector)

	sink := client.policy.AsyncReporting.SinkImpl.(*ChannelReportSink)
	ack, err := client.SubmitAsync(context.Background(), TaskRequest{
		WorkflowID: "wf-async",
		TeamID:     "team-async",
		StepID:     "step-async",
		AgentID:    "agent-async",
		PeerID:     "peer-async",
		Method:     "async.dispatch",
	}, nil)
	if err != nil {
		t.Fatalf("SubmitAsync failed: %v", err)
	}
	if ack.TaskID == "" || ack.AcceptedAt.IsZero() {
		t.Fatalf("invalid async ack: %#v", ack)
	}

	select {
	case report := <-sink.Channel():
		if report.TaskID != ack.TaskID {
			t.Fatalf("report task_id = %q, want %q", report.TaskID, ack.TaskID)
		}
		if report.Status != StatusSucceeded {
			t.Fatalf("report status = %q, want succeeded", report.Status)
		}
		if report.WorkflowID != "wf-async" || report.TeamID != "team-async" || report.StepID != "step-async" {
			t.Fatalf("correlation fields mismatch: %#v", report)
		}
		if report.AttemptID != "attempt-1" {
			t.Fatalf("attempt_id = %q, want attempt-1", report.AttemptID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for async report")
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		reasons := map[string]int{}
		for _, ev := range collector.Snapshot() {
			reason, _ := ev.Payload["reason"].(string)
			reasons[reason]++
		}
		if reasons[ReasonAsyncSubmit] > 0 && reasons[ReasonAsyncReportDeliver] > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("missing timeline reasons %q and/or %q", ReasonAsyncSubmit, ReasonAsyncReportDeliver)
}

func TestSubmitAsyncRetriesReportDeliveryThenSucceeds(t *testing.T) {
	collector := &timelineCollector{}
	server := NewInMemoryServer(HandlerFunc(func(ctx context.Context, req TaskRequest) (map[string]any, error) {
		return map[string]any{"ok": true}, nil
	}), collector)
	var attempts atomic.Int32
	callback := NewCallbackReportSink(func(ctx context.Context, report AsyncReport) error {
		if attempts.Add(1) < 3 {
			return &AsyncReportDeliveryError{
				Cause:     errors.New("transient sink error"),
				Retryable: true,
			}
		}
		return nil
	})
	client := NewClient(server, nil, nil, ClientPolicy{
		Timeout:            300 * time.Millisecond,
		RequestMaxAttempts: 1,
		AsyncReporting: AsyncReportingPolicy{
			Enabled:  true,
			Sink:     AsyncReportSinkCallback,
			SinkImpl: callback,
			Retry: AsyncReportingRetryPolicy{
				MaxAttempts:    3,
				BackoffInitial: time.Millisecond,
				BackoffMax:     3 * time.Millisecond,
			},
			JitterRatio: 0,
		},
	}, collector)

	if _, err := client.SubmitAsync(context.Background(), TaskRequest{
		AgentID: "agent-retry",
		PeerID:  "peer-retry",
		Method:  "retry.dispatch",
	}, nil); err != nil {
		t.Fatalf("SubmitAsync failed: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && attempts.Load() < 3 {
		time.Sleep(10 * time.Millisecond)
	}
	if got := attempts.Load(); got != 3 {
		t.Fatalf("callback attempts = %d, want 3", got)
	}
	retryEvents := 0
	deliverEvents := 0
	for _, ev := range collector.Snapshot() {
		reason, _ := ev.Payload["reason"].(string)
		switch reason {
		case ReasonAsyncReportRetry:
			retryEvents++
		case ReasonAsyncReportDeliver:
			deliverEvents++
		}
	}
	if retryEvents != 2 {
		t.Fatalf("retry events = %d, want 2", retryEvents)
	}
	if deliverEvents == 0 {
		t.Fatalf("missing timeline reason %q", ReasonAsyncReportDeliver)
	}
}

func TestSubmitAsyncDropsReportAfterRetryExhausted(t *testing.T) {
	collector := &timelineCollector{}
	server := NewInMemoryServer(HandlerFunc(func(ctx context.Context, req TaskRequest) (map[string]any, error) {
		return map[string]any{"ok": true}, nil
	}), collector)
	client := NewClient(server, nil, nil, ClientPolicy{
		Timeout:            300 * time.Millisecond,
		RequestMaxAttempts: 1,
		AsyncReporting: AsyncReportingPolicy{
			Enabled: true,
			Sink:    AsyncReportSinkCallback,
			SinkImpl: NewCallbackReportSink(func(ctx context.Context, report AsyncReport) error {
				return &AsyncReportDeliveryError{
					Cause:     errors.New("hard fail"),
					Retryable: false,
				}
			}),
			Retry: AsyncReportingRetryPolicy{
				MaxAttempts:    2,
				BackoffInitial: time.Millisecond,
				BackoffMax:     2 * time.Millisecond,
			},
			JitterRatio: 0,
		},
	}, collector)

	if _, err := client.SubmitAsync(context.Background(), TaskRequest{
		AgentID: "agent-drop",
		PeerID:  "peer-drop",
		Method:  "drop.dispatch",
	}, nil); err != nil {
		t.Fatalf("SubmitAsync failed: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		foundDrop := false
		for _, ev := range collector.Snapshot() {
			reason, _ := ev.Payload["reason"].(string)
			if reason == ReasonAsyncReportDrop {
				foundDrop = true
				break
			}
		}
		if foundDrop {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("missing timeline reason %q", ReasonAsyncReportDrop)
}

func TestAsyncReportDeliveryDedupConverges(t *testing.T) {
	collector := &timelineCollector{}
	client := NewClient(nil, nil, nil, ClientPolicy{
		AsyncReporting: AsyncReportingPolicy{
			Retry: AsyncReportingRetryPolicy{
				MaxAttempts:    1,
				BackoffInitial: 0,
				BackoffMax:     0,
			},
			JitterRatio: 0,
		},
	}, collector)
	sink := NewChannelReportSink(8)
	report := BuildAsyncReport(TaskRecord{
		TaskID:       "task-dedup",
		WorkflowID:   "wf-dedup",
		TeamID:       "team-dedup",
		StepID:       "step-dedup",
		AgentID:      "agent-dedup",
		PeerID:       "peer-dedup",
		Status:       StatusSucceeded,
		Result:       map[string]any{"ok": true},
		ErrorMessage: "",
		UpdatedAt:    time.Now(),
	})
	client.deliverAsyncReport(context.Background(), sink, report)
	client.deliverAsyncReport(context.Background(), sink, report)

	select {
	case <-sink.Channel():
	case <-time.After(time.Second):
		t.Fatal("first delivery missing")
	}
	select {
	case second := <-sink.Channel():
		t.Fatalf("unexpected second delivery: %#v", second)
	case <-time.After(80 * time.Millisecond):
	}

	dedupEvents := 0
	for _, ev := range collector.Snapshot() {
		reason, _ := ev.Payload["reason"].(string)
		if reason == ReasonAsyncReportDedup {
			dedupEvents++
		}
	}
	if dedupEvents == 0 {
		t.Fatalf("missing timeline reason %q", ReasonAsyncReportDedup)
	}
}

func TestAsyncReportFailureDoesNotMutateBusinessTerminalStatus(t *testing.T) {
	server := NewInMemoryServer(HandlerFunc(func(ctx context.Context, req TaskRequest) (map[string]any, error) {
		return map[string]any{"ok": true}, nil
	}), nil)
	client := NewClient(server, nil, nil, ClientPolicy{
		Timeout:            300 * time.Millisecond,
		RequestMaxAttempts: 1,
		AsyncReporting: AsyncReportingPolicy{
			Enabled: true,
			Sink:    AsyncReportSinkCallback,
			SinkImpl: NewCallbackReportSink(func(ctx context.Context, report AsyncReport) error {
				return &AsyncReportDeliveryError{
					Cause:     errors.New("sink offline"),
					Retryable: false,
				}
			}),
			Retry: AsyncReportingRetryPolicy{
				MaxAttempts:    1,
				BackoffInitial: 0,
				BackoffMax:     0,
			},
			JitterRatio: 0,
		},
	}, nil)

	ack, err := client.SubmitAsync(context.Background(), TaskRequest{
		AgentID: "agent-business",
		PeerID:  "peer-business",
		Method:  "business.ok",
	}, nil)
	if err != nil {
		t.Fatalf("SubmitAsync failed: %v", err)
	}
	terminal := waitForTerminal(t, server, ack.TaskID)
	if terminal.Status != StatusSucceeded {
		t.Fatalf("business terminal status = %q, want succeeded", terminal.Status)
	}
}
