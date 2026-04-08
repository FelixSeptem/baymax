package config

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const (
	RuntimeMemoryModeExternalSPI       = "external_spi"
	RuntimeMemoryModeBuiltinFilesystem = "builtin_filesystem"
)

const (
	RuntimeMemoryFallbackPolicyFailFast             = "fail_fast"
	RuntimeMemoryFallbackPolicyDegradeToBuiltin     = "degrade_to_builtin"
	RuntimeMemoryFallbackPolicyDegradeWithoutMemory = "degrade_without_memory"
)

const (
	RuntimeMemoryContractVersionV1 = "memory.v1"
	RuntimeMemoryProviderGeneric   = "generic"
	RuntimeMemoryProfileGeneric    = "generic"
)

const (
	RuntimeMemoryScopeSession = "session"
	RuntimeMemoryScopeProject = "project"
	RuntimeMemoryScopeGlobal  = "global"
)

const (
	RuntimeMemoryWriteModeAutomatic = "automatic"
	RuntimeMemoryWriteModeAgentic   = "agentic"
)

const (
	RuntimeMemoryInjectionTruncatePolicyScoreThenRecency = "score_then_recency"
	RuntimeMemoryInjectionTruncatePolicyRecencyThenID    = "recency_then_id"
)

const (
	RuntimeMemorySearchIndexUpdatePolicyIncremental               = "incremental"
	RuntimeMemorySearchIndexUpdatePolicyFullRebuildOnProfileDrift = "full_rebuild_on_profile_drift"
)

const (
	RuntimeMemorySearchDriftRecoveryPolicyIncrementalThenFull = "incremental_then_full"
	RuntimeMemorySearchDriftRecoveryPolicyFullRebuild         = "full_rebuild"
)

type RuntimeMemoryConfig struct {
	Mode            string                             `json:"mode"`
	External        RuntimeMemoryExternalConfig        `json:"external"`
	Builtin         RuntimeMemoryBuiltinConfig         `json:"builtin"`
	Fallback        RuntimeMemoryFallbackConfig        `json:"fallback"`
	Scope           RuntimeMemoryScopeConfig           `json:"scope"`
	WriteMode       RuntimeMemoryWriteModeConfig       `json:"write_mode"`
	InjectionBudget RuntimeMemoryInjectionBudgetConfig `json:"injection_budget"`
	Lifecycle       RuntimeMemoryLifecycleConfig       `json:"lifecycle"`
	Search          RuntimeMemorySearchConfig          `json:"search"`
}

type RuntimeMemoryExternalConfig struct {
	Provider        string `json:"provider"`
	Profile         string `json:"profile"`
	ContractVersion string `json:"contract_version"`
}

type RuntimeMemoryBuiltinConfig struct {
	RootDir    string                               `json:"root_dir"`
	Compaction RuntimeMemoryBuiltinCompactionConfig `json:"compaction"`
}

type RuntimeMemoryBuiltinCompactionConfig struct {
	Enabled        bool  `json:"enabled"`
	MinOps         int   `json:"min_ops"`
	MaxWALBytes    int64 `json:"max_wal_bytes"`
	FsyncBatchSize int   `json:"fsync_batch_size"`
}

type RuntimeMemoryFallbackConfig struct {
	Policy string `json:"policy"`
}

type RuntimeMemoryScopeConfig struct {
	Default         string   `json:"default"`
	Allowed         []string `json:"allowed"`
	AllowOverride   bool     `json:"allow_override"`
	GlobalNamespace string   `json:"global_namespace"`
}

type RuntimeMemoryWriteModeConfig struct {
	Mode              string        `json:"mode"`
	AutomaticWindow   time.Duration `json:"automatic_window"`
	AgenticWindow     time.Duration `json:"agentic_window"`
	IdempotencyWindow time.Duration `json:"idempotency_window"`
}

type RuntimeMemoryInjectionBudgetConfig struct {
	MaxRecords     int    `json:"max_records"`
	MaxBytes       int    `json:"max_bytes"`
	TruncatePolicy string `json:"truncate_policy"`
}

