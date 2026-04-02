package config

import (
	"path/filepath"
	"testing"
	"time"
)

func TestManagerRuntimeEvalInvalidReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
runtime:
  eval:
    agent:
      enabled: true
      suite_id: eval-suite
      task_success_threshold: 0.9
      tool_correctness_threshold: 0.9
      deny_intercept_accuracy_threshold: 0.95
      cost_budget_threshold: 1.0
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
reload:
  enabled: true
  debounce: 20ms
`)

	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX_A61_TEST", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig()
	if before.Runtime.Eval.Execution.Mode != RuntimeEvalExecutionModeLocal {
		t.Fatalf("before runtime.eval.execution.mode = %q, want %q", before.Runtime.Eval.Execution.Mode, RuntimeEvalExecutionModeLocal)
	}

	writeConfig(t, file, `
runtime:
  eval:
    agent:
      enabled: true
      suite_id: eval-suite
      task_success_threshold: 0.9
      tool_correctness_threshold: 0.9
      deny_intercept_accuracy_threshold: 0.95
      cost_budget_threshold: 1.0
      latency_budget_threshold: 2s
    execution:
      mode: managed
      shard:
        total: 1
      retry:
        max_attempts: 2
      resume:
        enabled: true
        max_count: 3
      aggregation: weighted_mean
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)

	after := mgr.EffectiveConfig()
	if after.Runtime.Eval.Execution.Mode != before.Runtime.Eval.Execution.Mode {
		t.Fatalf(
			"invalid runtime.eval.execution.mode should rollback, got %q want %q",
			after.Runtime.Eval.Execution.Mode,
			before.Runtime.Eval.Execution.Mode,
		)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
	if got := reloads[0].Error; got == "" {
		t.Fatalf("expected reload error for invalid runtime.eval.execution.mode")
	}
}
