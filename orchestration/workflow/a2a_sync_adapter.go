package workflow

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/FelixSeptem/baymax/orchestration/collab"
	"github.com/FelixSeptem/baymax/orchestration/invoke"
)

var generatedA2ATaskCounter uint64

type A2AStepAdapterOptions struct {
	PollInterval    time.Duration
	Method          string
	TaskIDGenerator func(step Step, attempt int) string
	Retry           collab.RetryConfig
	RetryObserver   collab.RetryObserver
}

func NewA2AStepAdapter(client invoke.Client, opts A2AStepAdapterOptions) func(context.Context, string, Step, int) (StepOutput, error) {
	method := strings.TrimSpace(opts.Method)
	if method == "" {
		method = "workflow.dispatch"
	}
	return func(ctx context.Context, workflowID string, step Step, attempt int) (StepOutput, error) {
		if client == nil {
			return StepOutput{}, errors.New("a2a client is not configured")
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
			taskID = fmt.Sprintf(
				"%s-attempt-%d-%d-%d",
				baseID,
				attempt,
				time.Now().UnixNano(),
				atomic.AddUint64(&generatedA2ATaskCounter, 1),
			)
		}
		outcome, err := collab.DelegateSyncWithRetry(ctx, client, invoke.Request{
			TaskID:       taskID,
			WorkflowID:   strings.TrimSpace(workflowID),
			TeamID:       strings.TrimSpace(step.TeamID),
			StepID:       strings.TrimSpace(step.StepID),
			AgentID:      strings.TrimSpace(step.AgentID),
			PeerID:       strings.TrimSpace(step.PeerID),
			Method:       method,
			Payload:      clonePayload(step.Payload),
			PollInterval: opts.PollInterval,
		}, opts.Retry, opts.RetryObserver)
		if err != nil {
			return StepOutput{}, err
		}
		if outcome.Status != collab.StatusSucceeded {
			message := strings.TrimSpace(outcome.Error)
			if message == "" {
				message = fmt.Sprintf("a2a task status %q", outcome.Status)
			}
			return StepOutput{}, errors.New(message)
		}
		return StepOutput{Payload: clonePayload(outcome.Payload)}, nil
	}
}

func clonePayload(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