type RuntimeMemoryLifecycleConfig struct {
	RetentionDays    int           `json:"retention_days"`
	TTLEnabled       bool          `json:"ttl_enabled"`
	TTL              time.Duration `json:"ttl"`
	ForgetScopeAllow []string      `json:"forget_scope_allow"`
}

type RuntimeMemorySearchConfig struct {
	Hybrid              RuntimeMemorySearchHybridConfig        `json:"hybrid"`
	Rerank              RuntimeMemorySearchRerankConfig        `json:"rerank"`
	TemporalDecay       RuntimeMemorySearchTemporalDecayConfig `json:"temporal_decay"`
	IndexUpdatePolicy   string                                 `json:"index_update_policy"`
	DriftRecoveryPolicy string                                 `json:"drift_recovery_policy"`
}

type RuntimeMemorySearchHybridConfig struct {
	Enabled       bool    `json:"enabled"`
	KeywordWeight float64 `json:"keyword_weight"`
	VectorWeight  float64 `json:"vector_weight"`
}

type RuntimeMemorySearchRerankConfig struct {
	Enabled       bool `json:"enabled"`
	MaxCandidates int  `json:"max_candidates"`
}

type RuntimeMemorySearchTemporalDecayConfig struct {
	Enabled      bool          `json:"enabled"`
	HalfLife     time.Duration `json:"half_life"`
	MaxBoostRate float64       `json:"max_boost_rate"`
}

