package config

import "testing"

func TestArbitratePrimaryReasonNoCandidate(t *testing.T) {
	got := ArbitratePrimaryReason(PrimaryReasonArbitrationInput{})
	if got.Domain != "" || got.Code != "" || got.Source != "" || got.ConflictTotal != 0 ||
		got.SecondaryCount != 0 || len(got.SecondaryCodes) != 0 ||
		got.RuleVersion != RuntimeArbitrationRuleVersionA49V1 ||
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
		got.RuleVersion != RuntimeArbitrationRuleVersionA49V1 ||
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
