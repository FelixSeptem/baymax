package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

func TestLoadPrecedenceEnvOverFileOverDefault(t *testing.T) {
	t.Setenv("BAYMAX_MCP_PROFILES_DEFAULT_RETRY", "7")
	t.Setenv("BAYMAX_MCP_ACTIVE_PROFILE", "default")
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	content := `
mcp:
  active_profile: default
  profiles:
    default:
      retry: 3
`
	if err := os.WriteFile(file, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(LoadOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.MCP.Profiles["default"].Retry != 7 {
		t.Fatalf("retry = %d, want 7", cfg.MCP.Profiles["default"].Retry)
	}
	if cfg.MCP.Profiles["default"].CallTimeout <= 0 {
		t.Fatalf("call_timeout should use default fallback, got %v", cfg.MCP.Profiles["default"].CallTimeout)
	}
}

func TestConcurrencyCancelPropagationDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Concurrency.CancelPropagationTimeout <= 0 {
		t.Fatalf("concurrency.cancel_propagation_timeout = %v, want > 0", cfg.Concurrency.CancelPropagationTimeout)
	}
}

func TestConcurrencyCancelPropagationEnvOverridePrecedence(t *testing.T) {
	t.Setenv("BAYMAX_CONCURRENCY_CANCEL_PROPAGATION_TIMEOUT", "750ms")
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	content := `
concurrency:
  cancel_propagation_timeout: 2s
`
	if err := os.WriteFile(file, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := Load(LoadOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Concurrency.CancelPropagationTimeout != 750*time.Millisecond {
		t.Fatalf("concurrency.cancel_propagation_timeout = %v, want 750ms", cfg.Concurrency.CancelPropagationTimeout)
	}
}

func TestConcurrencyCancelPropagationValidationRejectsInvalidValue(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Concurrency.CancelPropagationTimeout = 0
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for concurrency.cancel_propagation_timeout")
	}
}

func TestConcurrencyBackpressureAllowsDropLowPriority(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Concurrency.Backpressure = types.BackpressureDropLowPriority
	if err := Validate(cfg); err != nil {
		t.Fatalf("Validate failed for drop_low_priority: %v", err)
	}
}

func TestMCPProfileBackpressureRejectsDropLowPriority(t *testing.T) {
	cfg := DefaultConfig()
	p := cfg.MCP.Profiles[ProfileDefault]
	p.Backpressure = types.BackpressureDropLowPriority
	cfg.MCP.Profiles[ProfileDefault] = p
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for mcp profile backpressure drop_low_priority")
	}
}

func TestConcurrencyDropLowPriorityDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if len(cfg.Concurrency.DropLowPriority.DroppablePriorities) != 1 || cfg.Concurrency.DropLowPriority.DroppablePriorities[0] != DropPriorityLow {
		t.Fatalf("droppable_priorities = %#v, want [low]", cfg.Concurrency.DropLowPriority.DroppablePriorities)
	}
}

func TestConcurrencyDropLowPriorityEnvOverridePrecedence(t *testing.T) {
	t.Setenv("BAYMAX_CONCURRENCY_DROP_LOW_PRIORITY_DROPPABLE_PRIORITIES", "low,normal")
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	content := `
concurrency:
  backpressure: drop_low_priority
  drop_low_priority:
    priority_by_tool:
      local.search: low
    priority_by_keyword:
      cache: low
    droppable_priorities: [low]
`
	if err := os.WriteFile(file, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := Load(LoadOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Concurrency.Backpressure != types.BackpressureDropLowPriority {
		t.Fatalf("backpressure = %q, want drop_low_priority", cfg.Concurrency.Backpressure)
	}
	if len(cfg.Concurrency.DropLowPriority.DroppablePriorities) != 2 {
		t.Fatalf("droppable_priorities = %#v", cfg.Concurrency.DropLowPriority.DroppablePriorities)
	}
}

func TestConcurrencyDropLowPriorityValidateRejectsInvalidPriority(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Concurrency.DropLowPriority.DroppablePriorities = []string{"critical"}
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for invalid droppable priority")
	}
	cfg = DefaultConfig()
	cfg.Concurrency.DropLowPriority.PriorityByTool = map[string]string{"local.search": "critical"}
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for invalid priority_by_tool value")
	}
}

func TestDiagnosticsTimelineTrendDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.Diagnostics.TimelineTrend.Enabled {
		t.Fatal("diagnostics.timeline_trend.enabled = false, want true")
	}
	if cfg.Diagnostics.TimelineTrend.LastNRuns != 100 {
		t.Fatalf("diagnostics.timeline_trend.last_n_runs = %d, want 100", cfg.Diagnostics.TimelineTrend.LastNRuns)
	}
	if cfg.Diagnostics.TimelineTrend.TimeWindow != 15*time.Minute {
		t.Fatalf("diagnostics.timeline_trend.time_window = %v, want 15m", cfg.Diagnostics.TimelineTrend.TimeWindow)
	}
}

func TestDiagnosticsTimelineTrendEnvOverridePrecedence(t *testing.T) {
	t.Setenv("BAYMAX_DIAGNOSTICS_TIMELINE_TREND_LAST_N_RUNS", "42")
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	content := `
diagnostics:
  timeline_trend:
    enabled: true
    last_n_runs: 7
    time_window: 20m
`
	if err := os.WriteFile(file, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := Load(LoadOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Diagnostics.TimelineTrend.LastNRuns != 42 {
		t.Fatalf("diagnostics.timeline_trend.last_n_runs = %d, want 42", cfg.Diagnostics.TimelineTrend.LastNRuns)
	}
	if cfg.Diagnostics.TimelineTrend.TimeWindow != 20*time.Minute {
		t.Fatalf("diagnostics.timeline_trend.time_window = %v, want 20m", cfg.Diagnostics.TimelineTrend.TimeWindow)
	}
}

func TestDiagnosticsTimelineTrendValidationRejectsInvalidValue(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Diagnostics.TimelineTrend.LastNRuns = 0
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for diagnostics.timeline_trend.last_n_runs")
	}
	cfg = DefaultConfig()
	cfg.Diagnostics.TimelineTrend.TimeWindow = 0
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for diagnostics.timeline_trend.time_window")
	}
}

