package diagnosticsreplay

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"
)

const ActionCapabilityAuditVersion = "action_capability_audit.v1"

const (
	ActionCapabilityVerdictCompliant            = "compliant"
	ActionCapabilityVerdictGap                  = "gap"
	ActionCapabilityVerdictInsufficientEvidence = "insufficient_evidence"
	ActionCapabilityVerdictNotApplicable        = "not_applicable"

	ReasonCodeActionCapabilitySchemaDrift               = "action_capability_schema_drift"
	ReasonCodeActionCapabilityUnknownVersion            = "action_capability_unknown_version"
	ReasonCodeActionCapabilityExecutionOrDiscovery      = "action_capability_execution_or_discovery_detected"
	ReasonCodeActionCapabilityPrivacyOrBoundViolation   = "action_capability_privacy_or_bound_violation"
	ReasonCodeActionCapabilityVerifyEvidenceMissing     = "action_capability_verify_evidence_missing"
	ReasonCodeActionCapabilityApprovalScopeDrift        = "action_capability_approval_scope_drift"
	ReasonCodeActionCapabilityDeclaredObservedDrift     = "action_capability_declared_observed_drift"
	ReasonCodeActionCapabilityDuplicateConflict         = "action_capability_duplicate_conflict"
	ReasonCodeActionCapabilityRunStreamParityDrift      = "action_capability_run_stream_evidence_parity_drift"
	ReasonCodeActionCapabilityLibraryFirstBoundary      = "action_capability_library_first_boundary_violation"
	ReasonCodeActionCapabilityMetadataMissing           = "action_capability_metadata_missing"
	ReasonCodeActionCapabilityRetryIdempotencyConflict  = "action_capability_retry_idempotency_conflict"
	ReasonCodeActionCapabilityEvidenceInsufficient      = "action_capability_evidence_insufficient"
	ReasonCodeActionCapabilityStageEvidenceMissing      = "action_capability_stage_evidence_missing"
	ReasonCodeActionCapabilityStageEvidenceInsufficient = "action_capability_stage_evidence_insufficient"
	ReasonCodeActionCapabilityVerdictDrift              = "action_capability_verdict_drift"
	ReasonCodeActionCapabilityEvidenceCorrelationDrift  = "action_capability_evidence_correlation_drift"
)

const (
	actionCapabilityMaxString       = 256
	actionCapabilityMaxItems        = 64
	actionCapabilityMaxSummary      = 256
	actionCapabilityMaxFixtureBytes = 1 << 20
	actionCapabilityMaxTimeoutMS    = int64(24 * 60 * 60 * 1000)
	actionCapabilityMaxRetry        = 10
)

// ActionCapabilityAuditFixture is a bounded, host-supplied offline replay input.
type ActionCapabilityAuditFixture struct {
	Version string                      `json:"version"`
	Cases   []ActionCapabilityAuditCase `json:"cases"`
}

type ActionCapabilityAuditCase struct {
	CaseID   string                         `json:"case_id"`
	Action   ActionCapabilityDescriptor     `json:"action"`
	Observed *ActionCapabilityObservedFacts `json:"observed,omitempty"`
	Evidence []ActionCapabilityEvidence     `json:"evidence,omitempty"`
	Run      *ActionCapabilityEvidenceLane  `json:"run,omitempty"`
	Stream   *ActionCapabilityEvidenceLane  `json:"stream,omitempty"`
	Expected ActionCapabilityAuditExpected  `json:"expected,omitempty"`
}

// ActionCapabilityDescriptor contains only normalized capability metadata.
// Pointer booleans preserve the difference between false and an omitted value.
type ActionCapabilityDescriptor struct {
	Identity      string   `json:"identity"`
	Source        string   `json:"source"`
	Namespace     string   `json:"namespace,omitempty"`
	Tool          string   `json:"tool,omitempty"`
	Version       string   `json:"version,omitempty"`
	Digest        string   `json:"digest,omitempty"`
	Owner         string   `json:"owner,omitempty"`
	Scope         string   `json:"scope,omitempty"`
	Effect        string   `json:"effect,omitempty"`
	SideEffect    string   `json:"side_effect,omitempty"`
	Risk          string   `json:"risk,omitempty"`
	Reversible    *bool    `json:"reversible,omitempty"`
	Idempotent    *bool    `json:"idempotent,omitempty"`
	Retryable     *bool    `json:"retryable,omitempty"`
	Preconditions []string `json:"preconditions,omitempty"`
	TimeoutMS     int64    `json:"timeout_ms,omitempty"`
	RetryMax      int      `json:"retry_max,omitempty"`
	Stages        []string `json:"stages,omitempty"`
}

