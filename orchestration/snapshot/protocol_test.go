package snapshot

import (
	"testing"
	"time"
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
