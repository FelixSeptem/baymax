package composer

import (
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	orchestrationsnapshot "github.com/FelixSeptem/baymax/orchestration/snapshot"
)

func TestProtocolCheckpointRefForRestorePreservesSourceContext(t *testing.T) {
	ref, err := ProtocolCheckpointRefForRestore(orchestrationsnapshot.Manifest{SchemaVersion: orchestrationsnapshot.ManifestSchemaVersionV1, ExportedAt: time.Now(), Source: orchestrationsnapshot.Source{Component: "composer"}, Digest: "digest"}, types.CheckpointRestoreSourceCrossSession, types.CheckpointRelationRoot, "", "", "", 0, nil)
	if err != nil {
		t.Fatalf("projection error = %v", err)
	}
	if ref.RestoreSource != types.CheckpointRestoreSourceCrossSession || ref.Relation != types.CheckpointRelationRoot {
		t.Fatalf("ref = %#v", ref)
	}
}
