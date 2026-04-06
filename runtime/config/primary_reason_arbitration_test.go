package config

import "testing"

func TestArbitratePrimaryReasonNoCandidate(t *testing.T) {
	got := ArbitratePrimaryReason(PrimaryReasonArbitrationInput{})
	if got.Domain != "" || got.Code != "" || got.Source != "" || got.ConflictTotal != 0 ||
		got.SecondaryCount != 0 || len(got.SecondaryCodes) != 0 ||
		got.RuleVersion != RuntimeArbitrationRuleVersionExplainabilityV1 ||
		got.RemediationHintCode != "" || got.RemediationHintDomain != "" {
		t.Fatalf("no-candidate arbitration should return zero value, got %#v", got)
	}
}

func TestArbitratePrimaryReasonTimeoutRejectOutranksReadinessBlocked(t *testing.T) {
	got := ArbitratePrimaryReason(PrimaryReasonArbitrationInput{
		TimeoutParentBudgetRejectTotal: 1,
		TimeoutResolutionSource:        TimeoutResolutionSourceRequest,
		ReadinessFindings: []ReadinessFinding{
			{
				Code:     ReadinessCodeRecoveryActivationError,
				Domain:   ReadinessDomainRecovery,
				Severity: ReadinessSeverityError,
			},
		},
	})
	if got.Domain != "timeout" ||
		got.Code != RuntimePrimaryCodeTimeoutRejected ||
		got.Source != RuntimePrimarySourceTimeout+".request" ||
		got.ConflictTotal != 0 ||
		got.SecondaryCount != 1 ||
		len(got.SecondaryCodes) != 1 ||
		got.SecondaryCodes[0] != ReadinessCodeRecoveryActivationError ||
		got.RuleVersion != RuntimeArbitrationRuleVersionExplainabilityV1 ||
		got.RemediationHintCode != "timeout.adjust_parent_budget" ||
		got.RemediationHintDomain != "timeout" {
		t.Fatalf("timeout reject must outrank blocked readiness, got %#v", got)
	}
}

func TestArbitratePrimaryReasonReadinessBlockedOutranksRequiredUnavailable(t *testing.T) {
	got := ArbitratePrimaryReason(PrimaryReasonArbitrationInput{
		ReadinessFindings: []ReadinessFinding{
			{
				Code:     ReadinessCodeAdapterRequiredUnavailable,
				Domain:   ReadinessDomainAdapter,
				Severity: ReadinessSeverityWarning,
			},
			{
				Code:     ReadinessCodeRecoveryActivationError,
				Domain:   ReadinessDomainRecovery,
				Severity: ReadinessSeverityError,
			},
		},
	})
	if got.Domain != ReadinessDomainRecovery || got.Code != ReadinessCodeRecoveryActivationError || got.Source != RuntimePrimarySourceReadiness {
		t.Fatalf("readiness blocked must outrank required unavailable, got %#v", got)
	}
}

func TestArbitratePrimaryReasonRequiredUnavailableOutranksDegradedAndOptional(t *testing.T) {
	got := ArbitratePrimaryReason(PrimaryReasonArbitrationInput{
		ReadinessFindings: []ReadinessFinding{
			{
				Code:     ReadinessCodeSchedulerFallback,
				Domain:   ReadinessDomainScheduler,
				Severity: ReadinessSeverityWarning,
			},
			{
				Code:     ReadinessCodeAdapterOptionalUnavailable,
				Domain:   ReadinessDomainAdapter,
				Severity: ReadinessSeverityWarning,
			},
			{
				Code:     ReadinessCodeAdapterRequiredUnavailable,
				Domain:   ReadinessDomainAdapter,
				Severity: ReadinessSeverityWarning,
			},
		},
	})
	if got.Domain != ReadinessDomainAdapter || got.Code != ReadinessCodeAdapterRequiredUnavailable || got.Source != RuntimePrimarySourceAdapter {
		t.Fatalf("required unavailable must outrank degraded/optional, got %#v", got)
	}
}

