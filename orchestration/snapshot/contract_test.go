package snapshot

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestExportImportRoundTripStable(t *testing.T) {
	raw, err := Export(ExportRequest{
		ExportedAt: time.Unix(1_700_000_000, 0).UTC(),
		Source: Source{
			Component: "composer",
			RunID:     "run-a66",
			SessionID: "session-a66",
		},
		RunnerSessionPayload: map[string]any{
			"run_id":     "run-a66",
			"session_id": "session-a66",
			"status":     "running",
		},
		SchedulerMailboxPayload: map[string]any{
			"scheduler": map[string]any{"tasks": []any{}},
			"mailbox":   map[string]any{"records": []any{}},
		},
		ComposerRecoveryPayload: map[string]any{
			"replay": map[string]any{"sequence": 1, "terminal_commit_count": 0},
		},
		MemoryPayload: map[string]any{
			"lifecycle": map[string]any{"retention_days": 30, "ttl": "168h"},
		},
	})
	if err != nil {
		t.Fatalf("export snapshot: %v", err)
	}

	importer := NewImporter()
	res, err := importer.Import(ImportRequest{
		Payload:      raw,
		RestoreMode:  RestoreModeStrict,
		CompatWindow: 1,
		OperationID:  "op-a66-roundtrip",
	})
	if err != nil {
		t.Fatalf("import snapshot: %v", err)
	}
	if !res.Applied || res.Idempotent {
		t.Fatalf("expected first import applied=true idempotent=false, got %#v", res)
	}

	raw2, err := MarshalManifest(res.Manifest)
	if err != nil {
		t.Fatalf("marshal manifest after import: %v", err)
	}
	if string(raw) != string(raw2) {
		t.Fatalf("export->import->export is not stable:\nfirst=%s\nsecond=%s", raw, raw2)
	}
}

func TestImportIdempotencyNoInflation(t *testing.T) {
	raw := mustExportSnapshot(t)
	importer := NewImporter()

	first, err := importer.Import(ImportRequest{
		Payload:      raw,
		RestoreMode:  RestoreModeStrict,
		CompatWindow: 1,
		OperationID:  "op-a66-idempotent",
	})
	if err != nil {
		t.Fatalf("first import failed: %v", err)
	}
	second, err := importer.Import(ImportRequest{
		Payload:      raw,
		RestoreMode:  RestoreModeStrict,
		CompatWindow: 1,
		OperationID:  "op-a66-idempotent",
	})
	if err != nil {
		t.Fatalf("second import failed: %v", err)
	}

	if !first.Applied || first.Idempotent {
		t.Fatalf("first import expected applied=true idempotent=false, got %#v", first)
	}
	if second.Applied || !second.Idempotent {
		t.Fatalf("second import expected applied=false idempotent=true, got %#v", second)
	}
	if second.RestoreAction != RestoreActionIdempotentNoop {
		t.Fatalf("second import restore action = %q, want %q", second.RestoreAction, RestoreActionIdempotentNoop)
	}
}

func TestImportStrictRejectsIncompatibleVersion(t *testing.T) {
	raw := mustExportSnapshot(t)
	manifest, err := decodeManifest(raw, true, false)
	if err != nil {
		t.Fatalf("decode snapshot: %v", err)
	}
	manifest.SchemaVersion = "state_session_snapshot.v2"
	manifest.Digest, err = ComputeManifestDigest(manifest)
	if err != nil {
		t.Fatalf("compute digest for tampered manifest: %v", err)
	}
	tampered, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal tampered snapshot: %v", err)
	}

	_, err = NewImporter().Import(ImportRequest{
		Payload:      tampered,
		RestoreMode:  RestoreModeStrict,
		CompatWindow: 1,
		OperationID:  "op-a66-strict-reject",
	})
	if err == nil {
		t.Fatalf("expected strict import incompatible error")
	}
	var importErr *ImportError
	if !asImportError(err, &importErr) {
		t.Fatalf("expected ImportError, got %T: %v", err, err)
	}
	if importErr.ConflictCode != ConflictCodeStrictIncompatible {
		t.Fatalf("conflict_code = %q, want %q", importErr.ConflictCode, ConflictCodeStrictIncompatible)
	}
}

