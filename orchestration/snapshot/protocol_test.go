package snapshot

import (
	"reflect"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

func TestProtocolCheckpointRefFromManifestPreservesDigestAndSource(t *testing.T) {
	manifest := Manifest{
		SchemaVersion: ManifestSchemaVersionV1,
		ExportedAt:    time.Date(2026, time.August, 12, 10, 0, 0, 0, time.UTC),
		Source:        Source{Component: "composer", RunID: "run-1", SessionID: "session-1"},
		Digest:        "sha256:abc",
	}
	ref, err := ProtocolCheckpointRef(manifest)
	if err != nil {
		t.Fatalf("ProtocolCheckpointRef() error = %v", err)
	}
	if ref.CheckpointID != manifest.Digest || ref.SchemaVersion != manifest.SchemaVersion || ref.SourceComponent != "composer" || ref.Digest != manifest.Digest || ref.RunID != "run-1" || ref.SessionID != "session-1" {
		t.Fatalf("ref = %#v", ref)
	}
}

func TestProtocolCheckpointRefWithProvenanceProjectsRecoveryContext(t *testing.T) {
	manifest := Manifest{SchemaVersion: ManifestSchemaVersionV1, ExportedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Source: Source{Component: "composer", RunID: "run-1", SessionID: "session-1"}, Digest: "digest-1"}
	ref, err := ProtocolCheckpointRefWithContext(manifest, CheckpointProjectionContext{Relation: types.CheckpointRelationDerived, ParentCheckpointID: "checkpoint-root", HistoryIndex: 1, RestoreSource: types.CheckpointRestoreSourceResume, ReplayKey: "replay-1", WorkspaceProvenance: &types.WorkspaceProvenance{WorkspaceID: "workspace-1", ChangeSetID: "change-1", BeforeIntegrity: "before", AfterIntegrity: "after", ProducedByRunID: "run-1", ProducedByStepID: "step-1"}})
	if err != nil {
		t.Fatalf("ProtocolCheckpointRefWithContext() error = %v", err)
	}
	if ref.Relation != types.CheckpointRelationDerived || ref.ParentCheckpointID != "checkpoint-root" || ref.RestoreSource != types.CheckpointRestoreSourceResume || ref.WorkspaceProvenance == nil {
		t.Fatalf("projection lost context: %#v", ref)
	}
}

func TestProtocolCheckpointProjectionDoesNotMutateManifestOnSuccess(t *testing.T) {
	manifest := Manifest{SchemaVersion: ManifestSchemaVersionV1, ExportedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Source: Source{Component: "composer", RunID: "run-1", SessionID: "session-1"}, Digest: "digest-1"}
	original := manifest
	workspace := &types.WorkspaceProvenance{WorkspaceID: "workspace-1", ChangeSetID: "change-1", BeforeIntegrity: "before", AfterIntegrity: "after", ProducedByRunID: "run-1", ProducedByStepID: "step-1"}
	if _, err := ProtocolCheckpointRefWithContext(manifest, CheckpointProjectionContext{Relation: types.CheckpointRelationDerived, ParentCheckpointID: "digest-root", HistoryIndex: 1, RestoreSource: types.CheckpointRestoreSourceResume, WorkspaceProvenance: workspace}); err != nil {
		t.Fatalf("projection error = %v", err)
	}
	if !reflect.DeepEqual(manifest, original) {
		t.Fatalf("projection mutated manifest: before=%#v after=%#v", original, manifest)
	}
}

func TestProtocolCheckpointProjectionRejectsInvalidProvenanceWithoutMutation(t *testing.T) {
	manifest := Manifest{SchemaVersion: ManifestSchemaVersionV1, ExportedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Source: Source{Component: "composer", RunID: "run-1", SessionID: "session-1"}, Digest: "digest-1"}
	original := manifest
	_, err := ProtocolCheckpointRefWithContext(manifest, CheckpointProjectionContext{Relation: types.CheckpointRelationRoot, RestoreSource: types.CheckpointRestoreSourceFresh, WorkspaceProvenance: &types.WorkspaceProvenance{WorkspaceID: "workspace-1", ChangeSetID: "change-1"}})
	if err == nil {
		t.Fatal("invalid provenance should fail")
	}
	if !reflect.DeepEqual(manifest, original) {
		t.Fatalf("failed projection mutated manifest: before=%#v after=%#v", original, manifest)
	}
}
