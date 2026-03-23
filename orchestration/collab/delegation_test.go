package collab

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/a2a"
	"github.com/FelixSeptem/baymax/orchestration/invoke"
	"github.com/FelixSeptem/baymax/orchestration/mailbox"
)

type fakeSyncClient struct {
	submit func(context.Context, a2a.TaskRequest) (a2a.TaskRecord, error)
	wait   func(context.Context, string, time.Duration, func(context.Context, a2a.TaskRecord) error) (a2a.TaskRecord, error)
}

func (f fakeSyncClient) Submit(ctx context.Context, req a2a.TaskRequest) (a2a.TaskRecord, error) {
	if f.submit != nil {
		return f.submit(ctx, req)
	}
	return a2a.TaskRecord{TaskID: req.TaskID, Status: a2a.StatusSubmitted}, nil
}

func (f fakeSyncClient) WaitResult(
	ctx context.Context,
	taskID string,
	pollInterval time.Duration,
	callback func(context.Context, a2a.TaskRecord) error,
) (a2a.TaskRecord, error) {
	if f.wait != nil {
		return f.wait(ctx, taskID, pollInterval, callback)
	}
	return a2a.TaskRecord{TaskID: taskID, Status: a2a.StatusSucceeded, Result: map[string]any{"ok": true}}, nil
}

func TestDelegateSyncSuccessAndFailure(t *testing.T) {
	success, err := DelegateSync(context.Background(), fakeSyncClient{}, invoke.Request{
		TaskID: "task-success",
	})
	if err != nil {
		t.Fatalf("DelegateSync success failed: %v", err)
	}
	if success.Status != StatusSucceeded || success.Payload["ok"] != true {
		t.Fatalf("success outcome mismatch: %#v", success)
	}

	failed, err := DelegateSync(context.Background(), fakeSyncClient{
		wait: func(context.Context, string, time.Duration, func(context.Context, a2a.TaskRecord) error) (a2a.TaskRecord, error) {
			return a2a.TaskRecord{
				TaskID:       "task-failed",
				Status:       a2a.StatusFailed,
				ErrorMessage: "peer failed",
			}, nil
		},
	}, invoke.Request{TaskID: "task-failed"})
	if err != nil {
		t.Fatalf("DelegateSync failed terminal should not return hard error, got %v", err)
	}
	if failed.Status != StatusFailed || failed.Error == "" {
		t.Fatalf("failed outcome mismatch: %#v", failed)
	}
}

func TestDelegateSyncWithRetryTransportOnly(t *testing.T) {
	var calls atomic.Int32
	client := fakeSyncClient{
		wait: func(context.Context, string, time.Duration, func(context.Context, a2a.TaskRecord) error) (a2a.TaskRecord, error) {
			cur := calls.Add(1)
			if cur == 1 {
				return a2a.TaskRecord{}, errors.New("connection refused")
			}
			return a2a.TaskRecord{TaskID: "task-retry", Status: a2a.StatusSucceeded, Result: map[string]any{"ok": true}}, nil
		},
	}
	events := make([]RetryEvent, 0, 2)
	out, err := DelegateSyncWithRetry(context.Background(), client, invoke.Request{TaskID: "task-retry"}, RetryConfig{
		Enabled:        true,
		MaxAttempts:    3,
		BackoffInitial: time.Millisecond,
		BackoffMax:     2 * time.Millisecond,
		Multiplier:     2,
		JitterRatio:    0,
		RetryOn:        RetryOnTransportOnly,
	}, func(ev RetryEvent) { events = append(events, ev) })
	if err != nil {
		t.Fatalf("DelegateSyncWithRetry failed: %v", err)
	}
	if calls.Load() != 2 {
		t.Fatalf("sync calls=%d, want 2", calls.Load())
	}
	if out.Status != StatusSucceeded {
		t.Fatalf("sync status=%q, want succeeded", out.Status)
	}
	if out.Payload["collab_retry_attempts"] != 1 {
		t.Fatalf("collab_retry_attempts=%v, want 1", out.Payload["collab_retry_attempts"])
	}
	if len(events) != 2 || events[0].Type != RetryEventAttempt || events[1].Type != RetryEventSuccess {
		t.Fatalf("retry events mismatch: %#v", events)
	}
}

