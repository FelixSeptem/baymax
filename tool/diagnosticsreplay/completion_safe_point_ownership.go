package diagnosticsreplay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

const CompletionSafePointOwnershipFixtureV1 = "completion_safe_point_ownership.v1"

const (
	ReasonCodeCompletionSafePointSchemaDrift      = "completion_safe_point_schema_drift"
	ReasonCodeCompletionSafePointCorrelationDrift = "completion_safe_point_correlation_drift"
	ReasonCodeCompletionSafePointParityDrift      = "completion_safe_point_run_stream_parity_drift"
	ReasonCodeCompletionSafePointPrivacyDrift     = "completion_safe_point_privacy_drift"
)

type CompletionSafePointOwnershipFixture struct {
	Version string                             `json:"version"`
	Cases   []CompletionSafePointOwnershipCase `json:"cases"`
}
type CompletionSafePointOwnershipCase struct {
	Name          string                         `json:"name"`
	RunID         string                         `json:"run_id"`
	SessionID     string                         `json:"session_id"`
	TaskID        string                         `json:"task_id"`
	AttemptID     string                         `json:"attempt_id"`
	MailboxID     string                         `json:"mailbox_id"`
	CorrelationID string                         `json:"correlation_id"`
	Outcome       string                         `json:"outcome"`
	Duplicate     bool                           `json:"duplicate,omitempty"`
	Late          bool                           `json:"late,omitempty"`
	Disconnected  bool                           `json:"disconnected,omitempty"`
	Recovery      string                         `json:"recovery,omitempty"`
	Run           CompletionSafePointObservation `json:"run"`
	Stream        CompletionSafePointObservation `json:"stream"`
}
type CompletionSafePointObservation struct {
	Status   string `json:"status"`
	Boundary string `json:"boundary,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

func ParseCompletionSafePointOwnershipFixtureJSON(raw []byte) (CompletionSafePointOwnershipFixture, error) {
	if len(raw) > 2<<20 {
		return CompletionSafePointOwnershipFixture{}, schemaCompletionError("fixture exceeds 2 MiB")
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return CompletionSafePointOwnershipFixture{}, schemaCompletionError(err.Error())
	}
	for _, key := range []string{"completion_body", "reasoning", "raw_payload"} {
		if _, ok := envelope[key]; ok || bytes.Contains(raw, []byte(`"`+key+`"`)) {
			return CompletionSafePointOwnershipFixture{}, &ValidationError{Code: ReasonCodeCompletionSafePointPrivacyDrift, Message: key + " is not permitted"}
		}
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var f CompletionSafePointOwnershipFixture
	if err := dec.Decode(&f); err != nil {
		return f, schemaCompletionError(err.Error())
	}
	if strings.TrimSpace(f.Version) != CompletionSafePointOwnershipFixtureV1 {
		return f, schemaCompletionError(fmt.Sprintf("unsupported fixture version %q", f.Version))
	}
	if len(f.Cases) == 0 || len(f.Cases) > 256 {
		return f, schemaCompletionError("cases must contain 1..256 items")
	}
	for i := range f.Cases {
		c := &f.Cases[i]
		for _, v := range []string{c.Name, c.RunID, c.SessionID, c.TaskID, c.AttemptID, c.MailboxID, c.CorrelationID, c.Outcome} {
			if strings.TrimSpace(v) == "" || len(v) > 256 {
				return f, schemaCompletionError(fmt.Sprintf("cases[%d] required identifier missing or oversized", i))
			}
		}
		if strings.TrimSpace(c.Run.Status) == "" || strings.TrimSpace(c.Stream.Status) == "" {
			return f, schemaCompletionError(fmt.Sprintf("cases[%d] run/stream status required", i))
		}
		c.Outcome = strings.ToLower(strings.TrimSpace(c.Outcome))
		switch c.Outcome {
		case "accepted", "duplicate", "late", "disconnected", "recovered", "not_applied", "terminal_conflict":
		default:
			return f, schemaCompletionError(fmt.Sprintf("cases[%d] unsupported outcome %q", i, c.Outcome))
		}
		c.Recovery = strings.ToLower(strings.TrimSpace(c.Recovery))
		if c.Recovery != "" && c.Recovery != "replayed_once" && c.Recovery != "not_replayed" {
			return f, schemaCompletionError(fmt.Sprintf("cases[%d] unsupported recovery %q", i, c.Recovery))
		}
		if c.Run.Status != c.Stream.Status || c.Run.Boundary != c.Stream.Boundary {
			return f, &ValidationError{Code: ReasonCodeCompletionSafePointParityDrift, Message: c.Name}
		}
	}
	return f, nil
}

// EvaluateCompletionSafePointOwnershipFixtureJSON is the offline replay entrypoint.
func EvaluateCompletionSafePointOwnershipFixtureJSON(raw []byte) (CompletionSafePointOwnershipFixture, error) {
	return ParseCompletionSafePointOwnershipFixtureJSON(raw)
}

func schemaCompletionError(msg string) *ValidationError {
	return &ValidationError{Code: ReasonCodeCompletionSafePointSchemaDrift, Message: msg}
}
