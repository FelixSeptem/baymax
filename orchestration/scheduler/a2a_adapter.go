package scheduler

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/FelixSeptem/baymax/a2a"
	"github.com/FelixSeptem/baymax/core/types"
)

type A2AClient interface {
	Submit(ctx context.Context, req a2a.TaskRequest) (a2a.TaskRecord, error)
	WaitResult(
		ctx context.Context,
		taskID string,
		pollInterval time.Duration,
		callback func(context.Context, a2a.TaskRecord) error,
	) (a2a.TaskRecord, error)
}

type A2AExecution struct {
	Commit    TerminalCommit
	Retryable bool
}

func ExecuteClaimWithA2A(
	ctx context.Context,
	client A2AClient,
	claimed ClaimedTask,
	pollInterval time.Duration,
) (A2AExecution, error) {
	if client == nil {
		return A2AExecution{}, fmt.Errorf("a2a client is required")
	}
	task := claimed.Record.Task
	attempt := claimed.Attempt
	req := a2a.TaskRequest{
		TaskID:     strings.TrimSpace(task.TaskID) + "-" + strings.TrimSpace(attempt.AttemptID),
		WorkflowID: strings.TrimSpace(task.WorkflowID),
		TeamID:     strings.TrimSpace(task.TeamID),
		StepID:     strings.TrimSpace(task.StepID),
		AgentID:    strings.TrimSpace(task.AgentID),
		PeerID:     strings.TrimSpace(task.PeerID),
		Method:     "scheduler.dispatch",
		Payload:    copyMap(task.Payload),
	}
	if req.AgentID == "" {
		req.AgentID = "scheduler-worker"
	}
	if req.PeerID == "" {
		req.PeerID = "scheduler-peer"
	}
	submitted, err := client.Submit(ctx, req)
	if err != nil {
		return failedExecutionFromA2AError(claimed, err), err
	}
	record, err := client.WaitResult(ctx, submitted.TaskID, pollInterval, nil)
	if err != nil {
		return failedExecutionFromA2AError(claimed, err), err
	}
	switch record.Status {
	case a2a.StatusSucceeded:
		return A2AExecution{
			Commit: TerminalCommit{
				TaskID:      task.TaskID,
				AttemptID:   attempt.AttemptID,
				Status:      TaskStateSucceeded,
				Result:      copyMap(record.Result),
				CommittedAt: time.Now(),
			},
			Retryable: false,
		}, nil
	case a2a.StatusFailed, a2a.StatusCanceled:
		layer := strings.TrimSpace(record.A2AErrorLayer)
		retryable := layer == string(a2a.ErrorLayerTransport)
		class := record.ErrorClass
		if class == "" {
			class = types.ErrMCP
		}
		return A2AExecution{
			Commit: TerminalCommit{
				TaskID:       task.TaskID,
				AttemptID:    attempt.AttemptID,
				Status:       TaskStateFailed,
				ErrorMessage: strings.TrimSpace(record.ErrorMessage),
				ErrorClass:   class,
				ErrorLayer:   layer,
				CommittedAt:  time.Now(),
			},
			Retryable: retryable,
		}, nil
	default:
		return failedExecutionFromA2AError(
			claimed,
			fmt.Errorf("a2a terminal status %q is unsupported", record.Status),
		), fmt.Errorf("a2a terminal status %q is unsupported", record.Status)
	}
}

func failedExecutionFromA2AError(claimed ClaimedTask, err error) A2AExecution {
	class, layer, _ := a2a.ClassifyError(err)
	if class == "" {
		class = types.ErrMCP
	}
	errorLayer := strings.TrimSpace(string(layer))
	if errorLayer == "" {
		errorLayer = string(a2a.ErrorLayerProtocol)
	}
	return A2AExecution{
		Commit: TerminalCommit{
			TaskID:       claimed.Record.Task.TaskID,
			AttemptID:    claimed.Attempt.AttemptID,
			Status:       TaskStateFailed,
			ErrorMessage: strings.TrimSpace(err.Error()),
			ErrorClass:   class,
			ErrorLayer:   errorLayer,
			CommittedAt:  time.Now(),
		},
		Retryable: errorLayer == string(a2a.ErrorLayerTransport),
	}
}
