package local

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/observability/event"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
	"go.opentelemetry.io/otel/attribute"
)

func attachLifecycle(
	call types.ToolCall,
	outcome *types.ToolCallOutcome,
	inputIndex int,
	origin types.ToolFailureOrigin,
	executionStarted bool,
) {
	if outcome == nil || (outcome.Lifecycle != nil && outcome.Lifecycle.InputIndex >= 0) {
		return
	}
	failed := outcome.Result.Error != nil
	stageOutcome := types.ToolLifecycleOutcomeSucceeded
	if failed {
		stageOutcome = types.ToolLifecycleOutcomeFailed
	}
	stages := []types.ToolLifecycleStageInput{
		{Stage: types.ToolLifecycleStagePrepare, Outcome: types.ToolLifecycleOutcomeSucceeded},
		{Stage: types.ToolLifecycleStageValidate, Outcome: types.ToolLifecycleOutcomeSucceeded},
		{Stage: types.ToolLifecycleStageAuthorize, Outcome: types.ToolLifecycleOutcomeNotApplicable, ReasonCode: "not_applicable"},
		{Stage: types.ToolLifecycleStageExecute, Outcome: stageOutcome, Started: executionStarted},
		{Stage: types.ToolLifecycleStageFinalize, Outcome: types.ToolLifecycleOutcomeSucceeded},
	}
	switch origin {
	case types.ToolFailureOriginLookup:
		stages[0].Outcome = types.ToolLifecycleOutcomeFailed
		stages[1].Outcome = types.ToolLifecycleOutcomeSkipped
		stages[2].Outcome = types.ToolLifecycleOutcomeSkipped
		stages[3].Outcome = types.ToolLifecycleOutcomeSkipped
	case types.ToolFailureOriginValidation:
		stages[1].Outcome = types.ToolLifecycleOutcomeFailed
		stages[2].Outcome = types.ToolLifecycleOutcomeSkipped
		stages[3].Outcome = types.ToolLifecycleOutcomeSkipped
	case types.ToolFailureOriginPolicyDenied, types.ToolFailureOriginSandboxDenied:
		stages[2] = types.ToolLifecycleStageInput{Stage: types.ToolLifecycleStageAuthorize, Outcome: types.ToolLifecycleOutcomeFailed, ReasonCode: lifecycleReason(outcome)}
		stages[3].Outcome = types.ToolLifecycleOutcomeSkipped
	case types.ToolFailureOriginUnknown:
		if failed {
			stages[3].Outcome = types.ToolLifecycleOutcomeFailed
		}
	case types.ToolFailureOriginTimeout, types.ToolFailureOriginCanceled, types.ToolFailureOriginPanic, types.ToolFailureOriginRetryExhausted, types.ToolFailureOriginExecution, types.ToolFailureOriginMiddlewareFailure, types.ToolFailureOriginMiddlewareShortCircuit, types.ToolFailureOriginSandboxFailure:
		stages[3].Outcome = stageOutcome
	default:
		stages[3].Outcome = stageOutcome
	}
	projection, err := types.NewToolLifecycleProjection(types.ToolLifecycleInput{
		CallID:        call.CallID,
		ToolName:      call.Name,
		Source:        "tool/local",
		InputIndex:    inputIndex,
		FailureOrigin: origin,
		AttemptCount:  lifecycleAttemptCount(outcome),
		Stages:        stages,
	})
	if err == nil {
		outcome.Lifecycle = &projection
	}
}

func (d *Dispatcher) recordToolDiag(call types.ToolCall, start time.Time, classifiedErr *types.ClassifiedError, lifecycle *types.ToolLifecycleProjection) {
	if d == nil || d.recorder == nil {
		return
	}
	errorClass := ""
	if classifiedErr != nil {
		errorClass = string(classifiedErr.Class)
	}
	payload := map[string]any{
		"transport": "local", "call_id": call.CallID, "tool_name": call.Name,
		"latency_ms": time.Since(start).Milliseconds(), "error_class": errorClass,
	}
	if lifecycle != nil {
		payload["lifecycle_stage"] = string(types.ToolLifecycleStageFinalize)
		payload["failure_origin"] = string(lifecycle.FailureOrigin)
		payload["execution_started"] = lifecycle.ExecutionStarted
		payload["finalized"] = lifecycle.Finalized
		payload["attempt_count"] = lifecycle.AttemptCount
	}
	d.recorder.OnEvent(context.Background(), types.Event{Version: types.EventSchemaVersionV1, Type: types.EventTypeToolLifecycleFinalized, Time: time.Now(), Payload: payload})
}

