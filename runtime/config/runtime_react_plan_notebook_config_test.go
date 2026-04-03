package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimeReactPlanNotebookConfigDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Runtime.React.PlanNotebook.Enabled {
		t.Fatal("runtime.react.plan_notebook.enabled = true, want false")
	}
	if cfg.Runtime.React.PlanNotebook.MaxHistory != 32 {
		t.Fatalf("runtime.react.plan_notebook.max_history = %d, want 32", cfg.Runtime.React.PlanNotebook.MaxHistory)
	}
	if cfg.Runtime.React.PlanNotebook.OnRecoverConflict != RuntimeReactPlanNotebookRecoverConflictReject {
		t.Fatalf(
			"runtime.react.plan_notebook.on_recover_conflict = %q, want %q",
			cfg.Runtime.React.PlanNotebook.OnRecoverConflict,
			RuntimeReactPlanNotebookRecoverConflictReject,
		)
	}
	if cfg.Runtime.React.PlanChangeHook.Enabled {
		t.Fatal("runtime.react.plan_change_hook.enabled = true, want false")
	}
	if cfg.Runtime.React.PlanChangeHook.FailMode != RuntimeReactPlanChangeHookFailModeFailFast {
		t.Fatalf(
			"runtime.react.plan_change_hook.fail_mode = %q, want %q",
			cfg.Runtime.React.PlanChangeHook.FailMode,
			RuntimeReactPlanChangeHookFailModeFailFast,
		)
	}
	if cfg.Runtime.React.PlanChangeHook.TimeoutMs != 2000 {
		t.Fatalf("runtime.react.plan_change_hook.timeout_ms = %d, want 2000", cfg.Runtime.React.PlanChangeHook.TimeoutMs)
	}
}

