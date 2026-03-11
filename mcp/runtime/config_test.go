package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

func TestLoadConfigPrecedenceEnvOverFileOverDefault(t *testing.T) {
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

	cfg, err := LoadConfig(LoadOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg.MCP.Profiles["default"].Retry != 7 {
		t.Fatalf("retry = %d, want 7", cfg.MCP.Profiles["default"].Retry)
	}
	if cfg.MCP.Profiles["default"].CallTimeout <= 0 {
		t.Fatalf("call_timeout should use default fallback, got %v", cfg.MCP.Profiles["default"].CallTimeout)
	}
}

func TestLoadConfigAllowsCustomProfileEnum(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	content := `
mcp:
  active_profile: burst
  profiles:
    burst:
      call_timeout: 6s
      retry: 1
      backoff: 10ms
      queue_size: 64
      backpressure: reject
      read_pool_size: 8
      write_pool_size: 2
`
	if err := os.WriteFile(file, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := LoadConfig(LoadOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg.MCP.ActiveProfile != "burst" {
		t.Fatalf("active profile = %q, want burst", cfg.MCP.ActiveProfile)
	}
	if _, ok := cfg.MCP.Profiles["burst"]; !ok {
		t.Fatalf("custom profile not loaded")
	}
}

func TestValidateConfigFailFast(t *testing.T) {
	cfg := DefaultConfig()
	p := cfg.MCP.Profiles[string(ProfileDefault)]
	p.Backpressure = "invalid"
	cfg.MCP.Profiles[string(ProfileDefault)] = p
	if err := ValidateConfig(cfg); err == nil {
		t.Fatalf("expected validation error, got nil")
	}
}

func TestResolvePolicyWithConfig(t *testing.T) {
	cfg := DefaultConfig()
	override := &types.MCPRuntimePolicy{Retry: 9, Backoff: 30 * time.Millisecond}
	p, err := ResolvePolicyWithConfig(cfg, ProfileDefault, override)
	if err != nil {
		t.Fatalf("ResolvePolicyWithConfig failed: %v", err)
	}
	if p.Retry != 9 {
		t.Fatalf("retry = %d, want 9", p.Retry)
	}
	if p.Backoff != 30*time.Millisecond {
		t.Fatalf("backoff = %v, want 30ms", p.Backoff)
	}
}
