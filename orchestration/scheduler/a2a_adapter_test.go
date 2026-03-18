package scheduler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/a2a"
)

type fakeA2AClient struct {
	submitErr error
	waitErr   error
	record    a2a.TaskRecord
}

func (c fakeA2AClient) Submit(ctx context.Context, req a2a.TaskRequest) (a2a.TaskRecord, error) {
	if c.submitErr != nil {
		return a2a.TaskRecord{}, c.submitErr
	}
	return a2a.TaskRecord{TaskID: req.TaskID}, nil
}

func (c fakeA2AClient) WaitResult(
	_ context.Context,
	_ string,
	_ time.Duration,
	_ func(context.Context, a2a.TaskRecord) error,
) (a2a.TaskRecord, error) {
	if c.waitErr != nil {
		return a2a.TaskRecord{}, c.waitErr
	}
	return c.record, nil
}

func TestExecuteClaimWithA2ASuccess(t *testing.T) {
	server := a2a.NewInMemoryServer(a2a.HandlerFunc(func(ctx context.Context, req a2a.TaskRequest) (map[string]any, error) {
		return map[string]any{"ok": true, "peer": req.PeerID}, nil
	}), nil)
	client := a2a.NewClient(server, []a2a.AgentCard{
		{
			AgentID:                "agent-1",
			PeerID:                 "peer-1",
			SchemaVersion:          "a2a.v1.0",
			SupportedDeliveryModes: []string{a2a.DeliveryModeCallback},
		},
	}, a2a.DeterministicRouter{RequireAll: true}, a2a.ClientPolicy{
		Timeout:            300 * time.Millisecond,
		RequestMaxAttempts: 1,
	}, nil)

	claimed := ClaimedTask{
		Record: TaskRecord{
			Task: Task{
				TaskID:     "task-a2a-success",
				RunID:      "run-a2a-success",
				WorkflowID: "wf-a2a-success",
				TeamID:     "team-a2a-success",
				StepID:     "step-a2a-success",
				AgentID:    "agent-main",
				PeerID:     "peer-1",
				Payload:    map[string]any{"query": "hello"},
			},
		},
		Attempt: Attempt{AttemptID: "task-a2a-success-attempt-1"},
	}

	result, err := ExecuteClaimWithA2A(context.Background(), client, claimed, 10*time.Millisecond)
	if err != nil {
		t.Fatalf("execute claim with a2a: %v", err)
	}
	if result.Retryable {
		t.Fatal("successful execution should not be retryable")
	}
	if result.Commit.Status != TaskStateSucceeded {
		t.Fatalf("commit status = %q, want succeeded", result.Commit.Status)
	}
	if got := result.Commit.Result["ok"]; got != true {
		t.Fatalf("commit result ok = %#v, want true", got)
	}
}

func TestExecuteClaimWithA2ASubmitTransportErrorIsRetryable(t *testing.T) {
	claimed := ClaimedTask{
		Record:  TaskRecord{Task: Task{TaskID: "task-a2a-submit-fail"}},
		Attempt: Attempt{AttemptID: "task-a2a-submit-fail-attempt-1"},
	}
	result, err := ExecuteClaimWithA2A(context.Background(), fakeA2AClient{
		submitErr: context.DeadlineExceeded,
	}, claimed, 5*time.Millisecond)
	if err == nil {
		t.Fatal("expected submit error")
	}
	if !result.Retryable {
		t.Fatal("transport error should be retryable")
	}
	if result.Commit.Status != TaskStateFailed {
		t.Fatalf("commit status = %q, want failed", result.Commit.Status)
	}
	if result.Commit.ErrorLayer != string(a2a.ErrorLayerTransport) {
		t.Fatalf("error layer = %q, want transport", result.Commit.ErrorLayer)
	}
}

func TestExecuteClaimWithA2AFailedTerminalMapsErrorLayer(t *testing.T) {
	claimed := ClaimedTask{
		Record:  TaskRecord{Task: Task{TaskID: "task-a2a-terminal-fail"}},
		Attempt: Attempt{AttemptID: "task-a2a-terminal-fail-attempt-1"},
	}
	result, err := ExecuteClaimWithA2A(context.Background(), fakeA2AClient{
		record: a2a.TaskRecord{
			TaskID:        "dummy",
			Status:        a2a.StatusFailed,
			ErrorClass:    "ErrMCP",
			A2AErrorLayer: string(a2a.ErrorLayerProtocol),
			ErrorMessage:  "unsupported method",
		},
	}, claimed, 5*time.Millisecond)
	if err != nil {
		t.Fatalf("failed terminal should not return execution error, got %v", err)
	}
	if result.Retryable {
		t.Fatal("protocol-layer failure should not be retryable")
	}
	if result.Commit.ErrorLayer != string(a2a.ErrorLayerProtocol) {
		t.Fatalf("error layer = %q, want protocol", result.Commit.ErrorLayer)
	}
}

func TestFailedExecutionFromA2AErrorFallbackClassification(t *testing.T) {
	claimed := ClaimedTask{
		Record:  TaskRecord{Task: Task{TaskID: "task-a2a-fallback"}},
		Attempt: Attempt{AttemptID: "task-a2a-fallback-attempt-1"},
	}
	result := failedExecutionFromA2AError(claimed, errors.New("unsupported method"))
	if result.Commit.Status != TaskStateFailed {
		t.Fatalf("status = %q, want failed", result.Commit.Status)
	}
	if result.Commit.ErrorLayer != string(a2a.ErrorLayerProtocol) {
		t.Fatalf("error layer = %q, want protocol", result.Commit.ErrorLayer)
	}
	if result.Retryable {
		t.Fatal("protocol failure should not be retryable")
	}
}
