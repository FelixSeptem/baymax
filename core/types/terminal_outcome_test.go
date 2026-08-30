package types

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestTerminalOutcomeValidateAcceptsCompletedSuccess(t *testing.T) {
	outcome := TerminalOutcome{
		RunID:         "run-1",
		State:         RunStateCompleted,
		FailureFamily: FailureFamilyNone,
		Phase:         ExecutionPhasePostStart,
	}
	if err := outcome.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestTerminalOutcomeValidateRejectsIncompatibleFamilyAndState(t *testing.T) {
	outcome := TerminalOutcome{
		RunID:         "run-1",
		State:         RunStateCompleted,
		FailureFamily: FailureFamilyRuntimeFailed,
		Phase:         ExecutionPhasePostStart,
	}
	if err := outcome.Validate(); err == nil || !strings.Contains(err.Error(), "terminal_outcome") {
		t.Fatalf("Validate() error = %v, want terminal_outcome validation error", err)
	}
}

func TestTerminalOutcomeArbiterFirstTerminalWinsAndRecordsConflict(t *testing.T) {
	arbiter := NewTerminalOutcomeArbiter()
	first := TerminalOutcome{RunID: "run-1", State: RunStateCompleted, FailureFamily: FailureFamilyNone, Phase: ExecutionPhasePostStart}
	duplicate := first
	conflict := TerminalOutcome{RunID: "run-1", State: RunStateFailed, FailureFamily: FailureFamilyRuntimeFailed, Phase: ExecutionPhasePostStart, SourceReason: "provider_error"}

	if result, err := arbiter.Publish(first); err != nil || result != TerminalPublishAccepted {
		t.Fatalf("first Publish() = %v, %v", result, err)
	}
	if result, err := arbiter.Publish(duplicate); err != nil || result != TerminalPublishIdempotent {
		t.Fatalf("duplicate Publish() = %v, %v", result, err)
	}
	if result, err := arbiter.Publish(conflict); err != nil || result != TerminalPublishConflict {
		t.Fatalf("conflict Publish() = %v, %v", result, err)
	}
	got, ok := arbiter.Terminal()
	if !ok || got.State != RunStateCompleted {
		t.Fatalf("Terminal() = %#v, %v", got, ok)
	}
	if conflicts := arbiter.Conflicts(); len(conflicts) != 1 || conflicts[0].FailureFamily != FailureFamilyRuntimeFailed {
		t.Fatalf("Conflicts() = %#v", conflicts)
	}
}

func TestRunResultJSONRoundTripPreservesAbsentTerminalOutcome(t *testing.T) {
	result := RunResult{RunID: "run-legacy", FinalAnswer: "ok"}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var decoded RunResult
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.TerminalOutcome != nil {
		t.Fatalf("legacy result TerminalOutcome = %#v, want nil", decoded.TerminalOutcome)
	}
}

func TestTerminalOutcomeFromRunResultMapsFailureAndCancellation(t *testing.T) {
	failed := RunResult{RunID: "run-failed", Error: &ClassifiedError{Class: ErrModel, Message: "provider failed", Retryable: true, Details: map[string]any{"reason_code": "provider_error"}}}
	outcome, err := TerminalOutcomeFromRunResult(failed, "session-1")
	if err != nil {
		t.Fatal(err)
	}
	if outcome.State != RunStateFailed || outcome.FailureFamily != FailureFamilyRuntimeFailed || outcome.Phase != ExecutionPhasePostStart || !outcome.Retryable {
		t.Fatalf("failure outcome = %#v", outcome)
	}

	canceled := RunResult{RunID: "run-canceled", Error: &ClassifiedError{Class: ErrPolicyTimeout, Message: "canceled", Details: map[string]any{"reason_code": "context_canceled"}}}
	outcome, err = TerminalOutcomeFromRunResult(canceled, "session-1")
	if err != nil {
		t.Fatal(err)
	}
	if outcome.State != RunStateCanceled || outcome.FailureFamily != FailureFamilyCanceled {
		t.Fatalf("canceled outcome = %#v", outcome)
	}
}

func TestTerminalOutcomeFromRunResultUsesProviderReason(t *testing.T) {
	result := RunResult{RunID: "run-rate-limit", Error: &ClassifiedError{
		Class:     ErrModel,
		Message:   "rate limited",
		Retryable: true,
		Details:   map[string]any{"provider_reason": "rate_limit"},
	}}
	outcome, err := TerminalOutcomeFromRunResult(result, "session-1")
	if err != nil {
		t.Fatal(err)
	}
	if outcome.SourceReason != "rate_limit" || outcome.FailureFamily != FailureFamilyRuntimeFailed || !outcome.Retryable {
		t.Fatalf("outcome = %#v", outcome)
	}
}

func TestTerminalOutcomeFromRunResultClassifiesStartedSandboxDenialAsPostStart(t *testing.T) {
	result := RunResult{RunID: "run-sandbox-denied", Error: &ClassifiedError{
		Class:   ErrSecurity,
		Message: "tool call denied by sandbox policy",
		Details: map[string]any{"reason_code": "sandbox.policy_deny", "dispatch_phase": "tool"},
	}}
	outcome, err := TerminalOutcomeFromRunResult(result, "session-1")
	if err != nil {
		t.Fatal(err)
	}
	if outcome.FailureFamily != FailureFamilyPolicyDenied || outcome.Phase != ExecutionPhasePostStart {
		t.Fatalf("outcome = %#v", outcome)
	}
}

func TestTerminalOutcomeFromRunResultMapsToolValidationToPreExecutionRejection(t *testing.T) {
	result := RunResult{RunID: "run-tool-validation", Error: &ClassifiedError{
		Class:   ErrTool,
		Message: "input validation failed",
		Details: map[string]any{"validation": "missing required field"},
	}}
	outcome, err := TerminalOutcomeFromRunResult(result, "session-1")
	if err != nil {
		t.Fatal(err)
	}
	if outcome.FailureFamily != FailureFamilyRejected || outcome.Phase != ExecutionPhasePreExecution {
		t.Fatalf("outcome = %#v, want rejected/pre_execution", outcome)
	}
}

func TestTerminalOutcomeFromRunResultMapsSandboxLaunchFailureToPostStartRuntimeFailure(t *testing.T) {
	result := RunResult{RunID: "run-sandbox-launch", Error: &ClassifiedError{
		Class:   ErrSecurity,
		Message: "sandbox launch failed",
		Details: map[string]any{"reason_code": "sandbox.launch_failed", "dispatch_phase": "tool"},
	}}
	outcome, err := TerminalOutcomeFromRunResult(result, "session-1")
	if err != nil {
		t.Fatal(err)
	}
	if outcome.FailureFamily != FailureFamilyRuntimeFailed || outcome.Phase != ExecutionPhasePostStart {
		t.Fatalf("outcome = %#v, want runtime_failed/post_start", outcome)
	}
}

func TestTerminalOutcomeFromRunResultMapsTimeoutAndRetryExhaustion(t *testing.T) {
	timeoutResult := RunResult{RunID: "run-timeout", Error: &ClassifiedError{
		Class:   ErrPolicyTimeout,
		Message: "step timed out",
		Details: map[string]any{"reason_code": "policy_timeout"},
	}}
	outcome, err := TerminalOutcomeFromRunResult(timeoutResult, "session-1")
	if err != nil {
		t.Fatal(err)
	}
	if outcome.FailureFamily != FailureFamilyTimedOut || outcome.State != RunStateFailed {
		t.Fatalf("timeout outcome = %#v, want timed_out/failed", outcome)
	}

	retryResult := RunResult{RunID: "run-retry-exhausted", Error: &ClassifiedError{
		Class:     ErrTool,
		Message:   "retry exhausted",
		Retryable: true,
		Details:   map[string]any{"reason_code": "retry_exhausted"},
	}}
	outcome, err = TerminalOutcomeFromRunResult(retryResult, "session-1")
	if err != nil {
		t.Fatal(err)
	}
	if outcome.FailureFamily != FailureFamilyRetryExhausted || !outcome.Retryable {
		t.Fatalf("retry outcome = %#v, want retry_exhausted/retryable", outcome)
	}
}

func TestTerminalOutcomeFromRunResultPreservesSourceOwnedAttemptMetadata(t *testing.T) {
	result := RunResult{RunID: "run-attempt-metadata", Error: &ClassifiedError{
		Class:     ErrModel,
		Message:   "provider failed",
		Retryable: true,
		Details: map[string]any{
			"reason_code":   "provider_error",
			"resumable":     true,
			"attempt":       2,
			"attempt_limit": 4,
			"causation_id":  "cause-2",
		},
	}}
	outcome, err := TerminalOutcomeFromRunResult(result, "session-1")
	if err != nil {
		t.Fatal(err)
	}
	if !outcome.Resumable || outcome.Attempt != 2 || outcome.AttemptLimit != 4 || outcome.CausationID != "cause-2" {
		t.Fatalf("source-owned metadata lost: %#v", outcome)
	}
}

func TestTerminalOutcomeDoesNotSynthesizeRetryOrResumeFromMessages(t *testing.T) {
	result := RunResult{RunID: "run-source-owned", Error: &ClassifiedError{
		Class:   ErrModel,
		Message: "retry exhausted; resume may be possible",
		Details: map[string]any{"reason_code": "provider_error"},
	}}
	outcome, err := TerminalOutcomeFromRunResult(result, "session-1")
	if err != nil {
		t.Fatal(err)
	}
	if outcome.FailureFamily != FailureFamilyRuntimeFailed {
		t.Fatalf("failure family = %q, want runtime_failed", outcome.FailureFamily)
	}
	if outcome.Resumable || outcome.Attempt != 0 || outcome.AttemptLimit != 0 || outcome.CausationID != "" {
		t.Fatalf("projection synthesized source-owned metadata: %#v", outcome)
	}
}