func TestRuntimeReactPlanNotebookConfigEnvOverridePrecedence(t *testing.T) {
	t.Setenv("BAYMAX_RUNTIME_REACT_PLAN_NOTEBOOK_ENABLED", "true")
	t.Setenv("BAYMAX_RUNTIME_REACT_PLAN_NOTEBOOK_MAX_HISTORY", "77")
	t.Setenv("BAYMAX_RUNTIME_REACT_PLAN_NOTEBOOK_ON_RECOVER_CONFLICT", RuntimeReactPlanNotebookRecoverConflictPreferLatest)
	t.Setenv("BAYMAX_RUNTIME_REACT_PLAN_CHANGE_HOOK_ENABLED", "true")
	t.Setenv("BAYMAX_RUNTIME_REACT_PLAN_CHANGE_HOOK_FAIL_MODE", RuntimeReactPlanChangeHookFailModeDegrade)
	t.Setenv("BAYMAX_RUNTIME_REACT_PLAN_CHANGE_HOOK_TIMEOUT_MS", "3500")

	file := filepath.Join(t.TempDir(), "runtime.yaml")
	content := `
runtime:
  react:
    plan_notebook:
      enabled: false
      max_history: 9
      on_recover_conflict: reject
    plan_change_hook:
      enabled: false
      fail_mode: fail_fast
      timeout_ms: 1200
`
	if err := os.WriteFile(file, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := Load(LoadOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if !cfg.Runtime.React.PlanNotebook.Enabled {
		t.Fatal("runtime.react.plan_notebook.enabled = false, want true from env")
	}
	if cfg.Runtime.React.PlanNotebook.MaxHistory != 77 {
		t.Fatalf("runtime.react.plan_notebook.max_history = %d, want 77 from env", cfg.Runtime.React.PlanNotebook.MaxHistory)
	}
	if cfg.Runtime.React.PlanNotebook.OnRecoverConflict != RuntimeReactPlanNotebookRecoverConflictPreferLatest {
		t.Fatalf(
			"runtime.react.plan_notebook.on_recover_conflict = %q, want %q from env",
			cfg.Runtime.React.PlanNotebook.OnRecoverConflict,
			RuntimeReactPlanNotebookRecoverConflictPreferLatest,
		)
	}
	if !cfg.Runtime.React.PlanChangeHook.Enabled {
		t.Fatal("runtime.react.plan_change_hook.enabled = false, want true from env")
	}
	if cfg.Runtime.React.PlanChangeHook.FailMode != RuntimeReactPlanChangeHookFailModeDegrade {
		t.Fatalf(
			"runtime.react.plan_change_hook.fail_mode = %q, want %q from env",
			cfg.Runtime.React.PlanChangeHook.FailMode,
			RuntimeReactPlanChangeHookFailModeDegrade,
		)
	}
	if cfg.Runtime.React.PlanChangeHook.TimeoutMs != 3500 {
		t.Fatalf("runtime.react.plan_change_hook.timeout_ms = %d, want 3500 from env", cfg.Runtime.React.PlanChangeHook.TimeoutMs)
	}
}

func TestRuntimeReactPlanNotebookConfigValidationRejectsInvalidValues(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Runtime.React.PlanNotebook.Enabled = true
	cfg.Runtime.React.PlanNotebook.MaxHistory = 0
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.react.plan_notebook.max_history") {
		t.Fatalf("expected runtime.react.plan_notebook.max_history validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.React.PlanNotebook.Enabled = true
	cfg.Runtime.React.PlanNotebook.OnRecoverConflict = "best_effort"
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.react.plan_notebook.on_recover_conflict") {
		t.Fatalf("expected runtime.react.plan_notebook.on_recover_conflict validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.React.PlanNotebook.Enabled = true
	cfg.Runtime.React.PlanChangeHook.Enabled = true
	cfg.Runtime.React.PlanChangeHook.FailMode = "warn"
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.react.plan_change_hook.fail_mode") {
		t.Fatalf("expected runtime.react.plan_change_hook.fail_mode validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.React.PlanNotebook.Enabled = true
	cfg.Runtime.React.PlanChangeHook.TimeoutMs = 0
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.react.plan_change_hook.timeout_ms") {
		t.Fatalf("expected runtime.react.plan_change_hook.timeout_ms validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.React.PlanNotebook.Enabled = false
	cfg.Runtime.React.PlanChangeHook.Enabled = true
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.react.plan_change_hook.enabled requires runtime.react.plan_notebook.enabled=true") {
		t.Fatalf("expected runtime.react.plan_change_hook.enabled compatibility error, got %v", err)
	}
}

func TestRuntimeReactPlanNotebookConfigInvalidPrimitiveFailsFast(t *testing.T) {
	t.Setenv("BAYMAX_RUNTIME_REACT_PLAN_NOTEBOOK_ENABLED", "not-a-bool")
	if _, err := Load(LoadOptions{EnvPrefix: "BAYMAX"}); err == nil || !strings.Contains(err.Error(), "runtime.react.plan_notebook.enabled") {
		t.Fatalf("expected strict bool parse error for runtime.react.plan_notebook.enabled, got %v", err)
	}

	t.Setenv("BAYMAX_RUNTIME_REACT_PLAN_NOTEBOOK_ENABLED", "true")
	t.Setenv("BAYMAX_RUNTIME_REACT_PLAN_NOTEBOOK_MAX_HISTORY", "not-int")
	if _, err := Load(LoadOptions{EnvPrefix: "BAYMAX"}); err == nil || !strings.Contains(err.Error(), "runtime.react.plan_notebook.max_history") {
		t.Fatalf("expected strict int parse error for runtime.react.plan_notebook.max_history, got %v", err)
	}

	t.Setenv("BAYMAX_RUNTIME_REACT_PLAN_NOTEBOOK_MAX_HISTORY", "32")
	t.Setenv("BAYMAX_RUNTIME_REACT_PLAN_CHANGE_HOOK_TIMEOUT_MS", "not-int")
	if _, err := Load(LoadOptions{EnvPrefix: "BAYMAX"}); err == nil || !strings.Contains(err.Error(), "runtime.react.plan_change_hook.timeout_ms") {
		t.Fatalf("expected strict int parse error for runtime.react.plan_change_hook.timeout_ms, got %v", err)
	}
}
