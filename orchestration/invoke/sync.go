package invoke

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/FelixSeptem/baymax/a2a"
	"github.com/FelixSeptem/baymax/core/types"
)

const DefaultPollInterval = 20 * time.Millisecond

type Client interface {
	Submit(ctx context.Context, req a2a.TaskRequest) (a2a.TaskRecord, error)
	WaitResult(
		ctx context.Context,
		taskID string,
		pollInterval time.Duration,
		callback func(context.Context, a2a.TaskRecord) error,
	) (a2a.TaskRecord, error)
}

type Request struct {
	TaskID       string
	WorkflowID   string
	TeamID       string
	StepID       string
	AgentID      string
	PeerID       string
	Method       string
	Payload      map[string]any
	PollInterval time.Duration
	Callback     func(context.Context, a2a.TaskRecord) error
}

type NormalizedError struct {
	Message   string
	Class     types.ErrorClass
	Layer     string
	Code      string
	Retryable bool
}

type Outcome struct {
	TaskID         string
	TerminalStatus a2a.TaskStatus
	Result         map[string]any
	Record         a2a.TaskRecord
	Error          *NormalizedError
}

// InvokeSync executes direct submit+wait invocation.
//
// Deprecated: Use MailboxBridge.InvokeSync as the canonical command->result path.
func InvokeSync(ctx context.Context, client Client, req Request) (Outcome, error) {
	if client == nil {
		err := errors.New("a2a client is required")
		return Outcome{Error: normalizeError(err)}, err
	}
	submitReq := a2a.TaskRequest{
		TaskID:     strings.TrimSpace(req.TaskID),
		WorkflowID: strings.TrimSpace(req.WorkflowID),
		TeamID:     strings.TrimSpace(req.TeamID),
		StepID:     strings.TrimSpace(req.StepID),
		AgentID:    strings.TrimSpace(req.AgentID),
		PeerID:     strings.TrimSpace(req.PeerID),
		Method:     strings.TrimSpace(req.Method),
		Payload:    cloneMap(req.Payload),
	}
	if submitReq.TaskID == "" {
		err := errors.New("task_id is required")
		return Outcome{Error: normalizeError(err)}, err
	}

	submitted, err := client.Submit(ctx, submitReq)
	if err != nil {
		return Outcome{
			TaskID: submitReq.TaskID,
			Error:  normalizeError(err),
		}, err
	}
	taskID := strings.TrimSpace(submitted.TaskID)
	if taskID == "" {
		taskID = submitReq.TaskID
	}
	pollInterval := req.PollInterval
	if pollInterval <= 0 {
		pollInterval = DefaultPollInterval
	}
	record, err := client.WaitResult(ctx, taskID, pollInterval, req.Callback)
	if err != nil {
		return Outcome{
			TaskID: taskID,
			Error:  normalizeError(err),
		}, err
	}

	out := Outcome{
		TaskID:         taskID,
		TerminalStatus: record.Status,
		Result:         cloneMap(record.Result),
		Record:         record,
	}
	switch record.Status {
	case a2a.StatusSucceeded:
		return out, nil
	case a2a.StatusFailed, a2a.StatusCanceled:
		out.Error = normalizeTerminalRecordError(record)
		return out, nil
	default:
		err := fmt.Errorf("a2a terminal status %q is unsupported", record.Status)
		out.Error = normalizeError(err)
		return out, err
	}
}

func normalizeTerminalRecordError(record a2a.TaskRecord) *NormalizedError {
	message := strings.TrimSpace(record.ErrorMessage)
	class := record.ErrorClass
	layer := strings.TrimSpace(record.A2AErrorLayer)
	code := ""
	if message != "" {
		fallbackClass, fallbackLayer, fallbackCode := a2a.ClassifyError(errors.New(message))
		if class == "" {
			class = fallbackClass
		}
		if layer == "" {
			layer = strings.TrimSpace(string(fallbackLayer))
		}
		code = strings.TrimSpace(fallbackCode)
	}
	if class == "" {
		class = types.ErrMCP
	}
	if layer == "" {
		layer = string(a2a.ErrorLayerProtocol)
	}
	if message == "" {
		message = fmt.Sprintf("a2a terminal status %s", record.Status)
	}
	return &NormalizedError{
		Message:   message,
		Class:     class,
		Layer:     layer,
		Code:      code,
		Retryable: layer == string(a2a.ErrorLayerTransport),
	}
}

func normalizeError(err error) *NormalizedError {
	if err == nil {
		return nil
	}
	class, layer, code := a2a.ClassifyError(err)
	if class == "" {
		class = types.ErrMCP
	}
	layerText := strings.TrimSpace(string(layer))
	if layerText == "" {
		layerText = string(a2a.ErrorLayerProtocol)
	}
	return &NormalizedError{
		Message:   strings.TrimSpace(err.Error()),
		Class:     class,
		Layer:     layerText,
		Code:      strings.TrimSpace(code),
		Retryable: layerText == string(a2a.ErrorLayerTransport),
	}
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
