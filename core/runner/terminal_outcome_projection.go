package runner

import (
	"encoding/json"

	"github.com/FelixSeptem/baymax/core/types"
)

func attachTerminalOutcome(result *types.RunResult, sessionID string) {
	if result == nil || result.TerminalOutcome != nil {
		return
	}
	outcome, err := types.TerminalOutcomeFromRunResult(*result, sessionID)
	if err != nil {
		return
	}
	result.TerminalOutcome = &outcome
}

func terminalResult(result types.RunResult, sessionID string) types.RunResult {
	attachTerminalOutcome(&result, sessionID)
	return result
}

func appendTerminalOutcomePayload(payload map[string]any, result types.RunResult) {
	if payload == nil {
		return
	}
	outcome, err := types.TerminalOutcomeFromRunResult(result, "")
	if err != nil {
		return
	}
	payload["terminal_state"] = string(outcome.State)
	payload["terminal_failure_family"] = string(outcome.FailureFamily)
	payload["terminal_phase"] = string(outcome.Phase)
	payload["terminal_source_reason"] = outcome.SourceReason
	payload["terminal_retryable"] = outcome.Retryable
	payload["terminal_resumable"] = outcome.Resumable
	payload["terminal_attempt"] = outcome.Attempt
	payload["terminal_attempt_limit"] = outcome.AttemptLimit
	payload["terminal_causation_id"] = outcome.CausationID
	encoded, err := json.Marshal(outcome)
	if err != nil {
		return
	}
	var projected map[string]any
	if err := json.Unmarshal(encoded, &projected); err == nil {
		payload["terminal_outcome"] = projected
	}
}
