package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSecuritySandboxExternalBackendDefaultsToDisabled(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Security.Sandbox.Executor.ExternalBackend != SecuritySandboxExternalBackendDisabled {
		t.Fatalf(
			"security.sandbox.executor.external_backend = %q, want disabled",
			cfg.Security.Sandbox.Executor.ExternalBackend,
		)
	}
	if _, _, ok := ResolveSandboxExternalBackend(cfg.Security.Sandbox); ok {
		t.Fatal("ResolveSandboxExternalBackend reported enabled by default, want disabled")
	}
	if err := Validate(cfg); err != nil {
		t.Fatalf("default config must stay valid with external backend disabled: %v", err)
	}
}

func TestSecuritySandboxExternalBackendAcceptsMappedCanonicalProfile(t *testing.T) {
	cases := []struct {
		external  string
		canonical string
	}{
		{SecuritySandboxExternalBackendE2B, SecuritySandboxBackendOCIRuntime},
		{SecuritySandboxExternalBackendOpenSandbox, SecuritySandboxBackendOCIRuntime},
		{SecuritySandboxExternalBackendAnthropicSandboxRuntime, SecuritySandboxBackendOCIRuntime},
	}
	for _, tc := range cases {
		t.Run(tc.external, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Security.Sandbox.Executor.Backend = tc.canonical
			cfg.Security.Sandbox.Executor.ExternalBackend = tc.external
			if err := Validate(cfg); err != nil {
				t.Fatalf("Validate rejected %s -> %s: %v", tc.external, tc.canonical, err)
			}
			external, canonical, ok := ResolveSandboxExternalBackend(cfg.Security.Sandbox)
			if !ok {
				t.Fatalf("ResolveSandboxExternalBackend reported disabled for %s", tc.external)
			}
			if external != tc.external || canonical != tc.canonical {
				t.Fatalf(
					"ResolveSandboxExternalBackend = (%q, %q), want (%q, %q)",
					external, canonical, tc.external, tc.canonical,
				)
			}
		})
	}
}

func TestSecuritySandboxExternalBackendRejectsCanonicalMismatch(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Security.Sandbox.Executor.Backend = SecuritySandboxBackendWindowsJob
	cfg.Security.Sandbox.Executor.ExternalBackend = SecuritySandboxExternalBackendE2B
	err := Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "security.sandbox.executor.external_backend") {
		t.Fatalf("expected external_backend mapping error, got %v", err)
	}
	if !strings.Contains(err.Error(), SecuritySandboxBackendOCIRuntime) {
		t.Fatalf("mapping error should name the required canonical backend, got %v", err)
	}
}

func TestSecuritySandboxExternalBackendRejectsUnknownRuntime(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Security.Sandbox.Executor.Backend = SecuritySandboxBackendOCIRuntime
	cfg.Security.Sandbox.Executor.ExternalBackend = "firecracker"
	err := Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "not a registered external runtime") {
		t.Fatalf("expected unregistered external runtime error, got %v", err)
	}
}

func TestSecuritySandboxExternalBackendLocalAcceptsAnyCanonicalProfile(t *testing.T) {
	for _, canonical := range []string{
		SecuritySandboxBackendOCIRuntime,
		SecuritySandboxBackendLinuxBwrap,
		SecuritySandboxBackendLinuxNSJail,
		SecuritySandboxBackendWindowsJob,
	} {
		t.Run(canonical, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Security.Sandbox.Executor.Backend = canonical
			cfg.Security.Sandbox.Executor.ExternalBackend = SecuritySandboxExternalBackendLocal
			if err := Validate(cfg); err != nil {
				t.Fatalf("local onboarding backend rejected for %s: %v", canonical, err)
			}
			_, resolved, ok := ResolveSandboxExternalBackend(cfg.Security.Sandbox)
			if !ok || resolved != canonical {
				t.Fatalf("ResolveSandboxExternalBackend = (%q, %v), want %q", resolved, ok, canonical)
			}
		})
	}
}

func TestSecuritySandboxExternalBackendEnvOverride(t *testing.T) {
	t.Setenv("BAYMAX_SECURITY_SANDBOX_EXECUTOR_BACKEND", SecuritySandboxBackendOCIRuntime)
	t.Setenv("BAYMAX_SECURITY_SANDBOX_EXECUTOR_EXTERNAL_BACKEND", SecuritySandboxExternalBackendE2B)
	t.Setenv("BAYMAX_SECURITY_SANDBOX_EXECUTOR_EXTERNAL_BASE_URL", "http://127.0.0.1:18080")

	file := filepath.Join(t.TempDir(), "runtime.yaml")
	content := `
security:
  sandbox:
    executor:
      backend: windows_job
`
	if err := os.WriteFile(file, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := Load(LoadOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Security.Sandbox.Executor.ExternalBackend != SecuritySandboxExternalBackendE2B {
		t.Fatalf(
			"security.sandbox.executor.external_backend = %q, want %q",
			cfg.Security.Sandbox.Executor.ExternalBackend,
			SecuritySandboxExternalBackendE2B,
		)
	}
	if cfg.Security.Sandbox.Executor.ExternalBaseURL != "http://127.0.0.1:18080" {
		t.Fatalf(
			"security.sandbox.executor.external_base_url = %q, want env override",
			cfg.Security.Sandbox.Executor.ExternalBaseURL,
		)
	}
}