func TestDiagnosticsCA2ExternalTrendDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.Diagnostics.CA2ExternalTrend.Enabled {
		t.Fatal("diagnostics.ca2_external_trend.enabled = false, want true")
	}
	if cfg.Diagnostics.CA2ExternalTrend.Window != 15*time.Minute {
		t.Fatalf("diagnostics.ca2_external_trend.window = %v, want 15m", cfg.Diagnostics.CA2ExternalTrend.Window)
	}
	if cfg.Diagnostics.CA2ExternalTrend.Thresholds.P95LatencyMs <= 0 {
		t.Fatalf("diagnostics.ca2_external_trend.thresholds.p95_latency_ms = %d, want > 0", cfg.Diagnostics.CA2ExternalTrend.Thresholds.P95LatencyMs)
	}
}

func TestDiagnosticsCA2ExternalTrendEnvOverridePrecedence(t *testing.T) {
	t.Setenv("BAYMAX_DIAGNOSTICS_CA2_EXTERNAL_TREND_WINDOW", "25m")
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	content := `
diagnostics:
  ca2_external_trend:
    enabled: true
    window: 10m
    thresholds:
      p95_latency_ms: 900
      error_rate: 0.12
      hit_rate: 0.35
`
	if err := os.WriteFile(file, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := Load(LoadOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Diagnostics.CA2ExternalTrend.Window != 25*time.Minute {
		t.Fatalf("diagnostics.ca2_external_trend.window = %v, want 25m", cfg.Diagnostics.CA2ExternalTrend.Window)
	}
	if cfg.Diagnostics.CA2ExternalTrend.Thresholds.P95LatencyMs != 900 {
		t.Fatalf("diagnostics.ca2_external_trend.thresholds.p95_latency_ms = %d, want 900", cfg.Diagnostics.CA2ExternalTrend.Thresholds.P95LatencyMs)
	}
}

func TestDiagnosticsCA2ExternalTrendValidationRejectsInvalidValue(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Diagnostics.CA2ExternalTrend.Window = 0
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for diagnostics.ca2_external_trend.window")
	}
	cfg = DefaultConfig()
	cfg.Diagnostics.CA2ExternalTrend.Thresholds.ErrorRate = 1.2
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for diagnostics.ca2_external_trend.thresholds.error_rate")
	}
	cfg = DefaultConfig()
	cfg.Diagnostics.CA2ExternalTrend.Thresholds.HitRate = -0.1
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for diagnostics.ca2_external_trend.thresholds.hit_rate")
	}
}

func TestValidateFailFast(t *testing.T) {
	cfg := DefaultConfig()
	p := cfg.MCP.Profiles[ProfileDefault]
	p.Backpressure = "invalid"
	cfg.MCP.Profiles[ProfileDefault] = p
	if err := Validate(cfg); err == nil {
		t.Fatalf("expected validation error, got nil")
	}
}

func TestResolveMCPPolicyWithConfig(t *testing.T) {
	cfg := DefaultConfig()
	override := &types.MCPRuntimePolicy{Retry: 9, Backoff: 30 * time.Millisecond}
	p, err := ResolveMCPPolicyWithConfig(cfg, ProfileDefault, override)
	if err != nil {
		t.Fatalf("ResolveMCPPolicyWithConfig failed: %v", err)
	}
	if p.Retry != 9 {
		t.Fatalf("retry = %d, want 9", p.Retry)
	}
}

