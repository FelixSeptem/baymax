package workflow

import (
	"context"
	"strings"

	"github.com/FelixSeptem/baymax/core/history"
	"github.com/FelixSeptem/baymax/core/types"
)

// WithNativeEventHistory binds an existing workflow-owned journal. The
// default engine remains history-unsupported until a source is explicitly
// configured.
func WithNativeEventHistory(journal *history.Journal) Option {
	return func(e *Engine) {
		if e != nil {
			e.nativeHistory = journal
		}
	}
}

func (e *Engine) emitNativeHistory(ctx context.Context, timeline workflowTimelineContext, status StepStatus, payload map[string]any) {
	if e == nil || e.nativeHistory == nil {
		return
	}
	source := e.nativeHistory.Source()
	if source.Kind != types.HistorySourceWorkflow || strings.TrimSpace(source.ID) != strings.TrimSpace(timeline.workflowID) {
		return
	}
	nativeType := "workflow." + string(status)
	if status == StepStatusSucceeded || status == StepStatusFailed || status == StepStatusCanceled {
		nativeType = "run.finished"
		payload = cloneWorkflowTimelinePayload(payload)
		payload["terminal_state"] = workflowHistoryTerminalState(string(status))
	}
	_, err := e.nativeHistory.Append(ctx, types.NativeExecutionEvent{
		Version:    types.NativeExecutionEventVersionV1,
		NativeType: nativeType,
		Event:      types.EventEnvelope{RunID: timeline.runID, Source: types.ProtocolSourceWorkflow, Time: e.now(), Payload: payload},
	})
	e.nativeHistoryHealth.Record(source, err)
}

// NativeEventHistoryError reports append failures from the explicitly bound
// workflow source journal for the lifetime of this Engine.
func (e *Engine) NativeEventHistoryError() error {
	if e == nil {
		return nil
	}
	return e.nativeHistoryHealth.Err()
}

func workflowHistoryTerminalState(status string) string {
	if status == string(StepStatusSucceeded) {
		return "completed"
	}
	return status
}

func cloneWorkflowTimelinePayload(payload map[string]any) map[string]any {
	result := make(map[string]any, len(payload))
	for key, value := range payload {
		result[key] = value
	}
	return result
}