type ActionCapabilityObservedFacts struct {
	Effect     string `json:"effect,omitempty"`
	SideEffect string `json:"side_effect,omitempty"`
	Risk       string `json:"risk,omitempty"`
	Idempotent *bool  `json:"idempotent,omitempty"`
	Started    *bool  `json:"started,omitempty"`
}

type ActionCapabilityEvidence struct {
	ID             string `json:"id"`
	Stage          string `json:"stage"`
	Status         string `json:"status"`
	ActionIdentity string `json:"action_identity,omitempty"`
	Version        string `json:"version,omitempty"`
	Scope          string `json:"scope,omitempty"`
	AttemptID      string `json:"attempt_id,omitempty"`
	CorrelationID  string `json:"correlation_id,omitempty"`
	Summary        string `json:"summary,omitempty"`
}

type ActionCapabilityEvidenceLane struct {
	Evidence []ActionCapabilityEvidence     `json:"evidence,omitempty"`
	Observed *ActionCapabilityObservedFacts `json:"observed,omitempty"`
}

type ActionCapabilityAuditExpected struct {
	Verdict    string   `json:"verdict,omitempty"`
	DriftCodes []string `json:"drift_codes,omitempty"`
}

type ActionCapabilityAuditResult struct {
	Version string                            `json:"version"`
	Cases   []ActionCapabilityAuditCaseResult `json:"cases"`
}

type ActionCapabilityAuditCaseResult struct {
	CaseID          string   `json:"case_id"`
	Verdict         string   `json:"verdict"`
	DriftCodes      []string `json:"drift_codes,omitempty"`
	EvidenceState   string   `json:"evidence_state,omitempty"`
	Digest          string   `json:"digest"`
	ReplayDigest    string   `json:"replay_digest"`
	Idempotent      bool     `json:"idempotent"`
	RunStreamParity string   `json:"run_stream_parity,omitempty"`
}

type actionCapabilityEvaluation struct {
	Verdict       string
	DriftCodes    []string
	EvidenceState string
	Digest        string
}

func validateActionCapabilityJSONBoundary(raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if err := scanActionCapabilityJSONValue(decoder, "root"); err != nil {
		if validationErr, ok := err.(*ValidationError); ok {
			return validationErr
		}
		return &ValidationError{Code: ReasonCodeInvalidJSON, Message: err.Error()}
	}
	if _, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return &ValidationError{Code: ReasonCodeActionCapabilitySchemaDrift, Message: "multiple JSON values are not allowed"}
		}
		return &ValidationError{Code: ReasonCodeInvalidJSON, Message: err.Error()}
	}
	return nil
}

func scanActionCapabilityJSONValue(decoder *json.Decoder, shape string) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delim, isDelim := token.(json.Delim)
	if !isDelim {
		return nil
	}
	switch delim {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return &ValidationError{Code: ReasonCodeActionCapabilitySchemaDrift, Message: "object key is not a string"}
			}
			if _, exists := seen[key]; exists {
				return &ValidationError{Code: ReasonCodeActionCapabilityDuplicateConflict, Message: key}
			}
			seen[key] = struct{}{}
			if sensitiveJSONField(key) {
				return &ValidationError{Code: ReasonCodeActionCapabilityPrivacyOrBoundViolation, Message: "raw or sensitive JSON field is not allowed"}
			}
			childShape, allowed := actionCapabilityJSONFieldShape(shape, key)
			if !allowed {
				return &ValidationError{Code: ReasonCodeActionCapabilitySchemaDrift, Message: "unknown JSON field: " + key}
			}
			if err := scanActionCapabilityJSONValue(decoder, childShape); err != nil {
				return err
			}
		}
		_, err = decoder.Token()
		return err
	case '[':
		for decoder.More() {
			if err := scanActionCapabilityJSONValue(decoder, actionCapabilityJSONArrayItemShape(shape)); err != nil {
				return err
			}
		}
		_, err = decoder.Token()
		return err
	default:
		return &ValidationError{Code: ReasonCodeActionCapabilitySchemaDrift, Message: "unexpected JSON delimiter"}
	}
}

func actionCapabilityJSONFieldShape(shape, key string) (string, bool) {
	fields := map[string]map[string]string{
		"root":     {"version": "scalar", "cases": "cases"},
		"case":     {"case_id": "scalar", "action": "action", "observed": "observed", "evidence": "evidence", "run": "lane", "stream": "lane", "expected": "expected"},
		"action":   {"identity": "scalar", "source": "scalar", "namespace": "scalar", "tool": "scalar", "version": "scalar", "digest": "scalar", "owner": "scalar", "scope": "scalar", "effect": "scalar", "side_effect": "scalar", "risk": "scalar", "reversible": "scalar", "idempotent": "scalar", "retryable": "scalar", "preconditions": "strings", "timeout_ms": "scalar", "retry_max": "scalar", "stages": "strings"},
		"observed": {"effect": "scalar", "side_effect": "scalar", "risk": "scalar", "idempotent": "scalar", "started": "scalar"},
		"lane":     {"evidence": "evidence", "observed": "observed"},
		"evidence": {"id": "scalar", "stage": "scalar", "status": "scalar", "action_identity": "scalar", "version": "scalar", "scope": "scalar", "attempt_id": "scalar", "correlation_id": "scalar", "summary": "scalar"},
		"expected": {"verdict": "scalar", "drift_codes": "strings"},
	}
	child, ok := fields[shape][key]
	return child, ok
}

