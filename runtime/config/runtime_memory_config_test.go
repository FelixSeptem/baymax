package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
	if cfg.Runtime.Memory.Scope.Default != RuntimeMemoryScopeSession {
		t.Fatalf("runtime.memory.scope.default = %q, want %q", cfg.Runtime.Memory.Scope.Default, RuntimeMemoryScopeSession)
	}
	if cfg.Runtime.Memory.WriteMode.Mode != RuntimeMemoryWriteModeAutomatic {
		t.Fatalf("runtime.memory.write_mode = %q, want %q", cfg.Runtime.Memory.WriteMode.Mode, RuntimeMemoryWriteModeAutomatic)
	}
	if cfg.Runtime.Memory.InjectionBudget.MaxRecords <= 0 || cfg.Runtime.Memory.InjectionBudget.MaxBytes <= 0 {
		t.Fatalf("runtime.memory.injection_budget defaults invalid: %#v", cfg.Runtime.Memory.InjectionBudget)
	}
	if cfg.Runtime.Memory.Search.IndexUpdatePolicy != RuntimeMemorySearchIndexUpdatePolicyIncremental {
		t.Fatalf(
			"runtime.memory.search.index_update_policy = %q, want %q",
			cfg.Runtime.Memory.Search.IndexUpdatePolicy,
			RuntimeMemorySearchIndexUpdatePolicyIncremental,
		)
	}
	if cfg.Runtime.Memory.Search.DriftRecoveryPolicy != RuntimeMemorySearchDriftRecoveryPolicyIncrementalThenFull {
		t.Fatalf(
			"runtime.memory.search.drift_recovery_policy = %q, want %q",
			cfg.Runtime.Memory.Search.DriftRecoveryPolicy,
			RuntimeMemorySearchDriftRecoveryPolicyIncrementalThenFull,
		)
	}
	if cfg.Runtime.Memory.Lifecycle.RetentionDays <= 0 {
		t.Fatalf("runtime.memory.lifecycle.retention_days = %d, want > 0", cfg.Runtime.Memory.Lifecycle.RetentionDays)
	}
	if cfg.Runtime.Memory.WriteMode.AutomaticWindow <= 0 || cfg.Runtime.Memory.WriteMode.AgenticWindow <= 0 || cfg.Runtime.Memory.WriteMode.IdempotencyWindow <= 0 {
		t.Fatalf("runtime.memory.write_mode windows must be > 0, got %#v", cfg.Runtime.Memory.WriteMode)
	}
}

