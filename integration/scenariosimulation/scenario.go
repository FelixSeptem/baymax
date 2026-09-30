// Package scenariosimulation contains bounded, offline-only contracts used by
// integration tests. It deliberately has no dependency on runtime execution
// or provider packages.
package scenariosimulation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const (
	ScenarioVersionV1     = "scenario.v1"
	RunResultVersionV1    = "run_result.v1"
	MaxScenarioEvents     = 128
	MaxEvidenceReferences = 128
	MaxIdentifierLength   = 128
	MaxReferenceLength    = 512
	MaxSerializedBytes    = 256 << 10
)

const (
	ReasonScenarioSchemaDrift    = "scenario_schema_drift"
	ReasonScenarioVersionDrift   = "scenario_version_drift"
	ReasonOfflineScope           = "offline_scope_violation"
	ReasonPrivacyViolation       = "evidence_privacy_violation"
	ReasonEvidenceIncomplete     = "evidence_incomplete"
	ReasonEvidenceConflict       = "evidence_conflict"
	ReasonOutcomeIndeterminate   = "outcome_indeterminate"
	ReasonAdmissionIndeterminate = "admission_indeterminate"
	ReasonCausationDrift         = "event_causation_drift"
	ReasonCompletionDrift        = "completion_drift"
	ReasonOutcomeDrift           = "outcome_drift"
	ReasonAdmissionDrift         = "admission_drift"
	ReasonDuplicateConflict      = "duplicate_identity_conflict"
	ReasonLateCompletion         = "late_completion"
	ReasonDuplicateCompletion    = "duplicate_completion"
)

const (
	EventApproval         = "approval"
	EventRecovery         = "recovery"
	EventTool             = "tool"
	EventCancel           = "cancel"
	EventModel            = "model"
	EventStreamTruncation = "stream_truncation"
	EventCompletion       = "completion"
)

const (
	VerdictPass          = "pass"
	VerdictFail          = "fail"
	VerdictIndeterminate = "indeterminate"
	VerdictNotApplicable = "not_applicable"
)

// Scenario is the versioned, reference-first input to an offline case.
// Unknown is intentionally excluded from the canonical representation: it is
// a compatibility escape hatch for additive fields and never affects a digest.
type Scenario struct {
	Version       string              `json:"version"`
	ID            string              `json:"id"`
	Events        []PlannedEvent      `json:"events,omitempty"`
	Evidence      []EvidenceReference `json:"evidence,omitempty"`
	Outcome       *OutcomeReference   `json:"outcome,omitempty"`
	Admission     *AdmissionPolicy    `json:"admission,omitempty"`
	ExecutionMode string              `json:"execution_mode,omitempty"`
	Unknown       map[string]any      `json:"-"`
}

type PlannedEvent struct {
	ID          string            `json:"id"`
	Sequence    int               `json:"sequence"`
	Kind        string            `json:"kind"`
	Owner       string            `json:"owner"`
	CausationID string            `json:"causation_id,omitempty"`
	Decision    string            `json:"decision,omitempty"`
	Reason      string            `json:"reason,omitempty"`
	Reference   EvidenceReference `json:"reference,omitempty"`
	Expected    bool              `json:"expected,omitempty"`
}

type EvidenceReference struct {
	Owner         string `json:"owner"`
	ID            string `json:"id"`
	Digest        string `json:"digest,omitempty"`
	Version       string `json:"version,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`
	// Body and Payload are accepted only to provide a deterministic privacy
	// negative path; they are never included in a normalized result.
	Body    any            `json:"body,omitempty"`
	Payload map[string]any `json:"payload,omitempty"`
}

type OutcomeReference struct {
	Owner   string `json:"owner"`
	ID      string `json:"id"`
	Digest  string `json:"digest,omitempty"`
	Version string `json:"version,omitempty"`
}

type AdmissionPolicy struct {
	RequiredEvidence []string `json:"required_evidence,omitempty"`
	RequireOutcome   bool     `json:"require_outcome,omitempty"`
}

type Verdict struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

type RunResult struct {
	Version            string               `json:"version"`
	ScenarioID         string               `json:"scenario_id"`
	ScenarioDigest     string               `json:"scenario_digest"`
	Execution          Verdict              `json:"execution"`
	Evidence           Verdict              `json:"evidence"`
	Outcome            Verdict              `json:"outcome"`
	Admission          Verdict              `json:"admission"`
	Events             []ObservedEvent      `json:"events,omitempty"`
	EvidenceReferences []EvidenceReference  `json:"evidence_references,omitempty"`
	OutcomeReference   *OutcomeReference    `json:"outcome_reference,omitempty"`
	Completion         *CompletionReference `json:"completion,omitempty"`
	Unknown            map[string]any       `json:"-"`
}

