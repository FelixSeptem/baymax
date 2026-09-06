package runner

import (
	"context"
	"errors"
	"strings"

	"github.com/FelixSeptem/baymax/context/handoff"
	"github.com/FelixSeptem/baymax/core/types"
)

type handoffBoundary struct {
	CheckpointID  string
	StreamFlushed bool
}

// handoffBoundaryFromModelEvent accepts only explicit source-owned metadata on
// finalized model events. Incremental deltas are intentionally ignored so an
// unflushed stream cannot become a restore cut.
func handoffBoundaryFromModelEvent(ev types.ModelEvent) handoffBoundary {
	switch ev.Type {
	case types.ModelEventTypeResponseCompleted,
		types.ModelEventTypeOutputTextDone,
		types.ModelEventTypeFunctionArgsDone,
		types.ModelEventTypeOutputItemDone:
	default:
		return handoffBoundary{}
	}
	boundary := handoffBoundary{}
	if ev.Meta == nil {
		return boundary
	}
	if raw, ok := ev.Meta["checkpoint_id"].(string); ok {
		boundary.CheckpointID = strings.TrimSpace(raw)
	}
	if raw, ok := ev.Meta["stream_flushed"].(bool); ok {
		boundary.StreamFlushed = raw
	}
	return boundary
}

func (e *Engine) rememberHandoffBoundary(runID string, boundary handoffBoundary) {
	if e == nil || strings.TrimSpace(runID) == "" || (boundary.CheckpointID == "" && !boundary.StreamFlushed) {
		return
	}
	e.handoffBoundaryMu.Lock()
	defer e.handoffBoundaryMu.Unlock()
	if e.handoffBoundaries == nil {
		e.handoffBoundaries = map[string]handoffBoundary{}
	}
	e.handoffBoundaries[strings.TrimSpace(runID)] = boundary
}

func (e *Engine) takeHandoffBoundary(runID string) handoffBoundary {
	if e == nil || strings.TrimSpace(runID) == "" {
		return handoffBoundary{}
	}
	e.handoffBoundaryMu.Lock()
	defer e.handoffBoundaryMu.Unlock()
	key := strings.TrimSpace(runID)
	boundary := e.handoffBoundaries[key]
	delete(e.handoffBoundaries, key)
	return boundary
}

func (e *Engine) clearHandoffBoundary(runID string) {
	if e == nil || strings.TrimSpace(runID) == "" {
		return
	}
	e.handoffBoundaryMu.Lock()
	delete(e.handoffBoundaries, strings.TrimSpace(runID))
	e.handoffBoundaryMu.Unlock()
}

func (e *Engine) assembleWithHandoff(
	ctx context.Context,
	assembleReq types.ContextAssembleRequest,
	modelReq types.ModelRequest,
	eventHandler types.EventHandler,
	iteration int,
) (types.ModelRequest, types.ContextAssembleResult, error) {
	if e == nil || e.assembler == nil {
		return modelReq, types.ContextAssembleResult{}, errors.New("context assembler is not configured")
	}
	if !e.runtimeEffectiveConfigSnapshot().Runtime.Context.JIT.Compaction.Handoff.Enabled {
		return e.assembler.Assemble(ctx, assembleReq, modelReq)
	}
	boundary := e.takeHandoffBoundary(assembleReq.RunID)
	assembleReq.FinalizedCheckpointID = boundary.CheckpointID
	assembleReq.StreamFlushed = boundary.StreamFlushed
	assembled, result, record, err := e.assembler.AssembleWithHandoff(ctx, assembleReq, modelReq)
	if err != nil {
		e.emitContextHandoff(ctx, eventHandler, assembleReq.RunID, iteration, handoff.Handoff{
			Version:  handoff.VersionV1,
			Fallback: &handoff.Fallback{Reason: handoff.FallbackGenerationFailure},
		}, "generation_failed")
		return assembled, result, err
	}
	if record != nil {
		event := "generated"
		if record.Fallback != nil {
			event = "fallback"
		}
		e.emitContextHandoff(ctx, eventHandler, assembleReq.RunID, iteration, *record, event)
		e.emitContextHandoff(ctx, eventHandler, assembleReq.RunID, iteration, *record, "validated")
	}
	return assembled, result, nil
}