func TestImportCompatibleWithinWindow(t *testing.T) {
	raw := mustExportSnapshot(t)
	manifest, err := decodeManifest(raw, true, false)
	if err != nil {
		t.Fatalf("decode snapshot: %v", err)
	}
	manifest.SchemaVersion = "state_session_snapshot.v2"
	manifest.Segments.RunnerSession.Version = "runner_session.v2"
	manifest.Segments.SchedulerMailbox.Version = "scheduler_mailbox.v2"
	manifest.Segments.ComposerRecovery.Version = "composer_recovery.v2"
	manifest.Segments.Memory.Version = "memory.v2"
	manifest.Digest, err = ComputeManifestDigest(manifest)
	if err != nil {
		t.Fatalf("compute digest for compatible tampered manifest: %v", err)
	}
	tampered, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal compatible tampered snapshot: %v", err)
	}

	res, err := NewImporter().Import(ImportRequest{
		Payload:      tampered,
		RestoreMode:  RestoreModeCompatible,
		CompatWindow: 1,
		OperationID:  "op-a66-compatible",
	})
	if err != nil {
		t.Fatalf("compatible import should pass, got %v", err)
	}
	if res.RestoreAction != RestoreActionCompatibleBounded {
		t.Fatalf("restore action = %q, want %q", res.RestoreAction, RestoreActionCompatibleBounded)
	}
}

func TestImportSameOperationDifferentDigestConflict(t *testing.T) {
	raw := mustExportSnapshot(t)
	importer := NewImporter()
	if _, err := importer.Import(ImportRequest{
		Payload:      raw,
		RestoreMode:  RestoreModeStrict,
		CompatWindow: 1,
		OperationID:  "op-a66-conflict",
	}); err != nil {
		t.Fatalf("first import failed: %v", err)
	}

	manifest, err := decodeManifest(raw, true, false)
	if err != nil {
		t.Fatalf("decode snapshot: %v", err)
	}
	manifest.Source.Component = "composer-updated"
	manifest.Digest, err = ComputeManifestDigest(manifest)
	if err != nil {
		t.Fatalf("compute digest after mutation: %v", err)
	}
	changed, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal changed manifest: %v", err)
	}

	_, err = importer.Import(ImportRequest{
		Payload:      changed,
		RestoreMode:  RestoreModeStrict,
		CompatWindow: 1,
		OperationID:  "op-a66-conflict",
	})
	if err == nil {
		t.Fatalf("expected operation conflict error")
	}
	var importErr *ImportError
	if !asImportError(err, &importErr) {
		t.Fatalf("expected ImportError, got %T: %v", err, err)
	}
	if importErr.ConflictCode != ConflictCodeOperationConflict {
		t.Fatalf("conflict_code = %q, want %q", importErr.ConflictCode, ConflictCodeOperationConflict)
	}
}

func mustExportSnapshot(t *testing.T) []byte {
	t.Helper()
	raw, err := Export(ExportRequest{
		ExportedAt: time.Unix(1_700_000_000, 0).UTC(),
		Source: Source{
			Component: "composer",
			RunID:     "run-a66",
			SessionID: "session-a66",
		},
		RunnerSessionPayload: map[string]any{
			"run_id":     "run-a66",
			"session_id": "session-a66",
			"status":     "running",
		},
		SchedulerMailboxPayload: map[string]any{
			"scheduler": map[string]any{"tasks": []any{}},
			"mailbox":   map[string]any{"records": []any{}},
		},
		ComposerRecoveryPayload: map[string]any{
			"replay": map[string]any{"sequence": 1, "terminal_commit_count": 0},
		},
		MemoryPayload: map[string]any{
			"lifecycle": map[string]any{"retention_days": 30, "ttl": "168h"},
		},
	})
	if err != nil {
		t.Fatalf("export snapshot: %v", err)
	}
	if !strings.Contains(string(raw), `"schema_version":"state_session_snapshot.v1"`) {
		t.Fatalf("exported snapshot missing schema_version v1: %s", raw)
	}
	return raw
}

func asImportError(err error, out **ImportError) bool {
	if err == nil {
		return false
	}
	typed, ok := err.(*ImportError)
	if !ok {
		return false
	}
	*out = typed
	return true
}