func TestArbitratePrimaryReasonSandboxRequiredOutranksOptional(t *testing.T) {
	got := ArbitratePrimaryReason(PrimaryReasonArbitrationInput{
		ReadinessFindings: []ReadinessFinding{
			{
				Code:     ReadinessCodeSandboxOptionalUnavailable,
				Domain:   ReadinessDomainRuntime,
				Severity: ReadinessSeverityWarning,
			},
			{
				Code:     ReadinessCodeSandboxRequiredUnavailable,
				Domain:   ReadinessDomainRuntime,
				Severity: ReadinessSeverityError,
			},
			{
				Code:     ReadinessCodeSchedulerFallback,
				Domain:   ReadinessDomainScheduler,
				Severity: ReadinessSeverityWarning,
			},
		},
	})
	if got.Domain != ReadinessDomainRuntime ||
		got.Code != ReadinessCodeSandboxRequiredUnavailable ||
		got.Source != RuntimePrimarySourceReadiness ||
		got.RemediationHintCode != "sandbox.restore_required_executor" ||
		got.RemediationHintDomain != ReadinessDomainRuntime {
		t.Fatalf("sandbox required unavailable must outrank optional/degraded, got %#v", got)
	}
}

func TestArbitratePrimaryReasonReactProviderUnsupportedOutranksRecoverableReactFindings(t *testing.T) {
	got := ArbitratePrimaryReason(PrimaryReasonArbitrationInput{
		ReadinessFindings: []ReadinessFinding{
			{
				Code:     ReadinessCodeReactToolRegistryUnavailable,
				Domain:   ReadinessDomainRuntime,
				Severity: ReadinessSeverityWarning,
			},
			{
				Code:     ReadinessCodeReactProviderToolCallingUnsupported,
				Domain:   ReadinessDomainRuntime,
				Severity: ReadinessSeverityError,
			},
		},
	})
	if got.Domain != ReadinessDomainRuntime ||
		got.Code != ReadinessCodeReactProviderToolCallingUnsupported ||
		got.Source != RuntimePrimarySourceReadiness ||
		got.SecondaryCount != 1 ||
		len(got.SecondaryCodes) != 1 ||
		got.SecondaryCodes[0] != ReadinessCodeReactToolRegistryUnavailable ||
		got.RemediationHintCode != "react.select_tool_calling_provider" ||
		got.RemediationHintDomain != ReadinessDomainRuntime {
		t.Fatalf("react precedence mismatch: %#v", got)
	}
}

func TestArbitratePrimaryReasonSandboxEgressPolicyInvalidOutranksRuleConflict(t *testing.T) {
	got := ArbitratePrimaryReason(PrimaryReasonArbitrationInput{
		ReadinessFindings: []ReadinessFinding{
			{
				Code:     ReadinessCodeSandboxEgressRuleConflict,
				Domain:   ReadinessDomainRuntime,
				Severity: ReadinessSeverityWarning,
			},
			{
				Code:     ReadinessCodeSandboxEgressPolicyInvalid,
				Domain:   ReadinessDomainRuntime,
				Severity: ReadinessSeverityError,
			},
		},
	})
	if got.Domain != ReadinessDomainRuntime ||
		got.Code != ReadinessCodeSandboxEgressPolicyInvalid ||
		got.Source != RuntimePrimarySourceReadiness ||
		got.SecondaryCount != 1 ||
		len(got.SecondaryCodes) != 1 ||
		got.SecondaryCodes[0] != ReadinessCodeSandboxEgressRuleConflict ||
		got.RemediationHintCode != "sandbox.egress.fix_policy" ||
		got.RemediationHintDomain != ReadinessDomainRuntime {
		t.Fatalf("sandbox egress precedence mismatch: %#v", got)
	}
}

func TestArbitratePrimaryReasonAdapterAllowlistMissingEntryUsesAdapterSource(t *testing.T) {
	got := ArbitratePrimaryReason(PrimaryReasonArbitrationInput{
		ReadinessFindings: []ReadinessFinding{
			{
				Code:     ReadinessCodeSchedulerFallback,
				Domain:   ReadinessDomainScheduler,
				Severity: ReadinessSeverityWarning,
			},
			{
				Code:     ReadinessCodeAdapterAllowlistMissingEntry,
				Domain:   ReadinessDomainAdapter,
				Severity: ReadinessSeverityError,
			},
		},
	})
	if got.Domain != ReadinessDomainAdapter ||
		got.Code != ReadinessCodeAdapterAllowlistMissingEntry ||
		got.Source != RuntimePrimarySourceAdapter ||
		got.RemediationHintCode != "adapter.allowlist.add_required_entry" ||
		got.RemediationHintDomain != ReadinessDomainAdapter {
		t.Fatalf("adapter allowlist arbitration mismatch: %#v", got)
	}
}