func TestProviderFallbackLoadAndValidation(t *testing.T) {
	t.Setenv("BAYMAX_PROVIDER_FALLBACK_ENABLED", "true")
	t.Setenv("BAYMAX_PROVIDER_FALLBACK_PROVIDERS", "openai,anthropic")
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	content := `
provider_fallback:
  enabled: false
  providers: [gemini]
  discovery_timeout: 2s
  discovery_cache_ttl: 3m
`
	if err := os.WriteFile(file, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(LoadOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if !cfg.ProviderFallback.Enabled {
		t.Fatalf("provider_fallback.enabled = false, want true from env")
	}
	if len(cfg.ProviderFallback.Providers) != 2 || cfg.ProviderFallback.Providers[0] != "openai" || cfg.ProviderFallback.Providers[1] != "anthropic" {
		t.Fatalf("provider_fallback.providers = %#v", cfg.ProviderFallback.Providers)
	}
}

func TestProviderFallbackValidateRejectsEnabledWithoutProviders(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ProviderFallback.Enabled = true
	cfg.ProviderFallback.Providers = nil
	if err := Validate(cfg); err == nil {
		t.Fatalf("expected validation error, got nil")
	}
}

func TestSkillTriggerScoringDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Skill.TriggerScoring.Strategy != SkillTriggerScoringStrategyLexicalWeightedKeywords {
		t.Fatalf("skill.trigger_scoring.strategy = %q", cfg.Skill.TriggerScoring.Strategy)
	}
	if cfg.Skill.TriggerScoring.TieBreak != SkillTriggerScoringTieBreakHighestPriority {
		t.Fatalf("skill.trigger_scoring.tie_break = %q", cfg.Skill.TriggerScoring.TieBreak)
	}
	if !cfg.Skill.TriggerScoring.SuppressLowConfidence {
		t.Fatal("skill.trigger_scoring.suppress_low_confidence = false, want true")
	}
	if cfg.Skill.TriggerScoring.ConfidenceThreshold <= 0 || cfg.Skill.TriggerScoring.ConfidenceThreshold > 1 {
		t.Fatalf("skill.trigger_scoring.confidence_threshold = %v, want in (0,1]", cfg.Skill.TriggerScoring.ConfidenceThreshold)
	}
	if len(cfg.Skill.TriggerScoring.KeywordWeights) == 0 {
		t.Fatal("skill.trigger_scoring.keyword_weights must not be empty")
	}
}

func TestSkillTriggerScoringLoadEnvOverFile(t *testing.T) {
	t.Setenv("BAYMAX_SKILL_TRIGGER_SCORING_TIE_BREAK", SkillTriggerScoringTieBreakFirstRegistered)
	t.Setenv("BAYMAX_SKILL_TRIGGER_SCORING_SUPPRESS_LOW_CONFIDENCE", "false")
	t.Setenv("BAYMAX_SKILL_TRIGGER_SCORING_CONFIDENCE_THRESHOLD", "0.61")

	file := filepath.Join(t.TempDir(), "runtime.yaml")
	content := `
skill:
  trigger_scoring:
    strategy: lexical_weighted_keywords
    tie_break: highest_priority
    suppress_low_confidence: true
    confidence_threshold: 0.33
    keyword_weights:
      db: 1.9
      api: 1.4
`
	if err := os.WriteFile(file, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := Load(LoadOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Skill.TriggerScoring.TieBreak != SkillTriggerScoringTieBreakFirstRegistered {
		t.Fatalf("skill.trigger_scoring.tie_break = %q, want env override", cfg.Skill.TriggerScoring.TieBreak)
	}
	if cfg.Skill.TriggerScoring.SuppressLowConfidence {
		t.Fatal("skill.trigger_scoring.suppress_low_confidence = true, want false from env")
	}
	if cfg.Skill.TriggerScoring.ConfidenceThreshold != 0.61 {
		t.Fatalf("skill.trigger_scoring.confidence_threshold = %v, want 0.61", cfg.Skill.TriggerScoring.ConfidenceThreshold)
	}
	if cfg.Skill.TriggerScoring.KeywordWeights["db"] != 1.9 {
		t.Fatalf("skill.trigger_scoring.keyword_weights.db = %v, want 1.9", cfg.Skill.TriggerScoring.KeywordWeights["db"])
	}
}

func TestSkillTriggerScoringValidateRejectsInvalidTieBreak(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Skill.TriggerScoring.TieBreak = "priority_only"
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for skill.trigger_scoring.tie_break")
	}
}

func TestSkillTriggerScoringValidateRejectsInvalidThreshold(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Skill.TriggerScoring.ConfidenceThreshold = 1.01
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for skill.trigger_scoring.confidence_threshold")
	}
}

func TestSkillTriggerScoringValidateRejectsEmptyOrNonPositiveWeights(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Skill.TriggerScoring.KeywordWeights = map[string]float64{}
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for empty skill.trigger_scoring.keyword_weights")
	}
	cfg = DefaultConfig()
	cfg.Skill.TriggerScoring.KeywordWeights = map[string]float64{"db": 0}
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for non-positive skill.trigger_scoring.keyword_weights value")
	}
}

func TestActionGateDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.ActionGate.Enabled {
		t.Fatal("action_gate.enabled = false, want true")
	}
	if cfg.ActionGate.Policy != ActionGatePolicyRequireConfirm {
		t.Fatalf("action_gate.policy = %q, want %q", cfg.ActionGate.Policy, ActionGatePolicyRequireConfirm)
	}
	if cfg.ActionGate.Timeout <= 0 {
		t.Fatalf("action_gate.timeout = %v, want > 0", cfg.ActionGate.Timeout)
	}
}

func TestActionGateValidationRejectsInvalidPolicy(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ActionGate.Policy = "warn"
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for action_gate.policy")
	}
}

