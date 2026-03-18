package workflow

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/FelixSeptem/baymax/a2a"
	"github.com/FelixSeptem/baymax/orchestration/invoke"
)

type A2AAsyncStepAdapterOptions struct {
	Method          string
	TaskIDGenerator func(step Step, attempt int) string
	ReportSink      a2a.ReportSink
}

func NewA2AAsyncStepAdapter(client invoke.AsyncClient, opts A2AAsyncStepAdapterOptions) func(context.Context, string, Step, int) (StepOutput, error) {
	method := strings.TrimSpace(opts.Method)
	if method == "" {
		method = "workflow.dispatch"
	}
	return func(ctx context.Context, workflowID string, step Step, attempt int) (StepOutput, error) {
		if client == nil {
			return StepOutput{}, errors.New("a2a async client is not configured")
		}
		taskID := ""
		if opts.TaskIDGenerator != nil {
			taskID = strings.TrimSpace(opts.TaskIDGenerator(step, attempt))
		}
		if taskID == "" {
			baseID := strings.TrimSpace(step.TaskID)
			if baseID == "" {
				baseID = strings.TrimSpace(step.StepID)
			}
			taskID = fmt.Sprintf("%s-attempt-%d-%d", baseID, attempt, time.Now().UnixNano())
		}
		ack, err := invoke.InvokeAsync(ctx, client, invoke.AsyncRequest{
			TaskID:     taskID,
			WorkflowID: strings.TrimSpace(workflowID),
			TeamID:     strings.TrimSpace(step.TeamID),
			StepID:     strings.TrimSpace(step.StepID),
			AttemptID:  fmt.Sprintf("%d", attempt),
			AgentID:    strings.TrimSpace(step.AgentID),
			PeerID:     strings.TrimSpace(step.PeerID),
			Method:     method,
			Payload:    clonePayload(step.Payload),
		}, opts.ReportSink)
		if err != nil {
			return StepOutput{}, err
		}
		return StepOutput{
			Payload: map[string]any{
				"async_accepted": true,
				"async_task_id":  strings.TrimSpace(ack.TaskID),
				"workflow_id":    strings.TrimSpace(ack.WorkflowID),
				"team_id":        strings.TrimSpace(ack.TeamID),
				"step_id":        strings.TrimSpace(ack.StepID),
				"peer_id":        strings.TrimSpace(ack.PeerID),
			},
		}, nil
	}
}
