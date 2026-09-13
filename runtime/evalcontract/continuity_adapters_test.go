package evalcontract

import (
	"testing"

	"github.com/FelixSeptem/baymax/context/handoff"
	"github.com/FelixSeptem/baymax/orchestration/snapshot"
)

func TestContinuityProjectionFromHandoffIsReferenceOnly(t *testing.T) {
	h := handoff.Handoff{Version: handoff.VersionV1, RunID: "run-1", SessionID: "session-1", Objective: "ship", Cut: handoff.CutCheckpoint, SourceCheckpointID: "cp-1", Quality: handoff.Quality{Score: 1, Threshold: .5}, References: []handoff.Reference{{Kind: handoff.ReferenceArtifact, ID: "artifact-1", Digest: "a1"}}}
	p, err := ContinuityProjectionFromHandoff(h, ContinuityPhaseBaseline)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Facts) != 3 || p.Facts[0].Kind != ContinuityKindArtifact {
		t.Fatalf("unexpected facts: %#v", p.Facts)
	}
	for _, fact := range p.Facts {
		if fact.Body != "" {
			t.Fatalf("body leaked: %#v", fact)
		}
	}
}

func TestContinuityProjectionFromSnapshotPreservesManifestReference(t *testing.T) {
	m := snapshot.Manifest{SchemaVersion: snapshot.ManifestSchemaVersionV1, Source: snapshot.Source{Component: "runner", RunID: "run-1", SessionID: "session-1"}, Digest: "digest-1"}
	p, err := ContinuityProjectionFromSnapshot(m, ContinuityPhaseCandidate)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Facts) != 1 || p.Facts[0].Kind != ContinuityKindCheckpoint || p.Facts[0].ID != "digest-1" || p.Facts[0].Owner != "snapshot" {
		t.Fatalf("unexpected projection: %#v", p)
	}
}
