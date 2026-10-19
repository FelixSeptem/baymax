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
	MailboxPlanBridgeReplayVersion          = "mailbox_plan_bridge.v1"
	ReasonMailboxPlanReplayDrift            = "mailbox_plan_bridge.replay_drift"
	ReasonMailboxPlanManifestDrift          = "mailbox_plan_bridge.manifest_drift"
	ReasonMailboxPlanBindingMismatch        = "mailbox_plan_bridge.binding_mismatch"
	ReasonMailboxPlanDuplicateConflict      = "mailbox_plan_bridge.duplicate_conflict"
	ReasonMailboxPlanSideEffectBeforeReject = "mailbox_plan_bridge.side_effect_before_rejection"
	ReasonMailboxPlanParityDrift            = "mailbox_plan_bridge.run_stream_parity_drift"
)

// MailboxPlanBridgeFixture is an offline, reference-only contract fixture.
// It deliberately contains no source, callback, handler or executable handle.
type MailboxPlanBridgeFixture struct {
	Version string                        `json:"version"`
	Cases   []MailboxPlanBridgeReplayCase `json:"cases"`
}

type MailboxPlanBridgeReplayCase struct {
	CaseID                  string                       `json:"case_id"`
	ManifestContractVersion string                       `json:"manifest_contract_version"`
	ManifestAPIVersion      string                       `json:"manifest_api_version"`
	EvidenceReference       string                       `json:"evidence_reference"`
	SourceAvailable         bool                         `json:"source_available"`
	BindingEquivalent       bool                         `json:"binding_equivalent"`
	SideEffectsBeforeReject bool                         `json:"side_effects_before_reject,omitempty"`
	LogicalAdmissions       int                          `json:"logical_admissions"`
	Observation             MailboxPlanBridgeObservation `json:"observation"`
	Run                     MailboxPlanBridgeObservation `json:"run"`
	Stream                  MailboxPlanBridgeObservation `json:"stream"`
	ExpectedReason          string                       `json:"expected_reason"`
}

// MailboxPlanBridgeObservation contains only the public, source-owned fields
// needed to compare one continuation. It excludes payloads, credentials,
// callbacks, application commits and opaque source state.
type MailboxPlanBridgeObservation struct {
	MessageID        string `json:"message_id"`
	RunID            string `json:"run_id"`
	DeliveryID       string `json:"delivery_id,omitempty"`
	LeaseID          string `json:"lease_id,omitempty"`
	AttemptCausation string `json:"attempt_causation,omitempty"`
	ParentRunID      string `json:"parent_run_id"`
	CausationID      string `json:"causation_id"`
	AuthorityDigest  string `json:"authority_digest"`
	PlanDigest       string `json:"plan_digest"`
	WriteScope       string `json:"write_scope"`
	EffectNamespace  string `json:"effect_namespace"`
	Occurrence       string `json:"occurrence"`
	HostCorrelation  string `json:"host_correlation"`
	LifecycleResult  string `json:"lifecycle_result"`
	TerminalOutcome  string `json:"terminal_outcome,omitempty"`
}

type MailboxPlanBridgeReplayResult struct {
	CaseID string `json:"case_id"`
	Reason string `json:"reason"`
	Digest string `json:"digest"`
}

func mailboxPlanReplayError(reason string) error {
	return &ValidationError{Code: reason, Message: "mailbox plan bridge conformance failed"}
}