func normalizeRuntimeMemoryConfig(in RuntimeMemoryConfig) RuntimeMemoryConfig {
	base := DefaultConfig().Runtime.Memory
	out := in
	out.Mode = strings.ToLower(strings.TrimSpace(out.Mode))
	if out.Mode == "" {
		out.Mode = base.Mode
	}
	out.External.Provider = strings.ToLower(strings.TrimSpace(out.External.Provider))
	out.External.Profile = strings.ToLower(strings.TrimSpace(out.External.Profile))
	out.External.ContractVersion = strings.ToLower(strings.TrimSpace(out.External.ContractVersion))
	if out.External.ContractVersion == "" {
		out.External.ContractVersion = base.External.ContractVersion
	}
	out.Builtin.RootDir = strings.TrimSpace(out.Builtin.RootDir)
	if out.Builtin.Compaction.MinOps <= 0 {
		out.Builtin.Compaction.MinOps = base.Builtin.Compaction.MinOps
	}
	if out.Builtin.Compaction.MaxWALBytes <= 0 {
		out.Builtin.Compaction.MaxWALBytes = base.Builtin.Compaction.MaxWALBytes
	}
	if out.Builtin.Compaction.FsyncBatchSize <= 0 {
		out.Builtin.Compaction.FsyncBatchSize = base.Builtin.Compaction.FsyncBatchSize
	}
	out.Fallback.Policy = strings.ToLower(strings.TrimSpace(out.Fallback.Policy))
	if out.Fallback.Policy == "" {
		out.Fallback.Policy = base.Fallback.Policy
	}
	out.Scope.Default = canonicalRuntimeMemoryScope(out.Scope.Default)
	if out.Scope.Default == "" {
		out.Scope.Default = canonicalRuntimeMemoryScope(base.Scope.Default)
	}
	out.Scope.Allowed = normalizeRuntimeMemoryScopeList(out.Scope.Allowed)
	if len(out.Scope.Allowed) == 0 {
		out.Scope.Allowed = append([]string(nil), normalizeRuntimeMemoryScopeList(base.Scope.Allowed)...)
	}
	out.Scope.GlobalNamespace = strings.TrimSpace(out.Scope.GlobalNamespace)
	if out.Scope.GlobalNamespace == "" {
		out.Scope.GlobalNamespace = strings.TrimSpace(base.Scope.GlobalNamespace)
	}

	out.WriteMode.Mode = strings.ToLower(strings.TrimSpace(out.WriteMode.Mode))
	if out.WriteMode.Mode == "" {
		out.WriteMode.Mode = strings.ToLower(strings.TrimSpace(base.WriteMode.Mode))
	}
	if out.WriteMode.AutomaticWindow == 0 {
		out.WriteMode.AutomaticWindow = base.WriteMode.AutomaticWindow
	}
	if out.WriteMode.AgenticWindow == 0 {
		out.WriteMode.AgenticWindow = base.WriteMode.AgenticWindow
	}
	if out.WriteMode.IdempotencyWindow == 0 {
		out.WriteMode.IdempotencyWindow = base.WriteMode.IdempotencyWindow
	}

	out.InjectionBudget.TruncatePolicy = strings.ToLower(strings.TrimSpace(out.InjectionBudget.TruncatePolicy))
	if out.InjectionBudget.MaxRecords == 0 {
		out.InjectionBudget.MaxRecords = base.InjectionBudget.MaxRecords
	}
	if out.InjectionBudget.MaxBytes == 0 {
		out.InjectionBudget.MaxBytes = base.InjectionBudget.MaxBytes
	}
	if out.InjectionBudget.TruncatePolicy == "" {
		out.InjectionBudget.TruncatePolicy = strings.ToLower(strings.TrimSpace(base.InjectionBudget.TruncatePolicy))
	}

	if out.Lifecycle.RetentionDays == 0 {
		out.Lifecycle.RetentionDays = base.Lifecycle.RetentionDays
	}
	if out.Lifecycle.TTL == 0 {
		out.Lifecycle.TTL = base.Lifecycle.TTL
	}
	out.Lifecycle.ForgetScopeAllow = normalizeRuntimeMemoryScopeList(out.Lifecycle.ForgetScopeAllow)
	if len(out.Lifecycle.ForgetScopeAllow) == 0 {
		out.Lifecycle.ForgetScopeAllow = append([]string(nil), normalizeRuntimeMemoryScopeList(base.Lifecycle.ForgetScopeAllow)...)
	}

	if out.Search.Hybrid.KeywordWeight == 0 && out.Search.Hybrid.VectorWeight == 0 {
		out.Search.Hybrid.KeywordWeight = base.Search.Hybrid.KeywordWeight
		out.Search.Hybrid.VectorWeight = base.Search.Hybrid.VectorWeight
	}
	if out.Search.Rerank.MaxCandidates == 0 {
		out.Search.Rerank.MaxCandidates = base.Search.Rerank.MaxCandidates
	}
	if out.Search.TemporalDecay.HalfLife == 0 {
		out.Search.TemporalDecay.HalfLife = base.Search.TemporalDecay.HalfLife
	}
	if out.Search.TemporalDecay.MaxBoostRate == 0 {
		out.Search.TemporalDecay.MaxBoostRate = base.Search.TemporalDecay.MaxBoostRate
	}
	out.Search.IndexUpdatePolicy = strings.ToLower(strings.TrimSpace(out.Search.IndexUpdatePolicy))
	if out.Search.IndexUpdatePolicy == "" {
		out.Search.IndexUpdatePolicy = strings.ToLower(strings.TrimSpace(base.Search.IndexUpdatePolicy))
	}
	out.Search.DriftRecoveryPolicy = strings.ToLower(strings.TrimSpace(out.Search.DriftRecoveryPolicy))
	if out.Search.DriftRecoveryPolicy == "" {
		out.Search.DriftRecoveryPolicy = strings.ToLower(strings.TrimSpace(base.Search.DriftRecoveryPolicy))
	}
	return out
}

