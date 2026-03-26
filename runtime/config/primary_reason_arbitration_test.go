package config

import "testing"

func TestArbitratePrimaryReasonNoCandidate(t *testing.T) {
	got := ArbitratePrimaryReason(PrimaryReasonArbitrationInput{})
	if got.Domain != "" || got.Code != "" || got.Source != "" || got.ConflictTotal != 0 {
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
		got.ConflictTotal != 0 {
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
}