// ReplayMailboxPlanBridgeFixtureJSON reads only a retained fixture. It has no
// import or call path to a mailbox owner or any executable runtime surface.
func ReplayMailboxPlanBridgeFixtureJSON(raw []byte) ([]MailboxPlanBridgeReplayResult, error) {
	if len(raw) == 0 || len(raw) > 4*1024*1024 || !utf8.Valid(raw) {
		return nil, mailboxPlanReplayError(ReasonMailboxPlanReplayDrift)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	var fixture MailboxPlanBridgeFixture
	if err := decoder.Decode(&fixture); err != nil {
		return nil, mailboxPlanReplayError(ReasonMailboxPlanReplayDrift)
	}
	if err := decoder.Decode(new(any)); err != io.EOF || fixture.Version != MailboxPlanBridgeReplayVersion || len(fixture.Cases) == 0 || len(fixture.Cases) > 256 {
		return nil, mailboxPlanReplayError(ReasonMailboxPlanReplayDrift)
	}
	seen := make(map[string]struct{}, len(fixture.Cases))
	results := make([]MailboxPlanBridgeReplayResult, 0, len(fixture.Cases))
	for _, item := range fixture.Cases {
		if !boundedMailboxPlanCase(item) {
			return nil, mailboxPlanReplayError(ReasonMailboxPlanReplayDrift)
		}
		if _, duplicate := seen[item.CaseID]; duplicate {
			return nil, mailboxPlanReplayError(ReasonMailboxPlanReplayDrift)
		}
		seen[item.CaseID] = struct{}{}
		reason := mailboxPlanReason(item)
		if reason != item.ExpectedReason {
			return nil, mailboxPlanReplayError(ReasonMailboxPlanReplayDrift)
		}
		encoded, err := json.Marshal(struct {
			CaseID, Reason           string
			Observation, Run, Stream MailboxPlanBridgeObservation
			LogicalAdmissions        int
		}{item.CaseID, reason, item.Observation, item.Run, item.Stream, item.LogicalAdmissions})
		if err != nil {
			return nil, mailboxPlanReplayError(ReasonMailboxPlanReplayDrift)
		}
		sum := sha256.Sum256(encoded)
		results = append(results, MailboxPlanBridgeReplayResult{CaseID: item.CaseID, Reason: reason, Digest: hex.EncodeToString(sum[:])})
	}
	return results, nil
}

func boundedMailboxPlanCase(item MailboxPlanBridgeReplayCase) bool {
	if strings.TrimSpace(item.CaseID) == "" || len(item.CaseID) > 256 || item.LogicalAdmissions < 0 {
		return false
	}
	for _, value := range []string{item.ManifestContractVersion, item.ManifestAPIVersion, item.EvidenceReference} {
		if strings.TrimSpace(value) == "" || len(value) > 1024 {
			return false
		}
	}
	return boundedMailboxPlanObservation(item.Observation) && boundedMailboxPlanObservation(item.Run) && boundedMailboxPlanObservation(item.Stream)
}

func boundedMailboxPlanObservation(item MailboxPlanBridgeObservation) bool {
	for _, value := range []string{item.MessageID, item.ParentRunID, item.CausationID, item.AuthorityDigest, item.PlanDigest, item.WriteScope, item.EffectNamespace, item.Occurrence, item.HostCorrelation, item.LifecycleResult} {
		if strings.TrimSpace(value) == "" || len(value) > 1024 {
			return false
		}
	}
	return strings.HasPrefix(item.PlanDigest, "sha256:") && len(item.PlanDigest) == len("sha256:")+64
}

func mailboxPlanReason(item MailboxPlanBridgeReplayCase) string {
	if item.SideEffectsBeforeReject {
		return ReasonMailboxPlanSideEffectBeforeReject
	}
	if item.ManifestContractVersion != "baymax.runtime.contract.v1" || item.ManifestAPIVersion != "host_runtime_capability_manifest.v1" || !item.SourceAvailable {
		return ReasonMailboxPlanManifestDrift
	}
	if !item.BindingEquivalent {
		return ReasonMailboxPlanBindingMismatch
	}
	if item.LogicalAdmissions != 1 {
		return ReasonMailboxPlanDuplicateConflict
	}
	if item.Observation != item.Run || item.Observation != item.Stream {
		return ReasonMailboxPlanParityDrift
	}
	return ""
}
