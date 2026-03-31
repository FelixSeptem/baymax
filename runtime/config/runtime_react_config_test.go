package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimeReactConfigDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.Runtime.React.Enabled {
		t.Fatal("runtime.react.enabled = false, want true")
	}
	if cfg.Runtime.React.MaxIterations != 12 {
		t.Fatalf("runtime.react.max_iterations = %d, want 12", cfg.Runtime.React.MaxIterations)
	}
	if cfg.Runtime.React.ToolCallLimit != 64 {
		t.Fatalf("runtime.react.tool_call_limit = %d, want 64", cfg.Runtime.React.ToolCallLimit)
	}
	if !cfg.Runtime.React.StreamToolDispatchEnabled {
		t.Fatal("runtime.react.stream_tool_dispatch_enabled = false, want true")
	}
	if cfg.Runtime.React.OnBudgetExhausted != RuntimeReactOnBudgetExhaustedFailFast {
		t.Fatalf(
			"runtime.react.on_budget_exhausted = %q, want %q",
			cfg.Runtime.React.OnBudgetExhausted,
			RuntimeReactOnBudgetExhaustedFailFast,
		)
	}
}

func TestRuntimeReactConfigEnvOverridePrecedence(t *testing.T) {
	t.Setenv("BAYMAX_RUNTIME_REACT_ENABLED", "true")
	t.Setenv("BAYMAX_RUNTIME_REACT_MAX_ITERATIONS", "21")
	t.Setenv("BAYMAX_RUNTIME_REACT_TOOL_CALL_LIMIT", "77")
	t.Setenv("BAYMAX_RUNTIME_REACT_STREAM_TOOL_DISPATCH_ENABLED", "false")
	t.Setenv("BAYMAX_RUNTIME_REACT_ON_BUDGET_EXHAUSTED", RuntimeReactOnBudgetExhaustedFailFast)

	file := filepath.Join(t.TempDir(), "runtime.yaml")
	content := `
runtime:
  react:
    enabled: false
    max_iterations: 9
    tool_call_limit: 11
    stream_tool_dispatch_enabled: true
    on_budget_exhausted: fail_fast
`
	if err := os.WriteFile(file, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := Load(LoadOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if !cfg.Runtime.React.Enabled {
		t.Fatal("runtime.react.enabled = false, want true from env")
	}
	if cfg.Runtime.React.MaxIterations != 21 {
		t.Fatalf("runtime.react.max_iterations = %d, want 21 from env", cfg.Runtime.React.MaxIterations)
	}
	if cfg.Runtime.React.ToolCallLimit != 77 {
		t.Fatalf("runtime.react.tool_call_limit = %d, want 77 from env", cfg.Runtime.React.ToolCallLimit)
	}
	if cfg.Runtime.React.StreamToolDispatchEnabled {
		t.Fatal("runtime.react.stream_tool_dispatch_enabled = true, want false from env")
	}
}

func TestRuntimeReactConfigValidationRejectsInvalidValues(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Runtime.React.MaxIterations = 0
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.react.max_iterations") {
		t.Fatalf("expected runtime.react.max_iterations validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.React.ToolCallLimit = 0
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.react.tool_call_limit") {
		t.Fatalf("expected runtime.react.tool_call_limit validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.React.OnBudgetExhausted = "best_effort"
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.react.on_budget_exhausted") {
		t.Fatalf("expected runtime.react.on_budget_exhausted validation error, got %v", err)
	}

	cfg = DefaultConfig()
	cfg.Runtime.React.Enabled = false
	cfg.Runtime.React.StreamToolDispatchEnabled = true
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "runtime.react.stream_tool_dispatch_enabled") {
		t.Fatalf("expected runtime.react.stream_tool_dispatch_enabled compatibility error, got %v", err)
	}
}

func TestRuntimeReactConfigInvalidBoolFailsFast(t *testing.T) {
	t.Setenv("BAYMAX_RUNTIME_REACT_ENABLED", "not-a-bool")
	if _, err := Load(LoadOptions{EnvPrefix: "BAYMAX"}); err == nil || !strings.Contains(err.Error(), "runtime.react.enabled") {
		t.Fatalf("expected strict bool parse error for runtime.react.enabled, got %v", err)
	}

	t.Setenv("BAYMAX_RUNTIME_REACT_ENABLED", "true")
	t.Setenv("BAYMAX_RUNTIME_REACT_STREAM_TOOL_DISPATCH_ENABLED", "not-a-bool")
	if _, err := Load(LoadOptions{EnvPrefix: "BAYMAX"}); err == nil || !strings.Contains(err.Error(), "runtime.react.stream_tool_dispatch_enabled") {
		t.Fatalf("expected strict bool parse error for runtime.react.stream_tool_dispatch_enabled, got %v", err)
	}
}