type ObservedEvent struct {
	ID          string `json:"id"`
	Sequence    int    `json:"sequence"`
	Kind        string `json:"kind"`
	Owner       string `json:"owner"`
	CausationID string `json:"causation_id,omitempty"`
}

type CompletionReference struct {
	ID            string `json:"id"`
	CorrelationID string `json:"correlation_id,omitempty"`
	Committed     bool   `json:"committed"`
	Late          bool   `json:"late,omitempty"`
	Duplicate     bool   `json:"duplicate,omitempty"`
}

// ClassifyCompletion is a bounded projection of the existing completion
// owner. It does not decide or promote a completion; callers pass the source
// owner's committed/late/duplicate facts through unchanged.
func ClassifyCompletion(ref CompletionReference) Verdict {
	if ref.ID == "" {
		return Verdict{Status: VerdictIndeterminate, Reason: ReasonCompletionDrift}
	}
	if ref.Duplicate {
		return Verdict{Status: VerdictFail, Reason: ReasonDuplicateCompletion}
	}
	if ref.Late {
		return Verdict{Status: VerdictFail, Reason: ReasonLateCompletion}
	}
	if ref.Committed {
		return Verdict{Status: VerdictPass}
	}
	return Verdict{Status: VerdictIndeterminate, Reason: ReasonCompletionDrift}
}

func clean(s string, limit int) string {
	s = strings.TrimSpace(s)
	if len(s) > limit {
		return ""
	}
	return s
}

func normalizeReference(in *EvidenceReference) error {
	in.Owner = clean(in.Owner, MaxIdentifierLength)
	in.ID = clean(in.ID, MaxIdentifierLength)
	in.Digest = clean(in.Digest, MaxReferenceLength)
	in.Version = clean(in.Version, MaxIdentifierLength)
	in.CorrelationID = clean(in.CorrelationID, MaxIdentifierLength)
	if in.Owner == "" || in.ID == "" || (in.Body != nil || in.Payload != nil) {
		if in.Body != nil || in.Payload != nil {
			return fmt.Errorf("%s", ReasonPrivacyViolation)
		}
		return fmt.Errorf("%s", ReasonScenarioSchemaDrift)
	}
	return nil
}

// NormalizeScenario validates and canonicalizes a scenario. On error it
// returns zero values, ensuring callers cannot accidentally use partial data.
func NormalizeScenario(in Scenario) (Scenario, string, error) {
	in.Version = strings.ToLower(clean(in.Version, MaxIdentifierLength))
	in.ID = clean(in.ID, MaxIdentifierLength)
	if in.Version != ScenarioVersionV1 {
		return Scenario{}, "", fmt.Errorf("%s", ReasonScenarioVersionDrift)
	}
	if in.ID == "" || len(in.Events) > MaxScenarioEvents || len(in.Evidence) > MaxEvidenceReferences {
		return Scenario{}, "", fmt.Errorf("%s", ReasonScenarioSchemaDrift)
	}
	if in.Unknown != nil {
		extensionBytes, err := json.Marshal(in.Unknown)
		if err != nil || len(extensionBytes) > MaxSerializedBytes {
			return Scenario{}, "", fmt.Errorf("%s", ReasonScenarioSchemaDrift)
		}
	}
	seen := make(map[string]bool, len(in.Events))
	for i := range in.Events {
		e := &in.Events[i]
		e.ID = clean(e.ID, MaxIdentifierLength)
		e.Kind = clean(e.Kind, MaxIdentifierLength)
		e.Owner = clean(e.Owner, MaxIdentifierLength)
		e.CausationID = clean(e.CausationID, MaxIdentifierLength)
		e.Decision = clean(e.Decision, MaxIdentifierLength)
		e.Reason = clean(e.Reason, MaxReferenceLength)
		if e.ID == "" || e.Kind == "" || e.Owner == "" || e.Sequence < 0 || seen[e.ID] {
			return Scenario{}, "", fmt.Errorf("%s", ReasonScenarioSchemaDrift)
		}
		seen[e.ID] = true
		if e.Reference.Owner != "" || e.Reference.ID != "" || e.Reference.Digest != "" || e.Reference.Version != "" || e.Reference.CorrelationID != "" || e.Reference.Body != nil || e.Reference.Payload != nil {
			if err := normalizeReference(&e.Reference); err != nil {
				return Scenario{}, "", err
			}
		}
	}
	sort.SliceStable(in.Events, func(i, j int) bool {
		if in.Events[i].Sequence == in.Events[j].Sequence {
			return in.Events[i].ID < in.Events[j].ID
		}
		return in.Events[i].Sequence < in.Events[j].Sequence
	})
	for i := range in.Evidence {
		if err := normalizeReference(&in.Evidence[i]); err != nil {
			return Scenario{}, "", err
		}
	}
	sort.SliceStable(in.Evidence, func(i, j int) bool {
		if in.Evidence[i].Owner == in.Evidence[j].Owner {
			return in.Evidence[i].ID < in.Evidence[j].ID
		}
		return in.Evidence[i].Owner < in.Evidence[j].Owner
	})
	if in.Outcome != nil {
		normalizeOutcome(in.Outcome)
		if in.Outcome.Owner == "" || in.Outcome.ID == "" {
			return Scenario{}, "", fmt.Errorf("%s", ReasonScenarioSchemaDrift)
		}
	}
	if in.Admission != nil {
		sort.Strings(in.Admission.RequiredEvidence)
	}
	b, err := json.Marshal(in)
	if err != nil || len(b) > MaxSerializedBytes {
		return Scenario{}, "", fmt.Errorf("%s", ReasonScenarioSchemaDrift)
	}
	d, err := digestBytes(b)
	if err != nil {
		return Scenario{}, "", err
	}
	in.Unknown = nil
	return in, d, nil
}

