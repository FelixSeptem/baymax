package scheduler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/a2a"
)

type fakeA2AClient struct {
	submitErr      error
	waitErr        error
	asyncSubmitErr error
	record         a2a.TaskRecord
	lastAsyncReq   a2a.TaskRequest
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

func (c *fakeA2AClient) SubmitAsync(ctx context.Context, req a2a.TaskRequest, sink a2a.ReportSink) (a2a.AsyncSubmitAck, error) {
	c.lastAsyncReq = req
	if c.asyncSubmitErr != nil {
		return a2a.AsyncSubmitAck{}, c.asyncSubmitErr
	}
	if sink != nil {
		_ = sink.Deliver(ctx, a2a.AsyncReport{
			ReportKey:        "report-key",
			WorkflowID:       req.WorkflowID,
			TeamID:           req.TeamID,
			StepID:           req.StepID,
			TaskID:           req.TaskID,
			AttemptID:        req.AttemptID,
			AgentID:          req.AgentID,
			PeerID:           req.PeerID,
			Status:           a2a.StatusSucceeded,
			Result:           map[string]any{"ok": true},
			BusinessTerminal: true,
			UpdatedAt:        time.Now(),
		})
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

func TestExecuteClaimWithA2ACanceledTerminalMapsToFailedDeterministically(t *testing.T) {
	claimed := ClaimedTask{
		Record:  TaskRecord{Task: Task{TaskID: "task-a2a-terminal-canceled"}},
		Attempt: Attempt{AttemptID: "task-a2a-terminal-canceled-attempt-1"},
	}
	result, err := ExecuteClaimWithA2A(context.Background(), fakeA2AClient{
		record: a2a.TaskRecord{
			TaskID:        "dummy",
			Status:        a2a.StatusCanceled,
			ErrorClass:    "ErrMCP",
			A2AErrorLayer: string(a2a.ErrorLayerProtocol),
			ErrorMessage:  "canceled by peer",
		},
	}, claimed, 5*time.Millisecond)
	if err != nil {
		t.Fatalf("canceled terminal should not return execution error, got %v", err)
	}
	if result.Commit.Status != TaskStateFailed {
		t.Fatalf("commit status = %q, want failed", result.Commit.Status)
	}
	if result.Retryable {
		t.Fatal("protocol-layer canceled terminal should not be retryable")
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

func TestSubmitClaimWithA2AAsyncPreservesAttemptCorrelation(t *testing.T) {
	claimed := ClaimedTask{
		Record: TaskRecord{
			Task: Task{
				TaskID:     "task-a2a-async",
				WorkflowID: "wf-a2a-async",
				TeamID:     "team-a2a-async",
				StepID:     "step-a2a-async",
				AgentID:    "agent-main",
				PeerID:     "peer-async",
				Payload:    map[string]any{"query": "hello"},
			},
		},
		Attempt: Attempt{AttemptID: "task-a2a-async-attempt-1"},
	}
	client := &fakeA2AClient{}
	ack, err := SubmitClaimWithA2AAsync(context.Background(), client, claimed, a2a.NewChannelReportSink(2))
	if err != nil {
		t.Fatalf("SubmitClaimWithA2AAsync failed: %v", err)
	}
	if ack.TaskID == "" {
		t.Fatalf("ack task_id should not be empty: %#v", ack)
	}
	if client.lastAsyncReq.AttemptID != claimed.Attempt.AttemptID {
		t.Fatalf("attempt_id = %q, want %q", client.lastAsyncReq.AttemptID, claimed.Attempt.AttemptID)
	}
	if got, _ := client.lastAsyncReq.Payload["attempt_id"].(string); got != claimed.Attempt.AttemptID {
		t.Fatalf("payload attempt_id = %q, want %q", got, claimed.Attempt.AttemptID)
	}
}

func TestExecutionFromAsyncReportMapsTerminalSemantics(t *testing.T) {
	claimed := ClaimedTask{
		Record: TaskRecord{
			Task: Task{
				TaskID:  "task-a2a-async-map",
				AgentID: "agent-main",
				PeerID:  "peer-main",
			},
		},
		Attempt: Attempt{AttemptID: "attempt-1"},
	}
	successExec, err := ExecutionFromAsyncReport(claimed, a2a.AsyncReport{
		ReportKey:  "rk-1",
		OutcomeKey: "ok",
		TaskID:     "task-a2a-async-map",
		AttemptID:  "attempt-1",
		Status:     a2a.StatusSucceeded,
		Result:     map[string]any{"ok": true},
	})
	if err != nil {
		t.Fatalf("ExecutionFromAsyncReport success failed: %v", err)
	}
	if successExec.Commit.Status != TaskStateSucceeded {
		t.Fatalf("commit status = %q, want succeeded", successExec.Commit.Status)
	}

	failedExec, err := ExecutionFromAsyncReport(claimed, a2a.AsyncReport{
		ReportKey:    "rk-2",
		OutcomeKey:   "failed|transport",
		TaskID:       "task-a2a-async-map",
		AttemptID:    "attempt-1",
		Status:       a2a.StatusFailed,
		ErrorClass:   "ErrMCP",
		ErrorLayer:   string(a2a.ErrorLayerTransport),
		ErrorMessage: "network timeout",
	})
	if err != nil {
		t.Fatalf("ExecutionFromAsyncReport failed terminal should not return err: %v", err)
	}
	if failedExec.Commit.Status != TaskStateFailed || !failedExec.Retryable {
		t.Fatalf("failed terminal mapping mismatch: %#v", failedExec)
	}
}
