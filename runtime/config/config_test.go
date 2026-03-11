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
