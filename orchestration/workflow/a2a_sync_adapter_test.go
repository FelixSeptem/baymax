package workflow

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/a2a"
	"github.com/FelixSeptem/baymax/orchestration/invoke"
)

type fakeInvokeClient struct {
	submitFn func(context.Context, a2a.TaskRequest) (a2a.TaskRecord, error)
	waitFn   func(context.Context, string, time.Duration, func(context.Context, a2a.TaskRecord) error) (a2a.TaskRecord, error)
}

func (f fakeInvokeClient) Submit(ctx context.Context, req a2a.TaskRequest) (a2a.TaskRecord, error) {
	if f.submitFn != nil {
		return f.submitFn(ctx, req)
	}
	return a2a.TaskRecord{TaskID: req.TaskID}, nil
}

func (f fakeInvokeClient) WaitResult(ctx context.Context, taskID string, pollInterval time.Duration, callback func(context.Context, a2a.TaskRecord) error) (a2a.TaskRecord, error) {
	if f.waitFn != nil {
		return f.waitFn(ctx, taskID, pollInterval, callback)
	}
	return a2a.TaskRecord{TaskID: taskID, Status: a2a.StatusSucceeded, Result: map[string]any{"ok": true}}, nil
}

func TestNewA2AStepAdapterSuccess(t *testing.T) {
	adapter := NewA2AStepAdapter(fakeInvokeClient{}, A2AStepAdapterOptions{
		Method:       "workflow.delegate",
		PollInterval: 5 * time.Millisecond,
	})
	out, err := adapter(context.Background(), "wf-1", Step{
		StepID:  "step-1",
		TaskID:  "task-1",
		TeamID:  "team-1",
		AgentID: "agent-1",
		PeerID:  "peer-1",
		Payload: map[string]any{"q": "ping"},
	}, 1)
	if err != nil {
		t.Fatalf("adapter execute failed: %v", err)
	}
	if out.Payload["ok"] != true {
		t.Fatalf("unexpected payload: %#v", out.Payload)
	}
}

func TestNewA2AStepAdapterFailedTerminalReturnsError(t *testing.T) {
	adapter := NewA2AStepAdapter(fakeInvokeClient{
		waitFn: func(context.Context, string, time.Duration, func(context.Context, a2a.TaskRecord) error) (a2a.TaskRecord, error) {
			return a2a.TaskRecord{
				Status:       a2a.StatusFailed,
				ErrorMessage: "unsupported method",
			}, nil
		},
	}, A2AStepAdapterOptions{})
	_, err := adapter(context.Background(), "wf-1", Step{
		StepID:  "step-1",
		TaskID:  "task-1",
		AgentID: "agent-1",
		PeerID:  "peer-1",
	}, 1)
	if err == nil {
		t.Fatal("expected failed terminal error")
	}
}

func TestNewA2AStepAdapterContextCancellation(t *testing.T) {
	adapter := NewA2AStepAdapter(fakeInvokeClient{
		waitFn: func(ctx context.Context, _ string, _ time.Duration, _ func(context.Context, a2a.TaskRecord) error) (a2a.TaskRecord, error) {
			<-ctx.Done()
			return a2a.TaskRecord{}, ctx.Err()
		},
	}, A2AStepAdapterOptions{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := adapter(ctx, "wf-1", Step{
		StepID:  "step-1",
		TaskID:  "task-1",
		AgentID: "agent-1",
		PeerID:  "peer-1",
	}, 1)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled, got %v", err)
	}
}

func TestNewA2AStepAdapterUsesCustomTaskIDGenerator(t *testing.T) {
	var gotTaskID string
	client := fakeInvokeClient{
		submitFn: func(_ context.Context, req a2a.TaskRequest) (a2a.TaskRecord, error) {
			gotTaskID = req.TaskID
			return a2a.TaskRecord{TaskID: req.TaskID}, nil
		},
	}
	adapter := NewA2AStepAdapter(client, A2AStepAdapterOptions{
		TaskIDGenerator: func(step Step, attempt int) string {
			return step.TaskID + "-custom"
		},
	})
	_, err := adapter(context.Background(), "wf-1", Step{
		StepID:  "step-1",
		TaskID:  "task-1",
		AgentID: "agent-1",
		PeerID:  "peer-1",
	}, 1)
	if err != nil {
		t.Fatalf("adapter execute failed: %v", err)
	}
	if gotTaskID != "task-1-custom" {
		t.Fatalf("task id = %q, want task-1-custom", gotTaskID)
	}
}

var _ invoke.Client = fakeInvokeClient{}