func ValidateRuntimeMemoryConfig(cfg RuntimeMemoryConfig) error {
	normalized := normalizeRuntimeMemoryConfig(cfg)
	switch normalized.Mode {
	case RuntimeMemoryModeExternalSPI, RuntimeMemoryModeBuiltinFilesystem:
	default:
		return fmt.Errorf(
			"runtime.memory.mode must be one of [%s,%s], got %q",
			RuntimeMemoryModeExternalSPI,
			RuntimeMemoryModeBuiltinFilesystem,
			cfg.Mode,
		)
	}
	switch normalized.Fallback.Policy {
	case RuntimeMemoryFallbackPolicyFailFast, RuntimeMemoryFallbackPolicyDegradeToBuiltin, RuntimeMemoryFallbackPolicyDegradeWithoutMemory:
	default:
		return fmt.Errorf(
			"runtime.memory.fallback.policy must be one of [%s,%s,%s], got %q",
			RuntimeMemoryFallbackPolicyFailFast,
			RuntimeMemoryFallbackPolicyDegradeToBuiltin,
			RuntimeMemoryFallbackPolicyDegradeWithoutMemory,
			cfg.Fallback.Policy,
		)
	}
	if normalized.External.ContractVersion != RuntimeMemoryContractVersionV1 {
		return fmt.Errorf(
			"runtime.memory.external.contract_version must be one of [%s], got %q",
			RuntimeMemoryContractVersionV1,
			cfg.External.ContractVersion,
		)
	}
	if normalized.Mode == RuntimeMemoryModeExternalSPI {
		if strings.TrimSpace(normalized.External.Provider) == "" {
			return fmt.Errorf("runtime.memory.external.provider is required when runtime.memory.mode=%s", RuntimeMemoryModeExternalSPI)
		}
		if strings.TrimSpace(normalized.External.Profile) == "" {
			return fmt.Errorf("runtime.memory.external.profile is required when runtime.memory.mode=%s", RuntimeMemoryModeExternalSPI)
		}
	}
	if normalized.Mode == RuntimeMemoryModeBuiltinFilesystem {
		if err := validateRuntimeMemoryRootDir(normalized.Builtin.RootDir); err != nil {
			return err
		}
		if strings.TrimSpace(cfg.External.Provider) != "" {
			return fmt.Errorf(
				"runtime.memory.external.provider must be empty when runtime.memory.mode=%s",
				RuntimeMemoryModeBuiltinFilesystem,
			)
		}
	}
	if normalized.Fallback.Policy == RuntimeMemoryFallbackPolicyDegradeToBuiltin {
		if err := validateRuntimeMemoryRootDir(normalized.Builtin.RootDir); err != nil {
			return err
		}
	}
	if normalized.Builtin.Compaction.MinOps <= 0 {
		return errorsRuntimeMemory("runtime.memory.builtin.compaction.min_ops must be > 0")
	}
	if normalized.Builtin.Compaction.MaxWALBytes <= 0 {
		return errorsRuntimeMemory("runtime.memory.builtin.compaction.max_wal_bytes must be > 0")
	}
	if normalized.Builtin.Compaction.FsyncBatchSize <= 0 {
		return errorsRuntimeMemory("runtime.memory.builtin.compaction.fsync_batch_size must be > 0")
	}
	if err := validateRuntimeMemoryScopeConfig(normalized.Scope); err != nil {
		return err
	}
	if err := validateRuntimeMemoryWriteModeConfig(normalized.WriteMode); err != nil {
		return err
	}
	if err := validateRuntimeMemoryInjectionBudgetConfig(normalized.InjectionBudget); err != nil {
		return err
	}
	if err := validateRuntimeMemoryLifecycleConfig(normalized.Lifecycle); err != nil {
		return err
	}
	return validateRuntimeMemorySearchConfig(normalized.Search)
}

func validateRuntimeMemoryRootDir(root string) error {
	path := strings.TrimSpace(root)
	if path == "" {
		return errorsRuntimeMemory("runtime.memory.builtin.root_dir is required")
	}
	if strings.ContainsRune(path, '\x00') {
		return errorsRuntimeMemory("runtime.memory.builtin.root_dir contains invalid null character")
	}
	clean := filepath.Clean(path)
	if clean == "." || strings.TrimSpace(clean) == "" {
		return errorsRuntimeMemory("runtime.memory.builtin.root_dir is malformed")
	}
	return nil
}

