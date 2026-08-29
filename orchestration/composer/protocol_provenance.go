package composer

import (
	"fmt"
	"strings"

	"github.com/FelixSeptem/baymax/core/types"
	orchestrationsnapshot "github.com/FelixSeptem/baymax/orchestration/snapshot"
)

// ProtocolCheckpointRefForRestore projects source-owned composer restore
// context without changing import or recovery state.
func ProtocolCheckpointRefForRestore(manifest orchestrationsnapshot.Manifest, restoreSource types.CheckpointRestoreSource, relation types.CheckpointRelation, parentID, branchID, replayKey string, historyIndex int, workspace *types.WorkspaceProvenance) (types.CheckpointRef, error) {
	if strings.TrimSpace(string(restoreSource)) == "" {
		restoreSource = types.CheckpointRestoreSourceFresh
	}
	ref, err := orchestrationsnapshot.ProtocolCheckpointRefWithContext(manifest, orchestrationsnapshot.CheckpointProjectionContext{Relation: relation, ParentCheckpointID: parentID, BranchID: branchID, ReplayKey: replayKey, HistoryIndex: historyIndex, RestoreSource: restoreSource, WorkspaceProvenance: workspace})
	if err != nil {
		return types.CheckpointRef{}, fmt.Errorf("composer protocol checkpoint restore projection: %w", err)
	}
	return ref, nil
}
