package snapshot

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestMarshalUnmarshalManifestRoundTrip(t *testing.T) {
	in := Manifest{
		SchemaVersion: ManifestSchemaVersionV1,
		ExportedAt:    time.Unix(1_700_000_000, 0).UTC(),
		Source: Source{
			Component: "composer",
			RunID:     "run-a66",
			SessionID: "session-a66",
		},
		Segments: Segments{
			RunnerSession: Segment{
				Version: RunnerSessionSegmentVersionV1,
				Payload: json.RawMessage(`{"run_id":"run-a66","status":"running"}`),
			},
			SchedulerMailbox: Segment{
				Version: SchedulerMailboxSegmentVersionV1,
				Payload: json.RawMessage(`{"scheduler":{"tasks":[]},"mailbox":{"records":[]}}`),
			},
			ComposerRecovery: Segment{
				Version: ComposerRecoverySegmentVersionV1,
				Payload: json.RawMessage(`{"replay":{"sequence":1,"terminal_commit_count":0}}`),
			},
			Memory: Segment{
				Version: MemorySegmentVersionV1,
				Payload: json.RawMessage(`{"lifecycle":{"retention_days":30,"ttl":"168h"}}`),
			},
		},
	}

	raw, err := MarshalManifest(in)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	out, err := UnmarshalManifest(raw)
	if err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	if out.Digest == "" {
		t.Fatalf("digest is empty after unmarshal")
	}
	if out.SchemaVersion != ManifestSchemaVersionV1 {
		t.Fatalf("schema_version = %q, want %q", out.SchemaVersion, ManifestSchemaVersionV1)
	}
	if out.Segments.Memory.Version != MemorySegmentVersionV1 {
		t.Fatalf("memory segment version = %q, want %q", out.Segments.Memory.Version, MemorySegmentVersionV1)
	}

	// Same semantic payload should be stable after decode+encode.
	raw2, err := MarshalManifest(out)
	if err != nil {
		t.Fatalf("marshal manifest second pass: %v", err)
	}
	if string(raw) != string(raw2) {
		t.Fatalf("manifest marshal is not stable:\nfirst=%s\nsecond=%s", raw, raw2)
	}
}

func TestUnmarshalManifestMissingRequiredFieldsFailsFast(t *testing.T) {
	raw := []byte(`{
		"schema_version":"state_session_snapshot.v1",
		"exported_at":"2024-01-01T00:00:00Z",
		"source":{"run_id":"run-a66"},
		"segments":{
			"runner_session":{"version":"runner_session.v1","payload":{"run_id":"run-a66"}},
			"scheduler_mailbox":{"version":"scheduler_mailbox.v1","payload":{"scheduler":{"tasks":[]},"mailbox":{"records":[]}}},
			"composer_recovery":{"version":"composer_recovery.v1","payload":{"replay":{"sequence":1,"terminal_commit_count":0}}},
			"memory":{"version":"memory.v1","payload":{"lifecycle":{"retention_days":30}}}
		},
		"digest":"x"
	}`)

	_, err := UnmarshalManifest(raw)
	if err == nil || !strings.Contains(err.Error(), "source.component") {
		t.Fatalf("expected source.component validation error, got %v", err)
	}
}

func TestUnmarshalManifestWrongVersionFailsFast(t *testing.T) {
	raw := []byte(`{
		"schema_version":"state_session_snapshot.v9",
		"exported_at":"2024-01-01T00:00:00Z",
		"source":{"component":"composer"},
		"segments":{
			"runner_session":{"version":"runner_session.v1","payload":{"run_id":"run-a66"}},
			"scheduler_mailbox":{"version":"scheduler_mailbox.v1","payload":{"scheduler":{"tasks":[]},"mailbox":{"records":[]}}},
			"composer_recovery":{"version":"composer_recovery.v1","payload":{"replay":{"sequence":1,"terminal_commit_count":0}}},
			"memory":{"version":"memory.v1","payload":{"lifecycle":{"retention_days":30}}}
		},
		"digest":"x"
	}`)

	_, err := UnmarshalManifest(raw)
	if err == nil || !strings.Contains(err.Error(), "schema_version") {
		t.Fatalf("expected schema_version validation error, got %v", err)
	}
}

func TestUnmarshalManifestDigestMismatchFailsFast(t *testing.T) {
	in := Manifest{
		SchemaVersion: ManifestSchemaVersionV1,
		ExportedAt:    time.Unix(1_700_000_000, 0).UTC(),
		Source:        Source{Component: "composer", RunID: "run-a66"},
		Segments: Segments{
			RunnerSession: Segment{
				Version: RunnerSessionSegmentVersionV1,
				Payload: json.RawMessage(`{"run_id":"run-a66"}`),
			},
			SchedulerMailbox: Segment{
				Version: SchedulerMailboxSegmentVersionV1,
				Payload: json.RawMessage(`{"scheduler":{"tasks":[]},"mailbox":{"records":[]}}`),
			},
			ComposerRecovery: Segment{
				Version: ComposerRecoverySegmentVersionV1,
				Payload: json.RawMessage(`{"replay":{"sequence":1,"terminal_commit_count":0}}`),
			},
			Memory: Segment{
				Version: MemorySegmentVersionV1,
				Payload: json.RawMessage(`{"lifecycle":{"retention_days":30}}`),
			},
		},
	}
	raw, err := MarshalManifest(in)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("decode marshaled manifest: %v", err)
	}
	doc["digest"] = "digest-mismatch"
	tampered, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("re-encode tampered manifest: %v", err)
	}

	_, err = UnmarshalManifest(tampered)
	if err == nil || !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("expected digest mismatch error, got %v", err)
	}
}