func TestActionGateEnvOverridePrecedence(t *testing.T) {
	t.Setenv("BAYMAX_ACTION_GATE_POLICY", "deny")
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	content := `
action_gate:
  policy: require_confirm
  timeout: 7s
`
	if err := os.WriteFile(file, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(LoadOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.ActionGate.Policy != ActionGatePolicyDeny {
		t.Fatalf("action_gate.policy = %q, want env override deny", cfg.ActionGate.Policy)
	}
	if cfg.ActionGate.Timeout != 7*time.Second {
		t.Fatalf("action_gate.timeout = %v, want 7s from file", cfg.ActionGate.Timeout)
	}
}

func TestActionGateParameterRulesLoadAndPrecedence(t *testing.T) {
	t.Setenv("BAYMAX_ACTION_GATE_POLICY", "deny")
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	content := `
action_gate:
  enabled: true
  policy: require_confirm
  parameter_rules:
    - id: allow-echo-q
      tool_names: [echo]
      action: allow
      condition:
        path: q
        operator: contains
        expected: tool-loop
`
	if err := os.WriteFile(file, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := Load(LoadOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.ActionGate.Policy != ActionGatePolicyDeny {
		t.Fatalf("action_gate.policy = %q, want env override deny", cfg.ActionGate.Policy)
	}
	if len(cfg.ActionGate.ParameterRules) != 1 {
		t.Fatalf("parameter_rules len = %d, want 1", len(cfg.ActionGate.ParameterRules))
	}
	rule := cfg.ActionGate.ParameterRules[0]
	if rule.ID != "allow-echo-q" {
		t.Fatalf("rule.id = %q", rule.ID)
	}
	if rule.Condition.Path != "q" || rule.Condition.Operator != types.ActionGateRuleOperatorContains {
		t.Fatalf("unexpected rule condition: %#v", rule.Condition)
	}
}

func TestActionGateParameterRulesValidationRejectsInvalidOperator(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ActionGate.ParameterRules = []types.ActionGateParameterRule{
		{
			ID: "bad-op",
			Condition: types.ActionGateRuleCondition{
				Path:     "q",
				Operator: "boom",
			},
		},
	}
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for invalid action_gate.parameter_rules operator")
	}
}

func TestActionGateParameterRulesValidationRejectsEmptyConditionTree(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ActionGate.ParameterRules = []types.ActionGateParameterRule{
		{
			ID: "empty",
			Condition: types.ActionGateRuleCondition{
				All: []types.ActionGateRuleCondition{{}},
			},
		},
	}
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for malformed condition tree")
	}
}

func TestClarificationDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.Clarification.Enabled {
		t.Fatal("clarification.enabled = false, want true")
	}
	if cfg.Clarification.Timeout <= 0 {
		t.Fatalf("clarification.timeout = %v, want > 0", cfg.Clarification.Timeout)
	}
	if cfg.Clarification.TimeoutPolicy != ClarificationTimeoutPolicyCancelByUser {
		t.Fatalf("clarification.timeout_policy = %q, want %q", cfg.Clarification.TimeoutPolicy, ClarificationTimeoutPolicyCancelByUser)
	}
}

func TestClarificationEnvOverridePrecedence(t *testing.T) {
	t.Setenv("BAYMAX_CLARIFICATION_TIMEOUT_POLICY", "cancel_by_user")
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	content := `
clarification:
  enabled: true
  timeout: 9s
  timeout_policy: cancel_by_user
`
	if err := os.WriteFile(file, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := Load(LoadOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Clarification.Timeout != 9*time.Second {
		t.Fatalf("clarification.timeout = %v, want 9s from file", cfg.Clarification.Timeout)
	}
	if cfg.Clarification.TimeoutPolicy != ClarificationTimeoutPolicyCancelByUser {
		t.Fatalf("clarification.timeout_policy = %q", cfg.Clarification.TimeoutPolicy)
	}
}

func TestClarificationValidationRejectsInvalidPolicy(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Clarification.TimeoutPolicy = "deny"
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for clarification.timeout_policy")
	}
}

func TestContextAssemblerDefaultsEnabledAndFailFast(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.ContextAssembler.Enabled {
		t.Fatal("context_assembler.enabled = false, want true")
	}
	if !cfg.ContextAssembler.Guard.FailFast {
		t.Fatal("context_assembler.guard.fail_fast = false, want true")
	}
	if cfg.ContextAssembler.Storage.Backend != "file" {
		t.Fatalf("context_assembler.storage.backend = %q, want file", cfg.ContextAssembler.Storage.Backend)
	}
}

func TestContextAssemblerValidateRejectsInvalidBackend(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ContextAssembler.Storage.Backend = "invalid"
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for invalid backend")
	}
}

func TestContextAssemblerCA2Defaults(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.ContextAssembler.CA2.Enabled {
		t.Fatal("context_assembler.ca2.enabled = true, want false by default")
	}
	if cfg.ContextAssembler.CA2.RoutingMode != "rules" {
		t.Fatalf("routing_mode = %q, want rules", cfg.ContextAssembler.CA2.RoutingMode)
	}
	if cfg.ContextAssembler.CA2.Stage2.Provider != "file" {
		t.Fatalf("stage2.provider = %q, want file", cfg.ContextAssembler.CA2.Stage2.Provider)
	}
}

func TestContextAssemblerCA2ValidationRejectsInvalidMode(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ContextAssembler.CA2.Enabled = true
	cfg.ContextAssembler.CA2.RoutingMode = "invalid"
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for invalid ca2 routing mode")
	}
}

