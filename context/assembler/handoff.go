package assembler

import (
	"context"
	"fmt"
	"strings"

	"github.com/FelixSeptem/baymax/context/handoff"
	"github.com/FelixSeptem/baymax/core/types"
)

// AssembleWithHandoff preserves the existing Assemble result and optionally
// emits a bounded handoff when the runtime handoff switch is enabled.
func (a *Assembler) AssembleWithHandoff(ctx context.Context, req types.ContextAssembleRequest, modelReq types.ModelRequest) (types.ModelRequest, types.ContextAssembleResult, *handoff.Handoff, error) {
	assembled, result, err := a.Assemble(ctx, req, modelReq)
	if err != nil {
		return assembled, result, nil, err
	}
	if a.runtimeContextConfig == nil || !a.runtimeContextConfig().JIT.Compaction.Handoff.Enabled {
		return assembled, result, nil, nil
	}
	h, err := a.BuildHandoff(req, assembled, result)
	if err != nil {
		return assembled, result, nil, err
	}
	return assembled, result, &h, nil
}

// BuildHandoff projects an assembled model request into the bounded handoff
// contract. It is deliberately side-effect-free; source history and checkpoint
// owners remain authoritative for bodies and lineage.
func (a *Assembler) BuildHandoff(req types.ContextAssembleRequest, modelReq types.ModelRequest, result types.ContextAssembleResult) (handoff.Handoff, error) {
	cut := selectHandoffCut(req, result)
	messages := make([]handoff.Message, 0, len(modelReq.Messages))
	for _, msg := range modelReq.Messages {
		messages = append(messages, handoff.Message{Role: msg.Role, Content: msg.Content})
	}
	objective := strings.TrimSpace(req.Input)
	if objective == "" {
		objective = strings.TrimSpace(modelReq.Input)
	}
	if objective == "" {
		return handoff.Handoff{}, fmt.Errorf("handoff objective is required")
	}
	projection := handoff.CompressionProjection{
		PressureZone:              result.Stage.PressureZone,
		PressureReason:            result.Stage.PressureReason,
		PressureTriggerSource:     result.Stage.PressureTriggerSource,
		CompactionFallback:        result.Stage.CompactionFallback,
		CompactionFallbackReason:  result.Stage.CompactionFallbackReason,
		CompactionQualityScore:    result.Stage.CompactionQualityScore,
		RetainedEvidenceCount:     result.Stage.RetainedEvidenceCount,
		SpillCount:                result.Stage.SpillCount,
		SwapBackCount:             result.Stage.SwapBackCount,
		LifecycleTierStats:        cloneHandoffIntMap(result.Stage.ContextLifecycleTierStats),
		TierTransitionReason:      result.Stage.ContextTierTransitionReason,
		ColdStoreGovernanceAction: result.Stage.ContextColdStoreGovernanceAction,
		RecoveryConsistencyMarker: result.Stage.ContextRecoveryConsistencyMarker,
	}
	h, err := handoff.Build(handoff.BuildRequest{
		RunID: req.RunID, SessionID: req.SessionID, Cut: cut, Objective: objective, Messages: messages,
	}, handoff.BuildOptions{MinQuality: runtimeHandoffQualityThreshold(a), Compression: projection})
	if err != nil {
		return handoff.Handoff{}, err
	}
	if projection.CompactionFallback && h.Fallback == nil {
		h.Fallback = &handoff.Fallback{Reason: normalizeHandoffFallbackReason(projection.CompactionFallbackReason)}
	}
	h.SourceCheckpointID = strings.TrimSpace(req.FinalizedCheckpointID)
	return h, nil
}

func runtimeHandoffQualityThreshold(a *Assembler) float64 {
	if a == nil || a.runtimeContextConfig == nil {
		return 0
	}
	threshold := a.runtimeContextConfig().JIT.Compaction.Handoff.QualityThreshold
	if threshold < 0 || threshold > 1 {
		return 0
	}
	return threshold
}

func normalizeHandoffFallbackReason(reason string) string {
	switch strings.TrimSpace(reason) {
	case "quality_below_threshold", handoff.FallbackQualityBelowThreshold:
		return handoff.FallbackQualityBelowThreshold
	case "invalid_cut", handoff.FallbackInvalidCut:
		return handoff.FallbackInvalidCut
	case "reference_loss", handoff.FallbackReferenceLoss:
		return handoff.FallbackReferenceLoss
	default:
		if strings.TrimSpace(reason) == "" {
			return handoff.FallbackGenerationFailure
		}
		return strings.TrimSpace(reason)
	}
}

func cloneHandoffIntMap(in map[string]int) map[string]int {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]int, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func selectHandoffCut(req types.ContextAssembleRequest, result types.ContextAssembleResult) handoff.Cut {
	if strings.TrimSpace(req.FinalizedCheckpointID) != "" {
		return handoff.CutCheckpoint
	}
	if req.StreamFlushed {
		return handoff.CutFlushedStream
	}
	if strings.EqualFold(strings.TrimSpace(result.Status), StatusBypass) {
		return handoff.CutCheckpoint
	}
	for _, outcome := range req.ToolResult {
		if outcome.Lifecycle != nil && outcome.Lifecycle.Finalized {
			return handoff.CutToolFinalized
		}
	}
	return handoff.CutFinalizedEvent
}

// RestoreHandoff validates and resolves a handoff before any caller-owned
// context mutation. The assembler remains stateless with respect to source
// history; callers decide how to inject the returned restore projection.
func (a *Assembler) RestoreHandoff(h handoff.Handoff, resolver handoff.Resolver) (handoff.RestoreResult, error) {
	return a.restoreHandoff(h, resolver, nil)
}

// RestoreHandoffWithStore performs reference validation and uses the injected
// durable operation store when available. The store is consulted only after
// schema validation and before any source-owner mutation.
func (a *Assembler) RestoreHandoffWithStore(h handoff.Handoff, resolver handoff.Resolver, store handoff.RestoreOperationStore) (handoff.RestoreResult, error) {
	return a.restoreHandoff(h, resolver, store)
}

func (a *Assembler) restoreHandoff(h handoff.Handoff, resolver handoff.Resolver, store handoff.RestoreOperationStore) (handoff.RestoreResult, error) {
	if err := h.Validate(handoff.DefaultLimits()); err != nil {
		return handoff.RestoreResult{}, err
	}
	id := handoff.ID(h)
	if a != nil {
		a.mu.Lock()
		if result, ok := a.restoredHandoffs[id]; ok {
			a.mu.Unlock()
			return result, nil
		}
		a.mu.Unlock()
	}
	if store == nil && a != nil {
		store = a.handoffRestoreStore
	}
	result, err := handoff.RestoreWithStore(h, resolver, store)
	if err != nil {
		return handoff.RestoreResult{}, err
	}
	if a != nil {
		a.mu.Lock()
		if a.restoredHandoffs == nil {
			a.restoredHandoffs = map[string]handoff.RestoreResult{}
		}
		if existing, ok := a.restoredHandoffs[id]; ok {
			result = existing
		} else {
			a.restoredHandoffs[id] = result
		}
		a.mu.Unlock()
	}
	return result, nil
}
