package config

import (
	"path/filepath"
	"testing"
	"time"
)

func TestManagerRuntimeMemoryInvalidReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	root := filepath.ToSlash(filepath.Join(t.TempDir(), "memory-store"))
	writeConfig(t, file, `
runtime:
  memory:
    mode: builtin_filesystem
    external:
      contract_version: memory.v1
    builtin:
      root_dir: `+root+`
      compaction:
        enabled: true
        min_ops: 32
        max_wal_bytes: 4096
    fallback:
      policy: fail_fast
reload:
  enabled: true
  debounce: 20ms
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().Runtime.Memory.Fallback.Policy
	if before != RuntimeMemoryFallbackPolicyFailFast {
		t.Fatalf("before runtime.memory.fallback.policy = %q, want %q", before, RuntimeMemoryFallbackPolicyFailFast)
	}

	writeConfig(t, file, `
runtime:
  memory:
    mode: builtin_filesystem
    external:
      contract_version: memory.v1
    builtin:
      root_dir: `+root+`
      compaction:
        enabled: true
        min_ops: 32
        max_wal_bytes: 4096
    fallback:
      policy: shadow_deny
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	after := mgr.EffectiveConfig().Runtime.Memory.Fallback.Policy
	if after != before {
		t.Fatalf("invalid runtime.memory reload should rollback, fallback.policy = %q, want %q", after, before)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
}

func TestManagerRuntimeMemoryWriteModeInvalidReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	root := filepath.ToSlash(filepath.Join(t.TempDir(), "memory-store"))
	writeConfig(t, file, `
runtime:
  memory:
    mode: builtin_filesystem
    external:
      contract_version: memory.v1
    builtin:
      root_dir: `+root+`
      compaction:
        enabled: true
        min_ops: 32
        max_wal_bytes: 4096
    fallback:
      policy: fail_fast
    write_mode: automatic
reload:
  enabled: true
  debounce: 20ms
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().Runtime.Memory.WriteMode.Mode
	if before != RuntimeMemoryWriteModeAutomatic {
		t.Fatalf("before runtime.memory.write_mode = %q, want %q", before, RuntimeMemoryWriteModeAutomatic)
	}

	writeConfig(t, file, `
runtime:
  memory:
    mode: builtin_filesystem
    external:
      contract_version: memory.v1
    builtin:
      root_dir: `+root+`
      compaction:
        enabled: true
        min_ops: 32
        max_wal_bytes: 4096
    fallback:
      policy: fail_fast
    write_mode: shadow
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	after := mgr.EffectiveConfig().Runtime.Memory.WriteMode.Mode
	if after != before {
		t.Fatalf("invalid runtime.memory.write_mode reload should rollback, got %q want %q", after, before)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
}
