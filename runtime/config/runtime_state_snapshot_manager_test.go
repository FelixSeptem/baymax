package config

import (
	"path/filepath"
	"testing"
	"time"
)

func TestManagerRuntimeStateSnapshotInvalidReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
runtime:
  state:
    snapshot:
      enabled: true
      restore_mode: strict
      compat_window: 1
      schema_version: state_session_snapshot.v1
  session:
    state:
      enabled: true
      partial_restore_policy: reject
reload:
  enabled: true
  debounce: 20ms
`)

	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX_A66_TEST", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	beforeMode := mgr.EffectiveConfig().Runtime.State.Snapshot.RestoreMode
	beforePolicy := mgr.EffectiveConfig().Runtime.Session.State.PartialRestorePolicy
	if beforeMode != RuntimeStateSnapshotRestoreModeStrict || beforePolicy != RuntimeSessionStatePartialRestorePolicyReject {
		t.Fatalf("unexpected before snapshot/session config: mode=%q policy=%q", beforeMode, beforePolicy)
	}

	writeConfig(t, file, `
runtime:
  state:
    snapshot:
      enabled: true
      restore_mode: best_effort
      compat_window: 1
      schema_version: state_session_snapshot.v1
  session:
    state:
      enabled: true
      partial_restore_policy: reject
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)

	after := mgr.EffectiveConfig()
	if after.Runtime.State.Snapshot.RestoreMode != beforeMode {
		t.Fatalf(
			"invalid runtime.state.snapshot.restore_mode should rollback, got %q want %q",
			after.Runtime.State.Snapshot.RestoreMode,
			beforeMode,
		)
	}
	if after.Runtime.Session.State.PartialRestorePolicy != beforePolicy {
		t.Fatalf(
			"snapshot/session reload should rollback atomically, got %q want %q",
			after.Runtime.Session.State.PartialRestorePolicy,
			beforePolicy,
		)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
}

func TestManagerRuntimeStateSnapshotEntropyInvalidReloadRollsBackAtomically(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
runtime:
  state:
    snapshot:
      enabled: true
      restore_mode: strict
      compat_window: 1
      schema_version: state_session_snapshot.v1
      entropy:
        retention:
          max_snapshots: 48
        quota:
          max_bytes: 2048
        cleanup:
          enabled: true
          batch_size: 8
  session:
    state:
      enabled: true
      partial_restore_policy: reject
reload:
  enabled: true
  debounce: 20ms
`)

	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX_A66_TEST", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().Runtime.State.Snapshot.Entropy
	if before.Retention.MaxSnapshots != 48 ||
		before.Quota.MaxBytes != 2048 ||
		!before.Cleanup.Enabled ||
		before.Cleanup.BatchSize != 8 {
		t.Fatalf("unexpected before snapshot entropy config: %#v", before)
	}

	writeConfig(t, file, `
runtime:
  state:
    snapshot:
      enabled: true
      restore_mode: strict
      compat_window: 1
      schema_version: state_session_snapshot.v1
      entropy:
        retention:
          max_snapshots: 64
        quota:
          max_bytes: 4096
        cleanup:
          enabled: true
          batch_size: 0
  session:
    state:
      enabled: true
      partial_restore_policy: reject
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)

	after := mgr.EffectiveConfig().Runtime.State.Snapshot.Entropy
	if after != before {
		t.Fatalf("invalid snapshot entropy reload should rollback atomically, got %#v want %#v", after, before)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
}
