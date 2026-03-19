package collab

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/FelixSeptem/baymax/a2a"
	"github.com/FelixSeptem/baymax/orchestration/invoke"
)

type DelegationRequest struct {
	TaskID       string
	WorkflowID   string
	TeamID       string
	StepID       string
	AttemptID    string
	AgentID      string
	PeerID       string
	Method       string
	Payload      map[string]any
	PollInterval int64
}

type DelegationAsyncAck struct {
	TaskID     string
	WorkflowID string
	TeamID     string
	StepID     string
	PeerID     string
}

func DelegateSync(ctx context.Context, client invoke.Client, req invoke.Request) (Outcome, error) {
	if client == nil {
		err := errors.New("a2a client is not configured")
		return Outcome{Status: StatusFailed, Error: err.Error()}, err
	}
	outcome, err := invoke.InvokeSync(ctx, client, req)
	if err != nil {
		return Outcome{
			Status:    StatusFailed,
			Retryable: outcome.Error != nil && outcome.Error.Retryable,
			Error:     normalizeDelegationError(err, outcome),
		}, err
	}
	if outcome.TerminalStatus == a2a.StatusSucceeded {
		return Outcome{
			Status:  StatusSucceeded,
			Payload: cloneMap(outcome.Result),
		}, nil
	}
	normalized := normalizeDelegationError(nil, outcome)
	if normalized == "" {
		normalized = fmt.Sprintf("a2a task status %q", outcome.TerminalStatus)
	}
	return Outcome{
		Status:    NormalizeStatus(Status(outcome.TerminalStatus)),
		Retryable: outcome.Error != nil && outcome.Error.Retryable,
		Error:     normalized,
		Payload:   cloneMap(outcome.Result),
	}, nil
}

func DelegateAsync(ctx context.Context, client invoke.AsyncClient, req invoke.AsyncRequest, sink a2a.ReportSink) (DelegationAsyncAck, error) {
	ack, err := invoke.InvokeAsync(ctx, client, req, sink)
	if err != nil {
		return DelegationAsyncAck{}, err
	}
	return DelegationAsyncAck{
		TaskID:     strings.TrimSpace(ack.TaskID),
		WorkflowID: strings.TrimSpace(ack.WorkflowID),
		TeamID:     strings.TrimSpace(ack.TeamID),
		StepID:     strings.TrimSpace(ack.StepID),
		PeerID:     strings.TrimSpace(ack.PeerID),
	}, nil
}

func normalizeDelegationError(err error, outcome invoke.Outcome) string {
	if err != nil {
		return strings.TrimSpace(err.Error())
	}
	if outcome.Error != nil && strings.TrimSpace(outcome.Error.Message) != "" {
		return strings.TrimSpace(outcome.Error.Message)
	}
	return ""
}

func cloneMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
