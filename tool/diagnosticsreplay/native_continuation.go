package diagnosticsreplay

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"strings"
	"unicode/utf8"
)

const (
	NativeContinuationReplayVersion               = "native_continuation.v1"
	ReasonNativeContinuationTerminalEvidenceDrift = "native_continuation.terminal_evidence_drift"
	ReasonNativeContinuationMissingBinding        = "native_continuation.missing_source_binding"
	ReasonNativeContinuationDuplicateExecution    = "native_continuation.duplicate_execution"
	ReasonNativeContinuationUnsupportedComposite  = "native_continuation.unsupported_composite_restore"
	ReasonNativeContinuationParityDrift           = "native_continuation.run_stream_mismatch"
	ReasonNativeContinuationIdentityDrift         = "native_continuation.identity_drift"
	ReasonNativeContinuationSideEffect            = "native_continuation.side_effect_before_claim"
	ReasonNativeContinuationStateDrift            = "native_continuation.source_state_drift"
	ReasonNativeContinuationReplayDrift           = "native_continuation.replay_drift"
)

// NativeContinuationFixture records bounded source facts, never continuation
// material, executable bindings, source store handles, or host callbacks.
type NativeContinuationFixture struct {
	Version string                         `json:"version"`
	Cases   []NativeContinuationReplayCase `json:"cases"`
}

type NativeContinuationProjection struct {
	Outcome   string `json:"outcome"`
	RunID     string `json:"run_id"`
	SessionID string `json:"session_id"`
}

type NativeContinuationReplayCase struct {
	CaseID             string                       `json:"case_id"`
	Form               string                       `json:"form"`
	PauseReason        string                       `json:"pause_reason"`
	Window             string                       `json:"window"`
	CheckpointID       string                       `json:"checkpoint_id"`
	RunID              string                       `json:"run_id"`
	SessionID          string                       `json:"session_id"`
	SourceNamespace    string                       `json:"source_namespace"`
	Generation         uint64                       `json:"generation"`
	SourceState        string                       `json:"source_state"`
	SourceBound        bool                         `json:"source_bound"`
	ClaimCommitted     bool                         `json:"claim_committed"`
	TerminalCommitted  bool                         `json:"terminal_committed"`
	SuccessorCommitted bool                         `json:"successor_committed"`
	TerminalOutcome    string                       `json:"terminal_outcome"`
	ExecutionCount     int                          `json:"execution_count"`
	Run                NativeContinuationProjection `json:"run"`
	Stream             NativeContinuationProjection `json:"stream"`
	ExpectedReason     string                       `json:"expected_reason"`
}

type NativeContinuationReplayResult struct {
	CaseID string `json:"case_id"`
	Reason string `json:"reason"`
	Digest string `json:"digest"`
}

func continuationReplayError() error {
	return &ValidationError{Code: ReasonNativeContinuationReplayDrift, Message: "native continuation conformance failed"}
}

