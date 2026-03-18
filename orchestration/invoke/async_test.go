package invoke

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/a2a"
)

type fakeAsyncClient struct {
	submitAsyncFn func(context.Context, a2a.TaskRequest, a2a.ReportSink) (a2a.AsyncSubmitAck, error)
}

func (f fakeAsyncClient) SubmitAsync(ctx context.Context, req a2a.TaskRequest, sink a2a.ReportSink) (a2a.AsyncSubmitAck, error) {
	if f.submitAsyncFn != nil {
		return f.submitAsyncFn(ctx, req, sink)
	}
	return a2a.AsyncSubmitAck{TaskID: req.TaskID, AcceptedAt: time.Now()}, nil
}

func TestInvokeAsyncSuccess(t *testing.T) {
	var captured a2a.TaskRequest
	sink := a2a.NewChannelReportSink(1)
	client := fakeAsyncClient{
		submitAsyncFn: func(_ context.Context, req a2a.TaskRequest, gotSink a2a.ReportSink) (a2a.AsyncSubmitAck, error) {
			captured = req
			if gotSink == nil {
				t.Fatal("expected non-nil report sink")
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
		},
	}

	ack, err := InvokeAsync(context.Background(), client, AsyncRequest{
		TaskID:     "task-async",
		WorkflowID: "wf-async",
		TeamID:     "team-async",
		StepID:     "step-async",
		AttemptID:  "attempt-1",
		AgentID:    "agent-async",
		PeerID:     "peer-async",
		Method:     "workflow.dispatch",
		Payload:    map[string]any{"q": "ping"},
	}, sink)
	if err != nil {
		t.Fatalf("InvokeAsync failed: %v", err)
	}
	if ack.TaskID != "task-async" {
		t.Fatalf("ack task_id = %q, want task-async", ack.TaskID)
	}
	if captured.AttemptID != "attempt-1" {
		t.Fatalf("captured attempt_id = %q, want attempt-1", captured.AttemptID)
	}
}

func TestInvokeAsyncValidationAndErrorPropagation(t *testing.T) {
	if _, err := InvokeAsync(context.Background(), nil, AsyncRequest{TaskID: "x"}, nil); err == nil {
		t.Fatal("expected error when client is nil")
	}
	if _, err := InvokeAsync(context.Background(), fakeAsyncClient{}, AsyncRequest{}, nil); err == nil {
		t.Fatal("expected error for missing task_id")
	}
	expected := errors.New("submit async failed")
	_, err := InvokeAsync(context.Background(), fakeAsyncClient{
		submitAsyncFn: func(context.Context, a2a.TaskRequest, a2a.ReportSink) (a2a.AsyncSubmitAck, error) {
			return a2a.AsyncSubmitAck{}, expected
		},
	}, AsyncRequest{TaskID: "task-err"}, nil)
	if !errors.Is(err, expected) {
		t.Fatalf("expected propagated error %v, got %v", expected, err)
	}
}