func normalizeOutcome(o *OutcomeReference) {
	o.Owner = clean(o.Owner, MaxIdentifierLength)
	o.ID = clean(o.ID, MaxIdentifierLength)
	o.Digest = clean(o.Digest, MaxReferenceLength)
	o.Version = clean(o.Version, MaxIdentifierLength)
}

func digestBytes(b []byte) (string, error) {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

func Digest(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return digestBytes(b)
}

// ValidateEvidence checks only references, never dereferencing or copying bodies.
func ValidateEvidence(refs []EvidenceReference, required []string) (Verdict, error) {
	if len(refs) > MaxEvidenceReferences {
		return Verdict{}, fmt.Errorf("%s", ReasonEvidenceIncomplete)
	}
	seen := map[string]EvidenceReference{}
	for _, ref := range refs {
		if err := normalizeReference(&ref); err != nil {
			if ref.Body != nil || ref.Payload != nil {
				return Verdict{Status: VerdictFail, Reason: ReasonPrivacyViolation}, fmt.Errorf("%s", ReasonPrivacyViolation)
			}
			return Verdict{Status: VerdictFail, Reason: ReasonEvidenceIncomplete}, err
		}
		key := ref.Owner + "\x00" + ref.ID
		if prior, ok := seen[key]; ok && (prior.Digest != ref.Digest || prior.Version != ref.Version || prior.CorrelationID != ref.CorrelationID) {
			return Verdict{Status: VerdictFail, Reason: ReasonEvidenceConflict}, fmt.Errorf("%s", ReasonEvidenceConflict)
		}
		seen[key] = ref
	}
	for _, req := range required {
		found := false
		for key := range seen {
			if key == req || strings.HasSuffix(key, "\x00"+req) {
				found = true
				break
			}
		}
		if !found {
			return Verdict{Status: VerdictIndeterminate, Reason: ReasonEvidenceIncomplete}, fmt.Errorf("%s", ReasonEvidenceIncomplete)
		}
	}
	return Verdict{Status: VerdictPass}, nil
}

// VerifyRunResult evaluates evidence, outcome and admission independently.
func VerifyRunResult(result RunResult, requiredEvidence []string, requireOutcome bool) (RunResult, error) {
	result.Version = strings.ToLower(clean(result.Version, MaxIdentifierLength))
	if result.Version == "" {
		result.Version = RunResultVersionV1
	}
	if result.ScenarioID == "" {
		return RunResult{}, fmt.Errorf("%s", ReasonScenarioSchemaDrift)
	}
	ev, err := ValidateEvidence(result.EvidenceReferences, requiredEvidence)
	result.Evidence = ev
	if err != nil && ev.Reason == ReasonPrivacyViolation {
		return result, err
	}
	if result.OutcomeReference == nil || result.OutcomeReference.ID == "" || result.OutcomeReference.Owner == "" {
		result.Outcome = Verdict{Status: VerdictIndeterminate, Reason: ReasonOutcomeIndeterminate}
	} else {
		result.Outcome = Verdict{Status: VerdictPass}
	}
	switch {
	case requireOutcome && result.Outcome.Status != VerdictPass:
		result.Admission = Verdict{Status: VerdictIndeterminate, Reason: ReasonAdmissionIndeterminate}
	case result.Evidence.Status != VerdictPass:
		result.Admission = Verdict{Status: VerdictIndeterminate, Reason: ReasonAdmissionIndeterminate}
	default:
		result.Admission = Verdict{Status: VerdictPass}
	}
	return result, err
}
