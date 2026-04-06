package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestManagerReadinessPreflightReactToolRegistryStrictMapping(t *testing.T) {
	cases := []struct {
		name       string
		strict     bool
		wantStatus ReadinessStatus
	}{
		{name: "non_strict", strict: false, wantStatus: ReadinessStatusDegraded},
		{name: "strict", strict: true, wantStatus: ReadinessStatusBlocked},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "runtime-react-tool-registry-"+tc.name+".yaml")
			writeConfig(t, file, `
runtime:
  readiness:
    enabled: true
    strict: `+boolString(tc.strict)+`
    remote_probe_enabled: false
  react:
    enabled: true
    stream_tool_dispatch_enabled: true
`)
			mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX_RUNTIME_REACT_READINESS_TEST"})
			if err != nil {
				t.Fatalf("NewManager failed: %v", err)
			}
			defer func() { _ = mgr.Close() }()

			mgr.SetReactReadinessDependencySnapshot(ReactReadinessDependencySnapshot{
				ToolRegistryChecked:   true,
				ToolRegistryAvailable: false,
				ToolRegistryReason:    "tool registry not initialized",
			})

			result := mgr.ReadinessPreflight()
			if result.Status != tc.wantStatus {
				t.Fatalf("status=%q, want %q", result.Status, tc.wantStatus)
			}
			assertReadinessFindingCode(t, result.Findings, ReadinessCodeReactToolRegistryUnavailable)
			if tc.strict {
				assertReadinessFindingCode(t, result.Findings, ReadinessCodeStrictEscalated)
			}

			summary := result.Summary()
			if !tc.strict {
				if summary.PrimaryCode != ReadinessCodeReactToolRegistryUnavailable ||
					summary.PrimarySource != RuntimePrimarySourceReadiness ||
					summary.RemediationHintCode != "react.restore_tool_registry" ||
					summary.RemediationHintDomain != ReadinessDomainRuntime {
					t.Fatalf("summary mismatch: %#v", summary)
				}
			}
		})
	}
}

func TestManagerReadinessPreflightReactProviderUnsupportedBlocked(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime-react-provider-unsupported.yaml")
	writeConfig(t, file, `
runtime:
  readiness:
    enabled: true
    strict: false
    remote_probe_enabled: false
  react:
    enabled: true
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX_RUNTIME_REACT_READINESS_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	mgr.SetReactReadinessDependencySnapshot(ReactReadinessDependencySnapshot{
		ProviderChecked:              true,
		ProviderName:                 "openai",
		ProviderToolCallingSupported: false,
		ProviderReason:               "tool calling capability not supported",
	})

	result := mgr.ReadinessPreflight()
	if result.Status != ReadinessStatusBlocked {
		t.Fatalf("status=%q, want blocked", result.Status)
	}
	assertReadinessFindingCode(t, result.Findings, ReadinessCodeReactProviderToolCallingUnsupported)
	summary := result.Summary()
	if summary.PrimaryCode != ReadinessCodeReactProviderToolCallingUnsupported ||
		summary.PrimarySource != RuntimePrimarySourceReadiness ||
		summary.RemediationHintCode != "react.select_tool_calling_provider" ||
		summary.RemediationHintDomain != ReadinessDomainRuntime {
		t.Fatalf("summary mismatch: %#v", summary)
	}
}

func TestManagerReadinessPreflightReactDeterministicAndCanonicalTaxonomy(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime-react-deterministic.yaml")
	writeConfig(t, file, `
runtime:
  readiness:
    enabled: true
    strict: false
    remote_probe_enabled: false
  react:
    enabled: true
    stream_tool_dispatch_enabled: false
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX_RUNTIME_REACT_READINESS_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	mgr.SetReactReadinessDependencySnapshot(ReactReadinessDependencySnapshot{
		ToolRegistryChecked:        true,
		ToolRegistryAvailable:      false,
		ToolRegistryReason:         "tool registry unavailable",
		SandboxDependencyChecked:   true,
		SandboxDependencyAvailable: false,
		SandboxDependencyReason:    "sandbox runtime unavailable",
	})

	first := mgr.ReadinessPreflight()
	second := mgr.ReadinessPreflight()
	if first.Status != ReadinessStatusDegraded || second.Status != ReadinessStatusDegraded {
		t.Fatalf("status should stay degraded, first=%q second=%q", first.Status, second.Status)
	}
	if readinessSemanticFingerprint(first) != readinessSemanticFingerprint(second) {
		t.Fatalf(
			"react readiness fingerprint drift first=%s second=%s",
			readinessSemanticFingerprint(first),
			readinessSemanticFingerprint(second),
		)
	}

	assertReadinessFindingCode(t, first.Findings, ReadinessCodeReactStreamDispatchUnavailable)
	assertReadinessFindingCode(t, first.Findings, ReadinessCodeReactToolRegistryUnavailable)
	assertReadinessFindingCode(t, first.Findings, ReadinessCodeReactSandboxDependencyUnavailable)

	allowed := map[string]struct{}{
		ReadinessCodeReactLoopDisabled:                   {},
		ReadinessCodeReactStreamDispatchUnavailable:      {},
		ReadinessCodeReactProviderToolCallingUnsupported: {},
		ReadinessCodeReactToolRegistryUnavailable:        {},
		ReadinessCodeReactSandboxDependencyUnavailable:   {},
	}
	for i := range first.Findings {
		code := strings.TrimSpace(first.Findings[i].Code)
		if !strings.HasPrefix(code, "react.") {
			continue
		}
		if _, ok := allowed[code]; !ok {
			t.Fatalf("non-canonical react readiness code detected: %q in %#v", code, first.Findings)
		}
	}
}

func TestManagerReadinessPreflightReactLoopDisabledFinding(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime-react-disabled.yaml")
	writeConfig(t, file, `
runtime:
  readiness:
    enabled: true
    strict: false
    remote_probe_enabled: false
  react:
    enabled: false
    stream_tool_dispatch_enabled: false
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX_RUNTIME_REACT_READINESS_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	result := mgr.ReadinessPreflight()
	if result.Status != ReadinessStatusDegraded {
		t.Fatalf("status=%q, want degraded", result.Status)
	}
	assertReadinessFindingCode(t, result.Findings, ReadinessCodeReactLoopDisabled)
	summary := result.Summary()
	if summary.PrimaryCode != ReadinessCodeReactLoopDisabled ||
		summary.PrimarySource != RuntimePrimarySourceReadiness ||
		summary.RemediationHintCode != "react.enable_loop" ||
		summary.RemediationHintDomain != ReadinessDomainRuntime {
		t.Fatalf("summary mismatch: %#v", summary)
	}
}

func boolString(v bool) string {
	if v {
		return "true"
	}
	return "false"
}
