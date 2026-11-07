package composer

import (
	"context"
	"strings"

	"github.com/FelixSeptem/baymax/core/types"
)

// ContinuationCapabilities publishes child-only evidence from the configured
// source owner. Memory stores and owner-less journals deliberately cannot
// claim fresh-process restore.
func (c *Composer) ContinuationCapabilities(ctx context.Context) (types.ContinuationCapabilities, error) {
	if err := ctx.Err(); err != nil {
		return types.ContinuationCapabilities{}, err
	}
	if c == nil || c.childContinuationNamespace != "child" || c.childContinuationExecutor == nil || c.nativeHistory == nil {
		return types.ContinuationCapabilities{}, types.ErrCheckpointUnsupported
	}
	if _, ok := c.childContinuationStore.(*FileChildContinuationStore); !ok {
		return types.ContinuationCapabilities{}, types.ErrCheckpointUnsupported
	}
	source := c.nativeHistory.Source()
	if source.Kind != types.HistorySourceChild || strings.TrimSpace(source.ID) == "" {
		return types.ContinuationCapabilities{}, types.ErrCheckpointUnsupported
	}
	historyCaps, err := c.nativeHistory.Capabilities(ctx, source)
	if err != nil {
		return types.ContinuationCapabilities{}, err
	}
	if !historyCaps.Supported || !historyCaps.Durable {
		return types.ContinuationCapabilities{}, types.ErrCheckpointUnsupported
	}
	deployment := historyCaps.Deployment
	if fileStore, ok := c.childContinuationStore.(*FileChildContinuationStore); ok && fileStore.DeploymentScope() != (types.DeploymentScope{}) {
		deployment = fileStore.DeploymentScope()
	}
	caps := types.ContinuationCapabilities{
		ProfileVersion:      types.NativeContinuationVersion,
		SourceNamespace:     "child",
		ContractVersion:     "baymax.runtime.contract.v1",
		ModuleRevision:      "composer-child-continuation",
		APIVersion:          "host_runtime_capability_manifest.v1",
		RetentionPolicy:     historyCaps.RetentionWindow,
		ExpiryPolicy:        "earliest child deadline or source retention",
		CheckpointCreation:  true,
		CrossProcessRestore: true,
		EvidenceReferences: map[string]string{
			"checkpoint_creation":   "test:TestComposerRestoresChildContinuationWithFreshOwnersForEachPauseReason",
			"cross_process_restore": "test:TestComposerRestoresChildContinuationWithFreshOwnersForEachPauseReason",
		},
		PauseReasonEvidence: map[string]types.PauseReasonCapability{
			"input_required":    {Supported: true, EvidenceReference: "test:TestComposerRestoresChildContinuationWithFreshOwnersForEachPauseReason/input_required"},
			"approval_required": {Supported: true, EvidenceReference: "test:TestComposerRestoresChildContinuationWithFreshOwnersForEachPauseReason/approval_required"},
			"budget_exhausted":  {Supported: true, EvidenceReference: "test:TestComposerRestoresChildContinuationWithFreshOwnersForEachPauseReason/budget_exhausted"},
		},
		Deployment: deployment,
	}
	if err := caps.Validate(); err != nil {
		return types.ContinuationCapabilities{}, types.ErrCheckpointUnsupported
	}
	return caps, nil
}
