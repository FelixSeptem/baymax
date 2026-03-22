package teams

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

var generatedRemoteTaskCounter uint64

type A2ARemoteRunnerOptions struct {
	PollInterval    time.Duration
	TaskIDGenerator func(plan Plan, task Task) string
	Retry           collab.RetryConfig
	RetryObserver   collab.RetryObserver
}

func NewA2ARemoteTaskRunner(client invoke.Client, opts A2ARemoteRunnerOptions) RemoteTaskRunnerFunc {
	return func(ctx context.Context, plan Plan, task Task) (TaskResult, error) {
		if client == nil {
			return TaskResult{}, errors.New("a2a client is not configured")
		}
		taskID := ""
		if opts.TaskIDGenerator != nil {
			taskID = strings.TrimSpace(opts.TaskIDGenerator(plan, task))
		}
		if taskID == "" {
			taskID = fmt.Sprintf(
				"%s-%d-%d",
				strings.TrimSpace(task.TaskID),
				time.Now().UnixNano(),
				atomic.AddUint64(&generatedRemoteTaskCounter, 1),
			)
		}
		method := strings.TrimSpace(task.Remote.Method)
		if method == "" {
			method = "team.dispatch"
		}
		outcome, err := collab.DelegateSyncWithRetry(ctx, client, invoke.Request{
			TaskID:       taskID,
			WorkflowID:   strings.TrimSpace(plan.WorkflowID),
			TeamID:       strings.TrimSpace(plan.TeamID),
			StepID:       strings.TrimSpace(plan.StepID),
			AgentID:      strings.TrimSpace(task.AgentID),
			PeerID:       strings.TrimSpace(task.Remote.PeerID),
			Method:       method,
			Payload:      cloneRemotePayload(task.Remote.Payload),
			PollInterval: opts.PollInterval,
		}, opts.Retry, opts.RetryObserver)
		if err != nil {
			return TaskResult{}, err
		}
		if outcome.Status != collab.StatusSucceeded {
			message := strings.TrimSpace(outcome.Error)
			if message == "" {
				message = fmt.Sprintf("a2a task status %q", outcome.Status)
			}
			return TaskResult{}, errors.New(message)
		}
		out := TaskResult{Output: cloneRemotePayload(outcome.Payload)}
		if vote, ok := outcome.Payload["vote"].(string); ok {
			out.Vote = strings.TrimSpace(vote)
		}
		return out, nil
	}
}

func cloneRemotePayload(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
