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

func TestTeamsConfigDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Teams.Enabled {
		t.Fatal("teams.enabled = true, want false")
	}
	if cfg.Teams.DefaultStrategy != TeamsStrategySerial {
		t.Fatalf("teams.default_strategy = %q, want %q", cfg.Teams.DefaultStrategy, TeamsStrategySerial)
	}
	if cfg.Teams.TaskTimeout <= 0 {
		t.Fatalf("teams.task_timeout = %v, want > 0", cfg.Teams.TaskTimeout)
	}
	if cfg.Teams.Parallel.MaxWorkers <= 0 {
		t.Fatalf("teams.parallel.max_workers = %d, want > 0", cfg.Teams.Parallel.MaxWorkers)
	}
	if cfg.Teams.Vote.TieBreak != TeamsVoteTieBreakHighestPriority {
		t.Fatalf("teams.vote.tie_break = %q, want %q", cfg.Teams.Vote.TieBreak, TeamsVoteTieBreakHighestPriority)
	}
	if cfg.Teams.Remote.Enabled {
		t.Fatal("teams.remote.enabled = true, want false")
	}
	if !cfg.Teams.Remote.RequirePeerID {
		t.Fatal("teams.remote.require_peer_id = false, want true")
	}
}

func TestTeamsConfigEnvOverridePrecedence(t *testing.T) {
	t.Setenv("BAYMAX_TEAMS_ENABLED", "true")
	t.Setenv("BAYMAX_TEAMS_DEFAULT_STRATEGY", TeamsStrategyParallel)
	t.Setenv("BAYMAX_TEAMS_TASK_TIMEOUT", "900ms")
	t.Setenv("BAYMAX_TEAMS_PARALLEL_MAX_WORKERS", "6")
	t.Setenv("BAYMAX_TEAMS_VOTE_TIE_BREAK", TeamsVoteTieBreakFirstTaskID)
	t.Setenv("BAYMAX_TEAMS_REMOTE_ENABLED", "true")
	t.Setenv("BAYMAX_TEAMS_REMOTE_REQUIRE_PEER_ID", "false")

	file := filepath.Join(t.TempDir(), "runtime.yaml")
	content := `
teams:
  enabled: false
  default_strategy: serial
  task_timeout: 2s
  parallel:
    max_workers: 3
  vote:
    tie_break: highest_priority
  remote:
    enabled: false
    require_peer_id: true
`
	if err := os.WriteFile(file, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := Load(LoadOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Teams.DefaultStrategy != TeamsStrategyParallel {
		t.Fatalf("teams.default_strategy = %q, want %q", cfg.Teams.DefaultStrategy, TeamsStrategyParallel)
	}
	if cfg.Teams.TaskTimeout != 900*time.Millisecond {
		t.Fatalf("teams.task_timeout = %v, want 900ms", cfg.Teams.TaskTimeout)
	}
	if cfg.Teams.Parallel.MaxWorkers != 6 {
		t.Fatalf("teams.parallel.max_workers = %d, want 6", cfg.Teams.Parallel.MaxWorkers)
	}
	if cfg.Teams.Vote.TieBreak != TeamsVoteTieBreakFirstTaskID {
		t.Fatalf("teams.vote.tie_break = %q, want %q", cfg.Teams.Vote.TieBreak, TeamsVoteTieBreakFirstTaskID)
	}
	if !cfg.Teams.Remote.Enabled {
		t.Fatal("teams.remote.enabled = false, want true from env")
	}
	if cfg.Teams.Remote.RequirePeerID {
		t.Fatal("teams.remote.require_peer_id = true, want false from env")
	}
}

func TestTeamsConfigValidationRejectsInvalidValues(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Teams.DefaultStrategy = "weighted"
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for teams.default_strategy")
	}

	cfg = DefaultConfig()
	cfg.Teams.TaskTimeout = 0
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for teams.task_timeout")
	}

	cfg = DefaultConfig()
	cfg.Teams.Parallel.MaxWorkers = 0
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for teams.parallel.max_workers")
	}

	cfg = DefaultConfig()
	cfg.Teams.Vote.TieBreak = "latest"
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for teams.vote.tie_break")
	}

	cfg = DefaultConfig()
	cfg.Teams.Remote.Enabled = true
	cfg.Teams.Enabled = false
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for teams.remote.enabled when teams.enabled=false")
	}
}

func TestWorkflowConfigDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Workflow.Enabled {
		t.Fatal("workflow.enabled = true, want false")
	}
	if cfg.Workflow.PlannerValidationMode != WorkflowValidationModeStrict {
		t.Fatalf("workflow.planner_validation_mode = %q, want %q", cfg.Workflow.PlannerValidationMode, WorkflowValidationModeStrict)
	}
	if cfg.Workflow.DefaultStepTimeout <= 0 {
		t.Fatalf("workflow.default_step_timeout = %v, want > 0", cfg.Workflow.DefaultStepTimeout)
	}
	if cfg.Workflow.CheckpointBackend != WorkflowCheckpointMemory {
		t.Fatalf("workflow.checkpoint_backend = %q, want %q", cfg.Workflow.CheckpointBackend, WorkflowCheckpointMemory)
	}
	if cfg.Workflow.Remote.Enabled {
		t.Fatal("workflow.remote.enabled = true, want false")
	}
	if !cfg.Workflow.Remote.RequirePeerID {
		t.Fatal("workflow.remote.require_peer_id = false, want true")
	}
	if cfg.Workflow.Remote.DefaultRetryMaxAttempts != 2 {
		t.Fatalf("workflow.remote.default_retry_max_attempts = %d, want 2", cfg.Workflow.Remote.DefaultRetryMaxAttempts)
	}
}

func TestWorkflowConfigEnvOverridePrecedence(t *testing.T) {
	t.Setenv("BAYMAX_WORKFLOW_ENABLED", "true")
	t.Setenv("BAYMAX_WORKFLOW_PLANNER_VALIDATION_MODE", WorkflowValidationModeWarn)
	t.Setenv("BAYMAX_WORKFLOW_DEFAULT_STEP_TIMEOUT", "1400ms")
	t.Setenv("BAYMAX_WORKFLOW_CHECKPOINT_BACKEND", WorkflowCheckpointFile)
	t.Setenv("BAYMAX_WORKFLOW_CHECKPOINT_PATH", "/tmp/workflow-checkpoints")
	t.Setenv("BAYMAX_WORKFLOW_REMOTE_ENABLED", "true")
	t.Setenv("BAYMAX_WORKFLOW_REMOTE_REQUIRE_PEER_ID", "false")
	t.Setenv("BAYMAX_WORKFLOW_REMOTE_DEFAULT_RETRY_MAX_ATTEMPTS", "5")

	file := filepath.Join(t.TempDir(), "runtime.yaml")
	content := `
workflow:
  enabled: false
  planner_validation_mode: strict
  default_step_timeout: 3s
  checkpoint_backend: memory
  checkpoint_path: /tmp/ignored
  remote:
    enabled: false
    require_peer_id: true
    default_retry_max_attempts: 1
`
	if err := os.WriteFile(file, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := Load(LoadOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Workflow.PlannerValidationMode != WorkflowValidationModeWarn {
		t.Fatalf("workflow.planner_validation_mode = %q, want %q", cfg.Workflow.PlannerValidationMode, WorkflowValidationModeWarn)
	}
	if cfg.Workflow.DefaultStepTimeout != 1400*time.Millisecond {
		t.Fatalf("workflow.default_step_timeout = %v, want 1400ms", cfg.Workflow.DefaultStepTimeout)
	}
	if cfg.Workflow.CheckpointBackend != WorkflowCheckpointFile {
		t.Fatalf("workflow.checkpoint_backend = %q, want %q", cfg.Workflow.CheckpointBackend, WorkflowCheckpointFile)
	}
	if cfg.Workflow.CheckpointPath != "/tmp/workflow-checkpoints" {
		t.Fatalf("workflow.checkpoint_path = %q, want /tmp/workflow-checkpoints", cfg.Workflow.CheckpointPath)
	}
	if !cfg.Workflow.Remote.Enabled {
		t.Fatal("workflow.remote.enabled = false, want true from env")
	}
	if cfg.Workflow.Remote.RequirePeerID {
		t.Fatal("workflow.remote.require_peer_id = true, want false from env")
	}
	if cfg.Workflow.Remote.DefaultRetryMaxAttempts != 5 {
		t.Fatalf("workflow.remote.default_retry_max_attempts = %d, want 5", cfg.Workflow.Remote.DefaultRetryMaxAttempts)
	}
}

func TestWorkflowConfigValidationRejectsInvalidValues(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Workflow.PlannerValidationMode = "relaxed"
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for workflow.planner_validation_mode")
	}

	cfg = DefaultConfig()
	cfg.Workflow.DefaultStepTimeout = 0
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for workflow.default_step_timeout")
	}

	cfg = DefaultConfig()
	cfg.Workflow.CheckpointBackend = "db"
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for workflow.checkpoint_backend")
	}

	cfg = DefaultConfig()
	cfg.Workflow.CheckpointBackend = WorkflowCheckpointFile
	cfg.Workflow.CheckpointPath = ""
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for workflow.checkpoint_path when backend=file")
	}

	cfg = DefaultConfig()
	cfg.Workflow.Remote.Enabled = true
	cfg.Workflow.Enabled = false
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for workflow.remote.enabled when workflow.enabled=false")
	}

	cfg = DefaultConfig()
	cfg.Workflow.Remote.DefaultRetryMaxAttempts = -1
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for workflow.remote.default_retry_max_attempts")
	}
}