func TestRuntimeMemoryConfigEnvOverridePrecedence(t *testing.T) {
	t.Setenv("BAYMAX_RUNTIME_MEMORY_MODE", RuntimeMemoryModeExternalSPI)
	t.Setenv("BAYMAX_RUNTIME_MEMORY_EXTERNAL_PROVIDER", "mem0")
	t.Setenv("BAYMAX_RUNTIME_MEMORY_EXTERNAL_PROFILE", "mem0")
	t.Setenv("BAYMAX_RUNTIME_MEMORY_EXTERNAL_CONTRACT_VERSION", RuntimeMemoryContractVersionV1)
	t.Setenv("BAYMAX_RUNTIME_MEMORY_FALLBACK_POLICY", RuntimeMemoryFallbackPolicyDegradeWithoutMemory)
	t.Setenv("BAYMAX_RUNTIME_MEMORY_BUILTIN_COMPACTION_ENABLED", "false")
	t.Setenv("BAYMAX_RUNTIME_MEMORY_SCOPE_DEFAULT", RuntimeMemoryScopeProject)
	t.Setenv("BAYMAX_RUNTIME_MEMORY_SCOPE_ALLOWED", "project,global")
	t.Setenv("BAYMAX_RUNTIME_MEMORY_SCOPE_ALLOW_OVERRIDE", "false")
	t.Setenv("BAYMAX_RUNTIME_MEMORY_SCOPE_GLOBAL_NAMESPACE", "global-memory")
	t.Setenv("BAYMAX_RUNTIME_MEMORY_WRITE_MODE", RuntimeMemoryWriteModeAgentic)
	t.Setenv("BAYMAX_RUNTIME_MEMORY_WRITE_MODE_AGENTIC_WINDOW", "3h")
	t.Setenv("BAYMAX_RUNTIME_MEMORY_INJECTION_BUDGET_MAX_RECORDS", "3")
	t.Setenv("BAYMAX_RUNTIME_MEMORY_INJECTION_BUDGET_MAX_BYTES", "2048")
	t.Setenv("BAYMAX_RUNTIME_MEMORY_INJECTION_BUDGET_TRUNCATE_POLICY", RuntimeMemoryInjectionTruncatePolicyRecencyThenID)
	t.Setenv("BAYMAX_RUNTIME_MEMORY_LIFECYCLE_RETENTION_DAYS", "14")
	t.Setenv("BAYMAX_RUNTIME_MEMORY_LIFECYCLE_TTL_ENABLED", "true")
	t.Setenv("BAYMAX_RUNTIME_MEMORY_LIFECYCLE_TTL", "72h")
	t.Setenv("BAYMAX_RUNTIME_MEMORY_LIFECYCLE_FORGET_SCOPE_ALLOW", "session,project")
	t.Setenv("BAYMAX_RUNTIME_MEMORY_SEARCH_HYBRID_ENABLED", "true")
	t.Setenv("BAYMAX_RUNTIME_MEMORY_SEARCH_HYBRID_KEYWORD_WEIGHT", "0.7")
	t.Setenv("BAYMAX_RUNTIME_MEMORY_SEARCH_HYBRID_VECTOR_WEIGHT", "0.3")
	t.Setenv("BAYMAX_RUNTIME_MEMORY_SEARCH_RERANK_ENABLED", "true")
	t.Setenv("BAYMAX_RUNTIME_MEMORY_SEARCH_RERANK_MAX_CANDIDATES", "16")
	t.Setenv("BAYMAX_RUNTIME_MEMORY_SEARCH_TEMPORAL_DECAY_ENABLED", "true")
	t.Setenv("BAYMAX_RUNTIME_MEMORY_SEARCH_TEMPORAL_DECAY_HALF_LIFE", "24h")
	t.Setenv("BAYMAX_RUNTIME_MEMORY_SEARCH_TEMPORAL_DECAY_MAX_BOOST_RATE", "0.4")
	t.Setenv("BAYMAX_RUNTIME_MEMORY_SEARCH_INDEX_UPDATE_POLICY", RuntimeMemorySearchIndexUpdatePolicyFullRebuildOnProfileDrift)
	t.Setenv("BAYMAX_RUNTIME_MEMORY_SEARCH_DRIFT_RECOVERY_POLICY", RuntimeMemorySearchDriftRecoveryPolicyFullRebuild)

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
    scope:
      default: session
      allowed: [session, project, global]
      allow_override: true
      global_namespace: global
    write_mode: automatic
    injection_budget:
      max_records: 12
      max_bytes: 8192
      truncate_policy: score_then_recency
    lifecycle:
      retention_days: 30
      ttl_enabled: false
      ttl: 168h
      forget_scope_allow: [session, project, global]
    search:
      hybrid:
        enabled: true
        keyword_weight: 0.6
        vector_weight: 0.4
      rerank:
        enabled: false
        max_candidates: 32
      temporal_decay:
        enabled: false
        half_life: 168h
        max_boost_rate: 0.2
      index_update_policy: incremental
      drift_recovery_policy: incremental_then_full
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
	if cfg.Runtime.Memory.Scope.Default != RuntimeMemoryScopeProject {
		t.Fatalf("runtime.memory.scope.default = %q, want %q", cfg.Runtime.Memory.Scope.Default, RuntimeMemoryScopeProject)
	}
	if cfg.Runtime.Memory.Scope.AllowOverride {
		t.Fatal("runtime.memory.scope.allow_override = true, want false from env")
	}
	if cfg.Runtime.Memory.Scope.GlobalNamespace != "global-memory" {
		t.Fatalf("runtime.memory.scope.global_namespace = %q, want global-memory", cfg.Runtime.Memory.Scope.GlobalNamespace)
	}
	if cfg.Runtime.Memory.WriteMode.Mode != RuntimeMemoryWriteModeAgentic {
		t.Fatalf("runtime.memory.write_mode = %q, want %q", cfg.Runtime.Memory.WriteMode.Mode, RuntimeMemoryWriteModeAgentic)
	}
	if cfg.Runtime.Memory.WriteMode.AgenticWindow != 3*time.Hour {
		t.Fatalf("runtime.memory.write_mode.agentic_window = %v, want 3h", cfg.Runtime.Memory.WriteMode.AgenticWindow)
	}
	if cfg.Runtime.Memory.InjectionBudget.MaxRecords != 3 || cfg.Runtime.Memory.InjectionBudget.MaxBytes != 2048 {
		t.Fatalf("runtime.memory.injection_budget env override mismatch: %#v", cfg.Runtime.Memory.InjectionBudget)
	}
	if cfg.Runtime.Memory.InjectionBudget.TruncatePolicy != RuntimeMemoryInjectionTruncatePolicyRecencyThenID {
		t.Fatalf(
			"runtime.memory.injection_budget.truncate_policy = %q, want %q",
			cfg.Runtime.Memory.InjectionBudget.TruncatePolicy,
			RuntimeMemoryInjectionTruncatePolicyRecencyThenID,
		)
	}
	if cfg.Runtime.Memory.Lifecycle.RetentionDays != 14 || !cfg.Runtime.Memory.Lifecycle.TTLEnabled || cfg.Runtime.Memory.Lifecycle.TTL != 72*time.Hour {
		t.Fatalf("runtime.memory.lifecycle env override mismatch: %#v", cfg.Runtime.Memory.Lifecycle)
	}
	if cfg.Runtime.Memory.Search.IndexUpdatePolicy != RuntimeMemorySearchIndexUpdatePolicyFullRebuildOnProfileDrift {
		t.Fatalf("runtime.memory.search.index_update_policy mismatch: %q", cfg.Runtime.Memory.Search.IndexUpdatePolicy)
	}
	if cfg.Runtime.Memory.Search.DriftRecoveryPolicy != RuntimeMemorySearchDriftRecoveryPolicyFullRebuild {
		t.Fatalf("runtime.memory.search.drift_recovery_policy mismatch: %q", cfg.Runtime.Memory.Search.DriftRecoveryPolicy)
	}
	if !cfg.Runtime.Memory.Search.Rerank.Enabled || cfg.Runtime.Memory.Search.Rerank.MaxCandidates != 16 {
		t.Fatalf("runtime.memory.search.rerank env override mismatch: %#v", cfg.Runtime.Memory.Search.Rerank)
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

	cfg = DefaultConfig()
	cfg.Runtime.Memory.WriteMode.Mode = "manual"
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.memory.write_mode") {
		t.Fatalf("expected runtime.memory.write_mode validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Memory.Scope.Default = RuntimeMemoryScopeSession
	cfg.Runtime.Memory.Scope.Allowed = []string{RuntimeMemoryScopeProject}
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.memory.scope.default") {
		t.Fatalf("expected runtime.memory.scope.default validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Memory.InjectionBudget.MaxRecords = -1
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.memory.injection_budget.max_records") {
		t.Fatalf("expected runtime.memory.injection_budget.max_records validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Memory.Search.Hybrid.Enabled = true
	cfg.Runtime.Memory.Search.Hybrid.KeywordWeight = -1
	cfg.Runtime.Memory.Search.Hybrid.VectorWeight = 0
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.memory.search.hybrid") {
		t.Fatalf("expected runtime.memory.search.hybrid validation error, got %v", err)
	}
}

func TestRuntimeMemoryConfigInvalidBoolFailsFast(t *testing.T) {
	t.Setenv("BAYMAX_RUNTIME_MEMORY_BUILTIN_COMPACTION_ENABLED", "definitely-not-bool")
	if _, err := Load(LoadOptions{EnvPrefix: "BAYMAX"}); err == nil || !strings.Contains(err.Error(), "runtime.memory.builtin.compaction.enabled") {
		t.Fatalf("expected strict bool parse error for runtime.memory.builtin.compaction.enabled, got %v", err)
	}
}

func TestRuntimeMemoryConfigGovernanceBoolFailsFast(t *testing.T) {
	t.Setenv("BAYMAX_RUNTIME_MEMORY_SCOPE_ALLOW_OVERRIDE", "definitely-not-bool")
	if _, err := Load(LoadOptions{EnvPrefix: "BAYMAX"}); err == nil || !strings.Contains(err.Error(), "runtime.memory.scope.allow_override") {
		t.Fatalf("expected strict bool parse error for runtime.memory.scope.allow_override, got %v", err)
	}
}
