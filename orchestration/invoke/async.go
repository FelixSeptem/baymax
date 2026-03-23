package invoke

import (
	"context"
	"errors"
	"strings"

	"github.com/FelixSeptem/baymax/a2a"
)

type AsyncClient interface {
	SubmitAsync(ctx context.Context, req a2a.TaskRequest, sink a2a.ReportSink) (a2a.AsyncSubmitAck, error)
}

type AsyncRequest struct {
	TaskID     string
	WorkflowID string
	TeamID     string
	StepID     string
	AttemptID  string
	AgentID    string
	PeerID     string
	Method     string
	Payload    map[string]any
}

// invokeAsync executes async submit+report-sink invocation for MailboxBridge canonical entrypoints.
func invokeAsync(ctx context.Context, client AsyncClient, req AsyncRequest, sink a2a.ReportSink) (a2a.AsyncSubmitAck, error) {
	if client == nil {
		return a2a.AsyncSubmitAck{}, errors.New("a2a async client is required")
	}
	submitReq := a2a.TaskRequest{
		TaskID:     strings.TrimSpace(req.TaskID),
		WorkflowID: strings.TrimSpace(req.WorkflowID),
		TeamID:     strings.TrimSpace(req.TeamID),
		StepID:     strings.TrimSpace(req.StepID),
		AttemptID:  strings.TrimSpace(req.AttemptID),
		AgentID:    strings.TrimSpace(req.AgentID),
		PeerID:     strings.TrimSpace(req.PeerID),
		Method:     strings.TrimSpace(req.Method),
		Payload:    cloneMap(req.Payload),
	}
	if submitReq.TaskID == "" {
		return a2a.AsyncSubmitAck{}, errors.New("task_id is required")
	}
	return client.SubmitAsync(ctx, submitReq, sink)
}
