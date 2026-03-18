package teams

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/a2a"
	"github.com/FelixSeptem/baymax/orchestration/invoke"
)

type fakeAsyncTeamsClient struct {
	submitFn func(context.Context, a2a.TaskRequest, a2a.ReportSink) (a2a.AsyncSubmitAck, error)
}

func (f fakeAsyncTeamsClient) SubmitAsync(ctx context.Context, req a2a.TaskRequest, sink a2a.ReportSink) (a2a.AsyncSubmitAck, error) {
	if f.submitFn != nil {
		return f.submitFn(ctx, req, sink)
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

func TestNewA2AAsyncRemoteTaskRunnerSuccess(t *testing.T) {
	var capturedTaskID string
	runner := NewA2AAsyncRemoteTaskRunner(fakeAsyncTeamsClient{
		submitFn: func(_ context.Context, req a2a.TaskRequest, _ a2a.ReportSink) (a2a.AsyncSubmitAck, error) {
			capturedTaskID = req.TaskID
			return a2a.AsyncSubmitAck{
				TaskID:     req.TaskID,
				WorkflowID: req.WorkflowID,
				TeamID:     req.TeamID,
				StepID:     req.StepID,
				PeerID:     req.PeerID,
				AcceptedAt: time.Now(),
			}, nil
		},
	}, A2AAsyncRemoteRunnerOptions{
		TaskIDGenerator: func(plan Plan, task Task) string {
			return task.TaskID + "-async"
		},
	})

	out, err := runner(context.Background(), Plan{
		TeamID:     "team-1",
		WorkflowID: "wf-1",
		StepID:     "step-1",
	}, Task{
		TaskID:  "task-1",
		AgentID: "agent-1",
		Target:  TaskTargetRemote,
		Remote: RemoteTarget{
			PeerID: "peer-1",
			Method: "team.delegate",
			Payload: map[string]any{
				"intent": "review",
			},
		},
	})
	if err != nil {
		t.Fatalf("async remote runner failed: %v", err)
	}
	if capturedTaskID != "task-1-async" {
		t.Fatalf("task id = %q, want task-1-async", capturedTaskID)
	}
	output, ok := out.Output.(map[string]any)
	if !ok {
		t.Fatalf("output type = %T, want map[string]any", out.Output)
	}
	if output["async_accepted"] != true || output["async_task_id"] != "task-1-async" {
		t.Fatalf("unexpected async output: %#v", out.Output)
	}
}

func TestNewA2AAsyncRemoteTaskRunnerErrors(t *testing.T) {
	runner := NewA2AAsyncRemoteTaskRunner(nil, A2AAsyncRemoteRunnerOptions{})
	if _, err := runner(context.Background(), Plan{TeamID: "team-1"}, Task{
		TaskID:  "task-1",
		AgentID: "agent-1",
		Remote:  RemoteTarget{PeerID: "peer-1"},
	}); err == nil {
		t.Fatal("expected error for nil async client")
	}
	runner = NewA2AAsyncRemoteTaskRunner(fakeAsyncTeamsClient{
		submitFn: func(context.Context, a2a.TaskRequest, a2a.ReportSink) (a2a.AsyncSubmitAck, error) {
			return a2a.AsyncSubmitAck{}, errors.New("submit async failed")
		},
	}, A2AAsyncRemoteRunnerOptions{})
	if _, err := runner(context.Background(), Plan{TeamID: "team-1"}, Task{
		TaskID:  "task-1",
		AgentID: "agent-1",
		Remote:  RemoteTarget{PeerID: "peer-1"},
	}); err == nil {
		t.Fatal("expected async submit error")
	}
}

var _ invoke.AsyncClient = fakeAsyncTeamsClient{}
