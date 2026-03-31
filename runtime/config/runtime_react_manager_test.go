package config

import (
	"path/filepath"
	"testing"
	"time"
)

func TestManagerRuntimeReactInvalidReloadRollsBack(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
runtime:
  react:
    enabled: true
    max_iterations: 12
    tool_call_limit: 64
    stream_tool_dispatch_enabled: true
    on_budget_exhausted: fail_fast
reload:
  enabled: true
  debounce: 20ms
`)

	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX", EnableHotReload: true})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	before := mgr.EffectiveConfig().Runtime.React.ToolCallLimit
	if before != 64 {
		t.Fatalf("before runtime.react.tool_call_limit = %d, want 64", before)
	}

	writeConfig(t, file, `
runtime:
  react:
    enabled: true
    max_iterations: 12
    tool_call_limit: 0
    stream_tool_dispatch_enabled: true
    on_budget_exhausted: fail_fast
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	afterLimit := mgr.EffectiveConfig().Runtime.React.ToolCallLimit
	if afterLimit != before {
		t.Fatalf("invalid runtime.react.tool_call_limit should rollback, got %d want %d", afterLimit, before)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) == 0 || reloads[0].Success {
		t.Fatalf("expected failed reload record, got %#v", reloads)
	}

	writeConfig(t, file, `
runtime:
  react:
    enabled: true
    max_iterations: 12
    tool_call_limit: 64
    stream_tool_dispatch_enabled: true
    on_budget_exhausted: best_effort
reload:
  enabled: true
  debounce: 20ms
`)
	time.Sleep(250 * time.Millisecond)
	afterEnum := mgr.EffectiveConfig().Runtime.React.OnBudgetExhausted
	if afterEnum != RuntimeReactOnBudgetExhaustedFailFast {
		t.Fatalf(
			"invalid runtime.react.on_budget_exhausted should rollback, got %q want %q",
			afterEnum,
			RuntimeReactOnBudgetExhaustedFailFast,
		)
	}
}
