package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRuntimeEvalConfigDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Runtime.Eval.Agent.Enabled {
		t.Fatal("runtime.eval.agent.enabled = true, want false")
	}
	if strings.TrimSpace(cfg.Runtime.Eval.Agent.SuiteID) == "" {
		t.Fatal("runtime.eval.agent.suite_id should not be empty by default")
	}
	if cfg.Runtime.Eval.Agent.TaskSuccessThreshold < 0 || cfg.Runtime.Eval.Agent.TaskSuccessThreshold > 1 {
		t.Fatalf("runtime.eval.agent.task_success_threshold = %v, want in [0,1]", cfg.Runtime.Eval.Agent.TaskSuccessThreshold)
	}
	if cfg.Runtime.Eval.Agent.ToolCorrectnessThreshold < 0 || cfg.Runtime.Eval.Agent.ToolCorrectnessThreshold > 1 {
		t.Fatalf("runtime.eval.agent.tool_correctness_threshold = %v, want in [0,1]", cfg.Runtime.Eval.Agent.ToolCorrectnessThreshold)
	}
	if cfg.Runtime.Eval.Agent.DenyInterceptAccuracyThreshold < 0 || cfg.Runtime.Eval.Agent.DenyInterceptAccuracyThreshold > 1 {
		t.Fatalf("runtime.eval.agent.deny_intercept_accuracy_threshold = %v, want in [0,1]", cfg.Runtime.Eval.Agent.DenyInterceptAccuracyThreshold)
	}
	if cfg.Runtime.Eval.Agent.CostBudgetThreshold <= 0 {
		t.Fatalf("runtime.eval.agent.cost_budget_threshold = %v, want >0", cfg.Runtime.Eval.Agent.CostBudgetThreshold)
	}
	if cfg.Runtime.Eval.Agent.LatencyBudgetThreshold <= 0 {
		t.Fatalf("runtime.eval.agent.latency_budget_threshold = %v, want >0", cfg.Runtime.Eval.Agent.LatencyBudgetThreshold)
	}
	if cfg.Runtime.Eval.Execution.Mode != RuntimeEvalExecutionModeLocal {
		t.Fatalf("runtime.eval.execution.mode = %q, want %q", cfg.Runtime.Eval.Execution.Mode, RuntimeEvalExecutionModeLocal)
	}
	if cfg.Runtime.Eval.Execution.Shard.Total <= 0 {
		t.Fatalf("runtime.eval.execution.shard.total = %d, want >0", cfg.Runtime.Eval.Execution.Shard.Total)
	}
	if cfg.Runtime.Eval.Execution.Retry.MaxAttempts <= 0 {
		t.Fatalf("runtime.eval.execution.retry.max_attempts = %d, want >0", cfg.Runtime.Eval.Execution.Retry.MaxAttempts)
	}
	if !cfg.Runtime.Eval.Execution.Resume.Enabled {
		t.Fatal("runtime.eval.execution.resume.enabled = false, want true")
	}
	if cfg.Runtime.Eval.Execution.Aggregation != RuntimeEvalExecutionAggregationWeightedMean {
		t.Fatalf(
			"runtime.eval.execution.aggregation = %q, want %q",
			cfg.Runtime.Eval.Execution.Aggregation,
			RuntimeEvalExecutionAggregationWeightedMean,
		)
	}
}