func TestA2AConfigDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.A2A.Enabled {
		t.Fatal("a2a.enabled = true, want false")
	}
	if cfg.A2A.ClientTimeout <= 0 {
		t.Fatalf("a2a.client_timeout = %v, want > 0", cfg.A2A.ClientTimeout)
	}
	if cfg.A2A.Delivery.Mode != A2ADeliveryModeCallback {
		t.Fatalf("a2a.delivery.mode = %q, want callback", cfg.A2A.Delivery.Mode)
	}
	if cfg.A2A.Delivery.FallbackMode != A2ADeliveryModeCallback {
		t.Fatalf("a2a.delivery.fallback_mode = %q, want callback", cfg.A2A.Delivery.FallbackMode)
	}
	if cfg.A2A.Delivery.CallbackRetry.MaxAttempts != 3 {
		t.Fatalf("a2a.delivery.callback_retry.max_attempts = %d, want 3", cfg.A2A.Delivery.CallbackRetry.MaxAttempts)
	}
	if cfg.A2A.Delivery.SSEReconnect.MaxAttempts != 3 {
		t.Fatalf("a2a.delivery.sse_reconnect.max_attempts = %d, want 3", cfg.A2A.Delivery.SSEReconnect.MaxAttempts)
	}
	if cfg.A2A.Card.VersionPolicy.Mode != A2ACardVersionPolicyStrictMajor {
		t.Fatalf("a2a.card.version_policy.mode = %q, want strict_major", cfg.A2A.Card.VersionPolicy.Mode)
	}
	if cfg.A2A.Card.VersionPolicy.MinSupportedMinor != 0 {
		t.Fatalf("a2a.card.version_policy.min_supported_minor = %d, want 0", cfg.A2A.Card.VersionPolicy.MinSupportedMinor)
	}
	if !cfg.A2A.CapabilityDiscovery.Enabled {
		t.Fatal("a2a.capability_discovery.enabled = false, want true")
	}
	if !cfg.A2A.CapabilityDiscovery.RequireAll {
		t.Fatal("a2a.capability_discovery.require_all = false, want true")
	}
	if cfg.A2A.CapabilityDiscovery.MaxCandidates <= 0 {
		t.Fatalf("a2a.capability_discovery.max_candidates = %d, want > 0", cfg.A2A.CapabilityDiscovery.MaxCandidates)
	}
	if cfg.A2A.AsyncReporting.Enabled {
		t.Fatal("a2a.async_reporting.enabled = true, want false")
	}
	if cfg.A2A.AsyncReporting.Sink != A2AAsyncReportingSinkCallback {
		t.Fatalf("a2a.async_reporting.sink = %q, want callback", cfg.A2A.AsyncReporting.Sink)
	}
	if cfg.A2A.AsyncReporting.Retry.MaxAttempts != 3 {
		t.Fatalf("a2a.async_reporting.retry.max_attempts = %d, want 3", cfg.A2A.AsyncReporting.Retry.MaxAttempts)
	}
	if cfg.A2A.AsyncReporting.Retry.BackoffInitial <= 0 || cfg.A2A.AsyncReporting.Retry.BackoffMax <= 0 {
		t.Fatalf(
			"a2a.async_reporting.retry backoff values must be > 0, got initial=%v max=%v",
			cfg.A2A.AsyncReporting.Retry.BackoffInitial,
			cfg.A2A.AsyncReporting.Retry.BackoffMax,
		)
	}
}