func (e *Engine) emitContextHandoff(ctx context.Context, eventHandler types.EventHandler, runID string, iteration int, record handoff.Handoff, event string) {
	payload := map[string]any{
		"context_handoff_event":                event,
		"context_handoff_version":              record.Version,
		"context_handoff_cut":                  string(record.Cut),
		"context_handoff_source_checkpoint_id": record.SourceCheckpointID,
		"context_handoff_quality_score":        record.Quality.Score,
		"context_handoff_restore_ready":        record.Quality.RestoreReady,
	}
	if record.Fallback != nil {
		payload["context_handoff_fallback_reason"] = record.Fallback.Reason
	}
	e.emit(ctx, eventHandler, types.Event{Version: types.EventSchemaVersionV1, Type: types.EventTypeContextHandoff, RunID: runID, Iteration: iteration, Time: e.now(), Payload: payload})
}

// RestoreHandoff validates and restores a handoff through the context
// assembler's reference-first path. Lifecycle diagnostics are emitted through
// the caller's EventHandler so RuntimeRecorder remains the single writer.
// Repeated restores for the same stable handoff identity are idempotent and do
// not emit duplicate logical lifecycle events.
func (e *Engine) RestoreHandoff(ctx context.Context, record handoff.Handoff, eventHandler types.EventHandler) (handoff.RestoreResult, error) {
	if e == nil || e.assembler == nil {
		return handoff.RestoreResult{}, errors.New("context assembler is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := record.Validate(handoff.DefaultLimits()); err != nil {
		failed := record
		failed.Fallback = &handoff.Fallback{Reason: handoffValidationReason(err)}
		e.emitContextHandoff(ctx, eventHandler, record.RunID, 0, failed, "validation_failed")
		return handoff.RestoreResult{}, err
	}
	id := handoff.ID(record)
	e.handoffRestoreMu.Lock()
	if result, ok := e.handoffRestored[id]; ok {
		e.handoffRestoreMu.Unlock()
		return result, nil
	}
	e.handoffRestoreMu.Unlock()

	e.emitContextHandoff(ctx, eventHandler, record.RunID, 0, record, "validated")
	result, err := e.assembler.RestoreHandoffWithStore(record, e.handoffResolver, e.handoffRestoreStore)
	if err != nil {
		failed := record
		failed.Fallback = &handoff.Fallback{Reason: handoffRestoreReason(err)}
		e.emitContextHandoff(ctx, eventHandler, record.RunID, 0, failed, "restore_failed")
		return handoff.RestoreResult{}, err
	}
	e.handoffRestoreMu.Lock()
	if e.handoffRestored == nil {
		e.handoffRestored = map[string]handoff.RestoreResult{}
	}
	if existing, ok := e.handoffRestored[id]; ok {
		result = existing
	} else {
		e.handoffRestored[id] = result
	}
	e.handoffRestoreMu.Unlock()
	if result.HandoffID == id {
		record.Quality.RestoreReady = true
		e.emitContextHandoffRestore(ctx, eventHandler, record, result, "restored")
	}
	return result, nil
}

func handoffValidationReason(err error) string {
	if err == nil {
		return ""
	}
	if strings.Contains(err.Error(), "invalid handoff cut") {
		return handoff.FallbackInvalidCut
	}
	return handoff.FallbackGenerationFailure
}

func handoffRestoreReason(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, handoff.ErrReferenceNotFound) || strings.Contains(err.Error(), "resolve ") {
		return handoff.FallbackReferenceLoss
	}
	return handoff.FallbackGenerationFailure
}

func (e *Engine) emitContextHandoffRestore(ctx context.Context, eventHandler types.EventHandler, record handoff.Handoff, result handoff.RestoreResult, event string) {
	payload := map[string]any{
		"context_handoff_event":                event,
		"context_handoff_version":              record.Version,
		"context_handoff_cut":                  string(record.Cut),
		"context_handoff_source_checkpoint_id": record.SourceCheckpointID,
		"context_handoff_quality_score":        record.Quality.Score,
		"context_handoff_restore_ready":        record.Quality.RestoreReady,
		"context_handoff_restore_status":       "restored",
		"context_handoff_restore_operation_id": result.HandoffID,
		"context_handoff_restore_reason":       "reference_first",
	}
	e.emit(ctx, eventHandler, types.Event{Version: types.EventSchemaVersionV1, Type: types.EventTypeContextHandoff, RunID: record.RunID, Time: e.now(), Payload: payload})
}
