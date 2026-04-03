package snapshot

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	ManifestSchemaVersionV1 = "state_session_snapshot.v1"

	RunnerSessionSegmentVersionV1    = "runner_session.v1"
	SchedulerMailboxSegmentVersionV1 = "scheduler_mailbox.v1"
	ComposerRecoverySegmentVersionV1 = "composer_recovery.v1"
	MemorySegmentVersionV1           = "memory.v1"
)

type Manifest struct {
	SchemaVersion string    `json:"schema_version"`
	ExportedAt    time.Time `json:"exported_at"`
	Source        Source    `json:"source"`
	Segments      Segments  `json:"segments"`
	Digest        string    `json:"digest"`
}

type Source struct {
	Component string `json:"component"`
	RunID     string `json:"run_id,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

type Segments struct {
	RunnerSession    Segment `json:"runner_session"`
	SchedulerMailbox Segment `json:"scheduler_mailbox"`
	ComposerRecovery Segment `json:"composer_recovery"`
	Memory           Segment `json:"memory"`
}

type Segment struct {
	Version string          `json:"version"`
	Payload json.RawMessage `json:"payload"`
}

func NewSegment(version string, payload any) (Segment, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return Segment{}, fmt.Errorf("marshal segment payload: %w", err)
	}
	return normalizeSegment(Segment{Version: version, Payload: raw}, "segment", version, true)
}

func MarshalManifest(in Manifest) ([]byte, error) {
	normalized, err := normalizeManifest(in, false, true)
	if err != nil {
		return nil, err
	}
	digest, err := ComputeManifestDigest(normalized)
	if err != nil {
		return nil, err
	}
	normalized.Digest = digest
	return json.Marshal(normalized)
}

func UnmarshalManifest(raw []byte) (Manifest, error) {
	normalized, err := decodeManifest(raw, true, true)
	if err != nil {
		return Manifest{}, err
	}
	expected, err := ComputeManifestDigest(normalized)
	if err != nil {
		return Manifest{}, err
	}
	if normalized.Digest != expected {
		return Manifest{}, fmt.Errorf("snapshot manifest digest mismatch: got=%q want=%q", normalized.Digest, expected)
	}
	return normalized, nil
}

func ComputeManifestDigest(in Manifest) (string, error) {
	normalized, err := normalizeManifest(in, false, false)
	if err != nil {
		return "", err
	}
	normalized.Digest = ""
	raw, err := json.Marshal(normalized)
	if err != nil {
		return "", fmt.Errorf("marshal snapshot manifest for digest: %w", err)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func decodeManifest(raw []byte, requireDigest bool, enforceCanonicalVersions bool) (Manifest, error) {
	if len(strings.TrimSpace(string(raw))) == 0 {
		return Manifest{}, fmt.Errorf("snapshot manifest payload is empty")
	}
	var parsed Manifest
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return Manifest{}, fmt.Errorf("decode snapshot manifest: %w", err)
	}
	return normalizeManifest(parsed, requireDigest, enforceCanonicalVersions)
}

func normalizeManifest(in Manifest, requireDigest bool, enforceCanonicalVersions bool) (Manifest, error) {
	out := in
	out.SchemaVersion = strings.TrimSpace(out.SchemaVersion)
	if out.SchemaVersion == "" {
		return Manifest{}, fmt.Errorf("snapshot manifest schema_version is required")
	}
	if enforceCanonicalVersions && out.SchemaVersion != ManifestSchemaVersionV1 {
		return Manifest{}, fmt.Errorf("snapshot manifest schema_version must be %q, got %q", ManifestSchemaVersionV1, out.SchemaVersion)
	}

	if out.ExportedAt.IsZero() {
		return Manifest{}, fmt.Errorf("snapshot manifest exported_at is required")
	}
	out.ExportedAt = out.ExportedAt.UTC()

	out.Source.Component = strings.TrimSpace(out.Source.Component)
	out.Source.RunID = strings.TrimSpace(out.Source.RunID)
	out.Source.SessionID = strings.TrimSpace(out.Source.SessionID)
	if out.Source.Component == "" {
		return Manifest{}, fmt.Errorf("snapshot manifest source.component is required")
	}

	var err error
	out.Segments.RunnerSession, err = normalizeSegment(
		out.Segments.RunnerSession,
		"segments.runner_session",
		RunnerSessionSegmentVersionV1,
		enforceCanonicalVersions,
	)
	if err != nil {
		return Manifest{}, err
	}
	out.Segments.SchedulerMailbox, err = normalizeSegment(
		out.Segments.SchedulerMailbox,
		"segments.scheduler_mailbox",
		SchedulerMailboxSegmentVersionV1,
		enforceCanonicalVersions,
	)
	if err != nil {
		return Manifest{}, err
	}
	out.Segments.ComposerRecovery, err = normalizeSegment(
		out.Segments.ComposerRecovery,
		"segments.composer_recovery",
		ComposerRecoverySegmentVersionV1,
		enforceCanonicalVersions,
	)
	if err != nil {
		return Manifest{}, err
	}
	out.Segments.Memory, err = normalizeSegment(
		out.Segments.Memory,
		"segments.memory",
		MemorySegmentVersionV1,
		enforceCanonicalVersions,
	)
	if err != nil {
		return Manifest{}, err
	}

	out.Digest = strings.TrimSpace(out.Digest)
	if requireDigest && out.Digest == "" {
		return Manifest{}, fmt.Errorf("snapshot manifest digest is required")
	}
	return out, nil
}

func normalizeSegment(in Segment, fieldPath, expectedVersion string, enforceVersion bool) (Segment, error) {
	out := in
	out.Version = strings.TrimSpace(out.Version)
	if out.Version == "" {
		return Segment{}, fmt.Errorf("snapshot manifest %s.version is required", fieldPath)
	}
	if enforceVersion && out.Version != expectedVersion {
		return Segment{}, fmt.Errorf("snapshot manifest %s.version must be %q, got %q", fieldPath, expectedVersion, out.Version)
	}
	if len(strings.TrimSpace(string(out.Payload))) == 0 {
		return Segment{}, fmt.Errorf("snapshot manifest %s.payload is required", fieldPath)
	}
	if !json.Valid(out.Payload) {
		return Segment{}, fmt.Errorf("snapshot manifest %s.payload must be valid JSON", fieldPath)
	}
	var decoded any
	if err := json.Unmarshal(out.Payload, &decoded); err != nil {
		return Segment{}, fmt.Errorf("snapshot manifest %s.payload decode failed: %w", fieldPath, err)
	}
	canonical, err := json.Marshal(decoded)
	if err != nil {
		return Segment{}, fmt.Errorf("snapshot manifest %s.payload normalize failed: %w", fieldPath, err)
	}
	out.Payload = canonical
	return out, nil
}
