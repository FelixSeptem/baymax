package checkpointworkspaceprovenance

import (
	"fmt"
	"strings"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/orchestration/snapshot"
)

func RunMinimal()    { run(false) }
func RunProduction() { run(true) }

func run(production bool) {
	variant := "minimal"
	markers := []string{"checkpoint_history_projected", "checkpoint_lineage_validated", "workspace_provenance_projected"}
	if production {
		variant = "production-ish"
		markers = append(markers, "checkpoint_replay_idempotent", "workspace_integrity_drift_classified", "governance_checkpoint_provenance_replay_bound")
	}
	root, err := snapshot.ProtocolCheckpointRefWithContext(snapshot.Manifest{SchemaVersion: snapshot.ManifestSchemaVersionV1, ExportedAt: time.Date(2026, 8, 29, 0, 0, 0, 0, time.UTC), Source: snapshot.Source{Component: "example", RunID: "run-1", SessionID: "session-1"}, Digest: "checkpoint-root"}, snapshot.CheckpointProjectionContext{Relation: types.CheckpointRelationRoot})
	if err != nil {
		panic(err)
	}
	derived, err := snapshot.ProtocolCheckpointRefWithContext(snapshot.Manifest{SchemaVersion: snapshot.ManifestSchemaVersionV1, ExportedAt: time.Date(2026, 8, 29, 0, 1, 0, 0, time.UTC), Source: snapshot.Source{Component: "example", RunID: "run-1", SessionID: "session-1"}, Digest: "checkpoint-derived"}, snapshot.CheckpointProjectionContext{Relation: types.CheckpointRelationDerived, ParentCheckpointID: root.CheckpointID, HistoryIndex: 1, RestoreSource: types.CheckpointRestoreSourceResume, ReplayKey: "example-replay", WorkspaceProvenance: &types.WorkspaceProvenance{WorkspaceID: "workspace-example", ChangeSetID: "change-1", BeforeIntegrity: "before", AfterIntegrity: "after", ProducedByRunID: "run-1", ProducedByStepID: "step-1"}})
	if err != nil || types.ValidateCheckpointHistory([]types.CheckpointRef{root, derived}) != nil {
		panic("checkpoint provenance validation failed")
	}
	if production && (types.ValidateCheckpointReplay(derived, derived) != nil || types.ValidateWorkspaceIntegrity("different", *derived.WorkspaceProvenance) == nil) {
		panic("provenance drift validation failed")
	}
	fmt.Printf("pattern=checkpoint-workspace-provenance\nvariant=%s\nverification.semantic.anchor=agent_runtime_protocol.checkpoint_history_workspace_provenance\nverification.semantic.expected_markers=%s\nresult.checkpoint_id=%s\n", variant, strings.Join(markers, ","), derived.CheckpointID)
}