func oteltraceAttrs(name string) []attribute.KeyValue {
	return []attribute.KeyValue{attribute.String("tool.name", name)}
}

func failedOutcome(call types.ToolCall, class types.ErrorClass, message string, retryable bool, details map[string]any) types.ToolCallOutcome {
	outcome := types.ToolCallOutcome{CallID: call.CallID, Name: call.Name, Result: types.ToolResult{Error: &types.ClassifiedError{Class: class, Message: message, Retryable: retryable, Details: details}}}
	attachLifecycle(call, &outcome, -1, classifyLifecycleFailure(nil, outcome.Result.Error), false)
	return outcome
}

func NewDispatcherWithRuntimeManager(registry *Registry, mgr *runtimeconfig.Manager) *Dispatcher {
	d := NewDispatcher(registry)
	d.runtimeMgr = mgr
	d.recorder = event.NewRuntimeRecorder(mgr)
	return d
}

func (d *Dispatcher) SetRuntimeManager(mgr *runtimeconfig.Manager) {
	d.runtimeMgr = mgr
	d.recorder = event.NewRuntimeRecorder(mgr)
}

// SetEventHandler sets the source-independent event sink used for bounded diagnostics projections.
func (d *Dispatcher) SetEventHandler(handler types.EventHandler) {
	if d != nil {
		d.recorder = handler
	}
}

func lifecycleAttemptCount(outcome *types.ToolCallOutcome) int {
	if outcome == nil || outcome.Result.Error == nil {
		return 1
	}
	if count, ok := intValue(outcome.Result.Error.Details, "retry_count"); ok && count >= 0 {
		return count + 1
	}
	return 1
}

func classifyLifecycleFailure(err error, classified *types.ClassifiedError) types.ToolFailureOrigin {
	if err != nil {
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			return types.ToolFailureOriginTimeout
		case errors.Is(err, context.Canceled):
			return types.ToolFailureOriginCanceled
		}
	}
	if classified == nil {
		return types.ToolFailureOriginUnknown
	}
	message := strings.ToLower(strings.TrimSpace(classified.Message))
	if strings.Contains(message, "not found") {
		return types.ToolFailureOriginLookup
	}
	if strings.Contains(message, "validation") {
		return types.ToolFailureOriginValidation
	}
	if strings.HasPrefix(message, "tool panic:") {
		return types.ToolFailureOriginPanic
	}
	details := classified.Details
	if boolValue(details, "panic_recovered") {
		return types.ToolFailureOriginPanic
	}
	if retryCount, ok := intValue(details, "retry_count"); ok && retryCount > 0 && classified.Retryable {
		return types.ToolFailureOriginRetryExhausted
	}
	reason := strings.ToLower(strings.TrimSpace(stringValue(details, "reason_code")))
	if strings.Contains(reason, "deny") || classified.Class == types.ErrSecurity {
		if strings.Contains(reason, "sandbox") || strings.Contains(reason, "egress") {
			return types.ToolFailureOriginSandboxDenied
		}
		return types.ToolFailureOriginPolicyDenied
	}
	return types.ToolFailureOriginExecution
}

func lifecycleReason(outcome *types.ToolCallOutcome) string {
	if outcome == nil || outcome.Result.Error == nil {
		return ""
	}
	return strings.TrimSpace(stringValue(outcome.Result.Error.Details, "reason_code"))
}

func stringValue(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	value, _ := values[key].(string)
	return value
}

func boolValue(values map[string]any, key string) bool {
	if values == nil {
		return false
	}
	value, _ := values[key].(bool)
	return value
}

func intValue(values map[string]any, key string) (int, bool) {
	if values == nil {
		return 0, false
	}
	value, ok := values[key].(int)
	return value, ok
}
