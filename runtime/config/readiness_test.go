package config

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	adapterhealth "github.com/FelixSeptem/baymax/adapter/health"
)

func TestManagerReadinessPreflightClassificationMatrix(t *testing.T) {
	mgr, err := NewManager(ManagerOptions{EnvPrefix: "BAYMAX_A40_TEST"})
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
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX_A40_TEST"})
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
		summary.ArbitrationRuleVersion != RuntimeArbitrationRuleVersionA49V1 ||
		summary.RemediationHintCode != "runtime.relax_strict_mode" ||
		summary.RemediationHintDomain != ReadinessDomainRuntime {
		t.Fatalf("summary primary arbitration mismatch: %#v", summary)
	}
}

func TestManagerReadinessPreflightDeterministicForEquivalentSnapshot(t *testing.T) {
	mgr, err := NewManager(ManagerOptions{EnvPrefix: "BAYMAX_A40_TEST"})
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
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX_A43_TEST"})
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
		summary.ArbitrationRuleVersion != RuntimeArbitrationRuleVersionA49V1 ||
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
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX_A43_TEST"})
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
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX_A43_TEST"})
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
	strictMgr, err := NewManager(ManagerOptions{FilePath: strictFile, EnvPrefix: "BAYMAX_A46_TEST"})
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
	nonStrictMgr, err := NewManager(ManagerOptions{FilePath: nonStrictFile, EnvPrefix: "BAYMAX_A46_TEST"})
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
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX_A46_TEST"})
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
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX_A44_TEST"})
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
	if first.ReadinessArbitrationRuleVersion != RuntimeArbitrationRuleVersionA49V1 {
		t.Fatalf("readiness_arbitration_rule_version = %q, want %q", first.ReadinessArbitrationRuleVersion, RuntimeArbitrationRuleVersionA49V1)
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
	readyMgr, err := NewManager(ManagerOptions{FilePath: readyFile, EnvPrefix: "BAYMAX_A44_TEST"})
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
	allowMgr, err := NewManager(ManagerOptions{FilePath: allowFile, EnvPrefix: "BAYMAX_A44_TEST"})
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
	if allowDecision.ReadinessArbitrationRuleVersion != RuntimeArbitrationRuleVersionA49V1 ||
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
	denyMgr, err := NewManager(ManagerOptions{FilePath: denyFile, EnvPrefix: "BAYMAX_A44_TEST"})
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
	if denyDecision.ReadinessArbitrationRuleVersion != RuntimeArbitrationRuleVersionA49V1 ||
		denyDecision.ReadinessRemediationHintCode != "scheduler.recover_backend" ||
		denyDecision.ReadinessRemediationHintDomain != ReadinessDomainScheduler {
		t.Fatalf("degraded deny explainability mismatch: %#v", denyDecision)
	}
}

func TestManagerReadinessAdmissionDisabledBypass(t *testing.T) {
	mgr, err := NewManager(ManagerOptions{EnvPrefix: "BAYMAX_A44_TEST"})
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
	if decision.ReadinessArbitrationRuleVersion != RuntimeArbitrationRuleVersionA49V1 ||
		decision.ReadinessRemediationHintCode != "readiness.admission_enable_if_required" ||
		decision.ReadinessRemediationHintDomain != ReadinessDomainRuntime {
		t.Fatalf("disabled bypass explainability mismatch: %#v", decision)
	}
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
	blob, _ := json.Marshal(decision)
	return string(blob)
}
