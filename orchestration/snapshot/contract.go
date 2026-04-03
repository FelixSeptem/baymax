package snapshot

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	RestoreModeStrict     = "strict"
	RestoreModeCompatible = "compatible"

	RestoreActionStrictExact       = "strict_exact_restore"
	RestoreActionCompatibleBounded = "compatible_bounded_restore"
	RestoreActionIdempotentNoop    = "idempotent_noop"
	RestoreActionCompatibleExact   = "compatible_exact_restore"
)

const (
	ConflictCodeInvalidPayload       = "state_snapshot_invalid_payload"
	ConflictCodeInvalidRestoreMode   = "state_snapshot_restore_mode_invalid"
	ConflictCodeStrictIncompatible   = "state_snapshot_strict_incompatible"
	ConflictCodeCompatWindowExceeded = "state_snapshot_compat_window_exceeded"
	ConflictCodeOperationConflict    = "state_snapshot_operation_conflict"
	ConflictCodeDigestMismatch       = "state_snapshot_digest_mismatch"
)

type ExportRequest struct {
	ExportedAt time.Time `json:"exported_at"`
	Source     Source    `json:"source"`

	RunnerSessionPayload    any `json:"runner_session_payload"`
	SchedulerMailboxPayload any `json:"scheduler_mailbox_payload"`
	ComposerRecoveryPayload any `json:"composer_recovery_payload"`
	MemoryPayload           any `json:"memory_payload"`
}

type ImportRequest struct {
	Payload      []byte `json:"payload"`
	RestoreMode  string `json:"restore_mode"`
	CompatWindow int    `json:"compat_window"`
	OperationID  string `json:"operation_id"`
}

type ImportResult struct {
	OperationID    string   `json:"operation_id"`
	RestoreMode    string   `json:"restore_mode"`
	RestoreAction  string   `json:"restore_action"`
	Idempotent     bool     `json:"idempotent"`
	Applied        bool     `json:"applied"`
	ManifestDigest string   `json:"manifest_digest"`
	Manifest       Manifest `json:"manifest"`
}

type ImportError struct {
	ConflictCode string
	Message      string
	Cause        error
}

func (e *ImportError) Error() string {
	if e == nil {
		return ""
	}
	msg := strings.TrimSpace(e.Message)
	if msg == "" && e.Cause != nil {
		msg = e.Cause.Error()
	}
	if msg == "" {
		msg = "state snapshot import failed"
	}
	code := strings.TrimSpace(e.ConflictCode)
	if code == "" {
		return msg
	}
	return fmt.Sprintf("%s: %s", code, msg)
}