func TestA2AConfigEnvOverridePrecedence(t *testing.T) {
	t.Setenv("BAYMAX_A2A_CLIENT_TIMEOUT", "900ms")
	t.Setenv("BAYMAX_A2A_DELIVERY_MODE", A2ADeliveryModeSSE)
	t.Setenv("BAYMAX_A2A_DELIVERY_CALLBACK_RETRY_MAX_ATTEMPTS", "5")
	t.Setenv("BAYMAX_A2A_CARD_VERSION_POLICY_MIN_SUPPORTED_MINOR", "2")
	t.Setenv("BAYMAX_A2A_CAPABILITY_DISCOVERY_REQUIRE_ALL", "false")
	t.Setenv("BAYMAX_A2A_ASYNC_REPORTING_SINK", A2AAsyncReportingSinkChannel)
	t.Setenv("BAYMAX_A2A_ASYNC_REPORTING_RETRY_MAX_ATTEMPTS", "6")

	file := filepath.Join(t.TempDir(), "runtime.yaml")
	content := `
a2a:
  enabled: true
  client_timeout: 2s
  delivery:
    mode: callback
    fallback_mode: callback
    callback_retry:
      max_attempts: 2
      backoff: 50ms
    sse_reconnect:
      max_attempts: 4
      backoff: 60ms
  card:
    version_policy:
      mode: strict_major
      min_supported_minor: 1
  capability_discovery:
    enabled: true
    require_all: true
    max_candidates: 9
  async_reporting:
    enabled: true
    sink: callback
    retry:
      max_attempts: 2
      backoff_initial: 40ms
      backoff_max: 200ms
`
	if err := os.WriteFile(file, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(LoadOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.A2A.ClientTimeout != 900*time.Millisecond {
		t.Fatalf("a2a.client_timeout = %v, want 900ms", cfg.A2A.ClientTimeout)
	}
	if cfg.A2A.Delivery.Mode != A2ADeliveryModeSSE {
		t.Fatalf("a2a.delivery.mode = %q, want sse", cfg.A2A.Delivery.Mode)
	}
	if cfg.A2A.Delivery.CallbackRetry.MaxAttempts != 5 {
		t.Fatalf("a2a.delivery.callback_retry.max_attempts = %d, want 5", cfg.A2A.Delivery.CallbackRetry.MaxAttempts)
	}
	if cfg.A2A.Card.VersionPolicy.MinSupportedMinor != 2 {
		t.Fatalf("a2a.card.version_policy.min_supported_minor = %d, want 2", cfg.A2A.Card.VersionPolicy.MinSupportedMinor)
	}
	if cfg.A2A.CapabilityDiscovery.RequireAll {
		t.Fatal("a2a.capability_discovery.require_all = true, want false from env")
	}
	if cfg.A2A.CapabilityDiscovery.MaxCandidates != 9 {
		t.Fatalf("a2a.capability_discovery.max_candidates = %d, want 9", cfg.A2A.CapabilityDiscovery.MaxCandidates)
	}
	if cfg.A2A.AsyncReporting.Sink != A2AAsyncReportingSinkChannel {
		t.Fatalf("a2a.async_reporting.sink = %q, want channel from env", cfg.A2A.AsyncReporting.Sink)
	}
	if cfg.A2A.AsyncReporting.Retry.MaxAttempts != 6 {
		t.Fatalf("a2a.async_reporting.retry.max_attempts = %d, want 6 from env", cfg.A2A.AsyncReporting.Retry.MaxAttempts)
	}
	if cfg.A2A.AsyncReporting.Retry.BackoffInitial != 40*time.Millisecond || cfg.A2A.AsyncReporting.Retry.BackoffMax != 200*time.Millisecond {
		t.Fatalf(
			"a2a.async_reporting.retry backoff values = (%v,%v), want (40ms,200ms)",
			cfg.A2A.AsyncReporting.Retry.BackoffInitial,
			cfg.A2A.AsyncReporting.Retry.BackoffMax,
		)
	}
}

func TestA2AConfigValidationRejectsInvalidValues(t *testing.T) {
	cfg := DefaultConfig()
	cfg.A2A.ClientTimeout = 0
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for a2a.client_timeout")
	}

	cfg = DefaultConfig()
	cfg.A2A.Delivery.Mode = "websocket"
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for a2a.delivery.mode")
	}

	cfg = DefaultConfig()
	cfg.A2A.Delivery.SSEReconnect.MaxAttempts = 0
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for a2a.delivery.sse_reconnect.max_attempts")
	}

	cfg = DefaultConfig()
	cfg.A2A.Card.VersionPolicy.Mode = "compat"
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for a2a.card.version_policy.mode")
	}

	cfg = DefaultConfig()
	cfg.A2A.Card.VersionPolicy.MinSupportedMinor = -1
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for a2a.card.version_policy.min_supported_minor")
	}

	cfg = DefaultConfig()
	cfg.A2A.CapabilityDiscovery.MaxCandidates = 0
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for a2a.capability_discovery.max_candidates")
	}

	cfg = DefaultConfig()
	cfg.A2A.AsyncReporting.Sink = "webhook"
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for a2a.async_reporting.sink")
	}

	cfg = DefaultConfig()
	cfg.A2A.AsyncReporting.Retry.MaxAttempts = 0
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for a2a.async_reporting.retry.max_attempts")
	}

	cfg = DefaultConfig()
	cfg.A2A.AsyncReporting.Retry.BackoffInitial = 100 * time.Millisecond
	cfg.A2A.AsyncReporting.Retry.BackoffMax = 10 * time.Millisecond
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for a2a.async_reporting.retry.backoff_max")
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
	if cfg.Skill.TriggerScoring.Lexical.TokenizerMode != SkillTriggerScoringTokenizerMixedCJKEN {
		t.Fatalf("skill.trigger_scoring.lexical.tokenizer_mode = %q, want %q", cfg.Skill.TriggerScoring.Lexical.TokenizerMode, SkillTriggerScoringTokenizerMixedCJKEN)
	}
	if cfg.Skill.TriggerScoring.MaxSemanticCandidates != 5 {
		t.Fatalf("skill.trigger_scoring.max_semantic_candidates = %d, want 5", cfg.Skill.TriggerScoring.MaxSemanticCandidates)
	}
	if cfg.Skill.TriggerScoring.Budget.Mode != SkillTriggerScoringBudgetModeAdaptive {
		t.Fatalf("skill.trigger_scoring.budget.mode = %q, want %q", cfg.Skill.TriggerScoring.Budget.Mode, SkillTriggerScoringBudgetModeAdaptive)
	}
	if cfg.Skill.TriggerScoring.Budget.Adaptive.MinK != 1 || cfg.Skill.TriggerScoring.Budget.Adaptive.MaxK != 5 {
		t.Fatalf(
			"skill.trigger_scoring.budget.adaptive (min_k,max_k) = (%d,%d), want (1,5)",
			cfg.Skill.TriggerScoring.Budget.Adaptive.MinK,
			cfg.Skill.TriggerScoring.Budget.Adaptive.MaxK,
		)
	}
	if cfg.Skill.TriggerScoring.Budget.Adaptive.MinScoreMargin != 0.08 {
		t.Fatalf("skill.trigger_scoring.budget.adaptive.min_score_margin = %v, want 0.08", cfg.Skill.TriggerScoring.Budget.Adaptive.MinScoreMargin)
	}
	if cfg.Skill.TriggerScoring.Embedding.Timeout <= 0 {
		t.Fatalf("skill.trigger_scoring.embedding.timeout = %v, want > 0", cfg.Skill.TriggerScoring.Embedding.Timeout)
	}
	if cfg.Skill.TriggerScoring.Embedding.SimilarityMetric != SkillTriggerScoringSimilarityCosine {
		t.Fatalf("skill.trigger_scoring.embedding.similarity_metric = %q, want %q", cfg.Skill.TriggerScoring.Embedding.SimilarityMetric, SkillTriggerScoringSimilarityCosine)
	}
	if cfg.Skill.TriggerScoring.Embedding.LexicalWeight != 0.7 || cfg.Skill.TriggerScoring.Embedding.EmbeddingWeight != 0.3 {
		t.Fatalf("skill.trigger_scoring.embedding weights = (%v,%v), want (0.7,0.3)", cfg.Skill.TriggerScoring.Embedding.LexicalWeight, cfg.Skill.TriggerScoring.Embedding.EmbeddingWeight)
	}
}

