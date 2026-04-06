package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManagerReadinessPreflightObservabilitySinkUnavailableStrictMapping(t *testing.T) {
	nonStrictFile := filepath.Join(t.TempDir(), "runtime-observability-nonstrict.yaml")
	writeConfig(t, nonStrictFile, `
runtime:
  readiness:
    enabled: true
    strict: false
    remote_probe_enabled: false
  observability:
    export:
      enabled: true
      profile: otlp
      endpoint: http://127.0.0.1:9/v1/traces
      queue_capacity: 128
      on_error: degrade_and_record
`)
	nonStrictMgr, err := NewManager(ManagerOptions{FilePath: nonStrictFile, EnvPrefix: "BAYMAX_RUNTIME_OBSERVABILITY_TEST"})
	if err != nil {
		t.Fatalf("NewManager non-strict failed: %v", err)
	}
	defer func() { _ = nonStrictMgr.Close() }()

	nonStrictResult := nonStrictMgr.ReadinessPreflight()
	if nonStrictResult.Status != ReadinessStatusDegraded {
		t.Fatalf("non-strict status=%q, want degraded", nonStrictResult.Status)
	}
	assertReadinessFindingCode(t, nonStrictResult.Findings, ReadinessCodeObservabilityExportSinkUnavailable)
	nonStrictSummary := nonStrictResult.Summary()
	if nonStrictSummary.PrimaryCode != ReadinessCodeObservabilityExportSinkUnavailable ||
		nonStrictSummary.PrimarySource != RuntimePrimarySourceReadiness ||
		nonStrictSummary.RemediationHintCode != "runtime.observability.export.restore_sink" ||
		nonStrictSummary.RemediationHintDomain != ReadinessDomainRuntime {
		t.Fatalf("non-strict summary mismatch: %#v", nonStrictSummary)
	}

	strictFile := filepath.Join(t.TempDir(), "runtime-observability-strict.yaml")
	writeConfig(t, strictFile, `
runtime:
  readiness:
    enabled: true
    strict: true
    remote_probe_enabled: false
  observability:
    export:
      enabled: true
      profile: otlp
      endpoint: http://127.0.0.1:9/v1/traces
      queue_capacity: 128
      on_error: degrade_and_record
`)
	strictMgr, err := NewManager(ManagerOptions{FilePath: strictFile, EnvPrefix: "BAYMAX_RUNTIME_OBSERVABILITY_TEST"})
	if err != nil {
		t.Fatalf("NewManager strict failed: %v", err)
	}
	defer func() { _ = strictMgr.Close() }()

	strictResult := strictMgr.ReadinessPreflight()
	if strictResult.Status != ReadinessStatusBlocked {
		t.Fatalf("strict status=%q, want blocked", strictResult.Status)
	}
	assertReadinessFindingCode(t, strictResult.Findings, ReadinessCodeObservabilityExportSinkUnavailable)
	assertReadinessFindingCode(t, strictResult.Findings, ReadinessCodeStrictEscalated)
	strictSummary := strictResult.Summary()
	if strictSummary.PrimaryCode != ReadinessCodeStrictEscalated ||
		strictSummary.PrimarySource != RuntimePrimarySourceReadiness ||
		strictSummary.SecondaryReasonCount != 1 ||
		len(strictSummary.SecondaryReasonCodes) != 1 ||
		strictSummary.SecondaryReasonCodes[0] != ReadinessCodeObservabilityExportSinkUnavailable {
		t.Fatalf("strict summary mismatch: %#v", strictSummary)
	}
}