// ReplayNativeContinuationFixtureJSON validates recorded evidence offline. It
// cannot load, claim, execute or repair a checkpoint, and has no source handle.
func ReplayNativeContinuationFixtureJSON(raw []byte) ([]NativeContinuationReplayResult, error) {
	if len(raw) == 0 || len(raw) > 4*1024*1024 || !utf8.Valid(raw) {
		return nil, continuationReplayError()
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var fixture NativeContinuationFixture
	if err := decoder.Decode(&fixture); err != nil {
		return nil, continuationReplayError()
	}
	if err := decoder.Decode(new(any)); err != io.EOF || fixture.Version != NativeContinuationReplayVersion || len(fixture.Cases) == 0 || len(fixture.Cases) > 256 {
		return nil, continuationReplayError()
	}
	seen := make(map[string]struct{}, len(fixture.Cases))
	results := make([]NativeContinuationReplayResult, 0, len(fixture.Cases))
	for _, item := range fixture.Cases {
		if !validContinuationRecording(item) {
			return nil, continuationReplayError()
		}
		if _, exists := seen[item.CaseID]; exists {
			return nil, continuationReplayError()
		}
		seen[item.CaseID] = struct{}{}
		reason := continuationReason(item)
		if reason != item.ExpectedReason {
			return nil, continuationReplayError()
		}
		canonical, err := json.Marshal(item)
		if err != nil {
			return nil, continuationReplayError()
		}
		sum := sha256.Sum256(canonical)
		results = append(results, NativeContinuationReplayResult{CaseID: item.CaseID, Reason: reason, Digest: hex.EncodeToString(sum[:])})
	}
	return results, nil
}

func validContinuationRecording(item NativeContinuationReplayCase) bool {
	if strings.TrimSpace(item.CaseID) == "" || item.ExecutionCount < 0 {
		return false
	}
	for _, value := range []string{item.CaseID, item.Form, item.PauseReason, item.Window, item.CheckpointID, item.RunID, item.SessionID, item.SourceNamespace, item.SourceState, item.TerminalOutcome, item.Run.Outcome, item.Run.RunID, item.Run.SessionID, item.Stream.Outcome, item.Stream.RunID, item.Stream.SessionID, item.ExpectedReason} {
		if len(value) > 256 {
			return false
		}
	}
	switch item.Form {
	case "direct", "subagent", "child", "fanout", "team", "workflow", "mailbox_async", "planbridge":
	default:
		return false
	}
	switch item.PauseReason {
	case "input_required", "approval_required", "budget_exhausted":
	default:
		return false
	}
	_, _, known := continuationWindow(item.Window, item)
	return known
}

func continuationReason(item NativeContinuationReplayCase) string {
	if item.Form != "direct" {
		if item.ExecutionCount != 0 || item.ClaimCommitted || item.Run.Outcome != "unsupported_capability" || item.Stream.Outcome != "unsupported_capability" {
			return ReasonNativeContinuationUnsupportedComposite
		}
		return ""
	}
	if item.Run != item.Stream {
		return ReasonNativeContinuationParityDrift
	}
	if item.ExecutionCount > 1 {
		return ReasonNativeContinuationDuplicateExecution
	}
	if item.ExecutionCount > 0 && !item.ClaimCommitted {
		return ReasonNativeContinuationSideEffect
	}
	if item.Run.RunID != item.RunID || item.Run.SessionID != item.SessionID || strings.TrimSpace(item.RunID) == "" {
		return ReasonNativeContinuationIdentityDrift
	}
	if item.Window != "constructed_before_save" && item.Window != "material_before_publish" && item.Window != "foreign" && item.Window != "store_unavailable" {
		if !item.SourceBound || item.CheckpointID == "" || item.SourceNamespace == "" || item.Generation == 0 {
			return ReasonNativeContinuationMissingBinding
		}
	}
	state, outcome, _ := continuationWindow(item.Window, item)
	if item.SourceState != state {
		return ReasonNativeContinuationStateDrift
	}
	projectedTerminal := item.Run.Outcome == "completed" || item.Run.Outcome == "failed" || item.Run.Outcome == "canceled"
	projectedSuccessor := item.Run.Outcome == "input_required" || item.Run.Outcome == "approval_required" || item.Run.Outcome == "budget_exhausted"
	if projectedTerminal && (!item.TerminalCommitted || item.SourceState != "terminal") || projectedSuccessor && (!item.SuccessorCommitted || item.SourceState != "paused_successor") {
		return ReasonNativeContinuationTerminalEvidenceDrift
	}
	if item.Run.Outcome != outcome {
		return ReasonNativeContinuationStateDrift
	}
	if item.TerminalCommitted && item.SourceState != "terminal" || item.SuccessorCommitted && item.SourceState != "paused_successor" {
		return ReasonNativeContinuationTerminalEvidenceDrift
	}
	if (item.SourceState == "claimed" || item.SourceState == "running") && !item.ClaimCommitted {
		return ReasonNativeContinuationStateDrift
	}
	return ""
}

func continuationWindow(window string, item NativeContinuationReplayCase) (state, outcome string, known bool) {
	switch window {
	case "constructed_before_save":
		return "constructed", "not_found", true
	case "material_before_publish":
		return "material", "not_found", true
	case "persisted_response_lost", "verifier_before_claim":
		return "persisted", "paused", true
	case "claimed_before_start":
		return "claimed", "in_progress", true
	case "running_before_result", "duplicate_running":
		return "running", "recovery_required", true
	case "successor_committed", "duplicate_successor":
		return "paused_successor", item.PauseReason, true
	case "terminal_committed", "duplicate_terminal":
		return "terminal", item.TerminalOutcome, true
	case "terminal_write_failed":
		return "running", "source_unavailable", true
	case "expired":
		return "persisted", "expired", true
	case "corrupt":
		return "corrupt", "corrupt", true
	case "foreign":
		return "foreign", "foreign", true
	case "conflict":
		return "persisted", "conflict", true
	case "store_unavailable":
		return "unavailable", "source_unavailable", true
	case "canceled_before_claim", "terminal_before_claim":
		return "terminal", "terminal", true
	case "unsupported":
		return "unsupported", "unsupported_capability", true
	default:
		return "", "", false
	}
}
