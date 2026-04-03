package config

import (
	"path/filepath"
	"testing"
	"time"
)

func TestManagerRuntimeReactPlanNotebookInvalidReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
runtime:
  react:
    plan_notebook:
      enabled: true
      max_history: 64
      on_recover_conflict: reject
    plan_change_hook:
      enabled: true
      fail_mode: fail_fast
      timeout_ms: 1500
reload:
  enabled: true
  debounce: 20ms
`)

	mgr, err := NewManager(ManagerOptions{
		FilePath:        file,
		EnvPrefix:       "BAYMAX_A67_PLAN_NOTEBOOK_TEST",
		EnableHotReload: true,
	})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().Runtime.React
	if !before.PlanNotebook.Enabled || !before.PlanChangeHook.Enabled {
		t.Fatalf("expected notebook/hook enabled before reload, got %#v", before)
	}

	writeConfig(t, file, `
runtime:
  react:
    plan_notebook:
      enabled: true
      max_history: 64
      on_recover_conflict: reject
    plan_change_hook:
      enabled: true
      fail_mode: best_effort
      timeout_ms: 1500
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)

	after := mgr.EffectiveConfig().Runtime.React
	if after.PlanChangeHook.FailMode != before.PlanChangeHook.FailMode {
		t.Fatalf(
			"invalid runtime.react.plan_change_hook.fail_mode should rollback, got %q want %q",
			after.PlanChangeHook.FailMode,
			before.PlanChangeHook.FailMode,
		)
	}
	if after.PlanNotebook.MaxHistory != before.PlanNotebook.MaxHistory {
		t.Fatalf(
			"reload should rollback atomically for plan_notebook + plan_change_hook, max_history=%d want %d",
			after.PlanNotebook.MaxHistory,
			before.PlanNotebook.MaxHistory,
		)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
}

func TestManagerRuntimeReactPlanNotebookInvalidReloadRollsBackWithEnvPrecedence(t *testing.T) {
	t.Setenv("BAYMAX_A67_PLAN_NOTEBOOK_ENV_TEST_RUNTIME_REACT_PLAN_NOTEBOOK_MAX_HISTORY", "99")
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
runtime:
  react:
    plan_notebook:
      enabled: true
      max_history: 48
      on_recover_conflict: reject
    plan_change_hook:
      enabled: true
      fail_mode: fail_fast
      timeout_ms: 1200
reload:
  enabled: true
  debounce: 20ms
`)

	mgr, err := NewManager(ManagerOptions{
		FilePath:        file,
		EnvPrefix:       "BAYMAX_A67_PLAN_NOTEBOOK_ENV_TEST",
		EnableHotReload: true,
	})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().Runtime.React
	if before.PlanNotebook.MaxHistory != 99 {
		t.Fatalf("env precedence max_history=%d, want 99", before.PlanNotebook.MaxHistory)
	}

	writeConfig(t, file, `
runtime:
  react:
    plan_notebook:
      enabled: true
      max_history: 48
      on_recover_conflict: reject
    plan_change_hook:
      enabled: true
      fail_mode: fail_fast
      timeout_ms: 0
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)

	after := mgr.EffectiveConfig().Runtime.React
	if after.PlanNotebook.MaxHistory != 99 {
		t.Fatalf("env-derived max_history should remain 99 after failed reload, got %d", after.PlanNotebook.MaxHistory)
	}
	if after.PlanChangeHook.TimeoutMs != before.PlanChangeHook.TimeoutMs {
		t.Fatalf(
			"invalid runtime.react.plan_change_hook.timeout_ms should rollback, got %d want %d",
			after.PlanChangeHook.TimeoutMs,
			before.PlanChangeHook.TimeoutMs,
		)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}
}
