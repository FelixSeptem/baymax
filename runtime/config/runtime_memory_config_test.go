package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimeMemoryConfigDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Runtime.Memory.Mode != RuntimeMemoryModeBuiltinFilesystem {
		t.Fatalf("runtime.memory.mode = %q, want %q", cfg.Runtime.Memory.Mode, RuntimeMemoryModeBuiltinFilesystem)
	}
	if cfg.Runtime.Memory.External.ContractVersion != RuntimeMemoryContractVersionV1 {
		t.Fatalf(
			"runtime.memory.external.contract_version = %q, want %q",
			cfg.Runtime.Memory.External.ContractVersion,
			RuntimeMemoryContractVersionV1,
		)
	}
	if cfg.Runtime.Memory.Fallback.Policy != RuntimeMemoryFallbackPolicyFailFast {
		t.Fatalf(
			"runtime.memory.fallback.policy = %q, want %q",
			cfg.Runtime.Memory.Fallback.Policy,
			RuntimeMemoryFallbackPolicyFailFast,
		)
	}
	if strings.TrimSpace(cfg.Runtime.Memory.Builtin.RootDir) == "" {
		t.Fatal("runtime.memory.builtin.root_dir should not be empty by default")
	}
}

func TestRuntimeMemoryConfigEnvOverridePrecedence(t *testing.T) {
	t.Setenv("BAYMAX_RUNTIME_MEMORY_MODE", RuntimeMemoryModeExternalSPI)
	t.Setenv("BAYMAX_RUNTIME_MEMORY_EXTERNAL_PROVIDER", "mem0")
	t.Setenv("BAYMAX_RUNTIME_MEMORY_EXTERNAL_PROFILE", "mem0")
	t.Setenv("BAYMAX_RUNTIME_MEMORY_EXTERNAL_CONTRACT_VERSION", RuntimeMemoryContractVersionV1)
	t.Setenv("BAYMAX_RUNTIME_MEMORY_FALLBACK_POLICY", RuntimeMemoryFallbackPolicyDegradeWithoutMemory)
	t.Setenv("BAYMAX_RUNTIME_MEMORY_BUILTIN_COMPACTION_ENABLED", "false")

	file := filepath.Join(t.TempDir(), "runtime.yaml")
	content := `
runtime:
  memory:
    mode: builtin_filesystem
    external:
      provider: zep
      profile: zep
      contract_version: memory.v1
    builtin:
      root_dir: /tmp/builtin-file
      compaction:
        enabled: true
        min_ops: 16
        max_wal_bytes: 2048
    fallback:
      policy: fail_fast
`
	if err := os.WriteFile(file, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := Load(LoadOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Runtime.Memory.Mode != RuntimeMemoryModeExternalSPI {
		t.Fatalf("runtime.memory.mode = %q, want %q", cfg.Runtime.Memory.Mode, RuntimeMemoryModeExternalSPI)
	}
	if cfg.Runtime.Memory.External.Provider != "mem0" || cfg.Runtime.Memory.External.Profile != "mem0" {
		t.Fatalf("runtime.memory.external override mismatch: %#v", cfg.Runtime.Memory.External)
	}
	if cfg.Runtime.Memory.Fallback.Policy != RuntimeMemoryFallbackPolicyDegradeWithoutMemory {
		t.Fatalf(
			"runtime.memory.fallback.policy = %q, want %q",
			cfg.Runtime.Memory.Fallback.Policy,
			RuntimeMemoryFallbackPolicyDegradeWithoutMemory,
		)
	}
	if cfg.Runtime.Memory.Builtin.Compaction.Enabled {
		t.Fatalf("runtime.memory.builtin.compaction.enabled = true, want false from env")
	}
}

func TestRuntimeMemoryConfigValidationRejectsInvalidValues(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Runtime.Memory.Mode = "db"
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.memory.mode") {
		t.Fatalf("expected runtime.memory.mode validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Memory.Mode = RuntimeMemoryModeExternalSPI
	cfg.Runtime.Memory.External.Provider = ""
	cfg.Runtime.Memory.External.Profile = ""
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.memory.external.provider") {
		t.Fatalf("expected runtime.memory.external.provider validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Memory.Mode = RuntimeMemoryModeExternalSPI
	cfg.Runtime.Memory.External.Provider = "mem0"
	cfg.Runtime.Memory.External.Profile = "mem0"
	cfg.Runtime.Memory.External.ContractVersion = "memory.v2"
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.memory.external.contract_version") {
		t.Fatalf("expected runtime.memory.external.contract_version validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Memory.Mode = RuntimeMemoryModeBuiltinFilesystem
	cfg.Runtime.Memory.Builtin.RootDir = ""
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.memory.builtin.root_dir") {
		t.Fatalf("expected runtime.memory.builtin.root_dir validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Memory.Mode = RuntimeMemoryModeBuiltinFilesystem
	cfg.Runtime.Memory.External.Provider = "mem0"
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.memory.external.provider") {
		t.Fatalf("expected illegal mode-provider validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Memory.Fallback.Policy = "drop_new"
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.memory.fallback.policy") {
		t.Fatalf("expected runtime.memory.fallback.policy validation error, got %v", err)
	}
}

func TestRuntimeMemoryConfigInvalidBoolFailsFast(t *testing.T) {
	t.Setenv("BAYMAX_RUNTIME_MEMORY_BUILTIN_COMPACTION_ENABLED", "definitely-not-bool")
	if _, err := Load(LoadOptions{EnvPrefix: "BAYMAX"}); err == nil || !strings.Contains(err.Error(), "runtime.memory.builtin.compaction.enabled") {
		t.Fatalf("expected strict bool parse error for runtime.memory.builtin.compaction.enabled, got %v", err)
	}
}
