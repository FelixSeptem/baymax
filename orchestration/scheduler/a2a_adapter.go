package scheduler

import (
	"context"
	"errors"
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

type A2AAsyncClient interface {
	A2AClient
	SubmitAsync(ctx context.Context, req a2a.TaskRequest, sink a2a.ReportSink) (a2a.AsyncSubmitAck, error)
}

type A2AReconcilePollClient interface {
	Status(ctx context.Context, taskID string) (a2a.TaskRecord, error)
	Result(ctx context.Context, taskID string) (a2a.TaskRecord, error)
}

type A2AExecution struct {
	Commit        TerminalCommit
	Retryable     bool
	AsyncAccepted bool
	AsyncTaskID   string
}

type MailboxBridgeProvider func() (*invoke.MailboxBridge, error)

type A2AInvokeOption func(*a2aInvokeOptions)

type a2aInvokeOptions struct {
	bridgeProvider MailboxBridgeProvider
}

func WithMailboxBridgeProvider(provider MailboxBridgeProvider) A2AInvokeOption {
	return func(opts *a2aInvokeOptions) {
		if opts == nil {
			return
		}
		opts.bridgeProvider = provider
	}
}

func ExecuteClaimWithA2A(
	ctx context.Context,
	client A2AClient,
	claimed ClaimedTask,
	pollInterval time.Duration,
	opts ...A2AInvokeOption,
) (A2AExecution, error) {
	task := claimed.Record.Task
	attempt := claimed.Attempt
	req := buildInvokeRequest(claimed, pollInterval)
	if req.AgentID == "" {
		req.AgentID = "scheduler-worker"
	}
	if req.PeerID == "" {
		req.PeerID = "scheduler-peer"
	}

	cfg := resolveA2AInvokeOptions(opts...)
	bridge, err := cfg.resolveMailboxBridge()
	if err != nil {
		return failedExecutionFromA2AError(claimed, err), err
	}
	outcome, err := bridge.InvokeSync(ctx, client, req)
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

func SubmitClaimWithA2AAsync(
	ctx context.Context,
	client A2AAsyncClient,
	claimed ClaimedTask,
	sink a2a.ReportSink,
	opts ...A2AInvokeOption,
) (a2a.AsyncSubmitAck, error) {
	if client == nil {
		return a2a.AsyncSubmitAck{}, errors.New("a2a async client is required")
	}
	req := buildTaskRequest(claimed)
	if req.AgentID == "" {
		req.AgentID = "scheduler-worker"
	}
	if req.PeerID == "" {
		req.PeerID = "scheduler-peer"
	}
	if req.Payload == nil {
		req.Payload = map[string]any{}
	}
	req.Payload["attempt_id"] = strings.TrimSpace(claimed.Attempt.AttemptID)
	cfg := resolveA2AInvokeOptions(opts...)
	bridge, err := cfg.resolveMailboxBridge()
	if err != nil {
		return a2a.AsyncSubmitAck{}, err
	}
	ack, err := bridge.InvokeAsync(ctx, client, invoke.AsyncRequest{
		TaskID:     req.TaskID,
		WorkflowID: req.WorkflowID,
		TeamID:     req.TeamID,
		StepID:     req.StepID,
		AttemptID:  req.AttemptID,
		AgentID:    req.AgentID,
		PeerID:     req.PeerID,
		Method:     req.Method,
		Payload:    copyMap(req.Payload),
	}, sink)
	if err != nil {
		return a2a.AsyncSubmitAck{}, err
	}
	if strings.TrimSpace(ack.TaskID) == "" {
		ack.TaskID = req.TaskID
	}
	return ack, nil
}

func ExecutionFromAsyncReport(claimed ClaimedTask, report a2a.AsyncReport) (A2AExecution, error) {
	attemptID := strings.TrimSpace(report.AttemptID)
	if attemptID == "" {
		attemptID = strings.TrimSpace(claimed.Attempt.AttemptID)
	}
	if attemptID == "" {
		return failedExecutionFromA2AError(claimed, errors.New("async report attempt_id is required")), errors.New("async report attempt_id is required")
	}
	switch report.Status {
	case a2a.StatusSucceeded:
		return A2AExecution{
			Commit: TerminalCommit{
				TaskID:       claimed.Record.Task.TaskID,
				AttemptID:    attemptID,
				Status:       TaskStateSucceeded,
				Source:       AsyncResolutionSourceCallback,
				RemoteTaskID: strings.TrimSpace(report.TaskID),
				Result:       copyMap(report.Result),
				OutcomeKey:   strings.TrimSpace(report.OutcomeKey),
				CommittedAt:  time.Now(),
			},
		}, nil
	case a2a.StatusFailed, a2a.StatusCanceled:
		class := report.ErrorClass
		if class == "" {
			class = types.ErrMCP
		}
		layer := strings.TrimSpace(report.ErrorLayer)
		if layer == "" {
			layer = string(a2a.ErrorLayerProtocol)
		}
		message := strings.TrimSpace(report.ErrorMessage)
		if message == "" {
			message = fmt.Sprintf("a2a async terminal status %q", report.Status)
		}
		return A2AExecution{
			Commit: TerminalCommit{
				TaskID:       claimed.Record.Task.TaskID,
				AttemptID:    attemptID,
				Status:       TaskStateFailed,
				Source:       AsyncResolutionSourceCallback,
				RemoteTaskID: strings.TrimSpace(report.TaskID),
				ErrorMessage: message,
				ErrorClass:   class,
				ErrorLayer:   layer,
				OutcomeKey:   strings.TrimSpace(report.OutcomeKey),
				CommittedAt:  time.Now(),
			},
			Retryable: layer == string(a2a.ErrorLayerTransport),
		}, nil
	default:
		err := fmt.Errorf("unsupported async report status %q", report.Status)
		return failedExecutionFromA2AError(claimed, err), err
	}
}

func ClassifyReconcilePoll(
	ctx context.Context,
	client A2AReconcilePollClient,
	remoteTaskID string,
) (ReconcilePollClassification, a2a.TaskRecord, error) {
	if client == nil {
		return ReconcilePollClassificationNonRetryableErr, a2a.TaskRecord{}, errors.New("a2a reconcile poll client is required")
	}
	remoteTaskID = strings.TrimSpace(remoteTaskID)
	if remoteTaskID == "" {
		return ReconcilePollClassificationNonRetryableErr, a2a.TaskRecord{}, errors.New("remote_task_id is required")
	}
	statusRecord, err := client.Status(ctx, remoteTaskID)
	if err != nil {
		return classifyReconcilePollError(err), a2a.TaskRecord{}, err
	}
	switch statusRecord.Status {
	case a2a.StatusSucceeded, a2a.StatusFailed, a2a.StatusCanceled:
		resultRecord, resultErr := client.Result(ctx, remoteTaskID)
		if resultErr != nil {
			return classifyReconcilePollError(resultErr), a2a.TaskRecord{}, resultErr
		}
		if strings.TrimSpace(resultRecord.TaskID) == "" {
			resultRecord.TaskID = remoteTaskID
		}
		return ReconcilePollClassificationTerminal, resultRecord, nil
	default:
		return ReconcilePollClassificationPending, statusRecord, nil
	}
}

func classifyReconcilePollError(err error) ReconcilePollClassification {
	if err == nil {
		return ReconcilePollClassificationPending
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	if strings.Contains(message, "not found") {
		return ReconcilePollClassificationNotFound
	}
	_, layer, _ := a2a.ClassifyError(err)
	if layer == a2a.ErrorLayerTransport {
		return ReconcilePollClassificationRetryableError
	}
	return ReconcilePollClassificationNonRetryableErr
}

func ReconcileTerminalCommitFromRecord(
	taskID string,
	attemptID string,
	remoteTaskID string,
	record a2a.TaskRecord,
	committedAt time.Time,
) (TerminalCommit, error) {
	taskID = strings.TrimSpace(taskID)
	attemptID = strings.TrimSpace(attemptID)
	remoteTaskID = strings.TrimSpace(remoteTaskID)
	if taskID == "" || attemptID == "" {
		return TerminalCommit{}, errors.New("task_id and attempt_id are required")
	}
	if committedAt.IsZero() {
		committedAt = time.Now()
	}
	switch record.Status {
	case a2a.StatusSucceeded:
		return TerminalCommit{
			TaskID:       taskID,
			AttemptID:    attemptID,
			Status:       TaskStateSucceeded,
			Source:       AsyncResolutionSourceReconcilePoll,
			RemoteTaskID: remoteTaskID,
			Result:       copyMap(record.Result),
			CommittedAt:  committedAt,
		}, nil
	case a2a.StatusFailed, a2a.StatusCanceled:
		class := record.ErrorClass
		if class == "" {
			class = types.ErrMCP
		}
		layer := strings.TrimSpace(record.A2AErrorLayer)
		if layer == "" {
			layer = string(a2a.ErrorLayerProtocol)
		}
		message := strings.TrimSpace(record.ErrorMessage)
		if message == "" {
			message = fmt.Sprintf("a2a terminal status %q", record.Status)
		}
		return TerminalCommit{
			TaskID:       taskID,
			AttemptID:    attemptID,
			Status:       TaskStateFailed,
			Source:       AsyncResolutionSourceReconcilePoll,
			RemoteTaskID: remoteTaskID,
			ErrorMessage: message,
			ErrorClass:   class,
			ErrorLayer:   layer,
			CommittedAt:  committedAt,
		}, nil
	default:
		return TerminalCommit{}, fmt.Errorf("unsupported reconcile terminal status %q", record.Status)
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

func buildInvokeRequest(claimed ClaimedTask, pollInterval time.Duration) invoke.Request {
	task := claimed.Record.Task
	attempt := claimed.Attempt
	return invoke.Request{
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
}

func buildTaskRequest(claimed ClaimedTask) a2a.TaskRequest {
	req := buildInvokeRequest(claimed, 0)
	return a2a.TaskRequest{
		TaskID:     strings.TrimSpace(req.TaskID),
		WorkflowID: strings.TrimSpace(req.WorkflowID),
		TeamID:     strings.TrimSpace(req.TeamID),
		StepID:     strings.TrimSpace(req.StepID),
		AttemptID:  strings.TrimSpace(claimed.Attempt.AttemptID),
		AgentID:    strings.TrimSpace(req.AgentID),
		PeerID:     strings.TrimSpace(req.PeerID),
		Method:     strings.TrimSpace(req.Method),
		Payload:    copyMap(req.Payload),
	}
}

func resolveA2AInvokeOptions(opts ...A2AInvokeOption) a2aInvokeOptions {
	resolved := a2aInvokeOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(&resolved)
		}
	}
	return resolved
}

func (opts a2aInvokeOptions) resolveMailboxBridge() (*invoke.MailboxBridge, error) {
	if opts.bridgeProvider != nil {
		return opts.bridgeProvider()
	}
	return invoke.NewInMemoryMailboxBridge()
}