func TestArbitratePrimaryReasonTieBreakByLexicalCodeAndConflictCount(t *testing.T) {
	got := ArbitratePrimaryReason(PrimaryReasonArbitrationInput{
		ReadinessFindings: []ReadinessFinding{
			{
				Code:     ReadinessCodeSchedulerFallback,
				Domain:   ReadinessDomainScheduler,
				Severity: ReadinessSeverityWarning,
			},
			{
				Code:     ReadinessCodeMailboxFallback,
				Domain:   ReadinessDomainMailbox,
				Severity: ReadinessSeverityWarning,
			},
		},
	})
	if got.Code != ReadinessCodeMailboxFallback {
		t.Fatalf("tie-break should pick lexical min code, got %q", got.Code)
	}
	if got.ConflictTotal != 1 {
		t.Fatalf("top-level same-precedence conflict total = %d, want 1", got.ConflictTotal)
	}
}

func TestArbitratePrimaryReasonDuplicateCanonicalCodeDoesNotInflateConflict(t *testing.T) {
	got := ArbitratePrimaryReason(PrimaryReasonArbitrationInput{
		ReadinessFindings: []ReadinessFinding{
			{
				Code:     ReadinessCodeAdapterOptionalUnavailable,
				Domain:   ReadinessDomainAdapter,
				Severity: ReadinessSeverityWarning,
			},
			{
				Code:     ReadinessCodeAdapterOptionalUnavailable,
				Domain:   ReadinessDomainAdapter,
				Severity: ReadinessSeverityWarning,
			},
		},
	})
	if got.Code != ReadinessCodeAdapterOptionalUnavailable {
		t.Fatalf("expected optional unavailable as primary, got %#v", got)
	}
	if got.ConflictTotal != 0 {
		t.Fatalf("duplicate canonical code should not inflate conflict total, got %d", got.ConflictTotal)
	}
}

func TestArbitratePrimaryReasonTimeoutClampFallsBackToDegradedWhenPresent(t *testing.T) {
	got := ArbitratePrimaryReason(PrimaryReasonArbitrationInput{
		TimeoutParentBudgetClampTotal: 1,
		TimeoutResolutionSource:       TimeoutResolutionSourceDomain,
		ReadinessFindings: []ReadinessFinding{
			{
				Code:     ReadinessCodeSchedulerFallback,
				Domain:   ReadinessDomainScheduler,
				Severity: ReadinessSeverityWarning,
			},
		},
	})
	if got.Code != ReadinessCodeSchedulerFallback || got.Domain != ReadinessDomainScheduler || got.Source != RuntimePrimarySourceReadiness {
		t.Fatalf("clamp should not outrank degraded readiness, got %#v", got)
	}
	if got.SecondaryCount != 1 || len(got.SecondaryCodes) != 1 || got.SecondaryCodes[0] != RuntimePrimaryCodeTimeoutClamped {
		t.Fatalf("secondary reasons should keep timeout clamp candidate, got %#v", got)
	}
}

func TestArbitratePrimaryReasonSecondaryReasonsBoundedDeterministicAndDeduplicated(t *testing.T) {
	got := ArbitratePrimaryReason(PrimaryReasonArbitrationInput{
		ReadinessFindings: []ReadinessFinding{
			{Code: ReadinessCodeSchedulerFallback, Domain: ReadinessDomainScheduler, Severity: ReadinessSeverityWarning},
			{Code: ReadinessCodeMailboxFallback, Domain: ReadinessDomainMailbox, Severity: ReadinessSeverityWarning},
			{Code: ReadinessCodeRecoveryFallback, Domain: ReadinessDomainRecovery, Severity: ReadinessSeverityWarning},
			{Code: ReadinessCodeAdapterOptionalUnavailable, Domain: ReadinessDomainAdapter, Severity: ReadinessSeverityWarning},
			{Code: ReadinessCodeAdapterOptionalUnavailable, Domain: ReadinessDomainAdapter, Severity: ReadinessSeverityWarning},
		},
	})
	if got.Code != ReadinessCodeAdapterOptionalUnavailable {
		t.Fatalf("expected adapter optional unavailable as primary, got %#v", got)
	}
	if got.SecondaryCount != 3 {
		t.Fatalf("secondary count = %d, want 3", got.SecondaryCount)
	}
	want := []string{
		ReadinessCodeMailboxFallback,
		ReadinessCodeRecoveryFallback,
		ReadinessCodeSchedulerFallback,
	}
	if len(got.SecondaryCodes) != len(want) {
		t.Fatalf("secondary codes len = %d, want %d, got %#v", len(got.SecondaryCodes), len(want), got.SecondaryCodes)
	}
	for i := range want {
		if got.SecondaryCodes[i] != want[i] {
			t.Fatalf("secondary[%d]=%q, want %q (all=%#v)", i, got.SecondaryCodes[i], want[i], got.SecondaryCodes)
		}
	}
}

