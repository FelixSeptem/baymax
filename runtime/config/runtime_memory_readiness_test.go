package config

import (
	"path/filepath"
	"testing"
)

func TestManagerReadinessPreflightMemorySPIUnavailableNonStrict(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime-memory-nonstrict.yaml")
	journal := filepath.ToSlash(filepath.Join(t.TempDir(), "journal.jsonl"))
	memoryRoot := filepath.ToSlash(filepath.Join(t.TempDir(), "memory-store"))
	writeConfig(t, file, `
runtime:
  readiness:
    enabled: true
    strict: false
    remote_probe_enabled: false
  memory:
    mode: external_spi
    external:
      provider: mem0
      profile: mem0
      contract_version: memory.v1
    builtin:
      root_dir: `+memoryRoot+`
      compaction:
        enabled: true
        min_ops: 32
        max_wal_bytes: 4096
    fallback:
      policy: fail_fast
context_assembler:
  enabled: true
  journal_path: `+journal+`
  prefix_version: v1
  ca2:
    enabled: true
    stage2:
      provider: memory
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX_A54_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	result := mgr.ReadinessPreflight()
	if result.Status != ReadinessStatusDegraded {
		t.Fatalf("status = %q, want degraded", result.Status)
	}
	assertReadinessFindingCode(t, result.Findings, ReadinessCodeMemorySPIUnavailable)
}

func TestManagerReadinessPreflightMemorySPIUnavailableStrictEscalates(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime-memory-strict.yaml")
	journal := filepath.ToSlash(filepath.Join(t.TempDir(), "journal.jsonl"))
	memoryRoot := filepath.ToSlash(filepath.Join(t.TempDir(), "memory-store"))
	writeConfig(t, file, `
runtime:
  readiness:
    enabled: true
    strict: true
    remote_probe_enabled: false
  memory:
    mode: external_spi
    external:
      provider: mem0
      profile: mem0
      contract_version: memory.v1
    builtin:
      root_dir: `+memoryRoot+`
      compaction:
        enabled: true
        min_ops: 32
        max_wal_bytes: 4096
    fallback:
      policy: fail_fast
context_assembler:
  enabled: true
  journal_path: `+journal+`
  prefix_version: v1
  ca2:
    enabled: true
    stage2:
      provider: memory
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX_A54_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	result := mgr.ReadinessPreflight()
	if result.Status != ReadinessStatusBlocked {
		t.Fatalf("status = %q, want blocked", result.Status)
	}
	assertReadinessFindingCode(t, result.Findings, ReadinessCodeMemorySPIUnavailable)
	assertReadinessFindingCode(t, result.Findings, ReadinessCodeStrictEscalated)
}

func TestManagerReadinessPreflightMemoryFallbackPolicyConflict(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime-memory-fallback-conflict.yaml")
	memoryRoot := filepath.ToSlash(filepath.Join(t.TempDir(), "memory-store"))
	writeConfig(t, file, `
runtime:
  readiness:
    enabled: true
    strict: false
    remote_probe_enabled: false
  memory:
    mode: builtin_filesystem
    external:
      contract_version: memory.v1
    builtin:
      root_dir: `+memoryRoot+`
      compaction:
        enabled: true
        min_ops: 32
        max_wal_bytes: 4096
    fallback:
      policy: degrade_to_builtin
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX_A54_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	result := mgr.ReadinessPreflight()
	if result.Status != ReadinessStatusDegraded {
		t.Fatalf("status = %q, want degraded", result.Status)
	}
	assertReadinessFindingCode(t, result.Findings, ReadinessCodeMemoryFallbackPolicyConflict)
}
