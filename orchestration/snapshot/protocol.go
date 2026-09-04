package snapshot

import (
	"fmt"
	"strings"

	"github.com/FelixSeptem/baymax/core/types"
)

// ProtocolCheckpointRef projects the existing manifest as a reference. The
// manifest remains the owner of segments, digest validation, and restore mode.
func ProtocolCheckpointRef(manifest Manifest) (types.CheckpointRef, error) {
	return ProtocolCheckpointRefWithContext(manifest, CheckpointProjectionContext{})
}

// CheckpointProjectionContext carries source-owned recovery metadata into the
// reference-only protocol projection.
type CheckpointProjectionContext struct {
	Relation            types.CheckpointRelation
	ParentCheckpointID  string
	BranchID            string
	HistoryIndex        int
	HistoryRootID       string
	HistoryLeafID       string
	HistoryDigest       string
	RestoreSource       types.CheckpointRestoreSource
	ReplayKey           string
	WorkspaceProvenance *types.WorkspaceProvenance
}

// ProtocolCheckpointRefWithContext projects a manifest and optional recovery
// context without changing manifest storage or restore behavior.
func ProtocolCheckpointRefWithContext(manifest Manifest, context CheckpointProjectionContext) (types.CheckpointRef, error) {
	ref := types.CheckpointRef{
		CheckpointID:        strings.TrimSpace(manifest.Digest),
		SchemaVersion:       strings.TrimSpace(manifest.SchemaVersion),
		SourceComponent:     strings.TrimSpace(manifest.Source.Component),
		Digest:              strings.TrimSpace(manifest.Digest),
		RunID:               strings.TrimSpace(manifest.Source.RunID),
		SessionID:           strings.TrimSpace(manifest.Source.SessionID),
		Relation:            context.Relation,
		ParentCheckpointID:  strings.TrimSpace(context.ParentCheckpointID),
		BranchID:            strings.TrimSpace(context.BranchID),
		HistoryIndex:        context.HistoryIndex,
		HistoryRootID:       strings.TrimSpace(context.HistoryRootID),
		HistoryLeafID:       strings.TrimSpace(context.HistoryLeafID),
		HistoryDigest:       strings.TrimSpace(context.HistoryDigest),
		RestoreSource:       context.RestoreSource,
		ReplayKey:           strings.TrimSpace(context.ReplayKey),
		WorkspaceProvenance: context.WorkspaceProvenance,
	}
	if err := ref.ValidateProtocolReference(); err != nil {
		return types.CheckpointRef{}, fmt.Errorf("snapshot protocol checkpoint: %w", err)
	}
	return ref, nil
}
