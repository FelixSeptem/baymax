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

// HostExecutionPlanFixture is a bounded, reference-only recording. It is
// deliberately not a descriptor/binding: replay validates recorded evidence
// and never obtains a callback, runner, store, checkpoint, or resume handle.
type HostExecutionPlanFixture struct {
	Version string                        `json:"version"`
	Cases   []HostExecutionPlanReplayCase `json:"cases"`
}

type HostExecutionPlanReplayCase struct {
	CaseID                  string `json:"case_id"`
	PlanID                  string `json:"plan_id"`
	PlanVersion             int    `json:"plan_version"`
	PlanDigest              string `json:"plan_digest"`
	SourceOutcome           string `json:"source_outcome"`
	RunID                   string `json:"run_id"`
	ParentRunID             string `json:"parent_run_id,omitempty"`
	BudgetNanos             int64  `json:"budget_nanos"`
	SourceEvidence          bool   `json:"source_evidence"`
	SideEffectsBeforeReject bool   `json:"side_effects_before_reject,omitempty"`
	ExpectedReason          string `json:"expected_reason"`
}

type HostExecutionPlanReplayResult struct {
	CaseID string `json:"case_id"`
	Reason string `json:"reason"`
	Digest string `json:"digest"`
}

const (
	HostExecutionPlanReplayVersion          = "host_execution_plan.v1"
	ReasonHostExecutionPlanDigestDrift      = "host_execution_plan.digest_drift"
	ReasonHostExecutionPlanOutcomeDrift     = "host_execution_plan.outcome_drift"
	ReasonHostExecutionPlanCorrelationDrift = "host_execution_plan.correlation_drift"
	ReasonHostExecutionPlanBudgetDrift      = "host_execution_plan.budget_drift"
	ReasonHostExecutionPlanMissingEvidence  = "host_execution_plan.missing_source_evidence"
	ReasonHostExecutionPlanSideEffect       = "host_execution_plan.side_effect_before_rejection"
	ReasonHostExecutionPlanReplayDrift      = "host_execution_plan.replay_drift"
)

func hostPlanReplayError(reason string) error {
	return &ValidationError{Code: reason, Message: "host execution plan conformance failed"}
}

// ReplayHostExecutionPlanFixtureJSON validates source evidence offline. It
// cannot execute or resume an owner; every result is derived only from JSON.
func ReplayHostExecutionPlanFixtureJSON(raw []byte) ([]HostExecutionPlanReplayResult, error) {
	if len(raw) == 0 || len(raw) > 4*1024*1024 || !utf8.Valid(raw) {
		return nil, hostPlanReplayError(ReasonHostExecutionPlanReplayDrift)
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	d.UseNumber()
	var fixture HostExecutionPlanFixture
	if err := d.Decode(&fixture); err != nil {
		return nil, hostPlanReplayError(ReasonHostExecutionPlanReplayDrift)
	}
	if err := d.Decode(new(any)); err != io.EOF || fixture.Version != HostExecutionPlanReplayVersion || len(fixture.Cases) == 0 || len(fixture.Cases) > 256 {
		return nil, hostPlanReplayError(ReasonHostExecutionPlanReplayDrift)
	}
	seen := make(map[string]struct{}, len(fixture.Cases))
	results := make([]HostExecutionPlanReplayResult, 0, len(fixture.Cases))
	for _, item := range fixture.Cases {
		if len(item.CaseID) == 0 || len(item.CaseID) > 256 || len(item.PlanID) == 0 || len(item.PlanID) > 1024 || item.PlanVersion <= 0 || item.BudgetNanos < 0 {
			return nil, hostPlanReplayError(ReasonHostExecutionPlanReplayDrift)
		}
		if _, ok := seen[item.CaseID]; ok {
			return nil, hostPlanReplayError(ReasonHostExecutionPlanReplayDrift)
		}
		seen[item.CaseID] = struct{}{}
		reason := hostPlanReason(item)
		if reason != item.ExpectedReason {
			return nil, hostPlanReplayError(ReasonHostExecutionPlanReplayDrift)
		}
		encoded, err := json.Marshal(struct {
			CaseID, PlanID, PlanDigest, SourceOutcome, RunID, ParentRunID, Reason string
			PlanVersion                                                           int
			BudgetNanos                                                           int64
			SourceEvidence                                                        bool
		}{item.CaseID, item.PlanID, item.PlanDigest, item.SourceOutcome, item.RunID, item.ParentRunID, reason, item.PlanVersion, item.BudgetNanos, item.SourceEvidence})
		if err != nil {
			return nil, hostPlanReplayError(ReasonHostExecutionPlanReplayDrift)
		}
		sum := sha256.Sum256(encoded)
		results = append(results, HostExecutionPlanReplayResult{CaseID: item.CaseID, Reason: reason, Digest: hex.EncodeToString(sum[:])})
	}
	return results, nil
}

func hostPlanReason(item HostExecutionPlanReplayCase) string {
	if item.SideEffectsBeforeReject {
		return ReasonHostExecutionPlanSideEffect
	}
	if !item.SourceEvidence {
		return ReasonHostExecutionPlanMissingEvidence
	}
	if !strings.HasPrefix(item.PlanDigest, "sha256:") || len(item.PlanDigest) != len("sha256:")+64 {
		return ReasonHostExecutionPlanDigestDrift
	}
	for _, r := range item.PlanDigest[len("sha256:"):] {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return ReasonHostExecutionPlanDigestDrift
		}
	}
	if item.RunID == "" || (item.ParentRunID != "" && item.ParentRunID == item.RunID) {
		return ReasonHostExecutionPlanCorrelationDrift
	}
	if item.BudgetNanos == 0 {
		return ReasonHostExecutionPlanBudgetDrift
	}
	switch item.SourceOutcome {
	case "completed", "failed", "canceled", "input_required":
		return ""
	default:
		return ReasonHostExecutionPlanOutcomeDrift
	}
}
