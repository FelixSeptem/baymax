package workflow

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/a2a"
	"github.com/FelixSeptem/baymax/orchestration/invoke"
)

type fakeAsyncInvokeClient struct {
	submitFn func(context.Context, a2a.TaskRequest, a2a.ReportSink) (a2a.AsyncSubmitAck, error)
}

func (f fakeAsyncInvokeClient) SubmitAsync(ctx context.Context, req a2a.TaskRequest, sink a2a.ReportSink) (a2a.AsyncSubmitAck, error) {
	if f.submitFn != nil {
		return f.submitFn(ctx, req, sink)
	}
	return a2a.AsyncSubmitAck{
		TaskID:     req.TaskID,
		WorkflowID: req.WorkflowID,
		TeamID:     req.TeamID,
		StepID:     req.StepID,
		AgentID:    req.AgentID,
		PeerID:     req.PeerID,
		AcceptedAt: time.Now(),
	}, nil
}

func TestNewA2AAsyncStepAdapterSuccess(t *testing.T) {
	var gotTaskID string
	adapter := NewA2AAsyncStepAdapter(fakeAsyncInvokeClient{
		submitFn: func(_ context.Context, req a2a.TaskRequest, _ a2a.ReportSink) (a2a.AsyncSubmitAck, error) {
			gotTaskID = req.TaskID
			return a2a.AsyncSubmitAck{
				TaskID:     req.TaskID,
				WorkflowID: req.WorkflowID,
				TeamID:     req.TeamID,
				StepID:     req.StepID,
				PeerID:     req.PeerID,
				AcceptedAt: time.Now(),
			}, nil
		},
	}, A2AAsyncStepAdapterOptions{
		TaskIDGenerator: func(step Step, attempt int) string {
			return step.TaskID + "-async"
		},
	})
	out, err := adapter(context.Background(), "wf-async", Step{
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
	if gotTaskID != "task-1-async" {
		t.Fatalf("task id = %q, want task-1-async", gotTaskID)
	}
	if out.Payload["async_accepted"] != true {
		t.Fatalf("async_accepted = %#v, want true", out.Payload["async_accepted"])
	}
	if out.Payload["async_task_id"] != "task-1-async" {
		t.Fatalf("async_task_id = %#v, want task-1-async", out.Payload["async_task_id"])
	}
}

func TestNewA2AAsyncStepAdapterErrors(t *testing.T) {
	adapter := NewA2AAsyncStepAdapter(nil, A2AAsyncStepAdapterOptions{})
	if _, err := adapter(context.Background(), "wf-1", Step{
		StepID:  "step-1",
		TaskID:  "task-1",
		AgentID: "agent-1",
		PeerID:  "peer-1",
	}, 1); err == nil {
		t.Fatal("expected error for nil client")
	}
	adapter = NewA2AAsyncStepAdapter(fakeAsyncInvokeClient{
		submitFn: func(context.Context, a2a.TaskRequest, a2a.ReportSink) (a2a.AsyncSubmitAck, error) {
			return a2a.AsyncSubmitAck{}, errors.New("submit async failed")
		},
	}, A2AAsyncStepAdapterOptions{})
	if _, err := adapter(context.Background(), "wf-1", Step{
		StepID:  "step-1",
		TaskID:  "task-1",
		AgentID: "agent-1",
		PeerID:  "peer-1",
	}, 1); err == nil {
		t.Fatal("expected async submit error")
	}
}

var _ invoke.AsyncClient = fakeAsyncInvokeClient{}
