package scheduler

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/FelixSeptem/baymax/a2a"
	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/orchestration/invoke"
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
	task := claimed.Record.Task
	attempt := claimed.Attempt
	req := invoke.Request{
		TaskID:       strings.TrimSpace(task.TaskID) + "-" + strings.TrimSpace(attempt.AttemptID),
		WorkflowID:   strings.TrimSpace(task.WorkflowID),
		TeamID:       strings.TrimSpace(task.TeamID),
		StepID:       strings.TrimSpace(task.StepID),
		AgentID:      strings.TrimSpace(task.AgentID),
		PeerID:       strings.TrimSpace(task.PeerID),
		Method:       "scheduler.dispatch",
		Payload:      copyMap(task.Payload),
		PollInterval: pollInterval,
	}
	if req.AgentID == "" {
		req.AgentID = "scheduler-worker"
	}
	if req.PeerID == "" {
		req.PeerID = "scheduler-peer"
	}

	outcome, err := invoke.InvokeSync(ctx, client, req)
	if err != nil {
		return failedExecutionFromInvokeError(claimed, outcome, err), err
	}

	switch outcome.TerminalStatus {
	case a2a.StatusSucceeded:
		return A2AExecution{
			Commit: TerminalCommit{
				TaskID:      task.TaskID,
				AttemptID:   attempt.AttemptID,
				Status:      TaskStateSucceeded,
				Result:      copyMap(outcome.Result),
				CommittedAt: time.Now(),
			},
			Retryable: false,
		}, nil
	case a2a.StatusFailed, a2a.StatusCanceled:
		layer := ""
		retryable := false
		class := types.ErrMCP
		errorMessage := ""
		if outcome.Error != nil {
			layer = strings.TrimSpace(outcome.Error.Layer)
			retryable = outcome.Error.Retryable
			errorMessage = strings.TrimSpace(outcome.Error.Message)
			if outcome.Error.Class != "" {
				class = outcome.Error.Class
			}
		}
		if layer == "" {
			layer = strings.TrimSpace(outcome.Record.A2AErrorLayer)
			retryable = layer == string(a2a.ErrorLayerTransport)
		}
		if layer == "" {
			layer = string(a2a.ErrorLayerProtocol)
		}
		if errorMessage == "" {
			errorMessage = strings.TrimSpace(outcome.Record.ErrorMessage)
		}
		if errorMessage == "" {
			errorMessage = fmt.Sprintf("a2a terminal status %q", outcome.TerminalStatus)
		}
		return A2AExecution{
			Commit: TerminalCommit{
				TaskID:       task.TaskID,
				AttemptID:    attempt.AttemptID,
				Status:       TaskStateFailed,
				ErrorMessage: errorMessage,
				ErrorClass:   class,
				ErrorLayer:   layer,
				CommittedAt:  time.Now(),
			},
			Retryable: retryable,
		}, nil
	default:
		return failedExecutionFromA2AError(
			claimed,
			fmt.Errorf("a2a terminal status %q is unsupported", outcome.TerminalStatus),
		), fmt.Errorf("a2a terminal status %q is unsupported", outcome.TerminalStatus)
	}
}

func failedExecutionFromInvokeError(claimed ClaimedTask, outcome invoke.Outcome, err error) A2AExecution {
	if outcome.Error == nil {
		return failedExecutionFromA2AError(claimed, err)
	}
	layer := strings.TrimSpace(outcome.Error.Layer)
	if layer == "" {
		layer = string(a2a.ErrorLayerProtocol)
	}
	class := outcome.Error.Class
	if class == "" {
		class = types.ErrMCP
	}
	message := strings.TrimSpace(outcome.Error.Message)
	if message == "" && err != nil {
		message = strings.TrimSpace(err.Error())
	}
	return A2AExecution{
		Commit: TerminalCommit{
			TaskID:       claimed.Record.Task.TaskID,
			AttemptID:    claimed.Attempt.AttemptID,
			Status:       TaskStateFailed,
			ErrorMessage: message,
			ErrorClass:   class,
			ErrorLayer:   layer,
			CommittedAt:  time.Now(),
		},
		Retryable: outcome.Error.Retryable,
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