func TestManagerReadinessPreflightDiagnosticsBundleOutputUnavailableStrictMapping(t *testing.T) {
	unavailablePath := filepath.Join(t.TempDir(), "occupied")
	if err := os.WriteFile(unavailablePath, []byte("busy"), 0o600); err != nil {
		t.Fatalf("write occupied path: %v", err)
	}

	nonStrictFile := filepath.Join(t.TempDir(), "runtime-bundle-nonstrict.yaml")
	writeConfig(t, nonStrictFile, `
runtime:
  readiness:
    enabled: true
    strict: false
    remote_probe_enabled: false
  diagnostics:
    bundle:
      enabled: true
      output_dir: `+filepath.ToSlash(unavailablePath)+`
      max_size_mb: 64
      include_sections: [timeline, diagnostics]
`)
	nonStrictMgr, err := NewManager(ManagerOptions{FilePath: nonStrictFile, EnvPrefix: "BAYMAX_RUNTIME_OBSERVABILITY_TEST"})
	if err != nil {
		t.Fatalf("NewManager non-strict failed: %v", err)
	}
	defer func() { _ = nonStrictMgr.Close() }()

	nonStrictResult := nonStrictMgr.ReadinessPreflight()
	if nonStrictResult.Status != ReadinessStatusDegraded {
		t.Fatalf("non-strict status=%q, want degraded", nonStrictResult.Status)
	}
	assertReadinessFindingCode(t, nonStrictResult.Findings, ReadinessCodeDiagnosticsBundleOutputUnavailable)

	strictFile := filepath.Join(t.TempDir(), "runtime-bundle-strict.yaml")
	writeConfig(t, strictFile, `
runtime:
  readiness:
    enabled: true
    strict: true
    remote_probe_enabled: false
  diagnostics:
    bundle:
      enabled: true
      output_dir: `+filepath.ToSlash(unavailablePath)+`
      max_size_mb: 64
      include_sections: [timeline, diagnostics]
`)
	strictMgr, err := NewManager(ManagerOptions{FilePath: strictFile, EnvPrefix: "BAYMAX_RUNTIME_OBSERVABILITY_TEST"})
	if err != nil {
		t.Fatalf("NewManager strict failed: %v", err)
	}
	defer func() { _ = strictMgr.Close() }()

	strictResult := strictMgr.ReadinessPreflight()
	if strictResult.Status != ReadinessStatusBlocked {
		t.Fatalf("strict status=%q, want blocked", strictResult.Status)
	}
	assertReadinessFindingCode(t, strictResult.Findings, ReadinessCodeDiagnosticsBundleOutputUnavailable)
	assertReadinessFindingCode(t, strictResult.Findings, ReadinessCodeStrictEscalated)
}

func TestObservabilityReadinessFindingsCoverProfileAndPolicyInvalidCodes(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Runtime.Observability.Export.Profile = "jaeger"
	cfg.Runtime.Diagnostics.Bundle.IncludeSections = []string{"unknown_section"}

	findings := observabilityReadinessFindings(cfg)
	assertReadinessFindingCode(t, findings, ReadinessCodeObservabilityExportProfileInvalid)
	assertReadinessFindingCode(t, findings, ReadinessCodeDiagnosticsBundlePolicyInvalid)
}

func TestManagerReadinessPreflightObservabilityPrimaryDeterministic(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime-observability-deterministic.yaml")
	writeConfig(t, file, `
runtime:
  readiness:
    enabled: true
    strict: false
    remote_probe_enabled: false
  observability:
    export:
      enabled: true
      profile: otlp
      endpoint: http://127.0.0.1:9/v1/traces
      queue_capacity: 128
      on_error: degrade_and_record
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX_RUNTIME_OBSERVABILITY_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	first := mgr.ReadinessPreflight()
	second := mgr.ReadinessPreflight()
	if first.Status != second.Status {
		t.Fatalf("status mismatch first=%q second=%q", first.Status, second.Status)
	}
	if readinessSemanticFingerprint(first) != readinessSemanticFingerprint(second) {
		t.Fatalf("semantic fingerprint drift first=%s second=%s", readinessSemanticFingerprint(first), readinessSemanticFingerprint(second))
	}
	if first.Summary().PrimaryCode != second.Summary().PrimaryCode {
		t.Fatalf("primary code drift first=%q second=%q", first.Summary().PrimaryCode, second.Summary().PrimaryCode)
	}
}
