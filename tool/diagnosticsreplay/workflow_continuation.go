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
	WorkflowContinuationReplayVersion            = "workflow_continuation.v1"
	ReasonWorkflowContinuationReplayDrift        = "workflow_continuation.replay_drift"
	ReasonWorkflowContinuationIdentityDrift      = "workflow_continuation.identity_drift"
	ReasonWorkflowContinuationGraphDrift         = "workflow_continuation.graph_drift"
	ReasonWorkflowContinuationDuplicateExecution = "workflow_continuation.duplicate_execution"
	ReasonWorkflowContinuationClaimEvidence      = "workflow_continuation.claim_evidence_drift"
	ReasonWorkflowContinuationTerminalEvidence   = "workflow_continuation.terminal_evidence_drift"
	ReasonWorkflowContinuationUnsupported        = "workflow_continuation.unsupported_form"
)

// WorkflowContinuationReplayFixture holds reference-only source evidence. It
// cannot construct a workflow owner, read a store, or invoke Run/Stream.
type WorkflowContinuationReplayFixture struct {
	Version string                           `json:"version"`
	Cases   []WorkflowContinuationReplayCase `json:"cases"`
}

type WorkflowContinuationProjection struct {
	Outcome   string `json:"outcome"`
	RunID     string `json:"run_id"`
	SessionID string `json:"session_id"`
}

type WorkflowContinuationReplayCase struct {
	CaseID             string                         `json:"case_id"`
	Form               string                         `json:"form"`
	PauseReason        string                         `json:"pause_reason"`
	Window             string                         `json:"window"`
	CheckpointID       string                         `json:"checkpoint_id"`
	PredecessorID      string                         `json:"predecessor_id,omitempty"`
	GraphID            string                         `json:"graph_id"`
	GraphDigest        string                         `json:"graph_digest"`
	RunID              string                         `json:"run_id"`
	SessionID          string                         `json:"session_id"`
	SourceNamespace    string                         `json:"source_namespace"`
	Generation         uint64                         `json:"generation"`
	SourceState        string                         `json:"source_state"`
	ClaimCommitted     bool                           `json:"claim_committed"`
	TerminalCommitted  bool                           `json:"terminal_committed"`
	SuccessorCommitted bool                           `json:"successor_committed"`
	ExecutionCount     int                            `json:"execution_count"`
	Run                WorkflowContinuationProjection `json:"run"`
	Stream             WorkflowContinuationProjection `json:"stream"`
	ExpectedReason     string                         `json:"expected_reason"`
}

type WorkflowContinuationReplayResult struct {
	CaseID string `json:"case_id"`
	Reason string `json:"reason"`
	Digest string `json:"digest"`
}

func workflowContinuationReplayError() error {
	return &ValidationError{Code: ReasonWorkflowContinuationReplayDrift, Message: "workflow continuation conformance failed"}
}