func (e *ImportError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

type Importer struct {
	mu         sync.Mutex
	operations map[string]string
}

func NewImporter() *Importer {
	return &Importer{
		operations: map[string]string{},
	}
}

func ExportManifest(req ExportRequest) (Manifest, error) {
	exportedAt := req.ExportedAt
	if exportedAt.IsZero() {
		exportedAt = time.Now().UTC()
	}
	runnerSession, err := normalizeSegment(
		Segment{
			Version: RunnerSessionSegmentVersionV1,
			Payload: normalizePayload(req.RunnerSessionPayload),
		},
		"segments.runner_session",
		RunnerSessionSegmentVersionV1,
		true,
	)
	if err != nil {
		return Manifest{}, err
	}
	schedulerMailbox, err := normalizeSegment(
		Segment{
			Version: SchedulerMailboxSegmentVersionV1,
			Payload: normalizePayload(req.SchedulerMailboxPayload),
		},
		"segments.scheduler_mailbox",
		SchedulerMailboxSegmentVersionV1,
		true,
	)
	if err != nil {
		return Manifest{}, err
	}
	composerRecovery, err := normalizeSegment(
		Segment{
			Version: ComposerRecoverySegmentVersionV1,
			Payload: normalizePayload(req.ComposerRecoveryPayload),
		},
		"segments.composer_recovery",
		ComposerRecoverySegmentVersionV1,
		true,
	)
	if err != nil {
		return Manifest{}, err
	}
	memory, err := normalizeSegment(
		Segment{
			Version: MemorySegmentVersionV1,
			Payload: normalizePayload(req.MemoryPayload),
		},
		"segments.memory",
		MemorySegmentVersionV1,
		true,
	)
	if err != nil {
		return Manifest{}, err
	}

	normalized, err := normalizeManifest(
		Manifest{
			SchemaVersion: ManifestSchemaVersionV1,
			ExportedAt:    exportedAt.UTC(),
			Source:        req.Source,
			Segments: Segments{
				RunnerSession:    runnerSession,
				SchedulerMailbox: schedulerMailbox,
				ComposerRecovery: composerRecovery,
				Memory:           memory,
			},
		},
		false,
		true,
	)
	if err != nil {
		return Manifest{}, err
	}
	digest, err := ComputeManifestDigest(normalized)
	if err != nil {
		return Manifest{}, err
	}
	normalized.Digest = digest
	return normalized, nil
}

func Export(req ExportRequest) ([]byte, error) {
	manifest, err := ExportManifest(req)
	if err != nil {
		return nil, err
	}
	return json.Marshal(manifest)
}

func (i *Importer) Import(req ImportRequest) (ImportResult, error) {
	restoreMode := strings.ToLower(strings.TrimSpace(req.RestoreMode))
	if restoreMode == "" {
		restoreMode = RestoreModeStrict
	}
	if restoreMode != RestoreModeStrict && restoreMode != RestoreModeCompatible {
		return ImportResult{}, &ImportError{
			ConflictCode: ConflictCodeInvalidRestoreMode,
			Message:      fmt.Sprintf("restore_mode must be one of [%s,%s], got %q", RestoreModeStrict, RestoreModeCompatible, req.RestoreMode),
		}
	}
	if req.CompatWindow < 0 {
		return ImportResult{}, &ImportError{
			ConflictCode: ConflictCodeCompatWindowExceeded,
			Message:      "compat_window must be >= 0",
		}
	}

	manifest, err := decodeManifest(req.Payload, true, false)
	if err != nil {
		return ImportResult{}, &ImportError{
			ConflictCode: ConflictCodeInvalidPayload,
			Message:      "decode snapshot manifest failed",
			Cause:        err,
		}
	}
	expectedDigest, err := ComputeManifestDigest(manifest)
	if err != nil {
		return ImportResult{}, &ImportError{
			ConflictCode: ConflictCodeInvalidPayload,
			Message:      "compute snapshot digest failed",
			Cause:        err,
		}
	}
	if expectedDigest != manifest.Digest {
		return ImportResult{}, &ImportError{
			ConflictCode: ConflictCodeDigestMismatch,
			Message:      fmt.Sprintf("snapshot manifest digest mismatch: got=%q want=%q", manifest.Digest, expectedDigest),
		}
	}

	restoreAction, err := resolveRestoreAction(manifest, restoreMode, req.CompatWindow)
	if err != nil {
		return ImportResult{}, err
	}

	operationID := strings.TrimSpace(req.OperationID)
	if operationID == "" {
		operationID = manifest.Digest
	}

	if i == nil {
		return ImportResult{
			OperationID:    operationID,
			RestoreMode:    restoreMode,
			RestoreAction:  restoreAction,
			Idempotent:     false,
			Applied:        true,
			ManifestDigest: manifest.Digest,
			Manifest:       manifest,
		}, nil
	}

	i.mu.Lock()
	defer i.mu.Unlock()
	if i.operations == nil {
		i.operations = map[string]string{}
	}
	if knownDigest, ok := i.operations[operationID]; ok {
		if knownDigest == manifest.Digest {
			return ImportResult{
				OperationID:    operationID,
				RestoreMode:    restoreMode,
				RestoreAction:  RestoreActionIdempotentNoop,
				Idempotent:     true,
				Applied:        false,
				ManifestDigest: manifest.Digest,
				Manifest:       manifest,
			}, nil
		}
		return ImportResult{}, &ImportError{
			ConflictCode: ConflictCodeOperationConflict,
			Message:      fmt.Sprintf("operation_id %q already bound to digest %q, got %q", operationID, knownDigest, manifest.Digest),
		}
	}
	i.operations[operationID] = manifest.Digest
	return ImportResult{
		OperationID:    operationID,
		RestoreMode:    restoreMode,
		RestoreAction:  restoreAction,
		Idempotent:     false,
		Applied:        true,
		ManifestDigest: manifest.Digest,
		Manifest:       manifest,
	}, nil
}

func normalizePayload(payload any) json.RawMessage {
	if payload == nil {
		return json.RawMessage(`{}`)
	}
	switch typed := payload.(type) {
	case json.RawMessage:
		if len(strings.TrimSpace(string(typed))) == 0 {
			return json.RawMessage(`{}`)
		}
		return typed
	case []byte:
		if len(strings.TrimSpace(string(typed))) == 0 {
			return json.RawMessage(`{}`)
		}
		return json.RawMessage(typed)
	default:
		raw, err := json.Marshal(payload)
		if err != nil || len(strings.TrimSpace(string(raw))) == 0 {
			return json.RawMessage(`{}`)
		}
		return raw
	}
}

func resolveRestoreAction(manifest Manifest, restoreMode string, compatWindow int) (string, error) {
	versions := [][3]string{
		{"schema_version", ManifestSchemaVersionV1, manifest.SchemaVersion},
		{"segments.runner_session.version", RunnerSessionSegmentVersionV1, manifest.Segments.RunnerSession.Version},
		{"segments.scheduler_mailbox.version", SchedulerMailboxSegmentVersionV1, manifest.Segments.SchedulerMailbox.Version},
		{"segments.composer_recovery.version", ComposerRecoverySegmentVersionV1, manifest.Segments.ComposerRecovery.Version},
		{"segments.memory.version", MemorySegmentVersionV1, manifest.Segments.Memory.Version},
	}
	exact := true
	for _, version := range versions {
		if strings.TrimSpace(version[1]) != strings.TrimSpace(version[2]) {
			exact = false
			break
		}
	}

	if restoreMode == RestoreModeStrict {
		if exact {
			return RestoreActionStrictExact, nil
		}
		return "", &ImportError{
			ConflictCode: ConflictCodeStrictIncompatible,
			Message:      "strict restore rejected incompatible snapshot version",
		}
	}

	if exact {
		return RestoreActionCompatibleExact, nil
	}
	for _, version := range versions {
		if isWithinCompatibilityWindow(version[1], version[2], compatWindow) {
			continue
		}
		return "", &ImportError{
			ConflictCode: ConflictCodeCompatWindowExceeded,
			Message: fmt.Sprintf(
				"%s is outside compatibility window=%d expected=%q actual=%q",
				version[0],
				compatWindow,
				version[1],
				version[2],
			),
		}
	}
	return RestoreActionCompatibleBounded, nil
}

func isWithinCompatibilityWindow(expected, actual string, compatWindow int) bool {
	if compatWindow < 0 {
		return false
	}
	expectedPrefix, expectedVersion, ok := parseSnapshotVersion(expected)
	if !ok {
		return false
	}
	actualPrefix, actualVersion, ok := parseSnapshotVersion(actual)
	if !ok {
		return false
	}
	if expectedPrefix != actualPrefix {
		return false
	}
	delta := expectedVersion - actualVersion
	if delta < 0 {
		delta = -delta
	}
	return delta <= compatWindow
}

func parseSnapshotVersion(version string) (string, int, bool) {
	trimmed := strings.ToLower(strings.TrimSpace(version))
	idx := strings.LastIndex(trimmed, ".v")
	if idx <= 0 || idx+2 >= len(trimmed) {
		return "", 0, false
	}
	prefix := strings.TrimSpace(trimmed[:idx])
	if prefix == "" {
		return "", 0, false
	}
	numeric := strings.TrimSpace(trimmed[idx+2:])
	num, err := strconv.Atoi(numeric)
	if err != nil {
		return "", 0, false
	}
	return prefix, num, true
}
