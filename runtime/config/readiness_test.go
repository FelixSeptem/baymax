package config

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	adapterhealth "github.com/FelixSeptem/baymax/adapter/health"
	"github.com/FelixSeptem/baymax/core/types"
)

type fakeSandboxExecutor struct {
	probe   func(ctx context.Context) (types.SandboxCapabilityProbe, error)
	execute func(ctx context.Context, spec types.SandboxExecSpec) (types.SandboxExecResult, error)
}

func (f *fakeSandboxExecutor) Probe(ctx context.Context) (types.SandboxCapabilityProbe, error) {
	if f != nil && f.probe != nil {
		return f.probe(ctx)
	}
	return types.SandboxCapabilityProbe{}, nil
}

func (f *fakeSandboxExecutor) Execute(ctx context.Context, spec types.SandboxExecSpec) (types.SandboxExecResult, error) {
	if f != nil && f.execute != nil {
		return f.execute(ctx, spec)
	}
	return types.SandboxExecResult{}, nil
}

func TestManagerReadinessPreflightClassificationMatrix(t *testing.T) {
	mgr, err := NewManager(ManagerOptions{EnvPrefix: "BAYMAX_READINESS_BASELINE_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	ready := mgr.ReadinessPreflight()
	if ready.Status != ReadinessStatusReady {
		t.Fatalf("ready status = %q, want %q", ready.Status, ReadinessStatusReady)
	}
	if len(ready.Findings) != 0 {
		t.Fatalf("ready findings = %#v, want empty", ready.Findings)
	}

	mgr.SetReadinessComponentSnapshot(RuntimeReadinessComponentSnapshot{
		Scheduler: RuntimeReadinessComponentState{
			Enabled:           true,
			ConfiguredBackend: "file",
			EffectiveBackend:  "memory",
			Fallback:          true,
			FallbackReason:    "scheduler.backend.file_init_failed",
		},
	})
	degraded := mgr.ReadinessPreflight()
	if degraded.Status != ReadinessStatusDegraded {
		t.Fatalf("degraded status = %q, want %q", degraded.Status, ReadinessStatusDegraded)
	}
	assertReadinessFindingCode(t, degraded.Findings, ReadinessCodeSchedulerFallback)
	assertReadinessCanonicalFields(t, degraded.Findings)

	mgr.SetReadinessComponentSnapshot(RuntimeReadinessComponentSnapshot{
		Recovery: RuntimeReadinessComponentState{
			Enabled:           true,
			ConfiguredBackend: "file",
			EffectiveBackend:  "memory",
			ActivationError:   "permission denied",
		},
	})
	blocked := mgr.ReadinessPreflight()
	if blocked.Status != ReadinessStatusBlocked {
		t.Fatalf("blocked status = %q, want %q", blocked.Status, ReadinessStatusBlocked)
	}
	assertReadinessFindingCode(t, blocked.Findings, ReadinessCodeRecoveryActivationError)
	assertReadinessCanonicalFields(t, blocked.Findings)
}

func TestManagerReadinessPreflightStrictEscalation(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
runtime:
  readiness:
    enabled: true
    strict: true
    remote_probe_enabled: false
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX_READINESS_BASELINE_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	mgr.SetReadinessComponentSnapshot(RuntimeReadinessComponentSnapshot{
		Scheduler: RuntimeReadinessComponentState{
			Enabled:           true,
			ConfiguredBackend: "file",
			EffectiveBackend:  "memory",
			Fallback:          true,
			FallbackReason:    "scheduler.backend.file_init_failed",
		},
	})

	result := mgr.ReadinessPreflight()
	if result.Status != ReadinessStatusBlocked {
		t.Fatalf("strict status = %q, want %q", result.Status, ReadinessStatusBlocked)
	}
	assertReadinessFindingCode(t, result.Findings, ReadinessCodeSchedulerFallback)
	assertReadinessFindingCode(t, result.Findings, ReadinessCodeStrictEscalated)

	summary := result.Summary()
	if summary.Status != string(ReadinessStatusBlocked) {
		t.Fatalf("summary status = %q, want %q", summary.Status, ReadinessStatusBlocked)
	}
	if summary.FindingTotal < 2 || summary.BlockingTotal < 1 || summary.DegradedTotal < 1 {
		t.Fatalf("summary counts mismatch: %#v", summary)
	}
	if summary.PrimaryDomain != ReadinessDomainRuntime ||
		summary.PrimaryCode != ReadinessCodeStrictEscalated ||
		summary.PrimarySource != RuntimePrimarySourceReadiness ||
		summary.PrimaryConflictTotal != 0 ||
		summary.SecondaryReasonCount != 1 ||
		len(summary.SecondaryReasonCodes) != 1 ||
		summary.SecondaryReasonCodes[0] != ReadinessCodeSchedulerFallback ||
		summary.ArbitrationRuleVersion != RuntimeArbitrationRuleVersionExplainabilityV1 ||
		summary.RemediationHintCode != "runtime.relax_strict_mode" ||
		summary.RemediationHintDomain != ReadinessDomainRuntime {
		t.Fatalf("summary primary arbitration mismatch: %#v", summary)
	}
}

func TestManagerReadinessPreflightDeterministicForEquivalentSnapshot(t *testing.T) {
	mgr, err := NewManager(ManagerOptions{EnvPrefix: "BAYMAX_READINESS_BASELINE_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	mgr.SetReadinessComponentSnapshot(RuntimeReadinessComponentSnapshot{
		Scheduler: RuntimeReadinessComponentState{
			Enabled:           true,
			ConfiguredBackend: "file",
			EffectiveBackend:  "memory",
			Fallback:          true,
			FallbackReason:    "scheduler.backend.file_init_failed",
		},
		Mailbox: RuntimeReadinessComponentState{
			Enabled:           true,
			ConfiguredBackend: "file",
			EffectiveBackend:  "memory",
			Fallback:          true,
			FallbackReason:    "mailbox.backend.file_init_failed",
		},
	})

	first := mgr.ReadinessPreflight()
	second := mgr.ReadinessPreflight()
	if first.Status != second.Status {
		t.Fatalf("status mismatch first=%q second=%q", first.Status, second.Status)
	}
	if readinessSemanticFingerprint(first) != readinessSemanticFingerprint(second) {
		t.Fatalf("semantics changed across equivalent snapshots\nfirst=%s\nsecond=%s", readinessSemanticFingerprint(first), readinessSemanticFingerprint(second))
	}
}

func TestManagerReadinessPreflightAdapterHealthRequiredOptionalMapping(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
runtime:
  readiness:
    enabled: true
    strict: false
    remote_probe_enabled: false
adapter:
  health:
    enabled: true
    strict: false
    probe_timeout: 500ms
    cache_ttl: 30s
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX_READINESS_TIMEOUT_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	mgr.SetAdapterHealthTargets([]AdapterHealthTarget{
		{
			Name:     "required-x",
			Required: true,
			Probe: adapterhealth.ProbeFunc(func(_ context.Context) (adapterhealth.Result, error) {
				return adapterhealth.Result{Status: adapterhealth.StatusUnavailable, Code: "non_canonical.required"}, nil
			}),
		},
		{
			Name:     "optional-y",
			Required: false,
			Probe: adapterhealth.ProbeFunc(func(_ context.Context) (adapterhealth.Result, error) {
				return adapterhealth.Result{Status: adapterhealth.StatusUnavailable, Code: "non_canonical.optional"}, nil
			}),
		},
		{
			Name:     "optional-z",
			Required: false,
			Probe: adapterhealth.ProbeFunc(func(_ context.Context) (adapterhealth.Result, error) {
				return adapterhealth.Result{Status: adapterhealth.StatusDegraded, Code: "non_canonical.degraded"}, nil
			}),
		},
	})

	result := mgr.ReadinessPreflight()
	if result.Status != ReadinessStatusDegraded {
		t.Fatalf("status = %q, want degraded", result.Status)
	}
	assertReadinessFindingCode(t, result.Findings, ReadinessCodeAdapterRequiredUnavailable)
	assertReadinessFindingCode(t, result.Findings, ReadinessCodeAdapterOptionalUnavailable)
	assertReadinessFindingCode(t, result.Findings, ReadinessCodeAdapterDegraded)
	assertReadinessCanonicalFields(t, result.Findings)

	summary := result.Summary()
	if summary.AdapterHealthStatus != string(adapterhealth.StatusUnavailable) {
		t.Fatalf("adapter_health_status = %q, want unavailable", summary.AdapterHealthStatus)
	}
	if summary.AdapterHealthProbeTotal != 3 || summary.AdapterHealthDegradedTotal != 1 || summary.AdapterHealthUnavailableTotal != 2 {
		t.Fatalf("adapter health summary mismatch: %#v", summary)
	}
	if summary.PrimaryDomain != ReadinessDomainAdapter ||
		summary.PrimaryCode != ReadinessCodeAdapterRequiredUnavailable ||
		summary.PrimarySource != RuntimePrimarySourceAdapter ||
		summary.SecondaryReasonCount != 2 ||
		len(summary.SecondaryReasonCodes) != 2 ||
		summary.SecondaryReasonCodes[0] != ReadinessCodeAdapterDegraded ||
		summary.SecondaryReasonCodes[1] != ReadinessCodeAdapterOptionalUnavailable ||
		summary.ArbitrationRuleVersion != RuntimeArbitrationRuleVersionExplainabilityV1 ||
		summary.RemediationHintCode != "adapter.restore_required" ||
		summary.RemediationHintDomain != ReadinessDomainAdapter {
		t.Fatalf("adapter arbitration summary mismatch: %#v", summary)
	}
}

func TestManagerReadinessPreflightAdapterHealthStrictRequiredUnavailableBlocked(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
runtime:
  readiness:
    enabled: true
    strict: true
    remote_probe_enabled: false
adapter:
  health:
    enabled: true
    strict: false
    probe_timeout: 500ms
    cache_ttl: 30s
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX_READINESS_TIMEOUT_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	mgr.SetAdapterHealthTargets([]AdapterHealthTarget{
		{
			Name:     "required-x",
			Required: true,
			Probe: adapterhealth.ProbeFunc(func(_ context.Context) (adapterhealth.Result, error) {
				return adapterhealth.Result{Status: adapterhealth.StatusUnavailable, Code: adapterhealth.CodeProbeFailed}, nil
			}),
		},
	})

	result := mgr.ReadinessPreflight()
	if result.Status != ReadinessStatusBlocked {
		t.Fatalf("status = %q, want blocked", result.Status)
	}
	assertReadinessFindingCode(t, result.Findings, ReadinessCodeAdapterRequiredUnavailable)
	for _, finding := range result.Findings {
		if finding.Code == ReadinessCodeAdapterRequiredUnavailable && finding.Severity != ReadinessSeverityError {
			t.Fatalf("required unavailable must be blocking under strict policy: %#v", finding)
		}
	}
}

func TestManagerReadinessPreflightAdapterHealthDeterministicForEquivalentSnapshot(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
runtime:
  readiness:
    enabled: true
    strict: false
    remote_probe_enabled: false
adapter:
  health:
    enabled: true
    strict: false
    probe_timeout: 500ms
    cache_ttl: 30s
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX_READINESS_TIMEOUT_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	mgr.SetAdapterHealthTargets([]AdapterHealthTarget{
		{
			Name:     "adapter-a",
			Required: false,
			Probe: adapterhealth.ProbeFunc(func(_ context.Context) (adapterhealth.Result, error) {
				return adapterhealth.Result{Status: adapterhealth.StatusDegraded, Code: adapterhealth.CodeDegraded}, nil
			}),
		},
	})

	first := mgr.ReadinessPreflight()
	second := mgr.ReadinessPreflight()
	if first.Status != second.Status {
		t.Fatalf("status mismatch first=%q second=%q", first.Status, second.Status)
	}
	if readinessSemanticFingerprint(first) != readinessSemanticFingerprint(second) {
		t.Fatalf("semantic fingerprint drift first=%s second=%s", readinessSemanticFingerprint(first), readinessSemanticFingerprint(second))
	}
	if first.Summary().AdapterHealthPrimaryCode != second.Summary().AdapterHealthPrimaryCode {
		t.Fatalf("adapter primary code drift first=%q second=%q", first.Summary().AdapterHealthPrimaryCode, second.Summary().AdapterHealthPrimaryCode)
	}
}

func TestManagerReadinessPreflightAdapterHealthCircuitOpenStrictAndNonStrictMapping(t *testing.T) {
	strictFile := filepath.Join(t.TempDir(), "runtime-strict.yaml")
	writeConfig(t, strictFile, `
runtime:
  readiness:
    enabled: true
    strict: true
    remote_probe_enabled: false
adapter:
  health:
    enabled: true
    strict: false
    probe_timeout: 500ms
    cache_ttl: 1ms
    backoff:
      enabled: false
      initial: 200ms
      max: 5s
      multiplier: 2
      jitter_ratio: 0.2
    circuit:
      enabled: true
      failure_threshold: 1
      open_duration: 30s
      half_open_max_probe: 1
      half_open_success_threshold: 2
`)
	strictMgr, err := NewManager(ManagerOptions{FilePath: strictFile, EnvPrefix: "BAYMAX_ADAPTER_HEALTH_GOVERNANCE_TEST"})
	if err != nil {
		t.Fatalf("NewManager strict failed: %v", err)
	}
	defer func() { _ = strictMgr.Close() }()
	strictMgr.SetAdapterHealthTargets([]AdapterHealthTarget{
		{
			Name:     "required-a46",
			Required: true,
			Probe: adapterhealth.ProbeFunc(func(_ context.Context) (adapterhealth.Result, error) {
				return adapterhealth.Result{
					Status: adapterhealth.StatusUnavailable,
					Code:   adapterhealth.CodeProbeFailed,
				}, nil
			}),
		},
	})
	_ = strictMgr.ReadinessPreflight()
	strictResult := strictMgr.ReadinessPreflight()
	if strictResult.Status != ReadinessStatusBlocked {
		t.Fatalf("strict status=%q, want blocked", strictResult.Status)
	}
	assertReadinessFindingCode(t, strictResult.Findings, ReadinessCodeAdapterRequiredCircuitOpen)

	nonStrictFile := filepath.Join(t.TempDir(), "runtime-nonstrict.yaml")
	writeConfig(t, nonStrictFile, `
runtime:
  readiness:
    enabled: true
    strict: false
    remote_probe_enabled: false
adapter:
  health:
    enabled: true
    strict: false
    probe_timeout: 500ms
    cache_ttl: 1ms
    backoff:
      enabled: false
      initial: 200ms
      max: 5s
      multiplier: 2
      jitter_ratio: 0.2
    circuit:
      enabled: true
      failure_threshold: 1
      open_duration: 30s
      half_open_max_probe: 1
      half_open_success_threshold: 2
`)
	nonStrictMgr, err := NewManager(ManagerOptions{FilePath: nonStrictFile, EnvPrefix: "BAYMAX_ADAPTER_HEALTH_GOVERNANCE_TEST"})
	if err != nil {
		t.Fatalf("NewManager non-strict failed: %v", err)
	}
	defer func() { _ = nonStrictMgr.Close() }()
	nonStrictMgr.SetAdapterHealthTargets([]AdapterHealthTarget{
		{
			Name:     "optional-a46",
			Required: false,
			Probe: adapterhealth.ProbeFunc(func(_ context.Context) (adapterhealth.Result, error) {
				return adapterhealth.Result{
					Status: adapterhealth.StatusUnavailable,
					Code:   adapterhealth.CodeProbeFailed,
				}, nil
			}),
		},
	})
	_ = nonStrictMgr.ReadinessPreflight()
	nonStrictResult := nonStrictMgr.ReadinessPreflight()
	if nonStrictResult.Status != ReadinessStatusDegraded {
		t.Fatalf("non-strict status=%q, want degraded", nonStrictResult.Status)
	}
	assertReadinessFindingCode(t, nonStrictResult.Findings, ReadinessCodeAdapterOptionalCircuitOpen)
}

func TestManagerReadinessPreflightAdapterHealthHalfOpenDegradedAndGovernanceSummary(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
runtime:
  readiness:
    enabled: true
    strict: false
    remote_probe_enabled: false
adapter:
  health:
    enabled: true
    strict: false
    probe_timeout: 500ms
    cache_ttl: 1ms
    backoff:
      enabled: false
      initial: 200ms
      max: 5s
      multiplier: 2
      jitter_ratio: 0.2
    circuit:
      enabled: true
      failure_threshold: 1
      open_duration: 20ms
      half_open_max_probe: 1
      half_open_success_threshold: 2
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX_ADAPTER_HEALTH_GOVERNANCE_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	var calls int
	mgr.SetAdapterHealthTargets([]AdapterHealthTarget{
		{
			Name:     "half-open-a46",
			Required: false,
			Probe: adapterhealth.ProbeFunc(func(_ context.Context) (adapterhealth.Result, error) {
				calls++
				if calls == 1 {
					return adapterhealth.Result{
						Status: adapterhealth.StatusUnavailable,
						Code:   adapterhealth.CodeProbeFailed,
					}, nil
				}
				return adapterhealth.Result{
					Status: adapterhealth.StatusDegraded,
					Code:   adapterhealth.CodeDegraded,
				}, nil
			}),
		},
	})

	_ = mgr.ReadinessPreflight()
	time.Sleep(25 * time.Millisecond)
	result := mgr.ReadinessPreflight()
	if result.Status != ReadinessStatusDegraded {
		t.Fatalf("status=%q, want degraded", result.Status)
	}
	assertReadinessFindingCode(t, result.Findings, ReadinessCodeAdapterHalfOpenDegraded)
	summary := result.Summary()
	if summary.AdapterHealthBackoffAppliedTotal != 0 ||
		summary.AdapterHealthCircuitOpenTotal != 1 ||
		summary.AdapterHealthCircuitHalfOpenTotal != 1 ||
		summary.AdapterHealthCircuitRecoverTotal != 0 ||
		summary.AdapterHealthCircuitState != string(adapterhealth.CircuitStateHalfOpen) ||
		summary.AdapterHealthGovernancePrimaryCode != adapterhealth.CodeCircuitHalfOpen {
		t.Fatalf("governance summary mismatch: %#v", summary)
	}
}

func TestManagerReadinessPreflightSandboxRequiredUnavailableBlocked(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime-a51-required.yaml")
	writeConfig(t, file, `
runtime:
  readiness:
    enabled: true
    strict: false
    remote_probe_enabled: false
security:
  sandbox:
    enabled: true
    required: true
    mode: enforce
    policy:
      default_action: sandbox
      profile: default
      fallback_action: deny
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX_SANDBOX_EXECUTION_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	result := mgr.ReadinessPreflight()
	if result.Status != ReadinessStatusBlocked {
		t.Fatalf("status=%q, want blocked", result.Status)
	}
	assertReadinessFindingCode(t, result.Findings, ReadinessCodeSandboxRequiredUnavailable)
	for _, finding := range result.Findings {
		if finding.Code == ReadinessCodeSandboxRequiredUnavailable && finding.Severity != ReadinessSeverityError {
			t.Fatalf("required sandbox unavailable must be blocking, got %#v", finding)
		}
	}
	summary := result.Summary()
	if summary.PrimaryCode != ReadinessCodeSandboxRequiredUnavailable ||
		summary.PrimarySource != RuntimePrimarySourceReadiness ||
		summary.RemediationHintCode != "sandbox.restore_required_executor" ||
		summary.RemediationHintDomain != ReadinessDomainRuntime {
		t.Fatalf("sandbox required unavailable summary mismatch: %#v", summary)
	}
}

func TestManagerReadinessPreflightSandboxOptionalUnavailableStrictEscalation(t *testing.T) {
	nonStrictFile := filepath.Join(t.TempDir(), "runtime-a51-optional.yaml")
	writeConfig(t, nonStrictFile, `
runtime:
  readiness:
    enabled: true
    strict: false
    remote_probe_enabled: false
security:
  sandbox:
    enabled: true
    required: false
    mode: observe
    policy:
      default_action: sandbox
      profile: default
      fallback_action: allow_and_record
`)
	nonStrictMgr, err := NewManager(ManagerOptions{FilePath: nonStrictFile, EnvPrefix: "BAYMAX_SANDBOX_EXECUTION_TEST"})
	if err != nil {
		t.Fatalf("NewManager non-strict failed: %v", err)
	}
	defer func() { _ = nonStrictMgr.Close() }()
	nonStrictResult := nonStrictMgr.ReadinessPreflight()
	if nonStrictResult.Status != ReadinessStatusDegraded {
		t.Fatalf("non-strict status=%q, want degraded", nonStrictResult.Status)
	}
	assertReadinessFindingCode(t, nonStrictResult.Findings, ReadinessCodeSandboxOptionalUnavailable)
	for _, finding := range nonStrictResult.Findings {
		if finding.Code == ReadinessCodeSandboxOptionalUnavailable && finding.Severity != ReadinessSeverityWarning {
			t.Fatalf("optional sandbox unavailable must be degraded warning, got %#v", finding)
		}
	}

	strictFile := filepath.Join(t.TempDir(), "runtime-a51-optional-strict.yaml")
	writeConfig(t, strictFile, `
runtime:
  readiness:
    enabled: true
    strict: true
    remote_probe_enabled: false
security:
  sandbox:
    enabled: true
    required: false
    mode: observe
    policy:
      default_action: sandbox
      profile: default
      fallback_action: allow_and_record
`)
	strictMgr, err := NewManager(ManagerOptions{FilePath: strictFile, EnvPrefix: "BAYMAX_SANDBOX_EXECUTION_TEST"})
	if err != nil {
		t.Fatalf("NewManager strict failed: %v", err)
	}
	defer func() { _ = strictMgr.Close() }()
	strictResult := strictMgr.ReadinessPreflight()
	if strictResult.Status != ReadinessStatusBlocked {
		t.Fatalf("strict status=%q, want blocked", strictResult.Status)
	}
	assertReadinessFindingCode(t, strictResult.Findings, ReadinessCodeSandboxOptionalUnavailable)
	assertReadinessFindingCode(t, strictResult.Findings, ReadinessCodeStrictEscalated)
}

func TestManagerReadinessPreflightSandboxAdapterProfileMissingStrictMapping(t *testing.T) {
	nonStrictFile := filepath.Join(t.TempDir(), "runtime-a53-profile-missing-nonstrict.yaml")
	writeConfig(t, nonStrictFile, `
runtime:
  readiness:
    enabled: true
    strict: false
    remote_probe_enabled: false
security:
  sandbox:
    enabled: true
    required: false
    mode: observe
    policy:
      default_action: sandbox
      profile: missing-profile
      fallback_action: allow_and_record
`)
	nonStrictMgr, err := NewManager(ManagerOptions{FilePath: nonStrictFile, EnvPrefix: "BAYMAX_ADAPTER_ALLOWLIST_TEST"})
	if err != nil {
		t.Fatalf("NewManager non-strict failed: %v", err)
	}
	defer func() { _ = nonStrictMgr.Close() }()
	nonStrictResult := nonStrictMgr.ReadinessPreflight()
	if nonStrictResult.Status != ReadinessStatusDegraded {
		t.Fatalf("non-strict status=%q, want degraded", nonStrictResult.Status)
	}
	assertReadinessFindingCode(t, nonStrictResult.Findings, ReadinessCodeSandboxAdapterProfileMissing)

	strictFile := filepath.Join(t.TempDir(), "runtime-a53-profile-missing-strict.yaml")
	writeConfig(t, strictFile, `
runtime:
  readiness:
    enabled: true
    strict: true
    remote_probe_enabled: false
security:
  sandbox:
    enabled: true
    required: false
    mode: observe
    policy:
      default_action: sandbox
      profile: missing-profile
      fallback_action: allow_and_record
`)
	strictMgr, err := NewManager(ManagerOptions{FilePath: strictFile, EnvPrefix: "BAYMAX_ADAPTER_ALLOWLIST_TEST"})
	if err != nil {
		t.Fatalf("NewManager strict failed: %v", err)
	}
	defer func() { _ = strictMgr.Close() }()
	strictResult := strictMgr.ReadinessPreflight()
	if strictResult.Status != ReadinessStatusBlocked {
		t.Fatalf("strict status=%q, want blocked", strictResult.Status)
	}
	assertReadinessFindingCode(t, strictResult.Findings, ReadinessCodeSandboxAdapterProfileMissing)
	assertReadinessFindingCode(t, strictResult.Findings, ReadinessCodeStrictEscalated)
}

func TestManagerReadinessPreflightSandboxAdapterBackendUnsupportedAndHostMismatch(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime-a53-backend-host.yaml")
	unsupportedBackend := sandboxUnsupportedBackendForCurrentHost()
	writeConfig(t, file, fmt.Sprintf(`
runtime:
  readiness:
    enabled: true
    strict: false
    remote_probe_enabled: false
security:
  sandbox:
    enabled: true
    required: false
    mode: observe
    policy:
      default_action: sandbox
      profile: default
      fallback_action: allow_and_record
    executor:
      backend: %s
      session_mode: per_call
`, unsupportedBackend))
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX_ADAPTER_ALLOWLIST_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	result := mgr.ReadinessPreflight()
	if result.Status != ReadinessStatusDegraded {
		t.Fatalf("status=%q, want degraded", result.Status)
	}
	assertReadinessFindingCode(t, result.Findings, ReadinessCodeSandboxAdapterBackendNotSupported)
	assertReadinessFindingCode(t, result.Findings, ReadinessCodeSandboxAdapterHostMismatch)
}

func TestManagerReadinessPreflightSandboxCapabilityMismatchAndSessionUnsupported(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime-a51-capability.yaml")
	backend := sandboxBackendForCurrentHost()
	writeConfig(t, file, fmt.Sprintf(`
runtime:
  readiness:
    enabled: true
    strict: false
    remote_probe_enabled: false
security:
  sandbox:
    enabled: true
    required: true
    mode: enforce
    policy:
      default_action: sandbox
      profile: default
      fallback_action: deny
    executor:
      backend: %s
      session_mode: per_session
      required_capabilities: [network_off, stdout_stderr_capture]
`, backend))
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX_SANDBOX_EXECUTION_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()
	mgr.SetSandboxExecutor(&fakeSandboxExecutor{
		probe: func(ctx context.Context) (types.SandboxCapabilityProbe, error) {
			_ = ctx
			return types.SandboxCapabilityProbe{
				Backend:        backend,
				Capabilities:   []string{SecuritySandboxCapabilityStdoutStderrCapture},
				SupportedModes: []string{SecuritySandboxSessionModePerCall},
			}, nil
		},
	})

	result := mgr.ReadinessPreflight()
	if result.Status != ReadinessStatusBlocked {
		t.Fatalf("status=%q, want blocked", result.Status)
	}
	assertReadinessFindingCode(t, result.Findings, ReadinessCodeSandboxCapabilityMismatch)
	assertReadinessFindingCode(t, result.Findings, ReadinessCodeSandboxAdapterSessionModeUnsupported)

	summary := result.Summary()
	if summary.PrimaryCode != ReadinessCodeSandboxAdapterSessionModeUnsupported ||
		summary.PrimarySource != RuntimePrimarySourceReadiness ||
		summary.RemediationHintCode != "sandbox.adapter.adjust_session_mode" ||
		summary.RemediationHintDomain != ReadinessDomainRuntime {
		t.Fatalf("sandbox capability mismatch summary mismatch: %#v", summary)
	}
}

func TestManagerReadinessAdmissionSandboxRequiredDenyExplainability(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime-a51-admission.yaml")
	writeConfig(t, file, `
runtime:
  readiness:
    enabled: true
    strict: false
    remote_probe_enabled: false
    admission:
      enabled: true
      mode: fail_fast
      block_on: blocked_only
      degraded_policy: allow_and_record
security:
  sandbox:
    enabled: true
    required: true
    mode: enforce
    policy:
      default_action: sandbox
      profile: default
      fallback_action: deny
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX_SANDBOX_EXECUTION_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	first := mgr.EvaluateReadinessAdmission()
	second := mgr.EvaluateReadinessAdmission()
	if first.Outcome != ReadinessAdmissionOutcomeDeny {
		t.Fatalf("admission outcome=%q, want deny", first.Outcome)
	}
	if first.ReasonCode != ReadinessAdmissionCodeBlocked {
		t.Fatalf("admission reason_code=%q, want %q", first.ReasonCode, ReadinessAdmissionCodeBlocked)
	}
	if first.ReadinessPrimaryCode != ReadinessCodeSandboxRequiredUnavailable ||
		first.ReadinessPrimaryDomain != ReadinessDomainRuntime ||
		first.ReadinessPrimarySource != RuntimePrimarySourceReadiness ||
		first.ReadinessRemediationHintCode != "sandbox.restore_required_executor" ||
		first.ReadinessRemediationHintDomain != ReadinessDomainRuntime {
		t.Fatalf("sandbox admission explainability mismatch: %#v", first)
	}
	if readinessAdmissionFingerprint(first) != readinessAdmissionFingerprint(second) {
		t.Fatalf("sandbox admission must be deterministic first=%s second=%s", readinessAdmissionFingerprint(first), readinessAdmissionFingerprint(second))
	}
}

func TestManagerReadinessPreflightPolicyCandidatesWinnerMetadata(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime-a58-preflight.yaml")
	writeConfig(t, file, `
runtime:
  readiness:
    enabled: true
    strict: false
    remote_probe_enabled: false
security:
  sandbox:
    enabled: true
    required: true
    mode: enforce
    policy:
      default_action: sandbox
      profile: default
      fallback_action: deny
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX_POLICY_DECISION_PATH_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	first := mgr.ReadinessPreflight()
	second := mgr.ReadinessPreflight()
	if len(first.PolicyCandidates) == 0 {
		t.Fatalf("policy_candidates should not be empty: %#v", first)
	}
	if len(first.PolicyDecisionPath) == 0 {
		t.Fatalf("policy_decision_path should not be empty: %#v", first)
	}
	if first.PolicyPrecedenceVersion != RuntimePolicyPrecedenceVersionPolicyStackV1 {
		t.Fatalf("policy_precedence_version=%q, want %q", first.PolicyPrecedenceVersion, RuntimePolicyPrecedenceVersionPolicyStackV1)
	}
	if first.WinnerStage != RuntimePolicyStageSandboxAction {
		t.Fatalf("winner_stage=%q, want %q", first.WinnerStage, RuntimePolicyStageSandboxAction)
	}
	if strings.TrimSpace(first.DenySource) == "" {
		t.Fatalf("deny_source should be present: %#v", first)
	}

	firstDigest, _ := json.Marshal(struct {
		Candidates []RuntimePolicyCandidate `json:"policy_candidates"`
		Path       []RuntimePolicyCandidate `json:"policy_decision_path"`
		Version    string                   `json:"policy_precedence_version"`
		Winner     string                   `json:"winner_stage"`
		DenySource string                   `json:"deny_source"`
		TieBreak   string                   `json:"tie_break_reason"`
	}{
		Candidates: first.PolicyCandidates,
		Path:       first.PolicyDecisionPath,
		Version:    first.PolicyPrecedenceVersion,
		Winner:     first.WinnerStage,
		DenySource: first.DenySource,
		TieBreak:   first.TieBreakReason,
	})
	secondDigest, _ := json.Marshal(struct {
		Candidates []RuntimePolicyCandidate `json:"policy_candidates"`
		Path       []RuntimePolicyCandidate `json:"policy_decision_path"`
		Version    string                   `json:"policy_precedence_version"`
		Winner     string                   `json:"winner_stage"`
		DenySource string                   `json:"deny_source"`
		TieBreak   string                   `json:"tie_break_reason"`
	}{
		Candidates: second.PolicyCandidates,
		Path:       second.PolicyDecisionPath,
		Version:    second.PolicyPrecedenceVersion,
		Winner:     second.WinnerStage,
		DenySource: second.DenySource,
		TieBreak:   second.TieBreakReason,
	})
	if string(firstDigest) != string(secondDigest) {
		t.Fatalf("preflight policy metadata must be deterministic first=%s second=%s", string(firstDigest), string(secondDigest))
	}
}

func TestManagerReadinessAdmissionPolicyDecisionTraceFields(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime-a58-admission.yaml")
	writeConfig(t, file, `
runtime:
  readiness:
    enabled: true
    strict: false
    remote_probe_enabled: false
    admission:
      enabled: true
      mode: fail_fast
      block_on: blocked_only
      degraded_policy: allow_and_record
security:
  sandbox:
    enabled: true
    required: true
    mode: enforce
    policy:
      default_action: sandbox
      profile: default
      fallback_action: deny
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX_POLICY_DECISION_PATH_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	decision := mgr.EvaluateReadinessAdmission()
	if decision.Outcome != ReadinessAdmissionOutcomeDeny {
		t.Fatalf("outcome=%q, want deny", decision.Outcome)
	}
	if decision.PolicyPrecedenceVersion != RuntimePolicyPrecedenceVersionPolicyStackV1 {
		t.Fatalf("policy_precedence_version=%q, want %q", decision.PolicyPrecedenceVersion, RuntimePolicyPrecedenceVersionPolicyStackV1)
	}
	if decision.WinnerStage != RuntimePolicyStageSandboxAction {
		t.Fatalf("winner_stage=%q, want %q", decision.WinnerStage, RuntimePolicyStageSandboxAction)
	}
	if strings.TrimSpace(decision.DenySource) == "" {
		t.Fatalf("deny_source should be present: %#v", decision)
	}
	if len(decision.PolicyDecisionPath) == 0 {
		t.Fatalf("policy_decision_path should not be empty: %#v", decision)
	}
}

func TestManagerReadinessPreflightSandboxRolloutFrozenFinding(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime-a52-frozen.yaml")
	writeConfig(t, file, `
runtime:
  readiness:
    enabled: true
    strict: false
    remote_probe_enabled: false
security:
  sandbox:
    enabled: true
    mode: observe
    rollout:
      phase: frozen
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX_SANDBOX_ROLLOUT_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	result := mgr.ReadinessPreflight()
	if result.Status != ReadinessStatusBlocked {
		t.Fatalf("status=%q, want blocked", result.Status)
	}
	assertReadinessFindingCode(t, result.Findings, ReadinessCodeSandboxRolloutFrozen)
}

func TestManagerReadinessPreflightSandboxRolloutHealthBudgetBreachedFinding(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime-a52-health.yaml")
	writeConfig(t, file, `
runtime:
  readiness:
    enabled: true
    strict: false
    remote_probe_enabled: false
security:
  sandbox:
    enabled: true
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX_SANDBOX_ROLLOUT_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()
	mgr.SetSandboxRolloutRuntimeState(SandboxRolloutRuntimeState{
		HealthBudgetStatus:      SandboxHealthBudgetBreached,
		HealthBudgetBreachTotal: 3,
		FreezeState:             false,
	})

	result := mgr.ReadinessPreflight()
	if result.Status != ReadinessStatusDegraded {
		t.Fatalf("status=%q, want degraded", result.Status)
	}
	assertReadinessFindingCode(t, result.Findings, ReadinessCodeSandboxRolloutHealthBreached)
}

func TestManagerReadinessPreflightSandboxCapacityStrictMapping(t *testing.T) {
	nonStrictFile := filepath.Join(t.TempDir(), "runtime-a52-capacity-nonstrict.yaml")
	writeConfig(t, nonStrictFile, `
runtime:
  readiness:
    enabled: true
    strict: false
    remote_probe_enabled: false
security:
  sandbox:
    enabled: true
`)
	nonStrictMgr, err := NewManager(ManagerOptions{FilePath: nonStrictFile, EnvPrefix: "BAYMAX_SANDBOX_ROLLOUT_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = nonStrictMgr.Close() }()
	nonStrictMgr.SetSandboxRolloutRuntimeState(SandboxRolloutRuntimeState{CapacityAction: SandboxCapacityActionThrottle})

	nonStrictResult := nonStrictMgr.ReadinessPreflight()
	if nonStrictResult.Status != ReadinessStatusDegraded {
		t.Fatalf("non-strict status=%q, want degraded", nonStrictResult.Status)
	}
	assertReadinessFindingCode(t, nonStrictResult.Findings, ReadinessCodeSandboxRolloutCapacityBlocked)

	strictFile := filepath.Join(t.TempDir(), "runtime-a52-capacity-strict.yaml")
	writeConfig(t, strictFile, `
runtime:
  readiness:
    enabled: true
    strict: true
    remote_probe_enabled: false
security:
  sandbox:
    enabled: true
`)
	strictMgr, err := NewManager(ManagerOptions{FilePath: strictFile, EnvPrefix: "BAYMAX_SANDBOX_ROLLOUT_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = strictMgr.Close() }()
	strictMgr.SetSandboxRolloutRuntimeState(SandboxRolloutRuntimeState{CapacityAction: SandboxCapacityActionThrottle})

	strictResult := strictMgr.ReadinessPreflight()
	if strictResult.Status != ReadinessStatusBlocked {
		t.Fatalf("strict status=%q, want blocked", strictResult.Status)
	}
	assertReadinessFindingCode(t, strictResult.Findings, ReadinessCodeSandboxRolloutCapacityBlocked)
	assertReadinessFindingCode(t, strictResult.Findings, ReadinessCodeStrictEscalated)
}

func TestManagerReadinessAdmissionSandboxRolloutFrozenDeny(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime-a52-admission-frozen.yaml")
	writeConfig(t, file, `
runtime:
  readiness:
    enabled: true
    strict: false
    remote_probe_enabled: false
    admission:
      enabled: true
      mode: fail_fast
      block_on: blocked_only
      degraded_policy: allow_and_record
security:
  sandbox:
    enabled: true
    rollout:
      phase: frozen
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX_SANDBOX_ROLLOUT_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	decision := mgr.EvaluateReadinessAdmission()
	if decision.Outcome != ReadinessAdmissionOutcomeDeny {
		t.Fatalf("outcome=%q, want deny", decision.Outcome)
	}
	if decision.ReasonCode != ReadinessAdmissionCodeSandboxFrozen {
		t.Fatalf("reason_code=%q, want %q", decision.ReasonCode, ReadinessAdmissionCodeSandboxFrozen)
	}
}

func TestManagerReadinessAdmissionSandboxCapacityPolicyMapping(t *testing.T) {
	allowFile := filepath.Join(t.TempDir(), "runtime-a52-capacity-allow.yaml")
	writeConfig(t, allowFile, `
runtime:
  readiness:
    enabled: true
    strict: false
    remote_probe_enabled: false
    admission:
      enabled: true
      mode: fail_fast
      block_on: blocked_only
      degraded_policy: allow_and_record
security:
  sandbox:
    enabled: true
    capacity:
      degraded_policy: allow_and_record
`)
	allowMgr, err := NewManager(ManagerOptions{FilePath: allowFile, EnvPrefix: "BAYMAX_SANDBOX_ROLLOUT_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = allowMgr.Close() }()
	allowMgr.SetSandboxRolloutRuntimeState(SandboxRolloutRuntimeState{CapacityAction: SandboxCapacityActionThrottle})

	allowDecision := allowMgr.EvaluateReadinessAdmission()
	if allowDecision.Outcome != ReadinessAdmissionOutcomeAllow {
		t.Fatalf("allow policy outcome=%q, want allow", allowDecision.Outcome)
	}
	if allowDecision.ReasonCode != ReadinessAdmissionCodeSandboxThrottle {
		t.Fatalf("allow policy reason_code=%q, want %q", allowDecision.ReasonCode, ReadinessAdmissionCodeSandboxThrottle)
	}

	denyFile := filepath.Join(t.TempDir(), "runtime-a52-capacity-deny.yaml")
	writeConfig(t, denyFile, `
runtime:
  readiness:
    enabled: true
    strict: false
    remote_probe_enabled: false
    admission:
      enabled: true
      mode: fail_fast
      block_on: blocked_only
      degraded_policy: allow_and_record
security:
  sandbox:
    enabled: true
    capacity:
      degraded_policy: fail_fast
`)
	denyMgr, err := NewManager(ManagerOptions{FilePath: denyFile, EnvPrefix: "BAYMAX_SANDBOX_ROLLOUT_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = denyMgr.Close() }()
	denyMgr.SetSandboxRolloutRuntimeState(SandboxRolloutRuntimeState{CapacityAction: SandboxCapacityActionThrottle})

	denyDecision := denyMgr.EvaluateReadinessAdmission()
	if denyDecision.Outcome != ReadinessAdmissionOutcomeDeny {
		t.Fatalf("fail_fast policy outcome=%q, want deny", denyDecision.Outcome)
	}
	if denyDecision.ReasonCode != ReadinessAdmissionCodeSandboxThrottledDeny {
		t.Fatalf("fail_fast policy reason_code=%q, want %q", denyDecision.ReasonCode, ReadinessAdmissionCodeSandboxThrottledDeny)
	}

	capacityDenyMgr, err := NewManager(ManagerOptions{FilePath: allowFile, EnvPrefix: "BAYMAX_SANDBOX_ROLLOUT_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = capacityDenyMgr.Close() }()
	capacityDenyMgr.SetSandboxRolloutRuntimeState(SandboxRolloutRuntimeState{CapacityAction: SandboxCapacityActionDeny})
	capacityDenyDecision := capacityDenyMgr.EvaluateReadinessAdmission()
	if capacityDenyDecision.Outcome != ReadinessAdmissionOutcomeDeny {
		t.Fatalf("capacity deny outcome=%q, want deny", capacityDenyDecision.Outcome)
	}
	if capacityDenyDecision.ReasonCode != ReadinessAdmissionCodeSandboxCapacityDeny {
		t.Fatalf("capacity deny reason_code=%q, want %q", capacityDenyDecision.ReasonCode, ReadinessAdmissionCodeSandboxCapacityDeny)
	}
}

func TestManagerReadinessAdmissionBlockedDeny(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	writeConfig(t, file, `
runtime:
  readiness:
    enabled: true
    strict: false
    remote_probe_enabled: false
    admission:
      enabled: true
      mode: fail_fast
      block_on: blocked_only
      degraded_policy: allow_and_record
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX_READINESS_ADMISSION_CONTRACT_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	mgr.SetReadinessComponentSnapshot(RuntimeReadinessComponentSnapshot{
		Recovery: RuntimeReadinessComponentState{
			Enabled:           true,
			ConfiguredBackend: "file",
			EffectiveBackend:  "memory",
			ActivationError:   "permission denied",
		},
	})
	first := mgr.EvaluateReadinessAdmission()
	second := mgr.EvaluateReadinessAdmission()

	if first.Outcome != ReadinessAdmissionOutcomeDeny {
		t.Fatalf("admission outcome = %q, want %q", first.Outcome, ReadinessAdmissionOutcomeDeny)
	}
	if first.ReasonCode != ReadinessAdmissionCodeBlocked {
		t.Fatalf("admission reason_code = %q, want %q", first.ReasonCode, ReadinessAdmissionCodeBlocked)
	}
	if first.ReadinessStatus != ReadinessStatusBlocked {
		t.Fatalf("readiness_status = %q, want %q", first.ReadinessStatus, ReadinessStatusBlocked)
	}
	if first.ReadinessPrimaryDomain != ReadinessDomainRecovery {
		t.Fatalf("readiness_primary_domain = %q, want %q", first.ReadinessPrimaryDomain, ReadinessDomainRecovery)
	}
	if first.ReadinessPrimaryCode != ReadinessCodeRecoveryActivationError {
		t.Fatalf("readiness_primary_code = %q, want %q", first.ReadinessPrimaryCode, ReadinessCodeRecoveryActivationError)
	}
	if first.ReadinessPrimarySource != RuntimePrimarySourceReadiness {
		t.Fatalf("readiness_primary_source = %q, want %q", first.ReadinessPrimarySource, RuntimePrimarySourceReadiness)
	}
	if first.ReadinessSecondaryReasonCount != 0 || len(first.ReadinessSecondaryReasonCodes) != 0 {
		t.Fatalf("blocked admission should have empty secondary reasons, got %#v", first)
	}
	if first.ReadinessArbitrationRuleVersion != RuntimeArbitrationRuleVersionExplainabilityV1 {
		t.Fatalf("readiness_arbitration_rule_version = %q, want %q", first.ReadinessArbitrationRuleVersion, RuntimeArbitrationRuleVersionExplainabilityV1)
	}
	if first.ReadinessRemediationHintCode != "recovery.fix_activation" || first.ReadinessRemediationHintDomain != ReadinessDomainRecovery {
		t.Fatalf("readiness remediation hint mismatch: %#v", first)
	}
	if readinessAdmissionFingerprint(first) != readinessAdmissionFingerprint(second) {
		t.Fatalf("admission decision should be deterministic first=%s second=%s", readinessAdmissionFingerprint(first), readinessAdmissionFingerprint(second))
	}
}

func TestManagerReadinessAdmissionReadyAllowAndDegradedPolicyMapping(t *testing.T) {
	readyFile := filepath.Join(t.TempDir(), "runtime-ready.yaml")
	writeConfig(t, readyFile, `
runtime:
  readiness:
    enabled: true
    strict: false
    remote_probe_enabled: false
    admission:
      enabled: true
      mode: fail_fast
      block_on: blocked_only
      degraded_policy: allow_and_record
`)
	readyMgr, err := NewManager(ManagerOptions{FilePath: readyFile, EnvPrefix: "BAYMAX_READINESS_ADMISSION_CONTRACT_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = readyMgr.Close() }()

	readyDecision := readyMgr.EvaluateReadinessAdmission()
	if readyDecision.Outcome != ReadinessAdmissionOutcomeAllow {
		t.Fatalf("ready admission outcome = %q, want allow", readyDecision.Outcome)
	}
	if readyDecision.ReasonCode != ReadinessAdmissionCodeReady {
		t.Fatalf("ready admission reason_code = %q, want %q", readyDecision.ReasonCode, ReadinessAdmissionCodeReady)
	}

	allowFile := filepath.Join(t.TempDir(), "runtime-degraded-allow.yaml")
	writeConfig(t, allowFile, `
runtime:
  readiness:
    enabled: true
    strict: false
    remote_probe_enabled: false
    admission:
      enabled: true
      mode: fail_fast
      block_on: blocked_only
      degraded_policy: allow_and_record
`)
	allowMgr, err := NewManager(ManagerOptions{FilePath: allowFile, EnvPrefix: "BAYMAX_READINESS_ADMISSION_CONTRACT_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = allowMgr.Close() }()
	allowMgr.SetReadinessComponentSnapshot(RuntimeReadinessComponentSnapshot{
		Scheduler: RuntimeReadinessComponentState{
			Enabled:           true,
			ConfiguredBackend: "file",
			EffectiveBackend:  "memory",
			Fallback:          true,
			FallbackReason:    "scheduler.backend.file_init_failed",
		},
	})
	allowDecision := allowMgr.EvaluateReadinessAdmission()
	if allowDecision.Outcome != ReadinessAdmissionOutcomeAllow {
		t.Fatalf("degraded allow outcome = %q, want allow", allowDecision.Outcome)
	}
	if allowDecision.ReasonCode != ReadinessAdmissionCodeDegradedAllow {
		t.Fatalf("degraded allow reason_code = %q, want %q", allowDecision.ReasonCode, ReadinessAdmissionCodeDegradedAllow)
	}
	if allowDecision.ReadinessPrimaryCode != ReadinessCodeSchedulerFallback {
		t.Fatalf("degraded allow primary_code = %q, want %q", allowDecision.ReadinessPrimaryCode, ReadinessCodeSchedulerFallback)
	}
	if allowDecision.ReadinessPrimaryDomain != ReadinessDomainScheduler ||
		allowDecision.ReadinessPrimarySource != RuntimePrimarySourceReadiness {
		t.Fatalf("degraded allow primary metadata mismatch: %#v", allowDecision)
	}
	if allowDecision.ReadinessArbitrationRuleVersion != RuntimeArbitrationRuleVersionExplainabilityV1 ||
		allowDecision.ReadinessRemediationHintCode != "scheduler.recover_backend" ||
		allowDecision.ReadinessRemediationHintDomain != ReadinessDomainScheduler {
		t.Fatalf("degraded allow explainability mismatch: %#v", allowDecision)
	}

	denyFile := filepath.Join(t.TempDir(), "runtime-degraded-deny.yaml")
	writeConfig(t, denyFile, `
runtime:
  readiness:
    enabled: true
    strict: false
    remote_probe_enabled: false
    admission:
      enabled: true
      mode: fail_fast
      block_on: blocked_only
      degraded_policy: fail_fast
`)
	denyMgr, err := NewManager(ManagerOptions{FilePath: denyFile, EnvPrefix: "BAYMAX_READINESS_ADMISSION_CONTRACT_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = denyMgr.Close() }()
	denyMgr.SetReadinessComponentSnapshot(RuntimeReadinessComponentSnapshot{
		Scheduler: RuntimeReadinessComponentState{
			Enabled:           true,
			ConfiguredBackend: "file",
			EffectiveBackend:  "memory",
			Fallback:          true,
			FallbackReason:    "scheduler.backend.file_init_failed",
		},
	})
	denyDecision := denyMgr.EvaluateReadinessAdmission()
	if denyDecision.Outcome != ReadinessAdmissionOutcomeDeny {
		t.Fatalf("degraded fail_fast outcome = %q, want deny", denyDecision.Outcome)
	}
	if denyDecision.ReasonCode != ReadinessAdmissionCodeDegradedDeny {
		t.Fatalf("degraded fail_fast reason_code = %q, want %q", denyDecision.ReasonCode, ReadinessAdmissionCodeDegradedDeny)
	}
	if denyDecision.ReadinessPrimaryCode != ReadinessCodeSchedulerFallback {
		t.Fatalf("degraded fail_fast primary_code = %q, want %q", denyDecision.ReadinessPrimaryCode, ReadinessCodeSchedulerFallback)
	}
	if denyDecision.ReadinessPrimaryDomain != ReadinessDomainScheduler ||
		denyDecision.ReadinessPrimarySource != RuntimePrimarySourceReadiness {
		t.Fatalf("degraded fail_fast primary metadata mismatch: %#v", denyDecision)
	}
	if denyDecision.ReadinessArbitrationRuleVersion != RuntimeArbitrationRuleVersionExplainabilityV1 ||
		denyDecision.ReadinessRemediationHintCode != "scheduler.recover_backend" ||
		denyDecision.ReadinessRemediationHintDomain != ReadinessDomainScheduler {
		t.Fatalf("degraded deny explainability mismatch: %#v", denyDecision)
	}
}

func TestManagerReadinessAdmissionDisabledBypass(t *testing.T) {
	mgr, err := NewManager(ManagerOptions{EnvPrefix: "BAYMAX_READINESS_ADMISSION_CONTRACT_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	decision := mgr.EvaluateReadinessAdmission()
	if !decision.Bypass {
		t.Fatalf("admission bypass = false, want true: %#v", decision)
	}
	if decision.Outcome != ReadinessAdmissionOutcomeAllow {
		t.Fatalf("admission outcome = %q, want allow", decision.Outcome)
	}
	if decision.ReasonCode != ReadinessAdmissionCodeBypassDisabled {
		t.Fatalf("admission reason_code = %q, want %q", decision.ReasonCode, ReadinessAdmissionCodeBypassDisabled)
	}
	if decision.ReadinessPrimaryDomain != ReadinessDomainRuntime ||
		decision.ReadinessPrimaryCode != ReadinessAdmissionCodeBypassDisabled ||
		decision.ReadinessPrimarySource != RuntimePrimarySourceAdmission {
		t.Fatalf("disabled bypass decision primary metadata mismatch: %#v", decision)
	}
	if decision.ReadinessArbitrationRuleVersion != RuntimeArbitrationRuleVersionExplainabilityV1 ||
		decision.ReadinessRemediationHintCode != "readiness.admission_enable_if_required" ||
		decision.ReadinessRemediationHintDomain != ReadinessDomainRuntime {
		t.Fatalf("disabled bypass explainability mismatch: %#v", decision)
	}
}

func TestManagerReadinessAdmissionBudgetDecisionThresholdMapping(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime-a60-budget.yaml")
	writeConfig(t, file, `
runtime:
  readiness:
    enabled: true
    strict: false
    remote_probe_enabled: false
    admission:
      enabled: true
      mode: fail_fast
      block_on: blocked_only
      degraded_policy: allow_and_record
  admission:
    budget:
      cost:
        degrade_threshold: 0.75
        hard_threshold: 1.0
      latency:
        degrade_threshold: 1200ms
        hard_threshold: 2s
    degrade_policy:
      enabled: true
      action_order: [trim_memory_context, reduce_tool_call_limit]
      conflict_policy: first_action
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX_RUNTIME_MEMORY_GOVERNANCE_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	allowDecision := mgr.EvaluateReadinessAdmissionWithBudgetRequest("", ReadinessAdmissionRequest{
		ExplicitBudgetEstimate: &RuntimeAdmissionBudgetEstimate{
			TokenCost:      0.10,
			ToolCost:       0.08,
			SandboxCost:    0.06,
			MemoryCost:     0.04,
			TokenLatency:   200 * time.Millisecond,
			ToolLatency:    180 * time.Millisecond,
			SandboxLatency: 120 * time.Millisecond,
			MemoryLatency:  80 * time.Millisecond,
		},
	})
	if allowDecision.BudgetDecision != RuntimeAdmissionBudgetDecisionAllow ||
		allowDecision.Outcome != ReadinessAdmissionOutcomeAllow ||
		strings.TrimSpace(allowDecision.DegradeAction) != "" {
		t.Fatalf("budget allow decision mismatch: %#v", allowDecision)
	}
	if allowDecision.BudgetSnapshot == nil ||
		allowDecision.BudgetSnapshot.CostEstimate.Total <= 0 ||
		allowDecision.BudgetSnapshot.LatencyEstimate.TotalMs <= 0 {
		t.Fatalf("budget allow snapshot mismatch: %#v", allowDecision.BudgetSnapshot)
	}

	degradeDecision := mgr.EvaluateReadinessAdmissionWithBudgetRequest("", ReadinessAdmissionRequest{
		ExplicitBudgetEstimate: &RuntimeAdmissionBudgetEstimate{
			TokenCost:      0.36,
			ToolCost:       0.24,
			SandboxCost:    0.20,
			MemoryCost:     0.08,
			TokenLatency:   320 * time.Millisecond,
			ToolLatency:    260 * time.Millisecond,
			SandboxLatency: 210 * time.Millisecond,
			MemoryLatency:  140 * time.Millisecond,
		},
	})
	if degradeDecision.BudgetDecision != RuntimeAdmissionBudgetDecisionDegrade ||
		degradeDecision.Outcome != ReadinessAdmissionOutcomeAllow ||
		degradeDecision.ReasonCode != ReadinessAdmissionCodeBudgetDegradeAllow ||
		degradeDecision.DegradeAction != RuntimeAdmissionDegradeActionTrimMemoryContext {
		t.Fatalf("budget degrade decision mismatch: %#v", degradeDecision)
	}

	denyDecision := mgr.EvaluateReadinessAdmissionWithBudgetRequest("", ReadinessAdmissionRequest{
		ExplicitBudgetEstimate: &RuntimeAdmissionBudgetEstimate{
			TokenCost:      0.45,
			ToolCost:       0.30,
			SandboxCost:    0.22,
			MemoryCost:     0.18,
			TokenLatency:   700 * time.Millisecond,
			ToolLatency:    560 * time.Millisecond,
			SandboxLatency: 420 * time.Millisecond,
			MemoryLatency:  330 * time.Millisecond,
		},
	})
	if denyDecision.BudgetDecision != RuntimeAdmissionBudgetDecisionDeny ||
		denyDecision.Outcome != ReadinessAdmissionOutcomeDeny ||
		denyDecision.ReasonCode != ReadinessAdmissionCodeBudgetHardDeny ||
		strings.TrimSpace(denyDecision.DegradeAction) != "" {
		t.Fatalf("budget deny decision mismatch: %#v", denyDecision)
	}
}

func TestManagerReadinessAdmissionBudgetSnapshotDeterministicEquivalentInput(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime-a60-deterministic.yaml")
	writeConfig(t, file, `
runtime:
  readiness:
    enabled: true
    strict: false
    remote_probe_enabled: false
    admission:
      enabled: true
      mode: fail_fast
      block_on: blocked_only
      degraded_policy: allow_and_record
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX_RUNTIME_MEMORY_GOVERNANCE_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	req := ReadinessAdmissionRequest{
		InputChars:   420,
		MessageCount: 4,
		RequiredCapabilities: []types.ModelCapability{
			types.ModelCapabilityToolCall,
		},
	}
	first := mgr.EvaluateReadinessAdmissionWithBudgetRequest("", req)
	second := mgr.EvaluateReadinessAdmissionWithBudgetRequest("", req)

	if first.BudgetDecision != second.BudgetDecision || first.DegradeAction != second.DegradeAction {
		t.Fatalf("budget decision determinism mismatch first=%#v second=%#v", first, second)
	}
	if first.BudgetSnapshot == nil || second.BudgetSnapshot == nil {
		t.Fatalf("budget snapshot should be present first=%#v second=%#v", first.BudgetSnapshot, second.BudgetSnapshot)
	}
	firstSnapshot := *first.BudgetSnapshot
	secondSnapshot := *second.BudgetSnapshot
	firstSnapshot.EvaluatedAt = time.Time{}
	secondSnapshot.EvaluatedAt = time.Time{}
	firstBlob, _ := json.Marshal(firstSnapshot)
	secondBlob, _ := json.Marshal(secondSnapshot)
	if string(firstBlob) != string(secondBlob) {
		t.Fatalf("budget snapshot semantic drift first=%s second=%s", string(firstBlob), string(secondBlob))
	}
}

func TestManagerReadinessPreflightWithRequestArbitrationVersionUnsupported(t *testing.T) {
	mgr, err := NewManager(ManagerOptions{EnvPrefix: "BAYMAX_ARBITRATION_VERSION_GOVERNANCE_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	result := mgr.ReadinessPreflightWithRequest("a77.v9")
	if result.Status != ReadinessStatusBlocked {
		t.Fatalf("status=%q, want blocked", result.Status)
	}
	assertReadinessFindingCode(t, result.Findings, ReadinessCodeArbitrationVersionUnsupported)
	summary := result.Summary()
	if summary.PrimaryDomain != ReadinessDomainRuntime ||
		summary.PrimaryCode != ReadinessCodeArbitrationVersionUnsupported ||
		summary.PrimarySource != RuntimePrimarySourceArbitration ||
		summary.ArbitrationRuleVersion != "" ||
		summary.ArbitrationRuleRequestedVersion != "a77.v9" ||
		summary.ArbitrationRuleEffectiveVersion != "" ||
		summary.ArbitrationRuleVersionSource != RuntimeArbitrationVersionSourceRequested ||
		summary.ArbitrationRulePolicyAction != RuntimeArbitrationPolicyActionFailFastUnsupported ||
		summary.ArbitrationRuleUnsupportedTotal != 1 ||
		summary.ArbitrationRuleMismatchTotal != 0 {
		t.Fatalf("a50 preflight summary mismatch: %#v", summary)
	}
}

func TestManagerReadinessAdmissionWithRequestArbitrationVersionMismatchDeny(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime-a50.yaml")
	writeConfig(t, file, `
runtime:
  readiness:
    enabled: true
    strict: false
    remote_probe_enabled: false
    admission:
      enabled: true
      mode: fail_fast
      block_on: blocked_only
      degraded_policy: allow_and_record
  arbitration:
    version:
      enabled: true
      default: a49.v1
      compat_window: 0
      on_unsupported: fail_fast
      on_mismatch: fail_fast
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX_ARBITRATION_VERSION_GOVERNANCE_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	decision := mgr.EvaluateReadinessAdmissionWithRequest(RuntimeArbitrationRuleVersionPrimaryReasonV1)
	if decision.Outcome != ReadinessAdmissionOutcomeDeny ||
		decision.ReasonCode != ReadinessCodeArbitrationVersionMismatch ||
		decision.ReadinessStatus != ReadinessStatusBlocked ||
		decision.ReadinessPrimaryDomain != ReadinessDomainRuntime ||
		decision.ReadinessPrimaryCode != ReadinessCodeArbitrationVersionMismatch ||
		decision.ReadinessPrimarySource != RuntimePrimarySourceArbitration ||
		decision.ReadinessArbitrationRuleVersion != "" ||
		decision.ReadinessArbitrationRuleRequestedVersion != RuntimeArbitrationRuleVersionPrimaryReasonV1 ||
		decision.ReadinessArbitrationRuleEffectiveVersion != "" ||
		decision.ReadinessArbitrationRuleVersionSource != RuntimeArbitrationVersionSourceRequested ||
		decision.ReadinessArbitrationRulePolicyAction != RuntimeArbitrationPolicyActionFailFastMismatch ||
		decision.ReadinessArbitrationRuleUnsupportedTotal != 0 ||
		decision.ReadinessArbitrationRuleMismatchTotal != 1 ||
		decision.ReadinessRemediationHintCode != "runtime.align_arbitration_compat_window" ||
		decision.ReadinessRemediationHintDomain != ReadinessDomainRuntime {
		t.Fatalf("a50 admission mismatch deny decision: %#v", decision)
	}
}

func TestSandboxEgressReadinessFindingsPolicyAllowlistAndBudget(t *testing.T) {
	findings := sandboxEgressReadinessFindings(
		SecuritySandboxEgressConfig{
			Enabled:       true,
			DefaultAction: "invalid",
			ByTool: map[string]string{
				"bad selector": SecuritySandboxEgressActionDeny,
			},
			Allowlist:   []string{"bad/path", "api.example.com", "api.example.com"},
			OnViolation: "invalid",
		},
		SandboxRolloutRuntimeState{
			EgressViolationTotal:  3,
			EgressViolationBudget: 1,
			EgressBudgetBreached:  true,
		},
		map[string]any{"sandbox_enabled": true},
	)
	assertReadinessFindingCode(t, findings, ReadinessCodeSandboxEgressPolicyInvalid)
	assertReadinessFindingCode(t, findings, ReadinessCodeSandboxEgressAllowlistInvalid)
	assertReadinessFindingCode(t, findings, ReadinessCodeSandboxEgressViolationBudgetBreached)
	assertReadinessCanonicalFields(t, canonicalizeReadinessFindings(findings))
}

func TestSandboxEgressReadinessFindingsRuleConflictRecoverable(t *testing.T) {
	findings := sandboxEgressReadinessFindings(
		SecuritySandboxEgressConfig{
			Enabled:       true,
			DefaultAction: SecuritySandboxEgressActionAllow,
			Allowlist:     []string{"api.example.com"},
			OnViolation:   SecuritySandboxEgressOnViolationDeny,
		},
		SandboxRolloutRuntimeState{},
		map[string]any{"sandbox_enabled": true},
	)
	assertReadinessFindingCode(t, findings, ReadinessCodeSandboxEgressRuleConflict)
	for i := range findings {
		if findings[i].Code == ReadinessCodeSandboxEgressRuleConflict && findings[i].Severity != ReadinessSeverityWarning {
			t.Fatalf("sandbox egress rule conflict severity=%q, want warning", findings[i].Severity)
		}
	}
}

func TestManagerReadinessPreflightSandboxEgressViolationBudgetBreached(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime-a57-egress-budget.yaml")
	writeConfig(t, file, `
runtime:
  readiness:
    enabled: true
    strict: false
    remote_probe_enabled: false
security:
  sandbox:
    enabled: true
    required: false
    mode: observe
    policy:
      default_action: sandbox
      profile: default
      fallback_action: allow_and_record
    egress:
      enabled: true
      default_action: deny
      on_violation: deny
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX_SECURITY_EVENT_GOVERNANCE_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()
	mgr.SetSandboxExecutor(&fakeSandboxExecutor{
		probe: func(context.Context) (types.SandboxCapabilityProbe, error) {
			return types.SandboxCapabilityProbe{
				Backend: sandboxBackendForCurrentHost(),
			}, nil
		},
	})
	mgr.SetSandboxRolloutRuntimeState(SandboxRolloutRuntimeState{
		EgressViolationTotal:  2,
		EgressViolationBudget: 1,
		EgressBudgetBreached:  true,
	})

	result := mgr.ReadinessPreflight()
	if result.Status != ReadinessStatusDegraded {
		t.Fatalf("status=%q, want degraded", result.Status)
	}
	assertReadinessFindingCode(t, result.Findings, ReadinessCodeSandboxEgressViolationBudgetBreached)
}

func TestManagerReadinessPreflightAdapterAllowlistStrictEscalation(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime-a57-allowlist-strict.yaml")
	writeConfig(t, file, `
runtime:
  readiness:
    enabled: true
    strict: true
    remote_probe_enabled: false
adapter:
  allowlist:
    enabled: true
    enforcement_mode: observe
    on_unknown_signature: allow_and_record
    entries: []
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX_SECURITY_EVENT_GOVERNANCE_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()
	mgr.SetAdapterHealthTargets([]AdapterHealthTarget{
		{
			Name:     "required-adapter-a57",
			Required: true,
			Probe: adapterhealth.ProbeFunc(func(context.Context) (adapterhealth.Result, error) {
				return adapterhealth.Result{Status: adapterhealth.StatusHealthy, Code: adapterhealth.CodeHealthy}, nil
			}),
		},
	})

	result := mgr.ReadinessPreflight()
	if result.Status != ReadinessStatusBlocked {
		t.Fatalf("status=%q, want blocked", result.Status)
	}
	assertReadinessFindingCode(t, result.Findings, ReadinessCodeAdapterAllowlistMissingEntry)
	assertReadinessFindingCode(t, result.Findings, ReadinessCodeStrictEscalated)
}

func TestManagerReadinessAdmissionAdapterAllowlistMissingEntryDeny(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime-a57-admission-allowlist-missing.yaml")
	writeConfig(t, file, `
runtime:
  readiness:
    enabled: true
    strict: false
    remote_probe_enabled: false
    admission:
      enabled: true
      mode: fail_fast
      block_on: blocked_only
      degraded_policy: allow_and_record
adapter:
  allowlist:
    enabled: true
    enforcement_mode: enforce
    on_unknown_signature: deny
    entries: []
`)
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX_SECURITY_EVENT_GOVERNANCE_TEST"})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()
	mgr.SetAdapterHealthTargets([]AdapterHealthTarget{
		{
			Name:     "required-adapter-a57",
			Required: true,
			Probe: adapterhealth.ProbeFunc(func(context.Context) (adapterhealth.Result, error) {
				return adapterhealth.Result{Status: adapterhealth.StatusHealthy, Code: adapterhealth.CodeHealthy}, nil
			}),
		},
	})

	decision := mgr.EvaluateReadinessAdmission()
	if decision.Outcome != ReadinessAdmissionOutcomeDeny {
		t.Fatalf("outcome=%q, want deny", decision.Outcome)
	}
	if decision.ReasonCode != ReadinessAdmissionCodeBlocked {
		t.Fatalf("reason_code=%q, want %q", decision.ReasonCode, ReadinessAdmissionCodeBlocked)
	}
	if decision.ReadinessPrimaryDomain != ReadinessDomainAdapter ||
		decision.ReadinessPrimaryCode != ReadinessCodeAdapterAllowlistMissingEntry ||
		decision.ReadinessPrimarySource != RuntimePrimarySourceAdapter ||
		decision.ReadinessRemediationHintCode != "adapter.allowlist.add_required_entry" ||
		decision.ReadinessRemediationHintDomain != ReadinessDomainAdapter {
		t.Fatalf("allowlist admission explainability mismatch: %#v", decision)
	}
}

func sandboxBackendForCurrentHost() string {
	if strings.EqualFold(runtime.GOOS, "windows") {
		return SecuritySandboxBackendWindowsJob
	}
	return SecuritySandboxBackendLinuxNSJail
}

func sandboxUnsupportedBackendForCurrentHost() string {
	if strings.EqualFold(runtime.GOOS, "windows") {
		return SecuritySandboxBackendLinuxNSJail
	}
	return SecuritySandboxBackendWindowsJob
}

func assertReadinessFindingCode(t *testing.T, findings []ReadinessFinding, code string) {
	t.Helper()
	for i := range findings {
		if strings.TrimSpace(findings[i].Code) == strings.TrimSpace(code) {
			return
		}
	}
	t.Fatalf("finding code %q not found in %#v", code, findings)
}

func assertReadinessCanonicalFields(t *testing.T, findings []ReadinessFinding) {
	t.Helper()
	for i := range findings {
		item := findings[i]
		if strings.TrimSpace(item.Code) == "" {
			t.Fatalf("finding[%d] code is empty: %#v", i, item)
		}
		if strings.TrimSpace(item.Domain) == "" {
			t.Fatalf("finding[%d] domain is empty: %#v", i, item)
		}
		if strings.TrimSpace(item.Severity) == "" {
			t.Fatalf("finding[%d] severity is empty: %#v", i, item)
		}
		if strings.TrimSpace(item.Message) == "" {
			t.Fatalf("finding[%d] message is empty: %#v", i, item)
		}
		if item.Metadata == nil {
			t.Fatalf("finding[%d] metadata is nil: %#v", i, item)
		}
	}
}

func readinessSemanticFingerprint(result ReadinessResult) string {
	payload := struct {
		Status   ReadinessStatus    `json:"status"`
		Findings []ReadinessFinding `json:"findings"`
	}{
		Status:   result.Status,
		Findings: result.Findings,
	}
	blob, _ := json.Marshal(payload)
	return string(blob)
}

func readinessAdmissionFingerprint(decision ReadinessAdmissionDecision) string {
	clone := decision
	if clone.BudgetSnapshot != nil {
		snapshot := *clone.BudgetSnapshot
		snapshot.EvaluatedAt = time.Time{}
		clone.BudgetSnapshot = &snapshot
	}
	blob, _ := json.Marshal(clone)
	return string(blob)
}