func TestContextAssemblerCA2EnvOverride(t *testing.T) {
	t.Setenv("BAYMAX_CONTEXT_ASSEMBLER_CA2_ENABLED", "true")
	t.Setenv("BAYMAX_CONTEXT_ASSEMBLER_CA2_ROUTING_MODE", "rules")
	t.Setenv("BAYMAX_CONTEXT_ASSEMBLER_CA2_STAGE_POLICY_STAGE2", "fail_fast")
	t.Setenv("BAYMAX_CONTEXT_ASSEMBLER_CA2_STAGE2_PROVIDER", "file")
	t.Setenv("BAYMAX_CONTEXT_ASSEMBLER_CA2_STAGE2_FILE_PATH", filepath.Join(t.TempDir(), "stage2.jsonl"))
	cfg, err := Load(LoadOptions{EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if !cfg.ContextAssembler.CA2.Enabled {
		t.Fatal("ca2.enabled not loaded from env")
	}
	if cfg.ContextAssembler.CA2.StagePolicy.Stage2 != "fail_fast" {
		t.Fatalf("stage2 policy = %q, want fail_fast", cfg.ContextAssembler.CA2.StagePolicy.Stage2)
	}
}

func TestSecurityDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Security.Scan.Mode != SecurityScanModeStrict {
		t.Fatalf("security.scan.mode = %q, want %q", cfg.Security.Scan.Mode, SecurityScanModeStrict)
	}
	if !cfg.Security.Scan.GovulncheckEnable {
		t.Fatalf("security.scan.govulncheck_enabled = false, want true")
	}
	if !cfg.Security.Redaction.Enabled {
		t.Fatalf("security.redaction.enabled = false, want true")
	}
	if cfg.Security.Redaction.Strategy != SecurityRedactionKeyword {
		t.Fatalf("security.redaction.strategy = %q, want %q", cfg.Security.Redaction.Strategy, SecurityRedactionKeyword)
	}
	if len(cfg.Security.Redaction.Keywords) == 0 {
		t.Fatal("security.redaction.keywords should not be empty")
	}
}

func TestSecurityConfigEnvOverride(t *testing.T) {
	t.Setenv("BAYMAX_SECURITY_SCAN_MODE", "warn")
	t.Setenv("BAYMAX_SECURITY_REDACTION_KEYWORDS", "token,credential")
	cfg, err := Load(LoadOptions{EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Security.Scan.Mode != SecurityScanModeWarn {
		t.Fatalf("security.scan.mode = %q, want warn", cfg.Security.Scan.Mode)
	}
	if len(cfg.Security.Redaction.Keywords) != 2 || cfg.Security.Redaction.Keywords[1] != "credential" {
		t.Fatalf("security.redaction.keywords = %#v", cfg.Security.Redaction.Keywords)
	}
}

func TestValidateRejectsInvalidSecurityScanMode(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Security.Scan.Mode = "deny"
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for security.scan.mode")
	}
}

func TestValidateRejectsEmptyRedactionKeywordsWhenEnabled(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Security.Redaction.Enabled = true
	cfg.Security.Redaction.Keywords = []string{"   "}
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for security.redaction.keywords")
	}
}

func TestContextAssemblerCA2ProviderEnumAcceptsExternalProviders(t *testing.T) {
	for _, provider := range []string{"http", "rag", "db", "elasticsearch"} {
		cfg := DefaultConfig()
		cfg.ContextAssembler.CA2.Enabled = true
		cfg.ContextAssembler.CA2.Stage2.Provider = provider
		cfg.ContextAssembler.CA2.Stage2.External.Endpoint = "http://127.0.0.1:8080/retrieve"
		cfg.ContextAssembler.CA2.Stage2.External.Mapping.Request.QueryField = "query"
		cfg.ContextAssembler.CA2.Stage2.External.Mapping.Response.ChunksField = "chunks"
		if err := Validate(cfg); err != nil {
			t.Fatalf("provider=%s validate failed: %v", provider, err)
		}
	}
}

func TestContextAssemblerCA2ExternalValidationRejectsMissingEndpoint(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ContextAssembler.CA2.Enabled = true
	cfg.ContextAssembler.CA2.Stage2.Provider = "http"
	cfg.ContextAssembler.CA2.Stage2.External.Endpoint = ""
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for missing external endpoint")
	}
}

func TestContextAssemblerCA2ExternalValidationRejectsInvalidMappingMode(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ContextAssembler.CA2.Enabled = true
	cfg.ContextAssembler.CA2.Stage2.Provider = "http"
	cfg.ContextAssembler.CA2.Stage2.External.Endpoint = "http://127.0.0.1:8080/retrieve"
	cfg.ContextAssembler.CA2.Stage2.External.Mapping.Request.Mode = "custom"
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for invalid request mapping mode")
	}
}

func TestContextAssemblerCA2ExternalValidationRejectsMissingQueryOrChunksMapping(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ContextAssembler.CA2.Enabled = true
	cfg.ContextAssembler.CA2.Stage2.Provider = "http"
	cfg.ContextAssembler.CA2.Stage2.External.Endpoint = "http://127.0.0.1:8080/retrieve"
	cfg.ContextAssembler.CA2.Stage2.External.Mapping.Request.Mode = "jsonrpc2"
	cfg.ContextAssembler.CA2.Stage2.External.Mapping.Request.MethodName = ""
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for missing method_name in jsonrpc2 mode")
	}

	cfg = DefaultConfig()
	cfg.ContextAssembler.CA2.Enabled = true
	cfg.ContextAssembler.CA2.Stage2.Provider = "http"
	cfg.ContextAssembler.CA2.Stage2.External.Endpoint = "http://127.0.0.1:8080/retrieve"
	cfg.ContextAssembler.CA2.Stage2.External.Mapping.Request.QueryField = "payload.same"
	cfg.ContextAssembler.CA2.Stage2.External.Mapping.Response.ChunksField = "payload.same"
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for mapping field conflict")
	}
}

