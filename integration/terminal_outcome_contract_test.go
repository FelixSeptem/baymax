package integration

import (
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/orchestration/composer"
)

func TestTerminalOutcomeDriftGuards(t *testing.T) {
	arbiter := types.NewTerminalOutcomeArbiter()
	first := types.TerminalOutcome{RunID: "run-drift", State: types.RunStateCompleted, FailureFamily: types.FailureFamilyNone, Phase: types.ExecutionPhasePostStart}
	late := types.TerminalOutcome{RunID: "run-drift", State: types.RunStateFailed, FailureFamily: types.FailureFamilyRuntimeFailed, Phase: types.ExecutionPhasePostStart, SourceReason: "provider_error"}
	if _, err := arbiter.Publish(first); err != nil {
		t.Fatal(err)
	}
	if result, err := arbiter.Publish(late); err != nil || result != types.TerminalPublishConflict {
		t.Fatalf("late terminal publish = %v, %v", result, err)
	}
	terminal, ok := arbiter.Terminal()
	if !ok || terminal.State != types.RunStateCompleted {
		t.Fatalf("terminal overwritten: %#v, %v", terminal, ok)
	}
	if len(arbiter.Conflicts()) != 1 {
		t.Fatalf("conflict count = %d, want 1", len(arbiter.Conflicts()))
	}

	result := types.RunResult{RunID: "run-parity", FinalAnswer: "partial", ToolCalls: []types.ToolCallSummary{{CallID: "call-1", Name: "search"}}, Error: &types.ClassifiedError{Class: types.ErrModel, Message: "provider failed", Details: map[string]any{"reason_code": "provider_error"}}}
	outcome, err := types.TerminalOutcomeFromRunResult(result, "session-parity")
	if err != nil {
		t.Fatal(err)
	}
	if outcome.RunID != result.RunID || outcome.Phase != types.ExecutionPhasePostStart || result.FinalAnswer != "partial" || len(result.ToolCalls) != 1 {
		t.Fatalf("partial facts or correlation lost: outcome=%#v result=%#v", outcome, result)
	}

	recovery, err := composer.RecoveryTerminalOutcome("run-recovery", &composer.RecoveryError{Code: composer.RecoveryErrorConflict, Message: "snapshot conflict"}, types.ExecutionPhasePreExecution)
	if err != nil {
		t.Fatal(err)
	}
	if recovery.FailureFamily != types.FailureFamilyRecoveryConflict || recovery.Phase != types.ExecutionPhasePreExecution {
		t.Fatalf("recovery drift: %#v", recovery)
	}
}
