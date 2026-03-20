package teams

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/FelixSeptem/baymax/a2a"
	"github.com/FelixSeptem/baymax/orchestration/collab"
	"github.com/FelixSeptem/baymax/orchestration/invoke"
)

type A2AAsyncRemoteRunnerOptions struct {
	TaskIDGenerator func(plan Plan, task Task) string
	ReportSink      a2a.ReportSink
}

func NewA2AAsyncRemoteTaskRunner(client invoke.AsyncClient, opts A2AAsyncRemoteRunnerOptions) RemoteTaskRunnerFunc {
	return func(ctx context.Context, plan Plan, task Task) (TaskResult, error) {
		if client == nil {
			return TaskResult{}, errors.New("a2a async client is not configured")
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
		ack, err := collab.DelegateAsync(ctx, client, invoke.AsyncRequest{
			TaskID:     taskID,
			WorkflowID: strings.TrimSpace(plan.WorkflowID),
			TeamID:     strings.TrimSpace(plan.TeamID),
			StepID:     strings.TrimSpace(plan.StepID),
			AttemptID:  "1",
			AgentID:    strings.TrimSpace(task.AgentID),
			PeerID:     strings.TrimSpace(task.Remote.PeerID),
			Method:     method,
			Payload:    cloneRemotePayload(task.Remote.Payload),
		}, opts.ReportSink)
		if err != nil {
			return TaskResult{}, err
		}
		return TaskResult{
			Output: map[string]any{
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
