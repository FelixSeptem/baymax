package diagnostics

import (
	"errors"
	"sort"
)

// ErrAttentionInput indicates invalid snapshot input. No partial view is returned.
var ErrAttentionInput = errors.New("attention: invalid input")

// AttentionRequest supplies a snapshot, not a live owner-state or authorization proof.
// Runs IDs must be unique. Each list is bounded to 200 entries.
type AttentionRequest struct {
	Runs     []RunRecord         `json:"runs"`
	Evidence []AttentionEvidence `json:"evidence"`
	Limit    int                 `json:"limit"`
}

// AttentionEvidence is an opaque reference obtained by the caller from its owner.
// Only authorization-required, source-required and effect-pending are accepted.
type AttentionEvidence struct {
	RunID     string `json:"run_id"`
	Code      string `json:"code"`
	Reference string `json:"reference"`
}

// AttentionItem identifies an action for the owning host to consider, not execute.
type AttentionItem struct {
	RunID      string `json:"run_id"`
	Code       string `json:"code"`
	NextAction string `json:"next_action"`
	Owner      string `json:"owner"`
	Reference  string `json:"reference"`
}

// AttentionResult counts only candidates in the supplied snapshot.
type AttentionResult struct {
	Items     []AttentionItem `json:"items"`
	Total     int             `json:"total"`
	Truncated bool            `json:"truncated"`
}

// ProjectAttention validates and projects a snapshot without I/O or mutation.
// It preserves QueryRuns pagination; an empty view is not proof of completion.
func ProjectAttention(request AttentionRequest) (AttentionResult, error) {
	limit := request.Limit
	if limit == 0 {
		limit = 50
	}
	if limit < 1 || limit > 200 || len(request.Runs) > 200 || len(request.Evidence) > 200 {
		return AttentionResult{}, ErrAttentionInput
	}
	seen := make(map[string]bool, len(request.Runs))
	for _, run := range request.Runs {
		if !attentionReference(run.RunID) || seen[run.RunID] {
			return AttentionResult{}, ErrAttentionInput
		}
		seen[run.RunID] = true
	}
	for _, evidence := range request.Evidence {
		if !attentionReference(evidence.RunID) || !attentionReference(evidence.Reference) {
			return AttentionResult{}, ErrAttentionInput
		}
		switch evidence.Code {
		case "authorization-required", "source-required", "effect-pending":
		default:
			return AttentionResult{}, ErrAttentionInput
		}
	}
	unique := make(map[AttentionItem]struct{})
	add := func(runID, code, reference string) {
		action, owner := attentionMapping(code)
		unique[AttentionItem{RunID: runID, Code: code, NextAction: action, Owner: owner, Reference: reference}] = struct{}{}
	}
	for _, run := range request.Runs {
		if run.ProtocolCapabilityReason == "protocol.capability.missing_required" || run.ProviderCapabilityOutcome == "blocked" {
			add(run.RunID, "unsupported", run.RunID)
		}
		if run.RecoveryConflict || run.StreamRecoveryTerminalConflictRecorded {
			add(run.RunID, "recovery-required", run.RunID)
		}
		if run.GateDeniedCount > 0 || run.GateTimeoutCount > 0 {
			add(run.RunID, "gate-failed", run.RunID)
		}
	}
	for _, evidence := range request.Evidence {
		add(evidence.RunID, evidence.Code, evidence.Reference)
	}
	items := make([]AttentionItem, 0, len(unique))
	for item := range unique {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		a, b := items[i], items[j]
		if a.RunID != b.RunID {
			return a.RunID < b.RunID
		}
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		return a.Reference < b.Reference
	})
	result := AttentionResult{Items: items, Total: len(items), Truncated: len(items) > limit}
	if result.Truncated {
		result.Items = items[:limit:limit]
	}
	return result, nil
}

func attentionMapping(code string) (string, string) {
	switch code {
	case "unsupported":
		return "inspect_capability", "runtime/capability"
	case "recovery-required":
		return "inspect_recovery", "runtime/recovery"
	case "gate-failed":
		return "inspect_gate", "runtime/gate"
	case "authorization-required":
		return "revalidate_authorization", "host/authorization"
	case "source-required":
		return "inspect_source", "host/source"
	case "effect-pending":
		return "reconcile_effect", "host/effect"
	default:
		return "", ""
	}
}

func attentionReference(value string) bool {
	if len(value) < 1 || len(value) > 256 {
		return false
	}
	for i := range len(value) {
		c := value[i]
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' {
			continue
		}
		switch c {
		case '.', '_', ':', '/', '@', '-':
		default:
			return false
		}
	}
	return true
}
