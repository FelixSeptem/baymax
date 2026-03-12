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
}