// ReplayWorkflowContinuationFixtureJSON validates evidence deterministically
// without retaining or executing continuation material.
func ReplayWorkflowContinuationFixtureJSON(raw []byte) ([]WorkflowContinuationReplayResult, error) {
	if len(raw) == 0 || len(raw) > 4*1024*1024 || !utf8.Valid(raw) {
		return nil, workflowContinuationReplayError()
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var fixture WorkflowContinuationReplayFixture
	if err := decoder.Decode(&fixture); err != nil {
		return nil, workflowContinuationReplayError()
	}
	if err := decoder.Decode(new(any)); err != io.EOF || fixture.Version != WorkflowContinuationReplayVersion || len(fixture.Cases) == 0 || len(fixture.Cases) > 256 {
		return nil, workflowContinuationReplayError()
	}
	seen := make(map[string]struct{}, len(fixture.Cases))
	results := make([]WorkflowContinuationReplayResult, 0, len(fixture.Cases))
	for _, item := range fixture.Cases {
		if !validWorkflowContinuationRecording(item) {
			return nil, workflowContinuationReplayError()
		}
		if _, ok := seen[item.CaseID]; ok {
			return nil, workflowContinuationReplayError()
		}
		seen[item.CaseID] = struct{}{}
		reason := workflowContinuationReason(item)
		if reason != item.ExpectedReason {
			return nil, workflowContinuationReplayError()
		}
		canonical, err := json.Marshal(item)
		if err != nil {
			return nil, workflowContinuationReplayError()
		}
		sum := sha256.Sum256(canonical)
		results = append(results, WorkflowContinuationReplayResult{CaseID: item.CaseID, Reason: reason, Digest: hex.EncodeToString(sum[:])})
	}
	return results, nil
}

func validWorkflowContinuationRecording(item WorkflowContinuationReplayCase) bool {
	if strings.TrimSpace(item.CaseID) == "" || item.ExecutionCount < 0 {
		return false
	}
	for _, value := range []string{item.CaseID, item.Form, item.PauseReason, item.Window, item.CheckpointID, item.PredecessorID, item.GraphID, item.GraphDigest, item.RunID, item.SessionID, item.SourceNamespace, item.SourceState, item.Run.Outcome, item.Run.RunID, item.Run.SessionID, item.Stream.Outcome, item.Stream.RunID, item.Stream.SessionID, item.ExpectedReason} {
		if len(value) > 256 {
			return false
		}
	}
	switch item.Form {
	case "workflow", "child", "fanout", "team", "mailbox_async", "planbridge":
	default:
		return false
	}
	switch item.PauseReason {
	case "input_required", "approval_required", "budget_exhausted":
	default:
		return false
	}
	_, _, known := workflowContinuationWindow(item.Window, item.PauseReason)
	return known
}

func workflowContinuationReason(item WorkflowContinuationReplayCase) string {
	if item.Form != "workflow" {
		if item.ExecutionCount != 0 || item.ClaimCommitted || item.Run.Outcome != "unsupported_capability" || item.Stream.Outcome != "unsupported_capability" {
			return ReasonWorkflowContinuationUnsupported
		}
		return ""
	}
	if item.Run != item.Stream || item.Run.RunID != item.RunID || item.Run.SessionID != item.SessionID {
		return ReasonWorkflowContinuationIdentityDrift
	}
	if item.ExecutionCount > 1 {
		return ReasonWorkflowContinuationDuplicateExecution
	}
	state, outcome, _ := workflowContinuationWindow(item.Window, item.PauseReason)
	if item.SourceState != state || item.Run.Outcome != outcome {
		return ReasonWorkflowContinuationClaimEvidence
	}
	if item.Window != "unsupported" && (item.CheckpointID == "" || item.GraphID == "" || item.GraphDigest == "" || item.SourceNamespace != "workflow" || item.Generation == 0) {
		return ReasonWorkflowContinuationGraphDrift
	}
	if (state == "claimed" || state == "running" || state == "terminal" || state == "paused_successor") && !item.ClaimCommitted {
		return ReasonWorkflowContinuationClaimEvidence
	}
	if state == "terminal" && !item.TerminalCommitted || state == "paused_successor" && (!item.SuccessorCommitted || item.PredecessorID == "") {
		return ReasonWorkflowContinuationTerminalEvidence
	}
	return ""
}

func workflowContinuationWindow(window, pause string) (state, outcome string, known bool) {
	switch window {
	case "persisted":
		return "persisted", "paused", true
	case "claimed_before_start":
		return "claimed", "in_progress", true
	case "running_before_result":
		return "running", "recovery_required", true
	case "successor_committed":
		return "paused_successor", pause, true
	case "terminal_committed":
		return "terminal", "succeeded", true
	case "expired":
		return "persisted", "expired", true
	case "corrupt":
		return "corrupt", "corrupt", true
	case "foreign":
		return "foreign", "foreign", true
	case "conflict":
		return "persisted", "conflict", true
	case "unavailable":
		return "unavailable", "source_unavailable", true
	case "unsupported":
		return "unsupported", "unsupported_capability", true
	default:
		return "", "", false
	}
}
