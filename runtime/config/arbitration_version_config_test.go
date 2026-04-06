package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimeArbitrationVersionConfigDefaults(t *testing.T) {
	cfg := DefaultConfig()
	got := cfg.Runtime.Arbitration.Version
	if !got.Enabled {
		t.Fatal("runtime.arbitration.version.enabled = false, want true")
	}
	if got.Default != RuntimeArbitrationRuleVersionExplainabilityV1 {
		t.Fatalf("runtime.arbitration.version.default = %q, want %q", got.Default, RuntimeArbitrationRuleVersionExplainabilityV1)
	}
	if got.CompatWindow != 1 {
		t.Fatalf("runtime.arbitration.version.compat_window = %d, want 1", got.CompatWindow)
	}
	if got.OnUnsupported != RuntimeArbitrationVersionUnsupportedPolicyFailFast {
		t.Fatalf(
			"runtime.arbitration.version.on_unsupported = %q, want %q",
			got.OnUnsupported,
			RuntimeArbitrationVersionUnsupportedPolicyFailFast,
		)
	}
	if got.OnMismatch != RuntimeArbitrationVersionMismatchPolicyFailFast {
		t.Fatalf(
			"runtime.arbitration.version.on_mismatch = %q, want %q",
			got.OnMismatch,
			RuntimeArbitrationVersionMismatchPolicyFailFast,
		)
	}
}

func TestRuntimeArbitrationVersionConfigEnvOverridePrecedence(t *testing.T) {
	t.Setenv("BAYMAX_RUNTIME_ARBITRATION_VERSION_DEFAULT", RuntimeArbitrationRuleVersionPrimaryReasonV1)
	t.Setenv("BAYMAX_RUNTIME_ARBITRATION_VERSION_COMPAT_WINDOW", "0")
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	content := `
runtime:
  arbitration:
    version:
      enabled: true
      default: a49.v1
      compat_window: 1
      on_unsupported: fail_fast
      on_mismatch: fail_fast
`
	if err := os.WriteFile(file, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := Load(LoadOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Runtime.Arbitration.Version.Default != RuntimeArbitrationRuleVersionPrimaryReasonV1 {
		t.Fatalf("runtime.arbitration.version.default = %q, want %q", cfg.Runtime.Arbitration.Version.Default, RuntimeArbitrationRuleVersionPrimaryReasonV1)
	}
	if cfg.Runtime.Arbitration.Version.CompatWindow != 0 {
		t.Fatalf("runtime.arbitration.version.compat_window = %d, want 0", cfg.Runtime.Arbitration.Version.CompatWindow)
	}
}

func TestRuntimeArbitrationVersionConfigInvalidBoolFailsFast(t *testing.T) {
	t.Setenv("BAYMAX_RUNTIME_ARBITRATION_VERSION_ENABLED", "definitely-not-bool")
	_, err := Load(LoadOptions{EnvPrefix: "BAYMAX"})
	if err == nil {
		t.Fatal("expected runtime.arbitration.version.enabled invalid bool error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "runtime.arbitration.version.enabled") {
		t.Fatalf("unexpected error: %v", err)
	}
}
