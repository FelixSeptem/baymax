package collab

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/a2a"
	"github.com/FelixSeptem/baymax/orchestration/invoke"
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

type fakeAsyncClient struct{}

func (fakeAsyncClient) SubmitAsync(ctx context.Context, req a2a.TaskRequest, sink a2a.ReportSink) (a2a.AsyncSubmitAck, error) {
	if req.TaskID == "fail" {
		return a2a.AsyncSubmitAck{}, errors.New("submit failed")
	}
	return a2a.AsyncSubmitAck{
		TaskID:     req.TaskID,
		WorkflowID: req.WorkflowID,
		TeamID:     req.TeamID,
		StepID:     req.StepID,
		PeerID:     req.PeerID,
		AcceptedAt: time.Now(),
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
	if _, err := DelegateAsync(context.Background(), fakeAsyncClient{}, invoke.AsyncRequest{TaskID: "fail"}, nil); err == nil {
		t.Fatal("DelegateAsync should return error when submit fails")
	}
}
