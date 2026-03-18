package teams

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/FelixSeptem/baymax/a2a"
	"github.com/FelixSeptem/baymax/orchestration/invoke"
)

type A2ARemoteRunnerOptions struct {
	PollInterval    time.Duration
	TaskIDGenerator func(plan Plan, task Task) string
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
			taskID = fmt.Sprintf("%s-%d", strings.TrimSpace(task.TaskID), time.Now().UnixNano())
		}
		method := strings.TrimSpace(task.Remote.Method)
		if method == "" {
			method = "team.dispatch"
		}
		outcome, err := invoke.InvokeSync(ctx, client, invoke.Request{
			TaskID:       taskID,
			WorkflowID:   strings.TrimSpace(plan.WorkflowID),
			TeamID:       strings.TrimSpace(plan.TeamID),
			StepID:       strings.TrimSpace(plan.StepID),
			AgentID:      strings.TrimSpace(task.AgentID),
			PeerID:       strings.TrimSpace(task.Remote.PeerID),
			Method:       method,
			Payload:      cloneRemotePayload(task.Remote.Payload),
			PollInterval: opts.PollInterval,
		})
		if err != nil {
			return TaskResult{}, err
		}
		if outcome.TerminalStatus != a2a.StatusSucceeded {
			message := fmt.Sprintf("a2a task status %q", outcome.TerminalStatus)
			if outcome.Error != nil && strings.TrimSpace(outcome.Error.Message) != "" {
				message = strings.TrimSpace(outcome.Error.Message)
			}
			return TaskResult{}, errors.New(message)
		}
		out := TaskResult{Output: cloneRemotePayload(outcome.Result)}
		if vote, ok := outcome.Result["vote"].(string); ok {
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