func TestDelegateSyncWithRetrySkipsNonRetryableFailures(t *testing.T) {
	var calls atomic.Int32
	client := fakeSyncClient{
		wait: func(context.Context, string, time.Duration, func(context.Context, a2a.TaskRecord) error) (a2a.TaskRecord, error) {
			calls.Add(1)
			return a2a.TaskRecord{}, errors.New("invalid payload")
		},
	}
	_, err := DelegateSyncWithRetry(context.Background(), client, invoke.Request{TaskID: "task-no-retry"}, RetryConfig{
		Enabled:        true,
		MaxAttempts:    3,
		BackoffInitial: time.Millisecond,
		BackoffMax:     2 * time.Millisecond,
		Multiplier:     2,
		JitterRatio:    0,
		RetryOn:        RetryOnTransportOnly,
	}, nil)
	if err == nil {
		t.Fatal("expected hard error")
	}
	if calls.Load() != 1 {
		t.Fatalf("non-retryable calls=%d, want 1", calls.Load())
	}
}

type fakeAsyncClient struct {
	submit func(context.Context, a2a.TaskRequest, a2a.ReportSink) (a2a.AsyncSubmitAck, error)
}

func (f fakeAsyncClient) Submit(ctx context.Context, req a2a.TaskRequest) (a2a.TaskRecord, error) {
	return a2a.TaskRecord{TaskID: req.TaskID, Status: a2a.StatusSubmitted}, nil
}

func (f fakeAsyncClient) WaitResult(context.Context, string, time.Duration, func(context.Context, a2a.TaskRecord) error) (a2a.TaskRecord, error) {
	return a2a.TaskRecord{}, errors.New("not implemented")
}

func (f fakeAsyncClient) SubmitAsync(ctx context.Context, req a2a.TaskRequest, sink a2a.ReportSink) (a2a.AsyncSubmitAck, error) {
	if f.submit != nil {
		return f.submit(ctx, req, sink)
	}
	return a2a.AsyncSubmitAck{
		TaskID:     req.TaskID,
		WorkflowID: req.WorkflowID,
		TeamID:     req.TeamID,
		StepID:     req.StepID,
		PeerID:     req.PeerID,
	}, nil
}

func TestDelegateAsync(t *testing.T) {
	ack, err := DelegateAsync(context.Background(), fakeAsyncClient{}, invoke.AsyncRequest{
		TaskID:     "ok",
		WorkflowID: "wf",
		TeamID:     "team",
		StepID:     "step",
		PeerID:     "peer",
	}, nil)
	if err != nil {
		t.Fatalf("DelegateAsync failed: %v", err)
	}
	if ack.TaskID != "ok" || ack.WorkflowID != "wf" || ack.TeamID != "team" || ack.StepID != "step" || ack.PeerID != "peer" {
		t.Fatalf("DelegateAsync ack mismatch: %#v", ack)
	}
	if _, err := DelegateAsync(context.Background(), fakeAsyncClient{
		submit: func(context.Context, a2a.TaskRequest, a2a.ReportSink) (a2a.AsyncSubmitAck, error) {
			return a2a.AsyncSubmitAck{}, errors.New("submit failed")
		},
	}, invoke.AsyncRequest{TaskID: "fail"}, nil); err == nil {
		t.Fatal("DelegateAsync should return error when submit fails")
	}
}

