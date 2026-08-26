package snapshot

import (
	"fmt"
	"strings"

	"github.com/FelixSeptem/baymax/core/types"
)

// ProtocolCheckpointRef projects the existing manifest as a reference. The
// manifest remains the owner of segments, digest validation, and restore mode.
func ProtocolCheckpointRef(manifest Manifest) (types.CheckpointRef, error) {
	ref := types.CheckpointRef{
		CheckpointID:    strings.TrimSpace(manifest.Digest),
		SchemaVersion:   strings.TrimSpace(manifest.SchemaVersion),
		SourceComponent: strings.TrimSpace(manifest.Source.Component),
		Digest:          strings.TrimSpace(manifest.Digest),
		RunID:           strings.TrimSpace(manifest.Source.RunID),
		SessionID:       strings.TrimSpace(manifest.Source.SessionID),
	}
	if err := ref.ValidateProtocolReference(); err != nil {
		return types.CheckpointRef{}, fmt.Errorf("snapshot protocol checkpoint: %w", err)
	}
	return ref, nil
}
