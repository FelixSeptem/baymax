package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimeStateSnapshotSessionConfigDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Runtime.State.Snapshot.Enabled {
		t.Fatal("runtime.state.snapshot.enabled = true, want false")
	}
	if cfg.Runtime.State.Snapshot.RestoreMode != RuntimeStateSnapshotRestoreModeStrict {
		t.Fatalf(
			"runtime.state.snapshot.restore_mode = %q, want %q",
			cfg.Runtime.State.Snapshot.RestoreMode,
			RuntimeStateSnapshotRestoreModeStrict,
		)
	}
	if cfg.Runtime.State.Snapshot.CompatWindow != 1 {
		t.Fatalf("runtime.state.snapshot.compat_window = %d, want 1", cfg.Runtime.State.Snapshot.CompatWindow)
	}
	if cfg.Runtime.State.Snapshot.SchemaVersion != RuntimeStateSnapshotSchemaVersionV1 {
		t.Fatalf(
			"runtime.state.snapshot.schema_version = %q, want %q",
			cfg.Runtime.State.Snapshot.SchemaVersion,
			RuntimeStateSnapshotSchemaVersionV1,
		)
	}
	if cfg.Runtime.State.Snapshot.Entropy.Retention.MaxSnapshots != 0 {
		t.Fatalf("runtime.state.snapshot.entropy.retention.max_snapshots = %d, want 0", cfg.Runtime.State.Snapshot.Entropy.Retention.MaxSnapshots)
	}
	if cfg.Runtime.State.Snapshot.Entropy.Quota.MaxBytes != 0 {
		t.Fatalf("runtime.state.snapshot.entropy.quota.max_bytes = %d, want 0", cfg.Runtime.State.Snapshot.Entropy.Quota.MaxBytes)
	}
	if cfg.Runtime.State.Snapshot.Entropy.Cleanup.Enabled {
		t.Fatal("runtime.state.snapshot.entropy.cleanup.enabled = true, want false")
	}
	if cfg.Runtime.State.Snapshot.Entropy.Cleanup.BatchSize != 0 {
		t.Fatalf("runtime.state.snapshot.entropy.cleanup.batch_size = %d, want 0", cfg.Runtime.State.Snapshot.Entropy.Cleanup.BatchSize)
	}
	if cfg.Runtime.Session.State.Enabled {
		t.Fatal("runtime.session.state.enabled = true, want false")
	}
	if cfg.Runtime.Session.State.PartialRestorePolicy != RuntimeSessionStatePartialRestorePolicyReject {
		t.Fatalf(
			"runtime.session.state.partial_restore_policy = %q, want %q",
			cfg.Runtime.Session.State.PartialRestorePolicy,
			RuntimeSessionStatePartialRestorePolicyReject,
		)
	}
}