func TestDelegateAsyncWithRetryTransportOnlySubmit(t *testing.T) {
	var submits atomic.Int32
	client := fakeAsyncClient{submit: func(context.Context, a2a.TaskRequest, a2a.ReportSink) (a2a.AsyncSubmitAck, error) {
		cur := submits.Add(1)
		if cur == 1 {
			return a2a.AsyncSubmitAck{}, errors.New("connection reset by peer")
		}
		return a2a.AsyncSubmitAck{TaskID: "task-ok", WorkflowID: "wf", TeamID: "team", StepID: "step", PeerID: "peer"}, nil
	}}
	events := make([]RetryEvent, 0, 2)
	ack, err := DelegateAsyncWithRetry(context.Background(), client, invoke.AsyncRequest{
		TaskID:     "task-ok",
		WorkflowID: "wf",
		TeamID:     "team",
		StepID:     "step",
		PeerID:     "peer",
	}, nil, RetryConfig{
		Enabled:        true,
		MaxAttempts:    3,
		BackoffInitial: time.Millisecond,
		BackoffMax:     2 * time.Millisecond,
		Multiplier:     2,
		JitterRatio:    0,
		RetryOn:        RetryOnTransportOnly,
	}, func(ev RetryEvent) { events = append(events, ev) })
	if err != nil {
		t.Fatalf("DelegateAsyncWithRetry failed: %v", err)
	}
	if submits.Load() != 2 {
		t.Fatalf("async submit attempts=%d, want 2", submits.Load())
	}
	if ack.TaskID != "task-ok" {
		t.Fatalf("ack mismatch: %#v", ack)
	}
	if len(events) != 2 || events[0].Type != RetryEventAttempt || events[1].Type != RetryEventSuccess {
		t.Fatalf("retry events mismatch: %#v", events)
	}
}

func TestDelegateAsyncWithRetrySkipsProtocolFailures(t *testing.T) {
	var submits atomic.Int32
	client := fakeAsyncClient{submit: func(context.Context, a2a.TaskRequest, a2a.ReportSink) (a2a.AsyncSubmitAck, error) {
		submits.Add(1)
		return a2a.AsyncSubmitAck{}, errors.New("invalid request payload")
	}}
	if _, err := DelegateAsyncWithRetry(context.Background(), client, invoke.AsyncRequest{TaskID: "task-no-retry"}, nil, RetryConfig{
		Enabled:        true,
		MaxAttempts:    3,
		BackoffInitial: time.Millisecond,
		BackoffMax:     2 * time.Millisecond,
		Multiplier:     2,
		JitterRatio:    0,
		RetryOn:        RetryOnTransportOnly,
	}, nil); err == nil {
		t.Fatal("expected submit error")
	}
	if submits.Load() != 1 {
		t.Fatalf("protocol submit attempts=%d, want 1", submits.Load())
	}
}

func TestDelegateSyncWithRetryUsesInjectedMailboxBridgeProvider(t *testing.T) {
	mb, err := mailbox.New(mailbox.NewMemoryStore(mailbox.Policy{}))
	if err != nil {
		t.Fatalf("new mailbox failed: %v", err)
	}
	bridge := invoke.NewMailboxBridge(mb)
	providerCalls := 0
	_, err = DelegateSyncWithRetry(
		context.Background(),
		fakeSyncClient{},
		invoke.Request{TaskID: "task-provider", WorkflowID: "wf-provider", TeamID: "team-provider"},
		RetryConfig{Enabled: false},
		nil,
		WithMailboxBridgeProvider(func() (*invoke.MailboxBridge, error) {
			providerCalls++
			return bridge, nil
		}),
	)
	if err != nil {
		t.Fatalf("DelegateSyncWithRetry failed: %v", err)
	}
	if providerCalls != 1 {
		t.Fatalf("provider calls=%d, want 1", providerCalls)
	}
	page, err := mb.Query(context.Background(), mailbox.QueryRequest{TaskID: "task-provider"})
	if err != nil {
		t.Fatalf("mailbox query failed: %v", err)
	}
	if len(page.Items) != 2 {
		t.Fatalf("mailbox records len=%d, want 2", len(page.Items))
	}
}