func TestSkillTriggerScoringLoadEnvOverFile(t *testing.T) {
	t.Setenv("BAYMAX_SKILL_TRIGGER_SCORING_TIE_BREAK", SkillTriggerScoringTieBreakFirstRegistered)
	t.Setenv("BAYMAX_SKILL_TRIGGER_SCORING_SUPPRESS_LOW_CONFIDENCE", "false")
	t.Setenv("BAYMAX_SKILL_TRIGGER_SCORING_CONFIDENCE_THRESHOLD", "0.61")
	t.Setenv("BAYMAX_SKILL_TRIGGER_SCORING_LEXICAL_TOKENIZER_MODE", SkillTriggerScoringTokenizerMixedCJKEN)
	t.Setenv("BAYMAX_SKILL_TRIGGER_SCORING_MAX_SEMANTIC_CANDIDATES", "5")
	t.Setenv("BAYMAX_SKILL_TRIGGER_SCORING_BUDGET_MODE", SkillTriggerScoringBudgetModeFixed)
	t.Setenv("BAYMAX_SKILL_TRIGGER_SCORING_BUDGET_ADAPTIVE_MIN_K", "2")
	t.Setenv("BAYMAX_SKILL_TRIGGER_SCORING_BUDGET_ADAPTIVE_MAX_K", "4")
	t.Setenv("BAYMAX_SKILL_TRIGGER_SCORING_BUDGET_ADAPTIVE_MIN_SCORE_MARGIN", "0.11")

	file := filepath.Join(t.TempDir(), "runtime.yaml")
	content := `
skill:
  trigger_scoring:
    strategy: lexical_weighted_keywords
    tie_break: highest_priority
    suppress_low_confidence: true
    confidence_threshold: 0.33
    max_semantic_candidates: 2
    lexical:
      tokenizer_mode: mixed_cjk_en
    budget:
      mode: adaptive
      adaptive:
        min_k: 1
        max_k: 3
        min_score_margin: 0.08
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
	if cfg.Skill.TriggerScoring.MaxSemanticCandidates != 5 {
		t.Fatalf("skill.trigger_scoring.max_semantic_candidates = %d, want 5 from env", cfg.Skill.TriggerScoring.MaxSemanticCandidates)
	}
	if cfg.Skill.TriggerScoring.Lexical.TokenizerMode != SkillTriggerScoringTokenizerMixedCJKEN {
		t.Fatalf("skill.trigger_scoring.lexical.tokenizer_mode = %q, want %q", cfg.Skill.TriggerScoring.Lexical.TokenizerMode, SkillTriggerScoringTokenizerMixedCJKEN)
	}
	if cfg.Skill.TriggerScoring.Budget.Mode != SkillTriggerScoringBudgetModeFixed {
		t.Fatalf("skill.trigger_scoring.budget.mode = %q, want %q", cfg.Skill.TriggerScoring.Budget.Mode, SkillTriggerScoringBudgetModeFixed)
	}
	if cfg.Skill.TriggerScoring.Budget.Adaptive.MinK != 2 || cfg.Skill.TriggerScoring.Budget.Adaptive.MaxK != 4 {
		t.Fatalf(
			"skill.trigger_scoring.budget.adaptive (min_k,max_k) = (%d,%d), want (2,4)",
			cfg.Skill.TriggerScoring.Budget.Adaptive.MinK,
			cfg.Skill.TriggerScoring.Budget.Adaptive.MaxK,
		)
	}
	if cfg.Skill.TriggerScoring.Budget.Adaptive.MinScoreMargin != 0.11 {
		t.Fatalf("skill.trigger_scoring.budget.adaptive.min_score_margin = %v, want 0.11", cfg.Skill.TriggerScoring.Budget.Adaptive.MinScoreMargin)
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

func TestSkillTriggerScoringEmbeddingLoadEnvOverFile(t *testing.T) {
	t.Setenv("BAYMAX_SKILL_TRIGGER_SCORING_STRATEGY", SkillTriggerScoringStrategyLexicalPlusEmbedding)
	t.Setenv("BAYMAX_SKILL_TRIGGER_SCORING_EMBEDDING_TIMEOUT", "450ms")
	t.Setenv("BAYMAX_SKILL_TRIGGER_SCORING_EMBEDDING_LEXICAL_WEIGHT", "0.6")

	file := filepath.Join(t.TempDir(), "runtime.yaml")
	content := `
skill:
  trigger_scoring:
    strategy: lexical_weighted_keywords
    confidence_threshold: 0.42
    tie_break: highest_priority
    suppress_low_confidence: true
    keyword_weights:
      db: 1.9
    embedding:
      enabled: true
      provider: openai
      model: text-embedding-3-small
      timeout: 300ms
      similarity_metric: cosine
      lexical_weight: 0.7
      embedding_weight: 0.3
`
	if err := os.WriteFile(file, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := Load(LoadOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Skill.TriggerScoring.Strategy != SkillTriggerScoringStrategyLexicalPlusEmbedding {
		t.Fatalf("skill.trigger_scoring.strategy = %q, want %q", cfg.Skill.TriggerScoring.Strategy, SkillTriggerScoringStrategyLexicalPlusEmbedding)
	}
	if cfg.Skill.TriggerScoring.Embedding.Timeout != 450*time.Millisecond {
		t.Fatalf("skill.trigger_scoring.embedding.timeout = %v, want 450ms", cfg.Skill.TriggerScoring.Embedding.Timeout)
	}
	if cfg.Skill.TriggerScoring.Embedding.LexicalWeight != 0.6 {
		t.Fatalf("skill.trigger_scoring.embedding.lexical_weight = %v, want 0.6", cfg.Skill.TriggerScoring.Embedding.LexicalWeight)
	}
}

func TestSkillTriggerScoringValidateRejectsEmbeddingConfig(t *testing.T) {
	t.Run("strategy_requires_embedding_enabled", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Skill.TriggerScoring.Strategy = SkillTriggerScoringStrategyLexicalPlusEmbedding
		cfg.Skill.TriggerScoring.Embedding.Enabled = false
		if err := Validate(cfg); err == nil {
			t.Fatal("expected validation error for strategy requiring embedding.enabled=true")
		}
	})

	t.Run("invalid_similarity_metric", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Skill.TriggerScoring.Embedding.SimilarityMetric = "dot_product"
		if err := Validate(cfg); err == nil {
			t.Fatal("expected validation error for skill.trigger_scoring.embedding.similarity_metric")
		}
	})

	t.Run("invalid_weights", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Skill.TriggerScoring.Embedding.LexicalWeight = -0.1
		if err := Validate(cfg); err == nil {
			t.Fatal("expected validation error for skill.trigger_scoring.embedding.lexical_weight")
		}
		cfg = DefaultConfig()
		cfg.Skill.TriggerScoring.Embedding.EmbeddingWeight = -0.1
		if err := Validate(cfg); err == nil {
			t.Fatal("expected validation error for skill.trigger_scoring.embedding.embedding_weight")
		}
		cfg = DefaultConfig()
		cfg.Skill.TriggerScoring.Embedding.LexicalWeight = 0
		cfg.Skill.TriggerScoring.Embedding.EmbeddingWeight = 0
		if err := Validate(cfg); err == nil {
			t.Fatal("expected validation error for zero skill.trigger_scoring.embedding weights")
		}
	})
}

func TestSkillTriggerScoringValidateRejectsLexicalBudgetConfig(t *testing.T) {
	t.Run("invalid_tokenizer_mode", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Skill.TriggerScoring.Lexical.TokenizerMode = "unicode_all"
		if err := Validate(cfg); err == nil {
			t.Fatal("expected validation error for skill.trigger_scoring.lexical.tokenizer_mode")
		}
	})

	t.Run("non_positive_max_semantic_candidates", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Skill.TriggerScoring.MaxSemanticCandidates = 0
		if err := Validate(cfg); err == nil {
			t.Fatal("expected validation error for skill.trigger_scoring.max_semantic_candidates")
		}
	})

	t.Run("invalid_budget_mode", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Skill.TriggerScoring.Budget.Mode = "dynamic"
		if err := Validate(cfg); err == nil {
			t.Fatal("expected validation error for skill.trigger_scoring.budget.mode")
		}
	})

	t.Run("invalid_adaptive_k_range", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Skill.TriggerScoring.Budget.Adaptive.MinK = 3
		cfg.Skill.TriggerScoring.Budget.Adaptive.MaxK = 2
		if err := Validate(cfg); err == nil {
			t.Fatal("expected validation error for skill.trigger_scoring.budget.adaptive.max_k")
		}
	})

	t.Run("adaptive_max_k_exceeds_semantic_cap", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Skill.TriggerScoring.MaxSemanticCandidates = 4
		cfg.Skill.TriggerScoring.Budget.Adaptive.MaxK = 5
		if err := Validate(cfg); err == nil {
			t.Fatal("expected validation error for skill.trigger_scoring.budget.adaptive.max_k <= max_semantic_candidates")
		}
	})

	t.Run("invalid_adaptive_margin", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Skill.TriggerScoring.Budget.Adaptive.MinScoreMargin = 1.2
		if err := Validate(cfg); err == nil {
			t.Fatal("expected validation error for skill.trigger_scoring.budget.adaptive.min_score_margin")
		}
	})
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
	if cfg.ContextAssembler.CA2.Agentic.DecisionTimeout <= 0 {
		t.Fatalf("agentic.decision_timeout = %v, want > 0", cfg.ContextAssembler.CA2.Agentic.DecisionTimeout)
	}
	if cfg.ContextAssembler.CA2.Agentic.FailurePolicy != ContextCA2AgenticFailurePolicyBestEffortRules {
		t.Fatalf("agentic.failure_policy = %q, want %q", cfg.ContextAssembler.CA2.Agentic.FailurePolicy, ContextCA2AgenticFailurePolicyBestEffortRules)
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
	t.Setenv("BAYMAX_CONTEXT_ASSEMBLER_CA2_AGENTIC_DECISION_TIMEOUT", "150ms")
	t.Setenv("BAYMAX_CONTEXT_ASSEMBLER_CA2_AGENTIC_FAILURE_POLICY", ContextCA2AgenticFailurePolicyBestEffortRules)
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
	if cfg.ContextAssembler.CA2.Agentic.DecisionTimeout != 150*time.Millisecond {
		t.Fatalf("agentic.decision_timeout = %v, want 150ms", cfg.ContextAssembler.CA2.Agentic.DecisionTimeout)
	}
	if cfg.ContextAssembler.CA2.Agentic.FailurePolicy != ContextCA2AgenticFailurePolicyBestEffortRules {
		t.Fatalf("agentic.failure_policy = %q, want %q", cfg.ContextAssembler.CA2.Agentic.FailurePolicy, ContextCA2AgenticFailurePolicyBestEffortRules)
	}
}

func TestContextAssemblerCA2ValidationRejectsInvalidAgenticTimeout(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ContextAssembler.CA2.Enabled = true
	cfg.ContextAssembler.CA2.Agentic.DecisionTimeout = 0
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for invalid ca2 agentic timeout")
	}
}

func TestContextAssemblerCA2ValidationRejectsInvalidAgenticFailurePolicy(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ContextAssembler.CA2.Enabled = true
	cfg.ContextAssembler.CA2.Agentic.FailurePolicy = "deny"
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for invalid ca2 agentic failure policy")
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

func TestSecurityS2Defaults(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.Security.ToolGovernance.Enabled {
		t.Fatal("security.tool_governance.enabled = false, want true")
	}
	if cfg.Security.ToolGovernance.Mode != SecurityGovernanceModeEnforce {
		t.Fatalf("security.tool_governance.mode = %q, want enforce", cfg.Security.ToolGovernance.Mode)
	}
	if cfg.Security.ToolGovernance.RateLimit.Scope != SecurityToolRateLimitScopeProcess {
		t.Fatalf("security.tool_governance.rate_limit.scope = %q, want process", cfg.Security.ToolGovernance.RateLimit.Scope)
	}
	if cfg.Security.ToolGovernance.RateLimit.ExceedAction != SecurityToolPolicyDeny {
		t.Fatalf("security.tool_governance.rate_limit.exceed_action = %q, want deny", cfg.Security.ToolGovernance.RateLimit.ExceedAction)
	}
	if !cfg.Security.ModelIOFiltering.Enabled {
		t.Fatal("security.model_io_filtering.enabled = false, want true")
	}
	if cfg.Security.ModelIOFiltering.RequireRegisteredFilter {
		t.Fatal("security.model_io_filtering.require_registered_filter = true, want false")
	}
	if cfg.Security.ModelIOFiltering.Input.BlockAction != SecurityModelIOFilterBlockActionDeny {
		t.Fatalf("security.model_io_filtering.input.block_action = %q, want deny", cfg.Security.ModelIOFiltering.Input.BlockAction)
	}
	if cfg.Security.ModelIOFiltering.Output.BlockAction != SecurityModelIOFilterBlockActionDeny {
		t.Fatalf("security.model_io_filtering.output.block_action = %q, want deny", cfg.Security.ModelIOFiltering.Output.BlockAction)
	}
}

func TestSecurityS2EnvOverridePrecedence(t *testing.T) {
	t.Setenv("BAYMAX_SECURITY_TOOL_GOVERNANCE_RATE_LIMIT_LIMIT", "9")
	t.Setenv("BAYMAX_SECURITY_MODEL_IO_FILTERING_REQUIRE_REGISTERED_FILTER", "true")
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	content := `
security:
  tool_governance:
    rate_limit:
      limit: 3
  model_io_filtering:
    require_registered_filter: false
`
	if err := os.WriteFile(file, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := Load(LoadOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Security.ToolGovernance.RateLimit.Limit != 9 {
		t.Fatalf("security.tool_governance.rate_limit.limit = %d, want 9", cfg.Security.ToolGovernance.RateLimit.Limit)
	}
	if !cfg.Security.ModelIOFiltering.RequireRegisteredFilter {
		t.Fatal("security.model_io_filtering.require_registered_filter = false, want true")
	}
}

func TestSecurityEventS3Defaults(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.Security.SecurityEvent.Enabled {
		t.Fatal("security.security_event.enabled = false, want true")
	}
	if cfg.Security.SecurityEvent.Alert.TriggerPolicy != SecurityEventAlertPolicyDenyOnly {
		t.Fatalf("security.security_event.alert.trigger_policy = %q, want deny_only", cfg.Security.SecurityEvent.Alert.TriggerPolicy)
	}
	if cfg.Security.SecurityEvent.Alert.Sink != SecurityEventAlertSinkCallback {
		t.Fatalf("security.security_event.alert.sink = %q, want callback", cfg.Security.SecurityEvent.Alert.Sink)
	}
	if cfg.Security.SecurityEvent.Severity.Default != SecurityEventSeverityHigh {
		t.Fatalf("security.security_event.severity.default = %q, want high", cfg.Security.SecurityEvent.Severity.Default)
	}
	if cfg.Security.SecurityEvent.Severity.ByReasonCode["security.io_filter_match"] != SecurityEventSeverityMedium {
		t.Fatalf("security.security_event.severity.by_reason_code.security.io_filter_match = %q, want medium", cfg.Security.SecurityEvent.Severity.ByReasonCode["security.io_filter_match"])
	}
}

func TestSecurityEventS4DeliveryDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Security.SecurityEvent.Delivery.Mode != SecurityEventDeliveryModeAsync {
		t.Fatalf("security.security_event.delivery.mode = %q, want async", cfg.Security.SecurityEvent.Delivery.Mode)
	}
	if cfg.Security.SecurityEvent.Delivery.Queue.Size <= 0 {
		t.Fatalf("security.security_event.delivery.queue.size = %d, want > 0", cfg.Security.SecurityEvent.Delivery.Queue.Size)
	}
	if cfg.Security.SecurityEvent.Delivery.Queue.OverflowPolicy != SecurityEventDeliveryOverflowDropOld {
		t.Fatalf(
			"security.security_event.delivery.queue.overflow_policy = %q, want drop_old",
			cfg.Security.SecurityEvent.Delivery.Queue.OverflowPolicy,
		)
	}
	if cfg.Security.SecurityEvent.Delivery.Retry.MaxAttempts != 3 {
		t.Fatalf("security.security_event.delivery.retry.max_attempts = %d, want 3", cfg.Security.SecurityEvent.Delivery.Retry.MaxAttempts)
	}
	if cfg.Security.SecurityEvent.Delivery.CircuitBreaker.FailureThreshold <= 0 {
		t.Fatalf(
			"security.security_event.delivery.circuit_breaker.failure_threshold = %d, want > 0",
			cfg.Security.SecurityEvent.Delivery.CircuitBreaker.FailureThreshold,
		)
	}
}

func TestSecurityEventS3EnvOverridePrecedence(t *testing.T) {
	t.Setenv("BAYMAX_SECURITY_SECURITY_EVENT_ALERT_TRIGGER_POLICY", "deny_only")
	t.Setenv("BAYMAX_SECURITY_SECURITY_EVENT_SEVERITY_DEFAULT", "medium")
	t.Setenv("BAYMAX_SECURITY_SECURITY_EVENT_ALERT_CALLBACK_REQUIRE_REGISTERED", "true")
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	content := `
security:
  security_event:
    alert:
      trigger_policy: deny_only
      sink: callback
      callback:
        require_registered: false
    severity:
      default: high
`
	if err := os.WriteFile(file, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := Load(LoadOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Security.SecurityEvent.Alert.TriggerPolicy != SecurityEventAlertPolicyDenyOnly {
		t.Fatalf("security.security_event.alert.trigger_policy = %q, want deny_only", cfg.Security.SecurityEvent.Alert.TriggerPolicy)
	}
	if cfg.Security.SecurityEvent.Severity.Default != SecurityEventSeverityMedium {
		t.Fatalf("security.security_event.severity.default = %q, want env override medium", cfg.Security.SecurityEvent.Severity.Default)
	}
	if !cfg.Security.SecurityEvent.Alert.Callback.RequireRegistered {
		t.Fatal("security.security_event.alert.callback.require_registered = false, want true from env")
	}
}

func TestSecurityEventS4DeliveryEnvOverridePrecedence(t *testing.T) {
	t.Setenv("BAYMAX_SECURITY_SECURITY_EVENT_DELIVERY_MODE", "sync")
	t.Setenv("BAYMAX_SECURITY_SECURITY_EVENT_DELIVERY_RETRY_MAX_ATTEMPTS", "2")
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	content := `
security:
  security_event:
    delivery:
      mode: async
      queue:
        size: 17
        overflow_policy: drop_old
      timeout: 900ms
      retry:
        max_attempts: 3
        backoff_initial: 20ms
        backoff_max: 60ms
      circuit_breaker:
        failure_threshold: 7
        open_window: 3s
        half_open_probes: 1
`
	if err := os.WriteFile(file, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := Load(LoadOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Security.SecurityEvent.Delivery.Mode != SecurityEventDeliveryModeSync {
		t.Fatalf("security.security_event.delivery.mode = %q, want env override sync", cfg.Security.SecurityEvent.Delivery.Mode)
	}
	if cfg.Security.SecurityEvent.Delivery.Retry.MaxAttempts != 2 {
		t.Fatalf(
			"security.security_event.delivery.retry.max_attempts = %d, want env override 2",
			cfg.Security.SecurityEvent.Delivery.Retry.MaxAttempts,
		)
	}
	if cfg.Security.SecurityEvent.Delivery.Queue.Size != 17 {
		t.Fatalf("security.security_event.delivery.queue.size = %d, want 17 from file", cfg.Security.SecurityEvent.Delivery.Queue.Size)
	}
}

func TestSecurityDeliveryContractConfigDefaultsAndPrecedence(t *testing.T) {
	base := DefaultConfig()
	if base.Security.SecurityEvent.Delivery.Mode != SecurityEventDeliveryModeAsync {
		t.Fatalf("default delivery mode = %q, want async", base.Security.SecurityEvent.Delivery.Mode)
	}

	t.Setenv("BAYMAX_SECURITY_SECURITY_EVENT_DELIVERY_MODE", "sync")
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	content := `
security:
  security_event:
    delivery:
      mode: async
      queue:
        size: 9
        overflow_policy: drop_old
      timeout: 1s
      retry:
        max_attempts: 3
        backoff_initial: 20ms
        backoff_max: 100ms
      circuit_breaker:
        failure_threshold: 4
        open_window: 2s
        half_open_probes: 1
`
	if err := os.WriteFile(file, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := Load(LoadOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Security.SecurityEvent.Delivery.Mode != SecurityEventDeliveryModeSync {
		t.Fatalf("delivery.mode = %q, want env override sync", cfg.Security.SecurityEvent.Delivery.Mode)
	}
}

func TestSecurityEventContractValidateRejectsInvalidS3Config(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Security.SecurityEvent.Alert.TriggerPolicy = "all"
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "security.security_event.alert.trigger_policy") {
		t.Fatalf("expected invalid trigger_policy validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Security.SecurityEvent.Severity.ByPolicyKind = map[string]string{"unknown": "high"}
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "by_policy_kind") {
		t.Fatalf("expected invalid by_policy_kind validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Security.SecurityEvent.Severity.ByReasonCode = map[string]string{"security.permission_denied": "critical"}
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "by_reason_code") {
		t.Fatalf("expected invalid by_reason_code severity validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Security.SecurityEvent.Delivery.Mode = "queue"
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "security.security_event.delivery.mode") {
		t.Fatalf("expected invalid delivery mode validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Security.SecurityEvent.Delivery.Retry.MaxAttempts = 4
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "retry.max_attempts") {
		t.Fatalf("expected invalid retry.max_attempts validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Security.SecurityEvent.Delivery.CircuitBreaker.OpenWindow = 0
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "circuit_breaker.open_window") {
		t.Fatalf("expected invalid circuit_breaker.open_window validation error, got %v", err)
	}
}

func TestSecurityDeliveryContractValidateRejectsMalformedDeliveryConfig(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Security.SecurityEvent.Delivery.Queue.OverflowPolicy = "drop_new"
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "overflow_policy") {
		t.Fatalf("expected overflow_policy validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Security.SecurityEvent.Delivery.Retry.BackoffInitial = 50 * time.Millisecond
	cfg.Security.SecurityEvent.Delivery.Retry.BackoffMax = 10 * time.Millisecond
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "backoff_max") {
		t.Fatalf("expected retry backoff validation error, got %v", err)
	}
}

func TestValidateRejectsMalformedNamespaceToolSelector(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Security.ToolGovernance.Permission.ByTool = map[string]string{
		"local.echo": SecurityToolPolicyDeny,
	}
	err := Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "namespace+tool") {
		t.Fatalf("expected namespace+tool validation error, got %v", err)
	}
}

func TestValidateRejectsInvalidModelIOFilterBlockAction(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Security.ModelIOFiltering.Output.BlockAction = "warn"
	err := Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "security.model_io_filtering.output.block_action") {
		t.Fatalf("expected output block_action validation error, got %v", err)
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
	if cfg.ContextAssembler.CA2.Stage2.External.TemplateResolutionSource != Stage2TemplateResolutionProfileDefaultsWithOverride {
		t.Fatalf(
			"template_resolution_source = %q, want %q",
			cfg.ContextAssembler.CA2.Stage2.External.TemplateResolutionSource,
			Stage2TemplateResolutionProfileDefaultsWithOverride,
		)
	}
}

func TestContextAssemblerCA2ExternalExplicitOnlyProfileAccepted(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	content := `
context_assembler:
  ca2:
    enabled: true
    stage2:
      provider: http
      external:
        profile: explicit_only
        endpoint: http://file.example/retrieve
        mapping:
          request:
            mode: plain
            query_field: payload.query
          response:
            chunks_field: result.chunks
`
	if err := os.WriteFile(file, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := Load(LoadOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.ContextAssembler.CA2.Stage2.External.Profile != ContextStage2ExternalProfileExplicitOnly {
		t.Fatalf("profile = %q, want explicit_only", cfg.ContextAssembler.CA2.Stage2.External.Profile)
	}
	if cfg.ContextAssembler.CA2.Stage2.External.TemplateResolutionSource != Stage2TemplateResolutionExplicitOnly {
		t.Fatalf(
			"template_resolution_source = %q, want %q",
			cfg.ContextAssembler.CA2.Stage2.External.TemplateResolutionSource,
			Stage2TemplateResolutionExplicitOnly,
		)
	}
	if cfg.ContextAssembler.CA2.Stage2.External.Mapping.Request.QueryField != "payload.query" {
		t.Fatalf("query_field = %q, want payload.query", cfg.ContextAssembler.CA2.Stage2.External.Mapping.Request.QueryField)
	}
}

func TestContextAssemblerCA2ExternalHintsValidation(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ContextAssembler.CA2.Enabled = true
	cfg.ContextAssembler.CA2.Stage2.Provider = ContextStage2ProviderHTTP
	cfg.ContextAssembler.CA2.Stage2.External.Endpoint = "http://127.0.0.1:8080/retrieve"
	cfg.ContextAssembler.CA2.Stage2.External.Hints.Enabled = true
	cfg.ContextAssembler.CA2.Stage2.External.Hints.Capabilities = []string{}
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for empty hints.capabilities when hints.enabled=true")
	}

	cfg.ContextAssembler.CA2.Stage2.External.Hints.Capabilities = []string{"bad hint"}
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for malformed hint capability")
	}

	cfg.ContextAssembler.CA2.Stage2.External.Hints.Capabilities = []string{"rerank", "metadata_filter"}
	if err := Validate(cfg); err != nil {
		t.Fatalf("Validate failed for valid hints: %v", err)
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
	if cfg.ContextAssembler.CA3.Compaction.Reranker.Governance.Mode != CA3RerankerGovernanceModeEnforce {
		t.Fatalf("ca3.compaction.reranker.governance.mode = %q, want enforce", cfg.ContextAssembler.CA3.Compaction.Reranker.Governance.Mode)
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
        governance:
          mode: dry_run
          profile_version: e5-canary-v1
          rollout_provider_models:
            - openai:text-embedding-3-small
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
	if cfg.ContextAssembler.CA3.Compaction.Reranker.Governance.Mode != CA3RerankerGovernanceModeDryRun {
		t.Fatalf("governance.mode = %q, want dry_run", cfg.ContextAssembler.CA3.Compaction.Reranker.Governance.Mode)
	}
	if cfg.ContextAssembler.CA3.Compaction.Reranker.Governance.ProfileVersion != "e5-canary-v1" {
		t.Fatalf("governance.profile_version = %q, want e5-canary-v1", cfg.ContextAssembler.CA3.Compaction.Reranker.Governance.ProfileVersion)
	}
	if len(cfg.ContextAssembler.CA3.Compaction.Reranker.Governance.RolloutProviderModels) != 1 ||
		cfg.ContextAssembler.CA3.Compaction.Reranker.Governance.RolloutProviderModels[0] != "openai:text-embedding-3-small" {
		t.Fatalf("governance.rollout_provider_models = %#v", cfg.ContextAssembler.CA3.Compaction.Reranker.Governance.RolloutProviderModels)
	}
}

func TestContextAssemblerCA3RerankerGovernanceValidationRejectsInvalidMode(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ContextAssembler.CA3.Compaction.Embedding.Enabled = true
	cfg.ContextAssembler.CA3.Compaction.Embedding.Selector = "default"
	cfg.ContextAssembler.CA3.Compaction.Embedding.Provider = "openai"
	cfg.ContextAssembler.CA3.Compaction.Embedding.Model = "text-embedding-3-small"
	cfg.ContextAssembler.CA3.Compaction.Embedding.Timeout = 300 * time.Millisecond
	cfg.ContextAssembler.CA3.Compaction.Reranker.Enabled = true
	cfg.ContextAssembler.CA3.Compaction.Reranker.Governance.Mode = "shadow"
	cfg.ContextAssembler.CA3.Compaction.Reranker.ThresholdProfiles = map[string]float64{
		"openai:text-embedding-3-small": 0.62,
	}
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for invalid reranker governance mode")
	}
}

func TestContextAssemblerCA3RerankerGovernanceValidationRejectsInvalidRolloutKey(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ContextAssembler.CA3.Compaction.Embedding.Enabled = true
	cfg.ContextAssembler.CA3.Compaction.Embedding.Selector = "default"
	cfg.ContextAssembler.CA3.Compaction.Embedding.Provider = "openai"
	cfg.ContextAssembler.CA3.Compaction.Embedding.Model = "text-embedding-3-small"
	cfg.ContextAssembler.CA3.Compaction.Embedding.Timeout = 300 * time.Millisecond
	cfg.ContextAssembler.CA3.Compaction.Reranker.Enabled = true
	cfg.ContextAssembler.CA3.Compaction.Reranker.Governance.Mode = CA3RerankerGovernanceModeEnforce
	cfg.ContextAssembler.CA3.Compaction.Reranker.Governance.RolloutProviderModels = []string{"openai"}
	cfg.ContextAssembler.CA3.Compaction.Reranker.ThresholdProfiles = map[string]float64{
		"openai:text-embedding-3-small": 0.62,
	}
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for invalid reranker rollout_provider_models key")
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

func TestSchedulerAndSubagentConfigDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Scheduler.Enabled {
		t.Fatal("scheduler.enabled = true, want false")
	}
	if cfg.Scheduler.Backend != SchedulerBackendMemory {
		t.Fatalf("scheduler.backend = %q, want memory", cfg.Scheduler.Backend)
	}
	if cfg.Scheduler.LeaseTimeout <= 0 {
		t.Fatalf("scheduler.lease_timeout = %v, want > 0", cfg.Scheduler.LeaseTimeout)
	}
	if cfg.Scheduler.HeartbeatInterval <= 0 {
		t.Fatalf("scheduler.heartbeat_interval = %v, want > 0", cfg.Scheduler.HeartbeatInterval)
	}
	if cfg.Scheduler.HeartbeatInterval >= cfg.Scheduler.LeaseTimeout {
		t.Fatalf("scheduler heartbeat/lease relation invalid: heartbeat=%v lease=%v", cfg.Scheduler.HeartbeatInterval, cfg.Scheduler.LeaseTimeout)
	}
	if cfg.Scheduler.QueueLimit <= 0 || cfg.Scheduler.RetryMaxAttempts <= 0 {
		t.Fatalf("scheduler queue/retry defaults invalid: %#v", cfg.Scheduler)
	}
	if cfg.Scheduler.QoS.Mode != SchedulerQoSModeFIFO {
		t.Fatalf("scheduler.qos.mode = %q, want fifo", cfg.Scheduler.QoS.Mode)
	}
	if cfg.Scheduler.QoS.Fairness.MaxConsecutiveClaimsPerPriority != 3 {
		t.Fatalf(
			"scheduler.qos.fairness.max_consecutive_claims_per_priority = %d, want 3",
			cfg.Scheduler.QoS.Fairness.MaxConsecutiveClaimsPerPriority,
		)
	}
	if cfg.Scheduler.DLQ.Enabled {
		t.Fatal("scheduler.dlq.enabled = true, want false")
	}
	if cfg.Scheduler.Retry.Backoff.Initial <= 0 ||
		cfg.Scheduler.Retry.Backoff.Max < cfg.Scheduler.Retry.Backoff.Initial ||
		cfg.Scheduler.Retry.Backoff.Multiplier < 1 ||
		cfg.Scheduler.Retry.Backoff.JitterRatio < 0 ||
		cfg.Scheduler.Retry.Backoff.JitterRatio > 1 {
		t.Fatalf("scheduler.retry.backoff defaults invalid: %#v", cfg.Scheduler.Retry.Backoff)
	}
	if cfg.Recovery.Enabled {
		t.Fatal("recovery.enabled = true, want false")
	}
	if cfg.Recovery.Backend != RecoveryBackendMemory {
		t.Fatalf("recovery.backend = %q, want memory", cfg.Recovery.Backend)
	}
	if cfg.Recovery.ConflictPolicy != RecoveryConflictPolicyFailFast {
		t.Fatalf("recovery.conflict_policy = %q, want fail_fast", cfg.Recovery.ConflictPolicy)
	}
	if cfg.Subagent.MaxDepth <= 0 || cfg.Subagent.MaxActiveChildren <= 0 || cfg.Subagent.ChildTimeoutBudget <= 0 {
		t.Fatalf("subagent defaults invalid: %#v", cfg.Subagent)
	}
}

func TestSchedulerAndSubagentEnvOverridePrecedence(t *testing.T) {
	t.Setenv("BAYMAX_SCHEDULER_ENABLED", "true")
	t.Setenv("BAYMAX_SCHEDULER_BACKEND", SchedulerBackendFile)
	t.Setenv("BAYMAX_SCHEDULER_PATH", "/tmp/scheduler-state-override.json")
	t.Setenv("BAYMAX_SCHEDULER_LEASE_TIMEOUT", "3s")
	t.Setenv("BAYMAX_SCHEDULER_HEARTBEAT_INTERVAL", "800ms")
	t.Setenv("BAYMAX_SCHEDULER_QUEUE_LIMIT", "2048")
	t.Setenv("BAYMAX_SCHEDULER_RETRY_MAX_ATTEMPTS", "5")
	t.Setenv("BAYMAX_SCHEDULER_QOS_MODE", SchedulerQoSModePrio)
	t.Setenv("BAYMAX_SCHEDULER_QOS_FAIRNESS_MAX_CONSECUTIVE_CLAIMS_PER_PRIORITY", "4")
	t.Setenv("BAYMAX_SCHEDULER_DLQ_ENABLED", "true")
	t.Setenv("BAYMAX_SCHEDULER_RETRY_BACKOFF_ENABLED", "true")
	t.Setenv("BAYMAX_SCHEDULER_RETRY_BACKOFF_INITIAL", "80ms")
	t.Setenv("BAYMAX_SCHEDULER_RETRY_BACKOFF_MAX", "2s")
	t.Setenv("BAYMAX_SCHEDULER_RETRY_BACKOFF_MULTIPLIER", "2.5")
	t.Setenv("BAYMAX_SCHEDULER_RETRY_BACKOFF_JITTER_RATIO", "0.35")
	t.Setenv("BAYMAX_RECOVERY_ENABLED", "true")
	t.Setenv("BAYMAX_RECOVERY_BACKEND", RecoveryBackendFile)
	t.Setenv("BAYMAX_RECOVERY_PATH", "/tmp/recovery-override")
	t.Setenv("BAYMAX_RECOVERY_CONFLICT_POLICY", RecoveryConflictPolicyFailFast)
	t.Setenv("BAYMAX_SUBAGENT_MAX_DEPTH", "7")
	t.Setenv("BAYMAX_SUBAGENT_MAX_ACTIVE_CHILDREN", "11")
	t.Setenv("BAYMAX_SUBAGENT_CHILD_TIMEOUT_BUDGET", "9s")

	file := filepath.Join(t.TempDir(), "runtime.yaml")
	content := `
scheduler:
  enabled: false
  backend: memory
  path: /tmp/scheduler-state-file.json
  lease_timeout: 2s
  heartbeat_interval: 400ms
  queue_limit: 128
  retry_max_attempts: 2
  qos:
    mode: fifo
    fairness:
      max_consecutive_claims_per_priority: 2
  dlq:
    enabled: false
  retry:
    backoff:
      enabled: false
      initial: 40ms
      max: 1s
      multiplier: 2
      jitter_ratio: 0.20
recovery:
  enabled: false
  backend: memory
  path: /tmp/recovery-file
  conflict_policy: fail_fast
subagent:
  max_depth: 4
  max_active_children: 8
  child_timeout_budget: 5s
`
	if err := os.WriteFile(file, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := Load(LoadOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if !cfg.Scheduler.Enabled || cfg.Scheduler.Backend != SchedulerBackendFile {
		t.Fatalf("scheduler env override failed: %#v", cfg.Scheduler)
	}
	if cfg.Scheduler.Path != "/tmp/scheduler-state-override.json" {
		t.Fatalf("scheduler.path = %q, want /tmp/scheduler-state-override.json", cfg.Scheduler.Path)
	}
	if cfg.Scheduler.LeaseTimeout != 3*time.Second || cfg.Scheduler.HeartbeatInterval != 800*time.Millisecond {
		t.Fatalf("scheduler lease/heartbeat override mismatch: %#v", cfg.Scheduler)
	}
	if cfg.Scheduler.QueueLimit != 2048 || cfg.Scheduler.RetryMaxAttempts != 5 {
		t.Fatalf("scheduler queue/retry override mismatch: %#v", cfg.Scheduler)
	}
	if cfg.Scheduler.QoS.Mode != SchedulerQoSModePrio ||
		cfg.Scheduler.QoS.Fairness.MaxConsecutiveClaimsPerPriority != 4 {
		t.Fatalf("scheduler qos override mismatch: %#v", cfg.Scheduler.QoS)
	}
	if !cfg.Scheduler.DLQ.Enabled {
		t.Fatalf("scheduler dlq override mismatch: %#v", cfg.Scheduler.DLQ)
	}
	if cfg.Scheduler.Retry.Backoff.Initial != 80*time.Millisecond ||
		cfg.Scheduler.Retry.Backoff.Max != 2*time.Second ||
		cfg.Scheduler.Retry.Backoff.Multiplier != 2.5 ||
		cfg.Scheduler.Retry.Backoff.JitterRatio != 0.35 {
		t.Fatalf("scheduler retry backoff override mismatch: %#v", cfg.Scheduler.Retry.Backoff)
	}
	if !cfg.Scheduler.Retry.Backoff.Enabled {
		t.Fatalf("scheduler retry backoff enabled override mismatch: %#v", cfg.Scheduler.Retry.Backoff)
	}
	if !cfg.Recovery.Enabled || cfg.Recovery.Backend != RecoveryBackendFile || cfg.Recovery.Path != "/tmp/recovery-override" {
		t.Fatalf("recovery override mismatch: %#v", cfg.Recovery)
	}
	if cfg.Recovery.ConflictPolicy != RecoveryConflictPolicyFailFast {
		t.Fatalf("recovery conflict_policy override mismatch: %#v", cfg.Recovery)
	}
	if cfg.Subagent.MaxDepth != 7 || cfg.Subagent.MaxActiveChildren != 11 || cfg.Subagent.ChildTimeoutBudget != 9*time.Second {
		t.Fatalf("subagent override mismatch: %#v", cfg.Subagent)
	}
}

func TestSchedulerAndSubagentValidationRejectsInvalidValues(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Scheduler.Backend = "db"
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for scheduler.backend")
	}

	cfg = DefaultConfig()
	cfg.Scheduler.Backend = SchedulerBackendFile
	cfg.Scheduler.Path = ""
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for scheduler.path when backend=file")
	}

	cfg = DefaultConfig()
	cfg.Scheduler.HeartbeatInterval = cfg.Scheduler.LeaseTimeout
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error when heartbeat_interval >= lease_timeout")
	}

	cfg = DefaultConfig()
	cfg.Scheduler.QueueLimit = 0
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for scheduler.queue_limit")
	}

	cfg = DefaultConfig()
	cfg.Scheduler.QoS.Mode = "weighted"
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for scheduler.qos.mode")
	}

	cfg = DefaultConfig()
	cfg.Scheduler.QoS.Fairness.MaxConsecutiveClaimsPerPriority = 0
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for scheduler.qos.fairness.max_consecutive_claims_per_priority")
	}

	cfg = DefaultConfig()
	cfg.Scheduler.Retry.Backoff.Enabled = true
	cfg.Scheduler.Retry.Backoff.Initial = 0
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for scheduler.retry.backoff.initial")
	}

	cfg = DefaultConfig()
	cfg.Scheduler.Retry.Backoff.Enabled = true
	cfg.Scheduler.Retry.Backoff.Max = 10 * time.Millisecond
	cfg.Scheduler.Retry.Backoff.Initial = 20 * time.Millisecond
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for scheduler.retry.backoff.max")
	}

	cfg = DefaultConfig()
	cfg.Scheduler.Retry.Backoff.Enabled = true
	cfg.Scheduler.Retry.Backoff.Multiplier = 0.9
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for scheduler.retry.backoff.multiplier")
	}

	cfg = DefaultConfig()
	cfg.Scheduler.Retry.Backoff.Enabled = true
	cfg.Scheduler.Retry.Backoff.JitterRatio = 1.5
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for scheduler.retry.backoff.jitter_ratio")
	}

	cfg = DefaultConfig()
	cfg.Recovery.Backend = "db"
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for recovery.backend")
	}

	cfg = DefaultConfig()
	cfg.Recovery.Backend = RecoveryBackendFile
	cfg.Recovery.Path = ""
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for recovery.path when backend=file")
	}

	cfg = DefaultConfig()
	cfg.Recovery.ConflictPolicy = "merge"
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for recovery.conflict_policy")
	}

	cfg = DefaultConfig()
	cfg.Subagent.MaxDepth = 0
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error for subagent.max_depth")
	}
}