func TestRuntimeStateSnapshotSessionConfigEnvOverridePrecedence(t *testing.T) {
	t.Setenv("BAYMAX_RUNTIME_STATE_SNAPSHOT_ENABLED", "true")
	t.Setenv("BAYMAX_RUNTIME_STATE_SNAPSHOT_RESTORE_MODE", RuntimeStateSnapshotRestoreModeCompatible)
	t.Setenv("BAYMAX_RUNTIME_STATE_SNAPSHOT_COMPAT_WINDOW", "3")
	t.Setenv("BAYMAX_RUNTIME_STATE_SNAPSHOT_SCHEMA_VERSION", RuntimeStateSnapshotSchemaVersionV1)
	t.Setenv("BAYMAX_RUNTIME_STATE_SNAPSHOT_ENTROPY_RETENTION_MAX_SNAPSHOTS", "64")
	t.Setenv("BAYMAX_RUNTIME_STATE_SNAPSHOT_ENTROPY_QUOTA_MAX_BYTES", "1048576")
	t.Setenv("BAYMAX_RUNTIME_STATE_SNAPSHOT_ENTROPY_CLEANUP_ENABLED", "true")
	t.Setenv("BAYMAX_RUNTIME_STATE_SNAPSHOT_ENTROPY_CLEANUP_BATCH_SIZE", "16")
	t.Setenv("BAYMAX_RUNTIME_SESSION_STATE_ENABLED", "true")
	t.Setenv("BAYMAX_RUNTIME_SESSION_STATE_PARTIAL_RESTORE_POLICY", RuntimeSessionStatePartialRestorePolicyAllow)

	file := filepath.Join(t.TempDir(), "runtime.yaml")
	content := `
runtime:
  state:
    snapshot:
      enabled: false
      restore_mode: strict
      compat_window: 1
      schema_version: state_session_snapshot.v1
      entropy:
        retention:
          max_snapshots: 12
        quota:
          max_bytes: 4096
        cleanup:
          enabled: false
          batch_size: 4
  session:
    state:
      enabled: false
      partial_restore_policy: reject
`
	if err := os.WriteFile(file, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(LoadOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if !cfg.Runtime.State.Snapshot.Enabled {
		t.Fatal("runtime.state.snapshot.enabled = false, want true from env")
	}
	if cfg.Runtime.State.Snapshot.RestoreMode != RuntimeStateSnapshotRestoreModeCompatible {
		t.Fatalf(
			"runtime.state.snapshot.restore_mode = %q, want %q from env",
			cfg.Runtime.State.Snapshot.RestoreMode,
			RuntimeStateSnapshotRestoreModeCompatible,
		)
	}
	if cfg.Runtime.State.Snapshot.CompatWindow != 3 {
		t.Fatalf("runtime.state.snapshot.compat_window = %d, want 3 from env", cfg.Runtime.State.Snapshot.CompatWindow)
	}
	if cfg.Runtime.State.Snapshot.Entropy.Retention.MaxSnapshots != 64 {
		t.Fatalf("runtime.state.snapshot.entropy.retention.max_snapshots = %d, want 64 from env", cfg.Runtime.State.Snapshot.Entropy.Retention.MaxSnapshots)
	}
	if cfg.Runtime.State.Snapshot.Entropy.Quota.MaxBytes != 1048576 {
		t.Fatalf("runtime.state.snapshot.entropy.quota.max_bytes = %d, want 1048576 from env", cfg.Runtime.State.Snapshot.Entropy.Quota.MaxBytes)
	}
	if !cfg.Runtime.State.Snapshot.Entropy.Cleanup.Enabled {
		t.Fatal("runtime.state.snapshot.entropy.cleanup.enabled = false, want true from env")
	}
	if cfg.Runtime.State.Snapshot.Entropy.Cleanup.BatchSize != 16 {
		t.Fatalf("runtime.state.snapshot.entropy.cleanup.batch_size = %d, want 16 from env", cfg.Runtime.State.Snapshot.Entropy.Cleanup.BatchSize)
	}
	if !cfg.Runtime.Session.State.Enabled {
		t.Fatal("runtime.session.state.enabled = false, want true from env")
	}
	if cfg.Runtime.Session.State.PartialRestorePolicy != RuntimeSessionStatePartialRestorePolicyAllow {
		t.Fatalf(
			"runtime.session.state.partial_restore_policy = %q, want %q from env",
			cfg.Runtime.Session.State.PartialRestorePolicy,
			RuntimeSessionStatePartialRestorePolicyAllow,
		)
	}
}

func TestRuntimeStateSnapshotSessionConfigValidationRejectsInvalidValues(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Runtime.State.Snapshot.RestoreMode = "best_effort"
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.state.snapshot.restore_mode") {
		t.Fatalf("expected runtime.state.snapshot.restore_mode validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.State.Snapshot.CompatWindow = -1
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.state.snapshot.compat_window") {
		t.Fatalf("expected runtime.state.snapshot.compat_window validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.State.Snapshot.SchemaVersion = "state_session_snapshot.v9"
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.state.snapshot.schema_version") {
		t.Fatalf("expected runtime.state.snapshot.schema_version validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.State.Snapshot.Entropy.Retention.MaxSnapshots = -1
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.state.snapshot.entropy.retention.max_snapshots") {
		t.Fatalf("expected runtime.state.snapshot.entropy.retention.max_snapshots validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.State.Snapshot.Entropy.Quota.MaxBytes = -1
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.state.snapshot.entropy.quota.max_bytes") {
		t.Fatalf("expected runtime.state.snapshot.entropy.quota.max_bytes validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.State.Snapshot.Entropy.Cleanup.Enabled = true
	cfg.Runtime.State.Snapshot.Entropy.Cleanup.BatchSize = 0
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.state.snapshot.entropy.cleanup.batch_size") {
		t.Fatalf("expected runtime.state.snapshot.entropy.cleanup.batch_size validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Session.State.PartialRestorePolicy = "merge"
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.session.state.partial_restore_policy") {
		t.Fatalf("expected runtime.session.state.partial_restore_policy validation error, got %v", err)
	}
}

func TestRuntimeStateSnapshotSessionConfigValidationAllowsModesAndWindowBoundary(t *testing.T) {
	cases := []struct {
		name         string
		restoreMode  string
		compatWindow int
	}{
		{
			name:         "strict mode with zero window",
			restoreMode:  RuntimeStateSnapshotRestoreModeStrict,
			compatWindow: 0,
		},
		{
			name:         "compatible mode with zero window",
			restoreMode:  RuntimeStateSnapshotRestoreModeCompatible,
			compatWindow: 0,
		},
		{
			name:         "compatible mode with positive window",
			restoreMode:  RuntimeStateSnapshotRestoreModeCompatible,
			compatWindow: 2,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Runtime.State.Snapshot.RestoreMode = tt.restoreMode
			cfg.Runtime.State.Snapshot.CompatWindow = tt.compatWindow
			if err := Validate(cfg); err != nil {
				t.Fatalf(
					"Validate should accept restore_mode=%q compat_window=%d, got %v",
					tt.restoreMode,
					tt.compatWindow,
					err,
				)
			}
		})
	}
}

func TestRuntimeStateSnapshotSessionConfigInvalidBoolFailsFast(t *testing.T) {
	t.Setenv("BAYMAX_RUNTIME_STATE_SNAPSHOT_ENABLED", "not-a-bool")
	if _, err := Load(LoadOptions{EnvPrefix: "BAYMAX"}); err == nil || !strings.Contains(err.Error(), "runtime.state.snapshot.enabled") {
		t.Fatalf("expected strict bool parse error for runtime.state.snapshot.enabled, got %v", err)
	}

	t.Setenv("BAYMAX_RUNTIME_STATE_SNAPSHOT_ENABLED", "false")
	t.Setenv("BAYMAX_RUNTIME_SESSION_STATE_ENABLED", "not-a-bool")
	if _, err := Load(LoadOptions{EnvPrefix: "BAYMAX"}); err == nil || !strings.Contains(err.Error(), "runtime.session.state.enabled") {
		t.Fatalf("expected strict bool parse error for runtime.session.state.enabled, got %v", err)
	}
}

func TestRuntimeStateSnapshotConfigCompatWindowInvalidIntFailsFast(t *testing.T) {
	t.Setenv("BAYMAX_RUNTIME_STATE_SNAPSHOT_COMPAT_WINDOW", "abc")
	if _, err := Load(LoadOptions{EnvPrefix: "BAYMAX"}); err == nil || !strings.Contains(err.Error(), "runtime.state.snapshot.compat_window") {
		t.Fatalf("expected strict int parse error for runtime.state.snapshot.compat_window, got %v", err)
	}
}

func TestRuntimeStateSnapshotEntropyConfigInvalidIntFailsFast(t *testing.T) {
	t.Setenv("BAYMAX_RUNTIME_STATE_SNAPSHOT_ENTROPY_RETENTION_MAX_SNAPSHOTS", "abc")
	if _, err := Load(LoadOptions{EnvPrefix: "BAYMAX"}); err == nil || !strings.Contains(err.Error(), "runtime.state.snapshot.entropy.retention.max_snapshots") {
		t.Fatalf("expected strict int parse error for runtime.state.snapshot.entropy.retention.max_snapshots, got %v", err)
	}

	t.Setenv("BAYMAX_RUNTIME_STATE_SNAPSHOT_ENTROPY_RETENTION_MAX_SNAPSHOTS", "5")
	t.Setenv("BAYMAX_RUNTIME_STATE_SNAPSHOT_ENTROPY_CLEANUP_BATCH_SIZE", "xyz")
	if _, err := Load(LoadOptions{EnvPrefix: "BAYMAX"}); err == nil || !strings.Contains(err.Error(), "runtime.state.snapshot.entropy.cleanup.batch_size") {
		t.Fatalf("expected strict int parse error for runtime.state.snapshot.entropy.cleanup.batch_size, got %v", err)
	}
}
