package teams

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/a2a"
	"github.com/FelixSeptem/baymax/orchestration/invoke"
)

type fakeA2AInvokeClient struct {
	submitFn func(context.Context, a2a.TaskRequest) (a2a.TaskRecord, error)
	waitFn   func(context.Context, string, time.Duration, func(context.Context, a2a.TaskRecord) error) (a2a.TaskRecord, error)
}

func (f fakeA2AInvokeClient) Submit(ctx context.Context, req a2a.TaskRequest) (a2a.TaskRecord, error) {
	if f.submitFn != nil {
		return f.submitFn(ctx, req)
	}
	return a2a.TaskRecord{TaskID: req.TaskID}, nil
}

func (f fakeA2AInvokeClient) WaitResult(
	ctx context.Context,
	taskID string,
	pollInterval time.Duration,
	callback func(context.Context, a2a.TaskRecord) error,
) (a2a.TaskRecord, error) {
	if f.waitFn != nil {
		return f.waitFn(ctx, taskID, pollInterval, callback)
	}
	return a2a.TaskRecord{
		TaskID: taskID,
		Status: a2a.StatusSucceeded,
		Result: map[string]any{"vote": "yes", "ok": true},
	}, nil
}

func TestNewA2ARemoteTaskRunnerSuccess(t *testing.T) {
	runner := NewA2ARemoteTaskRunner(fakeA2AInvokeClient{}, A2ARemoteRunnerOptions{
		PollInterval: 5 * time.Millisecond,
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
		t.Fatalf("remote runner failed: %v", err)
	}
	if out.Vote != "yes" {
		t.Fatalf("vote = %q, want yes", out.Vote)
	}
}

func TestNewA2ARemoteTaskRunnerFailedTerminalReturnsError(t *testing.T) {
	runner := NewA2ARemoteTaskRunner(fakeA2AInvokeClient{
		waitFn: func(context.Context, string, time.Duration, func(context.Context, a2a.TaskRecord) error) (a2a.TaskRecord, error) {
			return a2a.TaskRecord{
				Status:       a2a.StatusFailed,
				ErrorMessage: "unsupported method",
			}, nil
		},
	}, A2ARemoteRunnerOptions{})
	_, err := runner(context.Background(), Plan{TeamID: "team-1"}, Task{
		TaskID:  "task-1",
		AgentID: "agent-1",
		Remote:  RemoteTarget{PeerID: "peer-1"},
	})
	if err == nil {
		t.Fatal("expected failed terminal error")
	}
}

func TestNewA2ARemoteTaskRunnerContextCancel(t *testing.T) {
	runner := NewA2ARemoteTaskRunner(fakeA2AInvokeClient{
		waitFn: func(ctx context.Context, _ string, _ time.Duration, _ func(context.Context, a2a.TaskRecord) error) (a2a.TaskRecord, error) {
			<-ctx.Done()
			return a2a.TaskRecord{}, ctx.Err()
		},
	}, A2ARemoteRunnerOptions{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := runner(ctx, Plan{TeamID: "team-1"}, Task{
		TaskID:  "task-1",
		AgentID: "agent-1",
		Remote:  RemoteTarget{PeerID: "peer-1"},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled, got %v", err)
	}
}

var _ invoke.Client = fakeA2AInvokeClient{}