func TestArbitratePrimaryReasonSecondaryCountPreservesTruncatedTotal(t *testing.T) {
	got := ArbitratePrimaryReason(PrimaryReasonArbitrationInput{
		ReadinessFindings: []ReadinessFinding{
			{Code: ReadinessCodeSchedulerFallback, Domain: ReadinessDomainScheduler, Severity: ReadinessSeverityWarning},
			{Code: ReadinessCodeMailboxFallback, Domain: ReadinessDomainMailbox, Severity: ReadinessSeverityWarning},
			{Code: ReadinessCodeRecoveryFallback, Domain: ReadinessDomainRecovery, Severity: ReadinessSeverityWarning},
			{Code: ReadinessCodeAdapterOptionalUnavailable, Domain: ReadinessDomainAdapter, Severity: ReadinessSeverityWarning},
			{Code: ReadinessCodeAdapterDegraded, Domain: ReadinessDomainAdapter, Severity: ReadinessSeverityWarning},
		},
	})
	if got.Code != ReadinessCodeAdapterDegraded {
		t.Fatalf("expected adapter degraded as primary, got %#v", got)
	}
	if got.SecondaryCount != 4 {
		t.Fatalf("secondary count should preserve total before truncation, got %d", got.SecondaryCount)
	}
	if len(got.SecondaryCodes) != RuntimeArbitrationMaxSecondary {
		t.Fatalf("secondary list should be capped at %d, got %#v", RuntimeArbitrationMaxSecondary, got.SecondaryCodes)
	}
}

func TestArbitratePrimaryReasonIncludesVersionGovernanceTraceability(t *testing.T) {
	got := ArbitratePrimaryReason(PrimaryReasonArbitrationInput{
		ReadinessFindings: []ReadinessFinding{
			{
				Code:     ReadinessCodeSchedulerFallback,
				Domain:   ReadinessDomainScheduler,
				Severity: ReadinessSeverityWarning,
			},
		},
		RequestedRuleVersion: RuntimeArbitrationRuleVersionPrimaryReasonV1,
		VersionConfig: RuntimeArbitrationVersionConfig{
			Enabled:       true,
			Default:       RuntimeArbitrationRuleVersionExplainabilityV1,
			CompatWindow:  1,
			OnUnsupported: RuntimeArbitrationVersionPolicyFailFast,
			OnMismatch:    RuntimeArbitrationVersionPolicyFailFast,
		},
	})
	if got.Code != ReadinessCodeSchedulerFallback ||
		got.Source != RuntimePrimarySourceReadiness ||
		got.RuleVersion != RuntimeArbitrationRuleVersionPrimaryReasonV1 ||
		got.RuleRequestedVersion != RuntimeArbitrationRuleVersionPrimaryReasonV1 ||
		got.RuleEffectiveVersion != RuntimeArbitrationRuleVersionPrimaryReasonV1 ||
		got.RuleVersionSource != RuntimeArbitrationVersionSourceRequested ||
		got.RulePolicyAction != RuntimeArbitrationPolicyActionNone ||
		got.RuleUnsupportedTotal != 0 ||
		got.RuleMismatchTotal != 0 {
		t.Fatalf("version governance traceability mismatch: %#v", got)
	}
}