func TestContextAssemblerCA2ExternalConfigLoadPrecedenceAndHeaders(t *testing.T) {
	t.Setenv("BAYMAX_CONTEXT_ASSEMBLER_CA2_STAGE2_EXTERNAL_ENDPOINT", "http://env.example/retrieve")
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	content := `
context_assembler:
  ca2:
    enabled: true
    stage2:
      provider: http
      external:
        endpoint: http://file.example/retrieve
        method: PUT
        headers:
          X-Tenant: tenant-a
        auth:
          bearer_token: file-token
          header_name: X-Auth
        mapping:
          request:
            mode: plain
            query_field: payload.query
          response:
            chunks_field: result.chunks
            source_field: result.source
            reason_field: result.reason
`
	if err := os.WriteFile(file, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := Load(LoadOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.ContextAssembler.CA2.Stage2.External.Endpoint != "http://env.example/retrieve" {
		t.Fatalf("endpoint = %q, want env override", cfg.ContextAssembler.CA2.Stage2.External.Endpoint)
	}
	if cfg.ContextAssembler.CA2.Stage2.External.Method != "PUT" {
		t.Fatalf("method = %q, want PUT", cfg.ContextAssembler.CA2.Stage2.External.Method)
	}
	if cfg.ContextAssembler.CA2.Stage2.External.Headers["x-tenant"] != "tenant-a" {
		t.Fatalf("headers = %#v, want X-Tenant=tenant-a", cfg.ContextAssembler.CA2.Stage2.External.Headers)
	}
	if cfg.ContextAssembler.CA2.Stage2.External.Auth.HeaderName != "X-Auth" {
		t.Fatalf("auth.header_name = %q, want X-Auth", cfg.ContextAssembler.CA2.Stage2.External.Auth.HeaderName)
	}
	if cfg.ContextAssembler.CA2.Stage2.External.Mapping.Request.QueryField != "payload.query" {
		t.Fatalf("query field = %q", cfg.ContextAssembler.CA2.Stage2.External.Mapping.Request.QueryField)
	}
	if cfg.ContextAssembler.CA2.Stage2.External.Mapping.Response.ChunksField != "result.chunks" {
		t.Fatalf("chunks field = %q", cfg.ContextAssembler.CA2.Stage2.External.Mapping.Response.ChunksField)
	}
}

func TestContextAssemblerCA2ExternalProfileDefaultsAndExplicitOverrides(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	content := `
context_assembler:
  ca2:
    enabled: true
    stage2:
      provider: http
      external:
        profile: ragflow_like
        endpoint: http://file.example/retrieve
        mapping:
          request:
            query_field: payload.query
`
	if err := os.WriteFile(file, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := Load(LoadOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.ContextAssembler.CA2.Stage2.External.Profile != ContextStage2ExternalProfileRAGFlowLike {
		t.Fatalf("profile = %q, want ragflow_like", cfg.ContextAssembler.CA2.Stage2.External.Profile)
	}
	// Explicit override should win over profile default.
	if cfg.ContextAssembler.CA2.Stage2.External.Mapping.Request.QueryField != "payload.query" {
		t.Fatalf("query_field = %q, want payload.query", cfg.ContextAssembler.CA2.Stage2.External.Mapping.Request.QueryField)
	}
	// Non-overridden fields should come from profile defaults.
	if cfg.ContextAssembler.CA2.Stage2.External.Mapping.Response.ChunksField != "data.chunks" {
		t.Fatalf("chunks_field = %q, want data.chunks", cfg.ContextAssembler.CA2.Stage2.External.Mapping.Response.ChunksField)
	}
}

func TestPrecheckStage2ExternalWarningAndError(t *testing.T) {
	okCfg := DefaultConfig().ContextAssembler.CA2.Stage2.External
	okCfg.Profile = ContextStage2ExternalProfileHTTPGeneric
	okCfg.Endpoint = "http://127.0.0.1:8080/retrieve"
	okCfg.Auth.BearerToken = ""
	res := PrecheckStage2External(ContextStage2ProviderHTTP, okCfg)
	if err := res.FirstError(); err != nil {
		t.Fatalf("FirstError() = %v, want nil", err)
	}
	if !res.HasWarnings() {
		t.Fatalf("expected warning findings, got %#v", res.Findings)
	}

	badCfg := okCfg
	badCfg.Profile = "unknown_profile"
	res = PrecheckStage2External(ContextStage2ProviderHTTP, badCfg)
	if err := res.FirstError(); err == nil {
		t.Fatal("expected blocking error for invalid profile")
	}
}

func TestContextAssemblerCA3Defaults(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.ContextAssembler.CA3.Enabled {
		t.Fatal("context_assembler.ca3.enabled = false, want true")
	}
	if cfg.ContextAssembler.CA3.Tokenizer.Mode != "sdk_preferred" {
		t.Fatalf("ca3.tokenizer.mode = %q, want sdk_preferred", cfg.ContextAssembler.CA3.Tokenizer.Mode)
	}
	if cfg.ContextAssembler.CA3.Compaction.Mode != "truncate" {
		t.Fatalf("ca3.compaction.mode = %q, want truncate", cfg.ContextAssembler.CA3.Compaction.Mode)
	}
	if cfg.ContextAssembler.CA3.Compaction.SemanticTimeout <= 0 {
		t.Fatalf("ca3.compaction.semantic_timeout = %v, want > 0", cfg.ContextAssembler.CA3.Compaction.SemanticTimeout)
	}
	if cfg.ContextAssembler.CA3.Compaction.Quality.Threshold <= 0 {
		t.Fatalf("ca3.compaction.quality.threshold = %v, want > 0", cfg.ContextAssembler.CA3.Compaction.Quality.Threshold)
	}
	if strings.TrimSpace(cfg.ContextAssembler.CA3.Compaction.SemanticTemplate.Prompt) == "" {
		t.Fatal("ca3.compaction.semantic_template.prompt should not be empty")
	}
	if cfg.ContextAssembler.CA3.Compaction.Embedding.SimilarityMetric != "cosine" {
		t.Fatalf("ca3.compaction.embedding.similarity_metric = %q, want cosine", cfg.ContextAssembler.CA3.Compaction.Embedding.SimilarityMetric)
	}
	if cfg.ContextAssembler.CA3.Compaction.Embedding.RuleWeight != 0.7 || cfg.ContextAssembler.CA3.Compaction.Embedding.EmbeddingWeight != 0.3 {
		t.Fatalf(
			"ca3.compaction.embedding default weights = (%v,%v), want (0.7,0.3)",
			cfg.ContextAssembler.CA3.Compaction.Embedding.RuleWeight,
			cfg.ContextAssembler.CA3.Compaction.Embedding.EmbeddingWeight,
		)
	}
	if cfg.ContextAssembler.CA3.Compaction.Reranker.Enabled {
		t.Fatal("ca3.compaction.reranker.enabled = true, want false")
	}
	if cfg.ContextAssembler.CA3.Compaction.Reranker.Timeout <= 0 {
		t.Fatalf("ca3.compaction.reranker.timeout = %v, want > 0", cfg.ContextAssembler.CA3.Compaction.Reranker.Timeout)
	}
}

func TestContextAssemblerCA3ValidationFailFastOnInvalidThresholds(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ContextAssembler.CA3.PercentThresholds.Warning = 10
	cfg.ContextAssembler.CA3.PercentThresholds.Comfort = 20
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for non-increasing ca3 percent thresholds")
	}
}

func TestContextAssemblerCA3EnvOverrideTokenizer(t *testing.T) {
	t.Setenv("BAYMAX_CONTEXT_ASSEMBLER_CA3_TOKENIZER_PROVIDER", "gemini")
	t.Setenv("BAYMAX_CONTEXT_ASSEMBLER_CA3_TOKENIZER_SMALL_DELTA_TOKENS", "64")
	cfg, err := Load(LoadOptions{EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.ContextAssembler.CA3.Tokenizer.Provider != "gemini" {
		t.Fatalf("tokenizer.provider = %q, want gemini", cfg.ContextAssembler.CA3.Tokenizer.Provider)
	}
	if cfg.ContextAssembler.CA3.Tokenizer.SmallDeltaTokens != 64 {
		t.Fatalf("tokenizer.small_delta_tokens = %d, want 64", cfg.ContextAssembler.CA3.Tokenizer.SmallDeltaTokens)
	}
}

func TestContextAssemblerCA3CompactionEnvOverride(t *testing.T) {
	t.Setenv("BAYMAX_CONTEXT_ASSEMBLER_CA3_COMPACTION_MODE", "semantic")
	t.Setenv("BAYMAX_CONTEXT_ASSEMBLER_CA3_COMPACTION_SEMANTIC_TIMEOUT", "1200ms")
	t.Setenv("BAYMAX_CONTEXT_ASSEMBLER_CA3_COMPACTION_QUALITY_THRESHOLD", "0.75")
	t.Setenv("BAYMAX_CONTEXT_ASSEMBLER_CA3_COMPACTION_EMBEDDING_ENABLED", "true")
	t.Setenv("BAYMAX_CONTEXT_ASSEMBLER_CA3_COMPACTION_EMBEDDING_SELECTOR", "sdk-default")
	t.Setenv("BAYMAX_CONTEXT_ASSEMBLER_CA3_COMPACTION_EMBEDDING_PROVIDER", "openai")
	t.Setenv("BAYMAX_CONTEXT_ASSEMBLER_CA3_COMPACTION_EMBEDDING_MODEL", "text-embedding-3-small")
	t.Setenv("BAYMAX_CONTEXT_ASSEMBLER_CA3_COMPACTION_EMBEDDING_TIMEOUT", "900ms")
	t.Setenv("BAYMAX_CONTEXT_ASSEMBLER_CA3_COMPACTION_EMBEDDING_SIMILARITY_METRIC", "cosine")
	t.Setenv("BAYMAX_CONTEXT_ASSEMBLER_CA3_COMPACTION_EMBEDDING_RULE_WEIGHT", "0.6")
	t.Setenv("BAYMAX_CONTEXT_ASSEMBLER_CA3_COMPACTION_EMBEDDING_EMBEDDING_WEIGHT", "0.4")
	t.Setenv("BAYMAX_CONTEXT_ASSEMBLER_CA3_COMPACTION_EMBEDDING_AUTH_API_KEY", "embed-key")
	t.Setenv("BAYMAX_CONTEXT_ASSEMBLER_CA3_COMPACTION_EVIDENCE_RECENT_WINDOW", "8")
	cfg, err := Load(LoadOptions{EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.ContextAssembler.CA3.Compaction.Mode != "semantic" {
		t.Fatalf("ca3.compaction.mode = %q, want semantic", cfg.ContextAssembler.CA3.Compaction.Mode)
	}
	if cfg.ContextAssembler.CA3.Compaction.SemanticTimeout != 1200*time.Millisecond {
		t.Fatalf("ca3.compaction.semantic_timeout = %v, want 1200ms", cfg.ContextAssembler.CA3.Compaction.SemanticTimeout)
	}
	if cfg.ContextAssembler.CA3.Compaction.Quality.Threshold != 0.75 {
		t.Fatalf("ca3.compaction.quality.threshold = %v, want 0.75", cfg.ContextAssembler.CA3.Compaction.Quality.Threshold)
	}
	if !cfg.ContextAssembler.CA3.Compaction.Embedding.Enabled {
		t.Fatal("ca3.compaction.embedding.enabled = false, want true")
	}
	if cfg.ContextAssembler.CA3.Compaction.Embedding.Selector != "sdk-default" {
		t.Fatalf("ca3.compaction.embedding.selector = %q, want sdk-default", cfg.ContextAssembler.CA3.Compaction.Embedding.Selector)
	}
	if cfg.ContextAssembler.CA3.Compaction.Embedding.Provider != "openai" {
		t.Fatalf("ca3.compaction.embedding.provider = %q, want openai", cfg.ContextAssembler.CA3.Compaction.Embedding.Provider)
	}
	if cfg.ContextAssembler.CA3.Compaction.Embedding.Model != "text-embedding-3-small" {
		t.Fatalf("ca3.compaction.embedding.model = %q, want text-embedding-3-small", cfg.ContextAssembler.CA3.Compaction.Embedding.Model)
	}
	if cfg.ContextAssembler.CA3.Compaction.Embedding.Timeout != 900*time.Millisecond {
		t.Fatalf("ca3.compaction.embedding.timeout = %v, want 900ms", cfg.ContextAssembler.CA3.Compaction.Embedding.Timeout)
	}
	if cfg.ContextAssembler.CA3.Compaction.Embedding.SimilarityMetric != "cosine" {
		t.Fatalf("ca3.compaction.embedding.similarity_metric = %q, want cosine", cfg.ContextAssembler.CA3.Compaction.Embedding.SimilarityMetric)
	}
	if cfg.ContextAssembler.CA3.Compaction.Embedding.RuleWeight != 0.6 || cfg.ContextAssembler.CA3.Compaction.Embedding.EmbeddingWeight != 0.4 {
		t.Fatalf("ca3.compaction.embedding weights = (%v,%v), want (0.6,0.4)",
			cfg.ContextAssembler.CA3.Compaction.Embedding.RuleWeight,
			cfg.ContextAssembler.CA3.Compaction.Embedding.EmbeddingWeight)
	}
	if cfg.ContextAssembler.CA3.Compaction.Embedding.Auth.APIKey != "embed-key" {
		t.Fatalf("ca3.compaction.embedding.auth.api_key = %q, want embed-key", cfg.ContextAssembler.CA3.Compaction.Embedding.Auth.APIKey)
	}
	if cfg.ContextAssembler.CA3.Compaction.Evidence.RecentWindow != 8 {
		t.Fatalf("ca3.compaction.evidence.recent_window = %d, want 8", cfg.ContextAssembler.CA3.Compaction.Evidence.RecentWindow)
	}
}

func TestContextAssemblerCA3CompactionValidationRejectsInvalidMode(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ContextAssembler.CA3.Compaction.Mode = "custom"
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for ca3.compaction.mode")
	}
}

func TestContextAssemblerCA3CompactionValidationRejectsInvalidQualityThreshold(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ContextAssembler.CA3.Compaction.Quality.Threshold = 1.5
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for quality threshold > 1")
	}
}

func TestContextAssemblerCA3CompactionValidationRejectsInvalidTemplatePlaceholder(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ContextAssembler.CA3.Compaction.SemanticTemplate.Prompt = "compact {{source}} and {{unknown}}"
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for invalid template placeholder")
	}
}

func TestContextAssemblerCA3CompactionValidationRejectsInvalidEmbeddingProvider(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ContextAssembler.CA3.Compaction.Embedding.Enabled = true
	cfg.ContextAssembler.CA3.Compaction.Embedding.Selector = "default"
	cfg.ContextAssembler.CA3.Compaction.Embedding.Provider = "custom"
	cfg.ContextAssembler.CA3.Compaction.Embedding.Model = "text-embedding-3-small"
	cfg.ContextAssembler.CA3.Compaction.Embedding.Timeout = 500 * time.Millisecond
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for invalid embedding provider")
	}
}

func TestContextAssemblerCA3CompactionValidationRejectsInvalidSimilarityMetric(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ContextAssembler.CA3.Compaction.Embedding.SimilarityMetric = "dot"
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for invalid embedding similarity metric")
	}
}

func TestContextAssemblerCA3RerankerValidationRejectsMissingProfile(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ContextAssembler.CA3.Compaction.Embedding.Enabled = true
	cfg.ContextAssembler.CA3.Compaction.Embedding.Selector = "default"
	cfg.ContextAssembler.CA3.Compaction.Embedding.Provider = "openai"
	cfg.ContextAssembler.CA3.Compaction.Embedding.Model = "text-embedding-3-small"
	cfg.ContextAssembler.CA3.Compaction.Embedding.Timeout = 400 * time.Millisecond
	cfg.ContextAssembler.CA3.Compaction.Reranker.Enabled = true
	cfg.ContextAssembler.CA3.Compaction.Reranker.Timeout = 300 * time.Millisecond
	cfg.ContextAssembler.CA3.Compaction.Reranker.ThresholdProfiles = map[string]float64{
		"gemini:text-embedding-004": 0.6,
	}
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for missing provider/model reranker threshold profile")
	}
}

func TestContextAssemblerCA3RerankerEnvOverride(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	content := `
context_assembler:
  ca3:
    compaction:
      embedding:
        enabled: true
        selector: default
        provider: openai
        model: text-embedding-3-small
        timeout: 300ms
      reranker:
        enabled: true
        timeout: 250ms
        max_retries: 2
        threshold_profiles:
          openai:text-embedding-3-small: 0.62
`
	if err := os.WriteFile(file, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := Load(LoadOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if !cfg.ContextAssembler.CA3.Compaction.Reranker.Enabled {
		t.Fatal("ca3.compaction.reranker.enabled = false, want true")
	}
	if cfg.ContextAssembler.CA3.Compaction.Reranker.MaxRetries != 2 {
		t.Fatalf("ca3.compaction.reranker.max_retries = %d, want 2", cfg.ContextAssembler.CA3.Compaction.Reranker.MaxRetries)
	}
	if cfg.ContextAssembler.CA3.Compaction.Reranker.ThresholdProfiles["openai:text-embedding-3-small"] != 0.62 {
		t.Fatalf("missing reranker threshold profile, got %#v", cfg.ContextAssembler.CA3.Compaction.Reranker.ThresholdProfiles)
	}
}

func TestContextAssemblerCA3ValidateRejectsPartialStageOverride(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ContextAssembler.CA3.Stage1.PercentThresholds = ContextAssemblerCA3Thresholds{
		Safe: 20, Comfort: 40, Warning: 60, Danger: 75, Emergency: 0,
	}
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for partial stage1.percent_thresholds override")
	}
}

func TestContextAssemblerCA3ValidateAcceptsCompleteStageOverride(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ContextAssembler.CA3.Stage2.PercentThresholds = ContextAssemblerCA3Thresholds{
		Safe: 10, Comfort: 20, Warning: 30, Danger: 40, Emergency: 50,
	}
	cfg.ContextAssembler.CA3.Stage2.AbsoluteThresholds = ContextAssemblerCA3Thresholds{
		Safe: 1000, Comfort: 2000, Warning: 3000, Danger: 4000, Emergency: 5000,
	}
	if err := Validate(cfg); err != nil {
		t.Fatalf("Validate returned error for complete stage2 override: %v", err)
	}
}
