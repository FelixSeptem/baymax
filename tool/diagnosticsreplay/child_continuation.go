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

const ChildContinuationReplayVersion = "child_continuation.v1"

type ChildContinuationReplayFixture struct {
	Version string                        `json:"version"`
	Cases   []ChildContinuationReplayCase `json:"cases"`
}

type ChildContinuationReplayCase struct {
	CaseID            string `json:"case_id"`
	Form              string `json:"form"`
	PauseReason       string `json:"pause_reason"`
	Window            string `json:"window"`
	SourceState       string `json:"source_state"`
	CheckpointID      string `json:"checkpoint_id"`
	GraphID           string `json:"graph_id"`
	GraphDigest       string `json:"graph_digest"`
	SourceNamespace   string `json:"source_namespace"`
	RunID             string `json:"run_id"`
	SessionID         string `json:"session_id"`
	ExecutionCount    int    `json:"execution_count"`
	ClaimCommitted    bool   `json:"claim_committed"`
	TerminalCommitted bool   `json:"terminal_committed"`
	RunOutcome        string `json:"run_outcome"`
	StreamOutcome     string `json:"stream_outcome"`
	ExpectedReason    string `json:"expected_reason"`
}

type ChildContinuationReplayResult struct{ CaseID, Reason, Digest string }

func ReplayChildContinuationFixtureJSON(raw []byte) ([]ChildContinuationReplayResult, error) {
	if len(raw) == 0 || len(raw) > 4*1024*1024 || !utf8.Valid(raw) {
		return nil, replayValidationError()
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var fixture ChildContinuationReplayFixture
	if err := dec.Decode(&fixture); err != nil || dec.Decode(new(any)) != io.EOF || fixture.Version != ChildContinuationReplayVersion || len(fixture.Cases) == 0 || len(fixture.Cases) > 256 {
		return nil, replayValidationError()
	}
	seen := map[string]struct{}{}
	result := make([]ChildContinuationReplayResult, 0, len(fixture.Cases))
	for _, item := range fixture.Cases {
		if !validChildContinuationCase(item) {
			return nil, replayValidationError()
		}
		if _, ok := seen[item.CaseID]; ok {
			return nil, replayValidationError()
		}
		seen[item.CaseID] = struct{}{}
		reason := childContinuationReason(item)
		if reason != item.ExpectedReason {
			return nil, replayValidationError()
		}
		canonical, _ := json.Marshal(item)
		digest := sha256.Sum256(canonical)
		result = append(result, ChildContinuationReplayResult{CaseID: item.CaseID, Reason: reason, Digest: hex.EncodeToString(digest[:])})
	}
	return result, nil
}

func validChildContinuationCase(item ChildContinuationReplayCase) bool {
	if strings.TrimSpace(item.CaseID) == "" || item.ExecutionCount < 0 {
		return false
	}
	for _, value := range []string{item.CaseID, item.Form, item.PauseReason, item.Window, item.SourceState, item.CheckpointID, item.GraphID, item.GraphDigest, item.SourceNamespace, item.RunID, item.SessionID, item.RunOutcome, item.StreamOutcome, item.ExpectedReason} {
		if len(value) > 256 {
			return false
		}
	}
	if item.Form != "child" && item.Form != "fanout" {
		return item.ExpectedReason == "child_continuation.unsupported_form" && item.RunOutcome == "unsupported_capability" && item.StreamOutcome == "unsupported_capability" && item.ExecutionCount == 0
	}
	if item.PauseReason != "input_required" && item.PauseReason != "approval_required" && item.PauseReason != "budget_exhausted" {
		return false
	}
	return item.Window != ""
}

func childContinuationReason(item ChildContinuationReplayCase) string {
	if item.Form != "child" && item.Form != "fanout" {
		return "child_continuation.unsupported_form"
	}
	if item.RunOutcome != item.StreamOutcome || item.ExecutionCount > 1 {
		return "child_continuation.identity_or_duplicate_drift"
	}
	if item.Window == "unsupported" {
		return "child_continuation.unsupported_form"
	}
	if item.SourceNamespace != "child" || item.CheckpointID == "" || item.GraphID == "" || item.GraphDigest == "" || item.RunID == "" || item.SessionID == "" {
		return "child_continuation.identity_drift"
	}
	if item.Window == "terminal_committed" && !item.TerminalCommitted {
		return "child_continuation.terminal_evidence_drift"
	}
	if item.Window == "claimed_before_start" && !item.ClaimCommitted {
		return "child_continuation.claim_evidence_drift"
	}
	return ""
}

func replayValidationError() error {
	return &ValidationError{Code: "child_continuation.replay_drift", Message: "child continuation conformance failed"}
}