func TestArbitratePrimaryReasonUnsupportedVersionFailFast(t *testing.T) {
	got := ArbitratePrimaryReason(PrimaryReasonArbitrationInput{
		RequestedRuleVersion: "a77.v9",
		VersionConfig: RuntimeArbitrationVersionConfig{
			Enabled:       true,
			Default:       RuntimeArbitrationRuleVersionExplainabilityV1,
			CompatWindow:  1,
			OnUnsupported: RuntimeArbitrationVersionPolicyFailFast,
			OnMismatch:    RuntimeArbitrationVersionPolicyFailFast,
		},
	})
	if got.Domain != ReadinessDomainRuntime ||
		got.Code != ReadinessCodeArbitrationVersionUnsupported ||
		got.Source != RuntimePrimarySourceArbitration ||
		got.RuleRequestedVersion != "a77.v9" ||
		got.RuleEffectiveVersion != "" ||
		got.RuleVersionSource != RuntimeArbitrationVersionSourceRequested ||
		got.RulePolicyAction != RuntimeArbitrationPolicyActionFailFastUnsupported ||
		got.RuleUnsupportedTotal != 1 ||
		got.RuleMismatchTotal != 0 ||
		got.RemediationHintCode != "runtime.select_supported_arbitration_version" ||
		got.RemediationHintDomain != ReadinessDomainRuntime {
		t.Fatalf("unsupported version fail-fast mismatch: %#v", got)
	}
}

func TestArbitratePrimaryReasonCompatibilityMismatchFailFast(t *testing.T) {
	got := ArbitratePrimaryReason(PrimaryReasonArbitrationInput{
		RequestedRuleVersion: RuntimeArbitrationRuleVersionPrimaryReasonV1,
		VersionConfig: RuntimeArbitrationVersionConfig{
			Enabled:       true,
			Default:       RuntimeArbitrationRuleVersionExplainabilityV1,
			CompatWindow:  0,
			OnUnsupported: RuntimeArbitrationVersionPolicyFailFast,
			OnMismatch:    RuntimeArbitrationVersionPolicyFailFast,
		},
	})
	if got.Domain != ReadinessDomainRuntime ||
		got.Code != ReadinessCodeArbitrationVersionMismatch ||
		got.Source != RuntimePrimarySourceArbitration ||
		got.RuleRequestedVersion != RuntimeArbitrationRuleVersionPrimaryReasonV1 ||
		got.RuleEffectiveVersion != "" ||
		got.RuleVersionSource != RuntimeArbitrationVersionSourceRequested ||
		got.RulePolicyAction != RuntimeArbitrationPolicyActionFailFastMismatch ||
		got.RuleUnsupportedTotal != 0 ||
		got.RuleMismatchTotal != 1 ||
		got.RemediationHintCode != "runtime.align_arbitration_compat_window" ||
		got.RemediationHintDomain != ReadinessDomainRuntime {
		t.Fatalf("version mismatch fail-fast mismatch: %#v", got)
	}
}

func TestArbitratePrimaryReasonUnknownPrimaryCodeFailFast(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered == nil {
			t.Fatal("expected panic for unknown primary code but function returned normally")
		}
	}()
	_ = ArbitratePrimaryReason(PrimaryReasonArbitrationInput{
		ReadinessFindings: []ReadinessFinding{
			{
				Code:     "runtime.readiness.unknown_future_code",
				Domain:   ReadinessDomainRuntime,
				Severity: ReadinessSeverityError,
			},
		},
	})
}

func TestArbitratePrimaryReasonObservabilityPolicyInvalidOutranksSinkUnavailable(t *testing.T) {
	got := ArbitratePrimaryReason(PrimaryReasonArbitrationInput{
		ReadinessFindings: []ReadinessFinding{
			{
				Code:     ReadinessCodeObservabilityExportSinkUnavailable,
				Domain:   ReadinessDomainRuntime,
				Severity: ReadinessSeverityWarning,
			},
			{
				Code:     ReadinessCodeDiagnosticsBundlePolicyInvalid,
				Domain:   ReadinessDomainRuntime,
				Severity: ReadinessSeverityError,
			},
		},
	})
	if got.Domain != ReadinessDomainRuntime ||
		got.Code != ReadinessCodeDiagnosticsBundlePolicyInvalid ||
		got.Source != RuntimePrimarySourceReadiness ||
		got.SecondaryCount != 1 ||
		len(got.SecondaryCodes) != 1 ||
		got.SecondaryCodes[0] != ReadinessCodeObservabilityExportSinkUnavailable ||
		got.RemediationHintCode != "runtime.diagnostics.bundle.fix_policy" ||
		got.RemediationHintDomain != ReadinessDomainRuntime {
		t.Fatalf("observability policy invalid precedence mismatch: %#v", got)
	}
}