func errorsRuntimeMemory(msg string) error {
	return fmt.Errorf("%s", strings.TrimSpace(msg))
}

func canonicalRuntimeMemoryScope(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func normalizeRuntimeMemoryScopeList(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, item := range in {
		scope := canonicalRuntimeMemoryScope(item)
		if scope == "" {
			continue
		}
		if _, ok := seen[scope]; ok {
			continue
		}
		seen[scope] = struct{}{}
		out = append(out, scope)
	}
	return out
}

func isRuntimeMemoryScope(scope string) bool {
	switch canonicalRuntimeMemoryScope(scope) {
	case RuntimeMemoryScopeSession, RuntimeMemoryScopeProject, RuntimeMemoryScopeGlobal:
		return true
	default:
		return false
	}
}

func validateRuntimeMemoryScopeConfig(cfg RuntimeMemoryScopeConfig) error {
	if !isRuntimeMemoryScope(cfg.Default) {
		return errorsRuntimeMemory(fmt.Sprintf("runtime.memory.scope.default must be one of [%s,%s,%s], got %q", RuntimeMemoryScopeSession, RuntimeMemoryScopeProject, RuntimeMemoryScopeGlobal, cfg.Default))
	}
	if len(cfg.Allowed) == 0 {
		return errorsRuntimeMemory("runtime.memory.scope.allowed must not be empty")
	}
	for i := range cfg.Allowed {
		if !isRuntimeMemoryScope(cfg.Allowed[i]) {
			return errorsRuntimeMemory(fmt.Sprintf("runtime.memory.scope.allowed[%d] must be one of [%s,%s,%s], got %q", i, RuntimeMemoryScopeSession, RuntimeMemoryScopeProject, RuntimeMemoryScopeGlobal, cfg.Allowed[i]))
		}
	}
	if !slices.Contains(cfg.Allowed, cfg.Default) {
		return errorsRuntimeMemory("runtime.memory.scope.default must be included in runtime.memory.scope.allowed")
	}
	if strings.TrimSpace(cfg.GlobalNamespace) == "" {
		return errorsRuntimeMemory("runtime.memory.scope.global_namespace is required")
	}
	return nil
}

func validateRuntimeMemoryWriteModeConfig(cfg RuntimeMemoryWriteModeConfig) error {
	switch cfg.Mode {
	case RuntimeMemoryWriteModeAutomatic, RuntimeMemoryWriteModeAgentic:
	default:
		return errorsRuntimeMemory(fmt.Sprintf("runtime.memory.write_mode must be one of [%s,%s], got %q", RuntimeMemoryWriteModeAutomatic, RuntimeMemoryWriteModeAgentic, cfg.Mode))
	}
	if cfg.AutomaticWindow <= 0 {
		return errorsRuntimeMemory("runtime.memory.write_mode.automatic_window must be > 0")
	}
	if cfg.AgenticWindow <= 0 {
		return errorsRuntimeMemory("runtime.memory.write_mode.agentic_window must be > 0")
	}
	if cfg.IdempotencyWindow <= 0 {
		return errorsRuntimeMemory("runtime.memory.write_mode.idempotency_window must be > 0")
	}
	return nil
}

func validateRuntimeMemoryInjectionBudgetConfig(cfg RuntimeMemoryInjectionBudgetConfig) error {
	if cfg.MaxRecords <= 0 {
		return errorsRuntimeMemory("runtime.memory.injection_budget.max_records must be > 0")
	}
	if cfg.MaxBytes <= 0 {
		return errorsRuntimeMemory("runtime.memory.injection_budget.max_bytes must be > 0")
	}
	switch cfg.TruncatePolicy {
	case RuntimeMemoryInjectionTruncatePolicyScoreThenRecency, RuntimeMemoryInjectionTruncatePolicyRecencyThenID:
	default:
		return errorsRuntimeMemory(fmt.Sprintf("runtime.memory.injection_budget.truncate_policy must be one of [%s,%s], got %q", RuntimeMemoryInjectionTruncatePolicyScoreThenRecency, RuntimeMemoryInjectionTruncatePolicyRecencyThenID, cfg.TruncatePolicy))
	}
	return nil
}

func validateRuntimeMemoryLifecycleConfig(cfg RuntimeMemoryLifecycleConfig) error {
	if cfg.RetentionDays <= 0 {
		return errorsRuntimeMemory("runtime.memory.lifecycle.retention_days must be > 0")
	}
	if cfg.TTLEnabled && cfg.TTL <= 0 {
		return errorsRuntimeMemory("runtime.memory.lifecycle.ttl must be > 0 when runtime.memory.lifecycle.ttl_enabled=true")
	}
	if len(cfg.ForgetScopeAllow) == 0 {
		return errorsRuntimeMemory("runtime.memory.lifecycle.forget_scope_allow must not be empty")
	}
	for i := range cfg.ForgetScopeAllow {
		if !isRuntimeMemoryScope(cfg.ForgetScopeAllow[i]) {
			return errorsRuntimeMemory(fmt.Sprintf("runtime.memory.lifecycle.forget_scope_allow[%d] must be one of [%s,%s,%s], got %q", i, RuntimeMemoryScopeSession, RuntimeMemoryScopeProject, RuntimeMemoryScopeGlobal, cfg.ForgetScopeAllow[i]))
		}
	}
	return nil
}

func validateRuntimeMemorySearchConfig(cfg RuntimeMemorySearchConfig) error {
	if cfg.Hybrid.KeywordWeight < 0 || cfg.Hybrid.VectorWeight < 0 {
		return errorsRuntimeMemory("runtime.memory.search.hybrid.keyword_weight and vector_weight must be >= 0")
	}
	if cfg.Hybrid.Enabled && (cfg.Hybrid.KeywordWeight+cfg.Hybrid.VectorWeight) <= 0 {
		return errorsRuntimeMemory("runtime.memory.search.hybrid keyword/vector weight sum must be > 0 when hybrid.enabled=true")
	}
	if cfg.Rerank.Enabled && cfg.Rerank.MaxCandidates <= 0 {
		return errorsRuntimeMemory("runtime.memory.search.rerank.max_candidates must be > 0 when rerank.enabled=true")
	}
	if cfg.TemporalDecay.Enabled {
		if cfg.TemporalDecay.HalfLife <= 0 {
			return errorsRuntimeMemory("runtime.memory.search.temporal_decay.half_life must be > 0 when temporal_decay.enabled=true")
		}
		if cfg.TemporalDecay.MaxBoostRate < 0 {
			return errorsRuntimeMemory("runtime.memory.search.temporal_decay.max_boost_rate must be >= 0")
		}
	}
	switch cfg.IndexUpdatePolicy {
	case RuntimeMemorySearchIndexUpdatePolicyIncremental, RuntimeMemorySearchIndexUpdatePolicyFullRebuildOnProfileDrift:
	default:
		return errorsRuntimeMemory(fmt.Sprintf("runtime.memory.search.index_update_policy must be one of [%s,%s], got %q", RuntimeMemorySearchIndexUpdatePolicyIncremental, RuntimeMemorySearchIndexUpdatePolicyFullRebuildOnProfileDrift, cfg.IndexUpdatePolicy))
	}
	switch cfg.DriftRecoveryPolicy {
	case RuntimeMemorySearchDriftRecoveryPolicyIncrementalThenFull, RuntimeMemorySearchDriftRecoveryPolicyFullRebuild:
	default:
		return errorsRuntimeMemory(fmt.Sprintf("runtime.memory.search.drift_recovery_policy must be one of [%s,%s], got %q", RuntimeMemorySearchDriftRecoveryPolicyIncrementalThenFull, RuntimeMemorySearchDriftRecoveryPolicyFullRebuild, cfg.DriftRecoveryPolicy))
	}
	return nil
}