func actionCapabilityJSONArrayItemShape(shape string) string {
	switch shape {
	case "cases":
		return "case"
	case "evidence":
		return "evidence"
	default:
		return "scalar"
	}
}

func sensitiveJSONField(key string) bool {
	lower := strings.ToLower(strings.TrimSpace(key))
	for _, marker := range []string{"payload", "credential", "password", "reasoning", "command", "response"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

// ReplayActionCapabilityAuditJSON replays a versioned action capability audit
// without invoking tools, providers, registries, networks, clocks, or stores.
func ReplayActionCapabilityAuditJSON(raw []byte) (ActionCapabilityAuditResult, error) {
	if len(raw) > actionCapabilityMaxFixtureBytes {
		return ActionCapabilityAuditResult{}, &ValidationError{Code: ReasonCodeActionCapabilityPrivacyOrBoundViolation, Message: "fixture exceeds byte bound"}
	}
	if err := validateActionCapabilityJSONBoundary(raw); err != nil {
		return ActionCapabilityAuditResult{}, err
	}
	var fixture ActionCapabilityAuditFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		return ActionCapabilityAuditResult{}, &ValidationError{Code: ReasonCodeInvalidJSON, Message: err.Error()}
	}
	if strings.TrimSpace(fixture.Version) == "" {
		return ActionCapabilityAuditResult{}, &ValidationError{Code: ReasonCodeActionCapabilitySchemaDrift, Message: "version is required"}
	}
	if fixture.Version != ActionCapabilityAuditVersion {
		return ActionCapabilityAuditResult{}, &ValidationError{Code: ReasonCodeActionCapabilityUnknownVersion, Message: fixture.Version}
	}
	if len(fixture.Cases) == 0 || len(fixture.Cases) > actionCapabilityMaxItems {
		return ActionCapabilityAuditResult{}, &ValidationError{Code: ReasonCodeActionCapabilitySchemaDrift, Message: "cases must contain between one and 64 items"}
	}

	result := ActionCapabilityAuditResult{Version: fixture.Version, Cases: make([]ActionCapabilityAuditCaseResult, 0, len(fixture.Cases))}
	for _, item := range fixture.Cases {
		first, err := replayActionCapabilityAuditCase(item)
		if err != nil {
			return ActionCapabilityAuditResult{}, err
		}
		second, err := replayActionCapabilityAuditCase(item)
		if err != nil {
			return ActionCapabilityAuditResult{}, err
		}
		if first.Digest != second.Digest {
			return ActionCapabilityAuditResult{}, &ValidationError{Code: ReasonCodeActionCapabilitySchemaDrift, Message: "replay digest changed"}
		}
		first.ReplayDigest = second.Digest
		first.Idempotent = first.Digest == first.ReplayDigest
		result.Cases = append(result.Cases, first)
	}
	return result, nil
}

func replayActionCapabilityAuditCase(input ActionCapabilityAuditCase) (ActionCapabilityAuditCaseResult, error) {
	caseID := strings.TrimSpace(input.CaseID)
	if caseID == "" {
		return ActionCapabilityAuditCaseResult{}, &ValidationError{Code: ReasonCodeActionCapabilitySchemaDrift, Message: "case_id is required"}
	}
	if err := validateBoundedString("case_id", caseID, actionCapabilityMaxString); err != nil {
		return ActionCapabilityAuditCaseResult{}, err
	}
	if err := validateActionCapabilityDescriptor(input.Action); err != nil {
		return ActionCapabilityAuditCaseResult{}, err
	}

	base, err := evaluateActionCapability(input.Action, input.Observed, input.Evidence)
	if err != nil {
		return ActionCapabilityAuditCaseResult{}, err
	}
	result := ActionCapabilityAuditCaseResult{
		CaseID:          caseID,
		Verdict:         base.Verdict,
		DriftCodes:      append([]string(nil), base.DriftCodes...),
		EvidenceState:   base.EvidenceState,
		Digest:          base.Digest,
		RunStreamParity: "not_applicable",
	}

	if input.Run != nil || input.Stream != nil {
		if input.Run == nil || input.Stream == nil {
			return ActionCapabilityAuditCaseResult{}, &ValidationError{Code: ReasonCodeActionCapabilityRunStreamParityDrift, Message: "both run and stream projections are required"}
		}
		run, err := evaluateActionCapability(input.Action, input.Run.Observed, input.Run.Evidence)
		if err != nil {
			return ActionCapabilityAuditCaseResult{}, err
		}
		stream, err := evaluateActionCapability(input.Action, input.Stream.Observed, input.Stream.Evidence)
		if err != nil {
			return ActionCapabilityAuditCaseResult{}, err
		}
		if run.Verdict != stream.Verdict || !reflect.DeepEqual(run.DriftCodes, stream.DriftCodes) || run.EvidenceState != stream.EvidenceState || digestActionCapabilityEvidenceSemantics(input.Run.Evidence) != digestActionCapabilityEvidenceSemantics(input.Stream.Evidence) {
			result.Verdict = ActionCapabilityVerdictGap
			result.DriftCodes = sortedUnique(append(append(append([]string(nil), result.DriftCodes...), run.DriftCodes...), stream.DriftCodes...))
			result.DriftCodes = sortedUnique(appendUnique(result.DriftCodes, ReasonCodeActionCapabilityRunStreamParityDrift))
			result.EvidenceState = "divergent"
			result.RunStreamParity = "drift"
		} else {
			result.Verdict = run.Verdict
			result.DriftCodes = append([]string(nil), run.DriftCodes...)
			result.EvidenceState = run.EvidenceState
			result.RunStreamParity = "equivalent"
		}
		result.Digest = digestActionCapabilityRunStream(base.Digest, run.Digest, stream.Digest)
	}

	if input.Expected.Verdict != "" && input.Expected.Verdict != result.Verdict {
		return ActionCapabilityAuditCaseResult{}, &ValidationError{Code: ReasonCodeActionCapabilityVerdictDrift, Message: fmt.Sprintf("case %q expected %q got %q", caseID, input.Expected.Verdict, result.Verdict)}
	}
	if input.Expected.DriftCodes != nil {
		want := sortedUnique(input.Expected.DriftCodes)
		if !reflect.DeepEqual(want, result.DriftCodes) {
			return ActionCapabilityAuditCaseResult{}, &ValidationError{Code: ReasonCodeActionCapabilityVerdictDrift, Message: fmt.Sprintf("case %q drift codes mismatch", caseID)}
		}
	}
	return result, nil
}

func validateActionCapabilityDescriptor(action ActionCapabilityDescriptor) error {
	if strings.TrimSpace(action.Identity) == "" && strings.TrimSpace(action.Source) == "" && strings.TrimSpace(action.Tool) == "" && strings.TrimSpace(action.Version) == "" && strings.TrimSpace(action.Digest) == "" {
		return nil // historical fixture with no extension fields
	}
	for name, value := range map[string]string{
		"identity": action.Identity, "source": action.Source, "namespace": action.Namespace, "tool": action.Tool,
		"version": action.Version, "digest": action.Digest, "owner": action.Owner, "scope": action.Scope,
		"effect": action.Effect, "side_effect": action.SideEffect, "risk": action.Risk,
	} {
		if err := validateBoundedString(name, value, actionCapabilityMaxString); err != nil {
			return err
		}
		if containsSensitiveMarker(value) {
			return &ValidationError{Code: ReasonCodeActionCapabilityPrivacyOrBoundViolation, Message: name + " contains sensitive or raw content"}
		}
	}
	if strings.Contains(strings.ToLower(action.Source), "discover") || strings.Contains(strings.ToLower(action.Source), "registry") || strings.Contains(strings.ToLower(action.Source), "network") || strings.Contains(strings.ToLower(action.Source), "provider") {
		return &ValidationError{Code: ReasonCodeActionCapabilityExecutionOrDiscovery, Message: "action source indicates dynamic discovery or execution"}
	}
	if strings.TrimSpace(action.Identity) == "" || strings.TrimSpace(action.Source) == "" {
		return &ValidationError{Code: ReasonCodeActionCapabilitySchemaDrift, Message: "action identity and source are required"}
	}
	if strings.TrimSpace(action.Version) == "" && strings.TrimSpace(action.Digest) == "" {
		return &ValidationError{Code: ReasonCodeActionCapabilitySchemaDrift, Message: "action version or digest is required"}
	}
	if len(action.Preconditions) > actionCapabilityMaxItems || len(action.Stages) > actionCapabilityMaxItems {
		return &ValidationError{Code: ReasonCodeActionCapabilityPrivacyOrBoundViolation, Message: "action list exceeds bound"}
	}
	for _, value := range append(append([]string(nil), action.Preconditions...), action.Stages...) {
		if err := validateBoundedString("action list item", value, actionCapabilityMaxString); err != nil {
			return err
		}
		if containsSensitiveMarker(value) {
			return &ValidationError{Code: ReasonCodeActionCapabilityPrivacyOrBoundViolation, Message: "action list item contains sensitive or raw content"}
		}
	}
	if action.TimeoutMS < 0 || action.TimeoutMS > actionCapabilityMaxTimeoutMS || action.RetryMax < 0 || action.RetryMax > actionCapabilityMaxRetry {
		return &ValidationError{Code: ReasonCodeActionCapabilityPrivacyOrBoundViolation, Message: "timeout or retry value exceeds bound"}
	}
	return nil
}

func evaluateActionCapability(action ActionCapabilityDescriptor, observed *ActionCapabilityObservedFacts, evidence []ActionCapabilityEvidence) (actionCapabilityEvaluation, error) {
	if strings.TrimSpace(action.Identity) == "" && strings.TrimSpace(action.Source) == "" {
		return actionCapabilityEvaluation{Verdict: ActionCapabilityVerdictNotApplicable, EvidenceState: "not_applicable", Digest: digestActionCapability(action, nil, nil)}, nil
	}
	normalizedAction := normalizeActionCapabilityDescriptor(action)
	normalizedEvidence, err := normalizeActionCapabilityEvidence(normalizedAction, evidence)
	if err != nil {
		return actionCapabilityEvaluation{}, err
	}
	drifts := make([]string, 0)
	metadataMissing := false
	for _, missing := range []bool{
		normalizedAction.Effect == "", normalizedAction.SideEffect == "", normalizedAction.Risk == "", normalizedAction.Owner == "", normalizedAction.Scope == "",
		normalizedAction.Reversible == nil, normalizedAction.Idempotent == nil, normalizedAction.Retryable == nil, normalizedAction.TimeoutMS <= 0,
	} {
		if missing {
			metadataMissing = true
			break
		}
	}
	if metadataMissing {
		drifts = appendUnique(drifts, ReasonCodeActionCapabilityMetadataMissing)
	}
	if normalizedAction.Retryable != nil && *normalizedAction.Retryable && normalizedAction.Idempotent != nil && !*normalizedAction.Idempotent {
		drifts = appendUnique(drifts, ReasonCodeActionCapabilityRetryIdempotencyConflict)
	}
	if observed != nil {
		if err := validateActionCapabilityObservedFacts(*observed); err != nil {
			return actionCapabilityEvaluation{}, err
		}
		if normalizedObserved := normalizeObserved(*observed); normalizedObserved != (ActionCapabilityObservedFacts{}) {
			if normalizedObserved.Effect != "" && normalizedObserved.Effect != normalizedAction.Effect || normalizedObserved.SideEffect != "" && normalizedObserved.SideEffect != normalizedAction.SideEffect || normalizedObserved.Risk != "" && normalizedObserved.Risk != normalizedAction.Risk || normalizedObserved.Idempotent != nil && (normalizedAction.Idempotent == nil || *normalizedObserved.Idempotent != *normalizedAction.Idempotent) {
				drifts = appendUnique(drifts, ReasonCodeActionCapabilityDeclaredObservedDrift)
			}
		}
	}

	stageSet := make(map[string]bool, len(normalizedEvidence))
	approvalScope := ""
	commitScope := ""
	intent, issued, confirmed := false, false, false
	issuedEvidence := make([]ActionCapabilityEvidence, 0)
	confirmedEvidence := make([]ActionCapabilityEvidence, 0)
	evidenceCorrelationIncomplete := false
	evidenceCorrelationConflict := false
	for _, item := range normalizedEvidence {
		stageSet[item.Stage] = true
		if (item.ActionIdentity != "" && item.ActionIdentity != normalizedAction.Identity) || (item.Version != "" && item.Version != actionCapabilityVersion(normalizedAction)) {
			evidenceCorrelationConflict = true
		}
		switch item.Stage {
		case "intent":
			intent = true
		case "issued":
			issued = true
			issuedEvidence = append(issuedEvidence, item)
		case "confirmed":
			if item.Status == "confirmed" {
				confirmedEvidence = append(confirmedEvidence, item)
			}
		case "approve":
			approvalScope = item.Scope
		case "commit":
			commitScope = item.Scope
		}
	}
	if observed != nil && observed.Started != nil && *observed.Started != issued {
		drifts = appendUnique(drifts, ReasonCodeActionCapabilityDeclaredObservedDrift)
		evidenceCorrelationConflict = true
	}
	if evidenceCorrelationConflict {
		drifts = appendUnique(drifts, ReasonCodeActionCapabilityEvidenceCorrelationDrift)
	}
	if len(confirmedEvidence) > 0 {
		if len(issuedEvidence) == 0 {
			evidenceCorrelationIncomplete = true
		} else {
			confirmed = true
			for _, issue := range issuedEvidence {
				matched := false
				for _, confirmation := range confirmedEvidence {
					if sameActionCapabilityOperation(normalizedAction, issue, confirmation) {
						matched = true
						break
					}
				}
				if !matched {
					confirmed = false
					if hasCompleteActionCapabilityCorrelation(issue) && hasCompleteActionCapabilityCorrelation(confirmedEvidence[0]) {
						evidenceCorrelationConflict = true
					} else {
						evidenceCorrelationIncomplete = true
					}
				}
			}
		}
	}
	if evidenceCorrelationConflict {
		drifts = appendUnique(drifts, ReasonCodeActionCapabilityEvidenceCorrelationDrift)
	}
	if evidenceCorrelationIncomplete {
		drifts = appendUnique(drifts, ReasonCodeActionCapabilityEvidenceInsufficient)
	}
	if approvalScope != "" && commitScope != "" && approvalScope != commitScope {
		drifts = appendUnique(drifts, ReasonCodeActionCapabilityApprovalScopeDrift)
	}
	if strings.EqualFold(normalizedAction.SideEffect, "external") || strings.EqualFold(normalizedAction.Risk, "high") || strings.EqualFold(normalizedAction.Risk, "critical") || normalizedAction.Reversible != nil && !*normalizedAction.Reversible {
		for _, stage := range []string{"preview", "approve", "commit", "verify"} {
			if !containsActionCapabilityString(normalizedAction.Stages, stage) || !stageSet[stage] {
				if stage == "verify" {
					drifts = appendUnique(drifts, ReasonCodeActionCapabilityVerifyEvidenceMissing)
				} else {
					drifts = appendUnique(drifts, ReasonCodeActionCapabilityStageEvidenceMissing)
				}
			}
			for _, item := range normalizedEvidence {
				if item.Stage == stage && !validActionCapabilityStageStatus(stage, item.Status) {
					evidenceCorrelationIncomplete = true
					drifts = appendUnique(drifts, ReasonCodeActionCapabilityStageEvidenceInsufficient)
				}
			}
			for _, item := range normalizedEvidence {
				if item.Stage != stage {
					continue
				}
				if item.ActionIdentity == "" || item.Version == "" || item.Scope == "" {
					evidenceCorrelationIncomplete = true
					continue
				}
				if item.ActionIdentity != normalizedAction.Identity || item.Version != actionCapabilityVersion(normalizedAction) || normalizedAction.Scope != "" && item.Scope != normalizedAction.Scope {
					evidenceCorrelationConflict = true
				}
			}
		}
	}
	if evidenceCorrelationConflict {
		drifts = appendUnique(drifts, ReasonCodeActionCapabilityEvidenceCorrelationDrift)
	}
	if evidenceCorrelationIncomplete {
		drifts = appendUnique(drifts, ReasonCodeActionCapabilityEvidenceInsufficient)
	}

	evidenceState := "none"
	if intent {
		evidenceState = "intent"
	}
	if issued {
		evidenceState = "issued"
	}
	if confirmed {
		evidenceState = "confirmed"
	}
	verdict := ActionCapabilityVerdictCompliant
	switch {
	case evidenceCorrelationConflict || containsActionCapabilityString(drifts, ReasonCodeActionCapabilityDeclaredObservedDrift) || containsActionCapabilityString(drifts, ReasonCodeActionCapabilityApprovalScopeDrift) || containsActionCapabilityString(drifts, ReasonCodeActionCapabilityRetryIdempotencyConflict):
		verdict = ActionCapabilityVerdictGap
	case evidenceCorrelationIncomplete || issued && !confirmed || len(confirmedEvidence) > 0 && !confirmed:
		verdict = ActionCapabilityVerdictInsufficientEvidence
		drifts = appendUnique(drifts, ReasonCodeActionCapabilityEvidenceInsufficient)
	case len(drifts) > 0:
		verdict = ActionCapabilityVerdictGap
	case intent && !issued:
		verdict = ActionCapabilityVerdictInsufficientEvidence
		drifts = appendUnique(drifts, ReasonCodeActionCapabilityEvidenceInsufficient)
	}
	return actionCapabilityEvaluation{Verdict: verdict, DriftCodes: sortedUnique(drifts), EvidenceState: evidenceState, Digest: digestActionCapability(normalizedAction, normalizedEvidence, normalizeObservedPointer(observed))}, nil
}

func normalizeActionCapabilityDescriptor(action ActionCapabilityDescriptor) ActionCapabilityDescriptor {
	action.Identity, action.Source, action.Namespace, action.Tool, action.Version, action.Digest, action.Owner, action.Scope, action.Effect, action.SideEffect, action.Risk = strings.TrimSpace(action.Identity), strings.TrimSpace(action.Source), strings.TrimSpace(action.Namespace), strings.TrimSpace(action.Tool), strings.TrimSpace(action.Version), strings.TrimSpace(action.Digest), strings.TrimSpace(action.Owner), strings.TrimSpace(action.Scope), strings.ToLower(strings.TrimSpace(action.Effect)), strings.ToLower(strings.TrimSpace(action.SideEffect)), strings.ToLower(strings.TrimSpace(action.Risk))
	action.Preconditions = sortedUnique(action.Preconditions)
	action.Stages = sortedUniqueNormalized(action.Stages)
	return action
}

func normalizeObserved(observed ActionCapabilityObservedFacts) ActionCapabilityObservedFacts {
	observed.Effect, observed.SideEffect, observed.Risk = strings.ToLower(strings.TrimSpace(observed.Effect)), strings.ToLower(strings.TrimSpace(observed.SideEffect)), strings.ToLower(strings.TrimSpace(observed.Risk))
	return observed
}

func validateActionCapabilityObservedFacts(observed ActionCapabilityObservedFacts) error {
	for name, value := range map[string]string{"observed effect": observed.Effect, "observed side_effect": observed.SideEffect, "observed risk": observed.Risk} {
		if err := validateBoundedString(name, value, actionCapabilityMaxString); err != nil {
			return err
		}
		if containsSensitiveMarker(value) {
			return &ValidationError{Code: ReasonCodeActionCapabilityPrivacyOrBoundViolation, Message: name + " contains sensitive or raw content"}
		}
	}
	return nil
}

func validActionCapabilityStageStatus(stage, status string) bool {
	switch stage {
	case "preview":
		return status == "observed" || status == "confirmed"
	case "approve":
		return status == "allowed" || status == "confirmed"
	case "commit":
		return status == "observed" || status == "confirmed"
	case "verify":
		return status == "confirmed"
	default:
		return true
	}
}

func actionCapabilityVersion(action ActionCapabilityDescriptor) string {
	if action.Version != "" {
		return action.Version
	}
	return action.Digest
}

func hasCompleteActionCapabilityCorrelation(item ActionCapabilityEvidence) bool {
	return item.ActionIdentity != "" && item.Version != "" && item.Scope != "" && item.AttemptID != "" && item.CorrelationID != ""
}

func sameActionCapabilityOperation(action ActionCapabilityDescriptor, issued, confirmed ActionCapabilityEvidence) bool {
	return hasCompleteActionCapabilityCorrelation(issued) && hasCompleteActionCapabilityCorrelation(confirmed) &&
		issued.ActionIdentity == confirmed.ActionIdentity && issued.ActionIdentity == action.Identity &&
		issued.Version == confirmed.Version && issued.Version == actionCapabilityVersion(action) &&
		issued.Scope == confirmed.Scope && issued.Scope == action.Scope &&
		issued.AttemptID == confirmed.AttemptID && issued.CorrelationID == confirmed.CorrelationID
}

func normalizeObservedPointer(observed *ActionCapabilityObservedFacts) *ActionCapabilityObservedFacts {
	if observed == nil {
		return nil
	}
	normalized := normalizeObserved(*observed)
	return &normalized
}

func normalizeActionCapabilityEvidence(action ActionCapabilityDescriptor, evidence []ActionCapabilityEvidence) ([]ActionCapabilityEvidence, error) {
	if len(evidence) > actionCapabilityMaxItems {
		return nil, &ValidationError{Code: ReasonCodeActionCapabilityPrivacyOrBoundViolation, Message: "evidence exceeds bound"}
	}
	byID := make(map[string]ActionCapabilityEvidence, len(evidence))
	for _, item := range evidence {
		item.ID, item.Stage, item.Status, item.ActionIdentity, item.Version, item.Scope, item.AttemptID, item.CorrelationID, item.Summary = strings.TrimSpace(item.ID), strings.ToLower(strings.TrimSpace(item.Stage)), strings.ToLower(strings.TrimSpace(item.Status)), strings.TrimSpace(item.ActionIdentity), strings.TrimSpace(item.Version), strings.TrimSpace(item.Scope), strings.TrimSpace(item.AttemptID), strings.TrimSpace(item.CorrelationID), strings.TrimSpace(item.Summary)
		if item.ID == "" || item.Stage == "" || item.Status == "" {
			return nil, &ValidationError{Code: ReasonCodeActionCapabilitySchemaDrift, Message: "evidence id, stage, and status are required"}
		}
		for name, value := range map[string]string{"evidence id": item.ID, "evidence stage": item.Stage, "evidence status": item.Status, "action identity": item.ActionIdentity, "evidence version": item.Version, "evidence scope": item.Scope, "attempt id": item.AttemptID, "correlation id": item.CorrelationID} {
			if err := validateBoundedString(name, value, actionCapabilityMaxString); err != nil {
				return nil, err
			}
			if containsSensitiveMarker(value) {
				return nil, &ValidationError{Code: ReasonCodeActionCapabilityPrivacyOrBoundViolation, Message: name + " contains sensitive or raw content"}
			}
		}
		if err := validateBoundedString("evidence summary", item.Summary, actionCapabilityMaxSummary); err != nil {
			return nil, err
		}
		if containsSensitiveMarker(item.Summary) {
			return nil, &ValidationError{Code: ReasonCodeActionCapabilityPrivacyOrBoundViolation, Message: "evidence summary contains sensitive or raw content"}
		}
		if item.ActionIdentity != "" && item.ActionIdentity != action.Identity {
			return nil, &ValidationError{Code: ReasonCodeActionCapabilityDeclaredObservedDrift, Message: "evidence action identity mismatch"}
		}
		if previous, ok := byID[item.ID]; ok && !reflect.DeepEqual(previous, item) {
			return nil, &ValidationError{Code: ReasonCodeActionCapabilityDuplicateConflict, Message: item.ID}
		} else if ok {
			continue
		}
		byID[item.ID] = item
	}
	out := make([]ActionCapabilityEvidence, 0, len(byID))
	for _, item := range byID {
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func validateBoundedString(name, value string, max int) error {
	if len(value) > max {
		return &ValidationError{Code: ReasonCodeActionCapabilityPrivacyOrBoundViolation, Message: name + " exceeds bound"}
	}
	return nil
}

func containsSensitiveMarker(value string) bool {
	lower := strings.ToLower(value)
	for _, marker := range []string{"bearer ", "password", "secret", "reasoning", "command output", "credential", "raw_payload"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	for offset := 0; ; {
		index := strings.Index(lower[offset:], "sk-")
		if index < 0 {
			break
		}
		start := offset + index + len("sk-")
		end := start
		for end < len(lower) && !strings.ContainsRune(" \t\r\n,;\"'", rune(lower[end])) {
			end++
		}
		if end-start >= 24 {
			return true
		}
		offset = start
		if offset >= len(lower) {
			break
		}
	}
	return false
}

func digestActionCapability(action ActionCapabilityDescriptor, evidence []ActionCapabilityEvidence, observed *ActionCapabilityObservedFacts) string {
	canonical := struct {
		Action   ActionCapabilityDescriptor     `json:"action"`
		Evidence []ActionCapabilityEvidence     `json:"evidence,omitempty"`
		Observed *ActionCapabilityObservedFacts `json:"observed,omitempty"`
	}{Action: normalizeActionCapabilityDescriptor(action), Evidence: evidence, Observed: normalizeObservedPointer(observed)}
	raw, _ := json.Marshal(canonical)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func digestActionCapabilityRunStream(baseDigest, runDigest, streamDigest string) string {
	canonical := struct {
		Base   string `json:"base"`
		Run    string `json:"run"`
		Stream string `json:"stream"`
	}{Base: baseDigest, Run: runDigest, Stream: streamDigest}
	raw, _ := json.Marshal(canonical)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func digestActionCapabilityEvidenceSemantics(evidence []ActionCapabilityEvidence) string {
	semantic := make([]struct {
		Stage          string `json:"stage"`
		Status         string `json:"status"`
		ActionIdentity string `json:"action_identity,omitempty"`
		Version        string `json:"version,omitempty"`
		Scope          string `json:"scope,omitempty"`
		AttemptID      string `json:"attempt_id,omitempty"`
		CorrelationID  string `json:"correlation_id,omitempty"`
		Summary        string `json:"summary,omitempty"`
	}, 0, len(evidence))
	for _, item := range evidence {
		semantic = append(semantic, struct {
			Stage          string `json:"stage"`
			Status         string `json:"status"`
			ActionIdentity string `json:"action_identity,omitempty"`
			Version        string `json:"version,omitempty"`
			Scope          string `json:"scope,omitempty"`
			AttemptID      string `json:"attempt_id,omitempty"`
			CorrelationID  string `json:"correlation_id,omitempty"`
			Summary        string `json:"summary,omitempty"`
		}{Stage: item.Stage, Status: item.Status, ActionIdentity: item.ActionIdentity, Version: item.Version, Scope: item.Scope, AttemptID: item.AttemptID, CorrelationID: item.CorrelationID, Summary: item.Summary})
	}
	sort.Slice(semantic, func(i, j int) bool {
		left, _ := json.Marshal(semantic[i])
		right, _ := json.Marshal(semantic[j])
		return string(left) < string(right)
	})
	raw, _ := json.Marshal(semantic)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
func sortedUnique(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = appendUnique(out, value)
		}
	}
	sort.Strings(out)
	return out
}
func sortedUniqueNormalized(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value != "" {
			out = appendUnique(out, value)
		}
	}
	sort.Strings(out)
	return out
}
func containsActionCapabilityString(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}
