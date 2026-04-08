package config

import (
	"fmt"
	"strings"
)

const (
	RuntimeStateSnapshotRestoreModeStrict     = "strict"
	RuntimeStateSnapshotRestoreModeCompatible = "compatible"
)

const (
	RuntimeStateSnapshotSchemaVersionV1 = "state_session_snapshot.v1"
)

const (
	RuntimeSessionStatePartialRestorePolicyReject = "reject"
	RuntimeSessionStatePartialRestorePolicyAllow  = "allow"
)

type RuntimeStateConfig struct {
	Snapshot RuntimeStateSnapshotConfig `json:"snapshot"`
}

type RuntimeStateSnapshotConfig struct {
	Enabled       bool                              `json:"enabled"`
	RestoreMode   string                            `json:"restore_mode"`
	CompatWindow  int                               `json:"compat_window"`
	SchemaVersion string                            `json:"schema_version"`
	Entropy       RuntimeStateSnapshotEntropyConfig `json:"entropy"`
}

type RuntimeStateSnapshotEntropyConfig struct {
	Retention RuntimeStateSnapshotEntropyRetentionConfig `json:"retention"`
	Quota     RuntimeStateSnapshotEntropyQuotaConfig     `json:"quota"`
	Cleanup   RuntimeStateSnapshotEntropyCleanupConfig   `json:"cleanup"`
}

type RuntimeStateSnapshotEntropyRetentionConfig struct {
	MaxSnapshots int `json:"max_snapshots"`
}

type RuntimeStateSnapshotEntropyQuotaConfig struct {
	MaxBytes int `json:"max_bytes"`
}

type RuntimeStateSnapshotEntropyCleanupConfig struct {
	Enabled   bool `json:"enabled"`
	BatchSize int  `json:"batch_size"`
}

type RuntimeSessionConfig struct {
	State RuntimeSessionStateConfig `json:"state"`
}

type RuntimeSessionStateConfig struct {
	Enabled              bool   `json:"enabled"`
	PartialRestorePolicy string `json:"partial_restore_policy"`
}

func normalizeRuntimeStateSnapshotConfig(in RuntimeStateSnapshotConfig) RuntimeStateSnapshotConfig {
	base := DefaultConfig().Runtime.State.Snapshot
	out := in
	out.RestoreMode = strings.ToLower(strings.TrimSpace(out.RestoreMode))
	if out.RestoreMode == "" {
		out.RestoreMode = strings.ToLower(strings.TrimSpace(base.RestoreMode))
	}
	out.SchemaVersion = strings.ToLower(strings.TrimSpace(out.SchemaVersion))
	if out.SchemaVersion == "" {
		out.SchemaVersion = strings.ToLower(strings.TrimSpace(base.SchemaVersion))
	}
	return out
}

func normalizeRuntimeSessionStateConfig(in RuntimeSessionStateConfig) RuntimeSessionStateConfig {
	base := DefaultConfig().Runtime.Session.State
	out := in
	out.PartialRestorePolicy = strings.ToLower(strings.TrimSpace(out.PartialRestorePolicy))
	if out.PartialRestorePolicy == "" {
		out.PartialRestorePolicy = strings.ToLower(strings.TrimSpace(base.PartialRestorePolicy))
	}
	return out
}

func ValidateRuntimeStateSnapshotConfig(cfg RuntimeStateSnapshotConfig) error {
	normalized := normalizeRuntimeStateSnapshotConfig(cfg)
	switch normalized.RestoreMode {
	case RuntimeStateSnapshotRestoreModeStrict, RuntimeStateSnapshotRestoreModeCompatible:
	default:
		return fmt.Errorf(
			"runtime.state.snapshot.restore_mode must be one of [%s,%s], got %q",
			RuntimeStateSnapshotRestoreModeStrict,
			RuntimeStateSnapshotRestoreModeCompatible,
			cfg.RestoreMode,
		)
	}
	if normalized.CompatWindow < 0 {
		return fmt.Errorf("runtime.state.snapshot.compat_window must be >= 0")
	}
	switch normalized.SchemaVersion {
	case RuntimeStateSnapshotSchemaVersionV1:
	default:
		return fmt.Errorf(
			"runtime.state.snapshot.schema_version must be one of [%s], got %q",
			RuntimeStateSnapshotSchemaVersionV1,
			cfg.SchemaVersion,
		)
	}
	if normalized.Entropy.Retention.MaxSnapshots < 0 {
		return fmt.Errorf("runtime.state.snapshot.entropy.retention.max_snapshots must be >= 0")
	}
	if normalized.Entropy.Quota.MaxBytes < 0 {
		return fmt.Errorf("runtime.state.snapshot.entropy.quota.max_bytes must be >= 0")
	}
	if normalized.Entropy.Cleanup.BatchSize < 0 {
		return fmt.Errorf("runtime.state.snapshot.entropy.cleanup.batch_size must be >= 0")
	}
	if normalized.Entropy.Cleanup.Enabled && normalized.Entropy.Cleanup.BatchSize <= 0 {
		return fmt.Errorf("runtime.state.snapshot.entropy.cleanup.batch_size must be > 0 when runtime.state.snapshot.entropy.cleanup.enabled=true")
	}
	return nil
}

func ValidateRuntimeSessionStateConfig(cfg RuntimeSessionStateConfig) error {
	normalized := normalizeRuntimeSessionStateConfig(cfg)
	switch normalized.PartialRestorePolicy {
	case RuntimeSessionStatePartialRestorePolicyReject, RuntimeSessionStatePartialRestorePolicyAllow:
	default:
		return fmt.Errorf(
			"runtime.session.state.partial_restore_policy must be one of [%s,%s], got %q",
			RuntimeSessionStatePartialRestorePolicyReject,
			RuntimeSessionStatePartialRestorePolicyAllow,
			cfg.PartialRestorePolicy,
		)
	}
	return nil
}
