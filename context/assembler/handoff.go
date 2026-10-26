package assembler

import (
	"context"
	"fmt"
	"strings"

	"github.com/FelixSeptem/baymax/context/handoff"
	"github.com/FelixSeptem/baymax/context/hostprojection"
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
	messageSource := modelReq.Messages
	if req.HostContextProjection != nil {
		// Assembled messages can contain host-resolved bodies or derivatives.
		// Only the original caller messages are eligible for legacy facts; host
		// projection material remains reference-only in the correlation.
		messageSource = req.Messages
	}
	messages := make([]handoff.Message, 0, len(messageSource))
	for _, msg := range messageSource {
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
	if req.HostContextProjection != nil {
		snapshot, err := hostprojection.Admit(*req.HostContextProjection)
		if err != nil {
			return handoff.Handoff{}, err
		}
		h.HostContextProjection = handoffHostProjectionCorrelation(snapshot.Descriptor())
		if err := h.Validate(handoff.DefaultLimits()); err != nil {
			return handoff.Handoff{}, err
		}
	}
	return h, nil
}

func handoffHostProjectionCorrelation(descriptor hostprojection.Descriptor) *handoff.HostContextProjectionCorrelation {
	correlation := &handoff.HostContextProjectionCorrelation{
		SchemaVersion:     descriptor.SchemaVersion,
		ProjectionID:      descriptor.ProjectionID,
		ProjectionVersion: descriptor.ProjectionVersion,
		Digest:            descriptor.Digest,
		CapabilityDigest:  descriptor.CapabilityDigest,
		Capabilities:      make([]string, len(descriptor.Capabilities)),
		SourceReferences:  make([]handoff.HostContextSourceReference, len(descriptor.SourceReferences)),
	}
	if descriptor.Binding != nil {
		binding := descriptor.Binding
		correlation.Binding = &handoff.HostContextSourceBinding{
			Form: string(binding.Form), OwnerNamespace: binding.OwnerNamespace, SourceID: binding.SourceID,
			NodeID: binding.NodeID, ContinuationID: binding.ContinuationID,
			ContinuationDigest: binding.ContinuationDigest, Generation: binding.Generation,
		}
	}
	for index, capability := range descriptor.Capabilities {
		correlation.Capabilities[index] = string(capability)
	}
	for index, reference := range descriptor.SourceReferences {
		correlation.SourceReferences[index] = handoff.HostContextSourceReference{Reference: reference.Reference, Digest: reference.Digest}
	}
	if descriptor.Parent != nil {
		correlation.Parent = &handoff.HostContextParentCorrelation{
			ProjectionID:     descriptor.Parent.ProjectionID,
			CapabilityDigest: descriptor.Parent.CapabilityDigest,
			Capabilities:     make([]string, len(descriptor.Parent.Capabilities)),
		}
		for index, capability := range descriptor.Parent.Capabilities {
			correlation.Parent.Capabilities[index] = string(capability)
		}
	}
	if descriptor.Plan != nil {
		correlation.Plan = &handoff.HostContextPlanCorrelation{PlanID: descriptor.Plan.PlanID, PlanVersion: descriptor.Plan.PlanVersion, Digest: descriptor.Plan.Digest}
	}
	return correlation
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
	if h.HostContextProjection != nil {
		return handoff.RestoreResult{}, fmt.Errorf("host context projection restore requires current host resolver")
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

// RestoreHandoffWithHostProjection preserves existing handoff schema/cut and
// owner-reference validation, then rechecks the bounded projection through the
// host's current resolver before consulting the normal durable restore identity
// path. It returns transient material for a caller-owned injection step; it
// never stores material, authorizes a business effect, or resumes execution.
func (a *Assembler) RestoreHandoffWithHostProjection(
	ctx context.Context,
	h handoff.Handoff,
	resolver handoff.Resolver,
	hostResolver hostprojection.Resolver,
	store handoff.RestoreOperationStore,
) (handoff.RestoreResult, hostprojection.Resolution, error) {
	if err := h.Validate(handoff.DefaultLimits()); err != nil {
		return handoff.RestoreResult{}, hostprojection.Resolution{}, err
	}
	if h.HostContextProjection == nil {
		return handoff.RestoreResult{}, hostprojection.Resolution{}, fmt.Errorf("host context projection correlation is required")
	}
	// The existing handoff source-owner validation always runs before projection
	// correlation and current authorization. This has no context injection.
	if _, err := handoff.Restore(h, resolver); err != nil {
		return handoff.RestoreResult{}, hostprojection.Resolution{}, err
	}
	descriptor, err := descriptorFromHandoffCorrelation(h.HostContextProjection)
	if err != nil {
		return handoff.RestoreResult{}, hostprojection.Resolution{}, err
	}
	_, resolution, err := hostprojection.Resolve(ctx, descriptor, hostResolver)
	if err != nil {
		return handoff.RestoreResult{}, hostprojection.Resolution{}, err
	}

	id := handoff.ID(h)
	if a != nil {
		a.mu.Lock()
		if result, ok := a.restoredHandoffs[id]; ok {
			a.mu.Unlock()
			return result, resolution, nil
		}
		a.mu.Unlock()
	}
	if store == nil && a != nil {
		store = a.handoffRestoreStore
	}
	result, err := handoff.RestoreWithStore(h, resolver, store)
	if err != nil {
		return handoff.RestoreResult{}, hostprojection.Resolution{}, err
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
	return result, resolution, nil
}

func descriptorFromHandoffCorrelation(correlation *handoff.HostContextProjectionCorrelation) (hostprojection.Descriptor, error) {
	if correlation == nil {
		return hostprojection.Descriptor{}, fmt.Errorf("host context projection correlation is required")
	}
	descriptor := hostprojection.Descriptor{
		SchemaVersion:     correlation.SchemaVersion,
		ProjectionID:      correlation.ProjectionID,
		ProjectionVersion: correlation.ProjectionVersion,
		Digest:            correlation.Digest,
		CapabilityDigest:  correlation.CapabilityDigest,
		Capabilities:      make([]hostprojection.Capability, len(correlation.Capabilities)),
		SourceReferences:  make([]hostprojection.SourceReference, len(correlation.SourceReferences)),
	}
	if correlation.Binding != nil {
		binding := correlation.Binding
		descriptor.Binding = &hostprojection.SourceBinding{
			Form: hostprojection.SourceForm(binding.Form), OwnerNamespace: binding.OwnerNamespace, SourceID: binding.SourceID,
			NodeID: binding.NodeID, ContinuationID: binding.ContinuationID,
			ContinuationDigest: binding.ContinuationDigest, Generation: binding.Generation,
		}
	}
	for index, capability := range correlation.Capabilities {
		descriptor.Capabilities[index] = hostprojection.Capability(capability)
	}
	for index, reference := range correlation.SourceReferences {
		descriptor.SourceReferences[index] = hostprojection.SourceReference{Reference: reference.Reference, Digest: reference.Digest}
	}
	if correlation.Parent != nil {
		descriptor.Parent = &hostprojection.ParentCapabilityEvidence{
			ProjectionID:     correlation.Parent.ProjectionID,
			CapabilityDigest: correlation.Parent.CapabilityDigest,
			Capabilities:     make([]hostprojection.Capability, len(correlation.Parent.Capabilities)),
		}
		for index, capability := range correlation.Parent.Capabilities {
			descriptor.Parent.Capabilities[index] = hostprojection.Capability(capability)
		}
	}
	if correlation.Plan != nil {
		descriptor.Plan = &hostprojection.PlanCorrelation{PlanID: correlation.Plan.PlanID, PlanVersion: correlation.Plan.PlanVersion, Digest: correlation.Plan.Digest}
	}
	return descriptor, nil
}