func TestRuntimeEvalConfigEnvOverridePrecedence(t *testing.T) {
	t.Setenv("BAYMAX_RUNTIME_EVAL_AGENT_ENABLED", "true")
	t.Setenv("BAYMAX_RUNTIME_EVAL_AGENT_SUITE_ID", "env-suite")
	t.Setenv("BAYMAX_RUNTIME_EVAL_AGENT_TASK_SUCCESS_THRESHOLD", "0.81")
	t.Setenv("BAYMAX_RUNTIME_EVAL_AGENT_TOOL_CORRECTNESS_THRESHOLD", "0.82")
	t.Setenv("BAYMAX_RUNTIME_EVAL_AGENT_DENY_INTERCEPT_ACCURACY_THRESHOLD", "0.83")
	t.Setenv("BAYMAX_RUNTIME_EVAL_AGENT_COST_BUDGET_THRESHOLD", "1.2")
	t.Setenv("BAYMAX_RUNTIME_EVAL_AGENT_LATENCY_BUDGET_THRESHOLD", "3500ms")
	t.Setenv("BAYMAX_RUNTIME_EVAL_EXECUTION_MODE", RuntimeEvalExecutionModeDistributed)
	t.Setenv("BAYMAX_RUNTIME_EVAL_EXECUTION_SHARD_TOTAL", "8")
	t.Setenv("BAYMAX_RUNTIME_EVAL_EXECUTION_RETRY_MAX_ATTEMPTS", "4")
	t.Setenv("BAYMAX_RUNTIME_EVAL_EXECUTION_RESUME_ENABLED", "false")
	t.Setenv("BAYMAX_RUNTIME_EVAL_EXECUTION_RESUME_MAX_COUNT", "5")
	t.Setenv("BAYMAX_RUNTIME_EVAL_EXECUTION_AGGREGATION", RuntimeEvalExecutionAggregationWorstCase)

	file := filepath.Join(t.TempDir(), "runtime.yaml")
	content := `
runtime:
  eval:
    agent:
      enabled: false
      suite_id: file-suite
      task_success_threshold: 0.7
      tool_correctness_threshold: 0.7
      deny_intercept_accuracy_threshold: 0.7
      cost_budget_threshold: 0.9
      latency_budget_threshold: 2s
    execution:
      mode: local
      shard:
        total: 1
      retry:
        max_attempts: 2
      resume:
        enabled: true
        max_count: 3
      aggregation: weighted_mean
`
	if err := os.WriteFile(file, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := Load(LoadOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if !cfg.Runtime.Eval.Agent.Enabled {
		t.Fatal("runtime.eval.agent.enabled = false, want true from env")
	}
	if cfg.Runtime.Eval.Agent.SuiteID != "env-suite" {
		t.Fatalf("runtime.eval.agent.suite_id = %q, want env-suite", cfg.Runtime.Eval.Agent.SuiteID)
	}
	if cfg.Runtime.Eval.Agent.TaskSuccessThreshold != 0.81 {
		t.Fatalf("runtime.eval.agent.task_success_threshold = %v, want 0.81", cfg.Runtime.Eval.Agent.TaskSuccessThreshold)
	}
	if cfg.Runtime.Eval.Agent.ToolCorrectnessThreshold != 0.82 {
		t.Fatalf("runtime.eval.agent.tool_correctness_threshold = %v, want 0.82", cfg.Runtime.Eval.Agent.ToolCorrectnessThreshold)
	}
	if cfg.Runtime.Eval.Agent.DenyInterceptAccuracyThreshold != 0.83 {
		t.Fatalf("runtime.eval.agent.deny_intercept_accuracy_threshold = %v, want 0.83", cfg.Runtime.Eval.Agent.DenyInterceptAccuracyThreshold)
	}
	if cfg.Runtime.Eval.Agent.CostBudgetThreshold != 1.2 {
		t.Fatalf("runtime.eval.agent.cost_budget_threshold = %v, want 1.2", cfg.Runtime.Eval.Agent.CostBudgetThreshold)
	}
	if cfg.Runtime.Eval.Agent.LatencyBudgetThreshold != 3500*time.Millisecond {
		t.Fatalf("runtime.eval.agent.latency_budget_threshold = %v, want 3.5s", cfg.Runtime.Eval.Agent.LatencyBudgetThreshold)
	}
	if cfg.Runtime.Eval.Execution.Mode != RuntimeEvalExecutionModeDistributed {
		t.Fatalf("runtime.eval.execution.mode = %q, want %q", cfg.Runtime.Eval.Execution.Mode, RuntimeEvalExecutionModeDistributed)
	}
	if cfg.Runtime.Eval.Execution.Shard.Total != 8 {
		t.Fatalf("runtime.eval.execution.shard.total = %d, want 8", cfg.Runtime.Eval.Execution.Shard.Total)
	}
	if cfg.Runtime.Eval.Execution.Retry.MaxAttempts != 4 {
		t.Fatalf("runtime.eval.execution.retry.max_attempts = %d, want 4", cfg.Runtime.Eval.Execution.Retry.MaxAttempts)
	}
	if cfg.Runtime.Eval.Execution.Resume.Enabled {
		t.Fatal("runtime.eval.execution.resume.enabled = true, want false from env")
	}
	if cfg.Runtime.Eval.Execution.Resume.MaxCount != 5 {
		t.Fatalf("runtime.eval.execution.resume.max_count = %d, want 5", cfg.Runtime.Eval.Execution.Resume.MaxCount)
	}
	if cfg.Runtime.Eval.Execution.Aggregation != RuntimeEvalExecutionAggregationWorstCase {
		t.Fatalf(
			"runtime.eval.execution.aggregation = %q, want %q",
			cfg.Runtime.Eval.Execution.Aggregation,
			RuntimeEvalExecutionAggregationWorstCase,
		)
	}
}

func TestRuntimeEvalConfigValidationRejectsInvalidValues(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Runtime.Eval.Agent.TaskSuccessThreshold = 1.2
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.eval.agent.task_success_threshold") {
		t.Fatalf("expected runtime.eval.agent.task_success_threshold validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Eval.Agent.ToolCorrectnessThreshold = -0.1
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.eval.agent.tool_correctness_threshold") {
		t.Fatalf("expected runtime.eval.agent.tool_correctness_threshold validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Eval.Agent.DenyInterceptAccuracyThreshold = 1.1
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.eval.agent.deny_intercept_accuracy_threshold") {
		t.Fatalf("expected runtime.eval.agent.deny_intercept_accuracy_threshold validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Eval.Agent.CostBudgetThreshold = 0
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.eval.agent.cost_budget_threshold") {
		t.Fatalf("expected runtime.eval.agent.cost_budget_threshold validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Eval.Agent.LatencyBudgetThreshold = 0
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.eval.agent.latency_budget_threshold") {
		t.Fatalf("expected runtime.eval.agent.latency_budget_threshold validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Eval.Execution.Mode = "managed"
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.eval.execution.mode") {
		t.Fatalf("expected runtime.eval.execution.mode validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Eval.Execution.Shard.Total = 0
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.eval.execution.shard.total") {
		t.Fatalf("expected runtime.eval.execution.shard.total validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Eval.Execution.Retry.MaxAttempts = 0
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.eval.execution.retry.max_attempts") {
		t.Fatalf("expected runtime.eval.execution.retry.max_attempts validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Eval.Execution.Resume.MaxCount = -1
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.eval.execution.resume.max_count") {
		t.Fatalf("expected runtime.eval.execution.resume.max_count validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.Eval.Execution.Aggregation = "sum"
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.eval.execution.aggregation") {
		t.Fatalf("expected runtime.eval.execution.aggregation validation error, got %v", err)
	}
}

func TestRuntimeEvalConfigInvalidBoolFailsFast(t *testing.T) {
	t.Setenv("BAYMAX_RUNTIME_EVAL_AGENT_ENABLED", "not-a-bool")
	if _, err := Load(LoadOptions{EnvPrefix: "BAYMAX"}); err == nil || !strings.Contains(err.Error(), "runtime.eval.agent.enabled") {
		t.Fatalf("expected strict bool parse error for runtime.eval.agent.enabled, got %v", err)
	}

	t.Setenv("BAYMAX_RUNTIME_EVAL_AGENT_ENABLED", "false")
	t.Setenv("BAYMAX_RUNTIME_EVAL_EXECUTION_RESUME_ENABLED", "not-a-bool")
	if _, err := Load(LoadOptions{EnvPrefix: "BAYMAX"}); err == nil || !strings.Contains(err.Error(), "runtime.eval.execution.resume.enabled") {
		t.Fatalf("expected strict bool parse error for runtime.eval.execution.resume.enabled, got %v", err)
	}
}
