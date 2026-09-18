package evalcontract

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"unicode"
)

const (
	FirstErrorAttributionVersionV1 = "eval_first_error_attribution.v1"
	FirstErrorComparisonVersionV1  = "eval_first_error_comparison.v1"

	FirstErrorMaxSecondaryCauses = 16
	FirstErrorMaxEvidence        = 32
	FirstErrorMaxBoundaryEntries = 32
	FirstErrorMaxSerializedSize  = 64 * 1024
)

const (
	FirstErrorKindDecision          = "decision"
	FirstErrorKindToolSelection     = "tool_selection"
	FirstErrorKindToolInput         = "tool_input"
	FirstErrorKindPolicy            = "policy"
	FirstErrorKindMemoryRetrieval   = "memory_retrieval"
	FirstErrorKindMemoryApplication = "memory_application"
	FirstErrorKindContext           = "context"
	FirstErrorKindProvider          = "provider"
	FirstErrorKindTermination       = "termination"
	FirstErrorKindEvidence          = "evidence"
)

const (
	AttributionOwnerModel    = "model"
	AttributionOwnerTool     = "tool"
	AttributionOwnerPolicy   = "policy"
	AttributionOwnerMemory   = "memory"
	AttributionOwnerContext  = "context"
	AttributionOwnerProvider = "provider"
	AttributionOwnerRuntime  = "runtime"
	AttributionOwnerHost     = "host"
	AttributionOwnerUnknown  = "unknown"
)

const (
	RecoverabilityRecoverable    = "recoverable"
	RecoverabilityNonRecoverable = "non_recoverable"
	RecoverabilityUnknown        = "unknown"
)

const (
	ReasonFirstErrorSchemaDrift             = "first_error_schema_drift"
	ReasonFirstErrorStepDrift               = "first_error_step_drift"
	ReasonFirstErrorKindDrift               = "first_error_kind_drift"
	ReasonFirstErrorOwnerDrift              = "first_error_owner_drift"
	ReasonFirstErrorCauseDrift              = "first_error_cause_drift"
	ReasonFirstErrorPrefixDrift             = "first_error_prefix_drift"
	ReasonTrajectoryActionBoundaryConflict  = "trajectory_action_boundary_conflict"
	ReasonTrajectoryActionBoundaryDrift     = "trajectory_action_boundary_drift"
	ReasonTrajectoryRequiredEvidenceMissing = "trajectory_required_evidence_missing"
	ReasonTrajectoryRequiredEvidenceDrift   = "trajectory_required_evidence_drift"
	ReasonFirstErrorEvidenceConflict        = "first_error_evidence_conflict"
	ReasonFirstErrorEvidenceDrift           = "first_error_evidence_drift"
	ReasonFirstErrorRecoverabilityDrift     = "first_error_recoverability_drift"
	ReasonFirstErrorConfidenceDrift         = "first_error_confidence_drift"
	ReasonFirstErrorCorrelationDrift        = "first_error_correlation_drift"
	ReasonFirstErrorPrivacyViolation        = "first_error_privacy_violation"
	ReasonFirstErrorRunStreamParityDrift    = "first_error_run_stream_parity_drift"
	ReasonFeedbackAutoApplyForbidden        = "feedback_auto_apply_forbidden"
)

const FeedbackApplicationReviewOnly = "review_only"

type AttributionCorrelation struct {
	CorpusItemID  string `json:"corpus_item_id"`
	BadcaseID     string `json:"badcase_id"`
	RunID         string `json:"run_id"`
	ExperimentID  string `json:"experiment_id,omitempty"`
	ExecutionMode string `json:"execution_mode,omitempty"`
}

type FirstErrorIdentity struct {
	StepID  string `json:"step_id"`
	Ordinal *int   `json:"ordinal"`
	Kind    string `json:"kind"`
}

type RankedCause struct {
	Rank int    `json:"rank"`
	Code string `json:"code"`
}

type FirstErrorCause struct {
	Owner     string        `json:"owner"`
	Primary   string        `json:"primary"`
	Secondary []RankedCause `json:"secondary,omitempty"`
}

type AttributionReference struct {
	Kind    string `json:"kind"`
	Owner   string `json:"owner"`
	ID      string `json:"id"`
	Digest  string `json:"digest,omitempty"`
	Version string `json:"version,omitempty"`
}

type AttributionEvidence struct {
	Reference AttributionReference `json:"reference"`
	Body      string               `json:"body,omitempty"`
}

type TrajectoryDecisionBoundary struct {
	AcceptableActions []AttributionReference `json:"acceptable_actions,omitempty"`
	ForbiddenActions  []AttributionReference `json:"forbidden_actions,omitempty"`
	RequiredEvidence  []AttributionReference `json:"required_evidence,omitempty"`
	SafetyConstraints []AttributionReference `json:"safety_constraints,omitempty"`
}

type FirstErrorAttribution struct {
	Version               string                     `json:"version"`
	Correlation           AttributionCorrelation     `json:"correlation"`
	FirstError            FirstErrorIdentity         `json:"first_error"`
	Cause                 FirstErrorCause            `json:"cause"`
	Recoverability        string                     `json:"recoverability"`
	ConfidenceBasisPoints int                        `json:"confidence_basis_points"`
	PrefixDigest          string                     `json:"prefix_digest"`
	Boundary              TrajectoryDecisionBoundary `json:"boundary"`
	Evidence              []AttributionEvidence      `json:"evidence"`
}

type FirstErrorAttributionReference struct {
	Version      string `json:"version"`
	ID           string `json:"id"`
	CorpusItemID string `json:"corpus_item_id"`
	BadcaseID    string `json:"badcase_id"`
	RunID        string `json:"run_id"`
	StepID       string `json:"step_id"`
	ExperimentID string `json:"experiment_id,omitempty"`
}

type FirstErrorDrift struct {
	Reason string `json:"reason"`
}

type FirstErrorComparison struct {
	Version           string            `json:"version"`
	ID                string            `json:"id"`
	BaselineIdentity  string            `json:"baseline_identity"`
	CandidateIdentity string            `json:"candidate_identity"`
	Passed            bool              `json:"passed"`
	Drifts            []FirstErrorDrift `json:"drifts,omitempty"`
}

var firstErrorKinds = stringSet(
	FirstErrorKindDecision,
	FirstErrorKindToolSelection,
	FirstErrorKindToolInput,
	FirstErrorKindPolicy,
	FirstErrorKindMemoryRetrieval,
	FirstErrorKindMemoryApplication,
	FirstErrorKindContext,
	FirstErrorKindProvider,
	FirstErrorKindTermination,
	FirstErrorKindEvidence,
)

var attributionOwners = stringSet(
	AttributionOwnerModel,
	AttributionOwnerTool,
	AttributionOwnerPolicy,
	AttributionOwnerMemory,
	AttributionOwnerContext,
	AttributionOwnerProvider,
	AttributionOwnerRuntime,
	AttributionOwnerHost,
	AttributionOwnerUnknown,
)

var recoverabilityValues = stringSet(
	RecoverabilityRecoverable,
	RecoverabilityNonRecoverable,
	RecoverabilityUnknown,
)

func NormalizeFirstErrorAttribution(input FirstErrorAttribution) (FirstErrorAttribution, string, error) {
	in := cloneFirstErrorAttributionInput(input)
	in.Version = normalizeToken(in.Version)
	in.Correlation.CorpusItemID = strings.TrimSpace(in.Correlation.CorpusItemID)
	in.Correlation.BadcaseID = strings.TrimSpace(in.Correlation.BadcaseID)
	in.Correlation.RunID = strings.TrimSpace(in.Correlation.RunID)
	in.Correlation.ExperimentID = strings.TrimSpace(in.Correlation.ExperimentID)
	in.Correlation.ExecutionMode = normalizeToken(in.Correlation.ExecutionMode)
	in.FirstError.StepID = strings.TrimSpace(in.FirstError.StepID)
	in.FirstError.Kind = normalizeToken(in.FirstError.Kind)
	in.Cause.Owner = normalizeToken(in.Cause.Owner)
	in.Cause.Primary = normalizeToken(in.Cause.Primary)
	in.Recoverability = normalizeToken(in.Recoverability)
	if in.Recoverability == "" {
		in.Recoverability = RecoverabilityUnknown
	}
	in.PrefixDigest = strings.TrimSpace(in.PrefixDigest)

	if in.Version != FirstErrorAttributionVersionV1 ||
		in.Correlation.CorpusItemID == "" || in.Correlation.BadcaseID == "" || in.Correlation.RunID == "" ||
		in.FirstError.StepID == "" || in.FirstError.Ordinal == nil || *in.FirstError.Ordinal <= 0 ||
		!contains(firstErrorKinds, in.FirstError.Kind) || !contains(attributionOwners, in.Cause.Owner) ||
		!validCauseCode(in.Cause.Primary) || !contains(recoverabilityValues, in.Recoverability) ||
		in.ConfidenceBasisPoints < 0 || in.ConfidenceBasisPoints > 10000 || in.PrefixDigest == "" ||
		(in.Correlation.ExecutionMode != "" && in.Correlation.ExecutionMode != "run" && in.Correlation.ExecutionMode != "stream") {
		return FirstErrorAttribution{}, "", fmt.Errorf("%s", ReasonFirstErrorSchemaDrift)
	}
	if len(in.Cause.Secondary) > FirstErrorMaxSecondaryCauses || len(in.Evidence) > FirstErrorMaxEvidence ||
		len(in.Boundary.AcceptableActions) > FirstErrorMaxBoundaryEntries ||
		len(in.Boundary.ForbiddenActions) > FirstErrorMaxBoundaryEntries ||
		len(in.Boundary.RequiredEvidence) > FirstErrorMaxBoundaryEntries ||
		len(in.Boundary.SafetyConstraints) > FirstErrorMaxBoundaryEntries {
		return FirstErrorAttribution{}, "", fmt.Errorf("%s", ReasonFirstErrorSchemaDrift)
	}

	seenRanks := make(map[int]struct{}, len(in.Cause.Secondary))
	for i := range in.Cause.Secondary {
		in.Cause.Secondary[i].Code = normalizeToken(in.Cause.Secondary[i].Code)
		if in.Cause.Secondary[i].Rank <= 0 || !validCauseCode(in.Cause.Secondary[i].Code) {
			return FirstErrorAttribution{}, "", fmt.Errorf("%s", ReasonFirstErrorSchemaDrift)
		}
		if _, duplicate := seenRanks[in.Cause.Secondary[i].Rank]; duplicate {
			return FirstErrorAttribution{}, "", fmt.Errorf("%s", ReasonFirstErrorSchemaDrift)
		}
		seenRanks[in.Cause.Secondary[i].Rank] = struct{}{}
	}
	sort.Slice(in.Cause.Secondary, func(i, j int) bool {
		if in.Cause.Secondary[i].Rank == in.Cause.Secondary[j].Rank {
			return in.Cause.Secondary[i].Code < in.Cause.Secondary[j].Code
		}
		return in.Cause.Secondary[i].Rank < in.Cause.Secondary[j].Rank
	})

	var err error
	if in.Boundary.AcceptableActions, err = normalizeAttributionReferences(in.Boundary.AcceptableActions); err != nil {
		return FirstErrorAttribution{}, "", err
	}
	if in.Boundary.ForbiddenActions, err = normalizeAttributionReferences(in.Boundary.ForbiddenActions); err != nil {
		return FirstErrorAttribution{}, "", err
	}
	if in.Boundary.RequiredEvidence, err = normalizeAttributionReferences(in.Boundary.RequiredEvidence); err != nil {
		return FirstErrorAttribution{}, "", err
	}
	if in.Boundary.SafetyConstraints, err = normalizeAttributionReferences(in.Boundary.SafetyConstraints); err != nil {
		return FirstErrorAttribution{}, "", err
	}
	if hasReferenceOverlap(in.Boundary.AcceptableActions, in.Boundary.ForbiddenActions) {
		return FirstErrorAttribution{}, "", fmt.Errorf("%s", ReasonTrajectoryActionBoundaryConflict)
	}

	in.Evidence, err = normalizeAttributionEvidence(in.Evidence)
	if err != nil {
		return FirstErrorAttribution{}, "", err
	}
	for _, required := range in.Boundary.RequiredEvidence {
		if !containsSatisfyingReference(in.Evidence, required) {
			return FirstErrorAttribution{}, "", fmt.Errorf("%s", ReasonTrajectoryRequiredEvidenceMissing)
		}
	}

	serialized, err := json.Marshal(in)
	if err != nil {
		return FirstErrorAttribution{}, "", err
	}
	if len(serialized) > FirstErrorMaxSerializedSize {
		return FirstErrorAttribution{}, "", fmt.Errorf("%s", ReasonFirstErrorSchemaDrift)
	}

	identityInput := in
	identityInput.Correlation.ExecutionMode = ""
	identity, err := digestValue(identityInput)
	if err != nil {
		return FirstErrorAttribution{}, "", err
	}
	return in, identity, nil
}

func CompareFirstErrorAttribution(baselineInput, candidateInput FirstErrorAttribution) (FirstErrorComparison, error) {
	baseline, baselineIdentity, err := NormalizeFirstErrorAttribution(baselineInput)
	if err != nil {
		return FirstErrorComparison{}, err
	}
	candidate, candidateIdentity, err := NormalizeFirstErrorAttribution(candidateInput)
	if err != nil {
		return FirstErrorComparison{}, err
	}

	reasons := make(map[string]struct{})
	add := func(reason string) { reasons[reason] = struct{}{} }
	if baseline.FirstError.StepID != candidate.FirstError.StepID || !equalOrdinal(baseline.FirstError.Ordinal, candidate.FirstError.Ordinal) {
		add(ReasonFirstErrorStepDrift)
	}
	if baseline.FirstError.Kind != candidate.FirstError.Kind {
		add(ReasonFirstErrorKindDrift)
	}
	if baseline.PrefixDigest != candidate.PrefixDigest {
		add(ReasonFirstErrorPrefixDrift)
	}
	if baseline.Cause.Owner != candidate.Cause.Owner {
		add(ReasonFirstErrorOwnerDrift)
	}
	if baseline.Cause.Primary != candidate.Cause.Primary || !reflect.DeepEqual(baseline.Cause.Secondary, candidate.Cause.Secondary) {
		add(ReasonFirstErrorCauseDrift)
	}
	if !reflect.DeepEqual(baseline.Boundary.AcceptableActions, candidate.Boundary.AcceptableActions) ||
		!reflect.DeepEqual(baseline.Boundary.ForbiddenActions, candidate.Boundary.ForbiddenActions) ||
		!reflect.DeepEqual(baseline.Boundary.SafetyConstraints, candidate.Boundary.SafetyConstraints) {
		add(ReasonTrajectoryActionBoundaryDrift)
	}
	if !reflect.DeepEqual(baseline.Boundary.RequiredEvidence, candidate.Boundary.RequiredEvidence) {
		add(ReasonTrajectoryRequiredEvidenceDrift)
	}
	if !reflect.DeepEqual(baseline.Evidence, candidate.Evidence) {
		add(ReasonFirstErrorEvidenceDrift)
	}
	if baseline.Recoverability != candidate.Recoverability {
		add(ReasonFirstErrorRecoverabilityDrift)
	}
	if baseline.ConfidenceBasisPoints != candidate.ConfidenceBasisPoints {
		add(ReasonFirstErrorConfidenceDrift)
	}
	if !equalAttributionCorrelation(baseline.Correlation, candidate.Correlation) {
		add(ReasonFirstErrorCorrelationDrift)
	}
	if isRunStreamPair(baseline.Correlation.ExecutionMode, candidate.Correlation.ExecutionMode) && len(reasons) > 0 {
		add(ReasonFirstErrorRunStreamParityDrift)
	}

	reasonValues := make([]string, 0, len(reasons))
	for reason := range reasons {
		reasonValues = append(reasonValues, reason)
	}
	sort.Strings(reasonValues)
	drifts := make([]FirstErrorDrift, len(reasonValues))
	for i, reason := range reasonValues {
		drifts[i] = FirstErrorDrift{Reason: reason}
	}
	comparison := FirstErrorComparison{
		Version:           FirstErrorComparisonVersionV1,
		BaselineIdentity:  baselineIdentity,
		CandidateIdentity: candidateIdentity,
		Passed:            len(drifts) == 0,
		Drifts:            drifts,
	}
	comparison.ID, err = digestValue(struct {
		Version           string
		BaselineIdentity  string
		CandidateIdentity string
		Drifts            []FirstErrorDrift
	}{comparison.Version, comparison.BaselineIdentity, comparison.CandidateIdentity, comparison.Drifts})
	if err != nil {
		return FirstErrorComparison{}, err
	}
	return comparison, nil
}

func NewFirstErrorAttributionReference(attribution FirstErrorAttribution) (FirstErrorAttributionReference, error) {
	normalized, identity, err := NormalizeFirstErrorAttribution(attribution)
	if err != nil {
		return FirstErrorAttributionReference{}, err
	}
	return FirstErrorAttributionReference{
		Version:      FirstErrorAttributionVersionV1,
		ID:           identity,
		CorpusItemID: normalized.Correlation.CorpusItemID,
		BadcaseID:    normalized.Correlation.BadcaseID,
		RunID:        normalized.Correlation.RunID,
		StepID:       normalized.FirstError.StepID,
		ExperimentID: normalized.Correlation.ExperimentID,
	}, nil
}

func ValidateFirstErrorAttributionAssociation(reference FirstErrorAttributionReference, attribution FirstErrorAttribution) error {
	normalizedReference, err := normalizeFirstErrorAttributionReference(reference)
	if err != nil {
		return err
	}
	normalizedAttribution, identity, err := NormalizeFirstErrorAttribution(attribution)
	if err != nil {
		return err
	}
	if normalizedReference.ID != identity ||
		normalizedReference.CorpusItemID != normalizedAttribution.Correlation.CorpusItemID ||
		normalizedReference.BadcaseID != normalizedAttribution.Correlation.BadcaseID ||
		normalizedReference.RunID != normalizedAttribution.Correlation.RunID ||
		normalizedReference.StepID != normalizedAttribution.FirstError.StepID ||
		normalizedReference.ExperimentID != normalizedAttribution.Correlation.ExperimentID {
		return fmt.Errorf("%s", ReasonFirstErrorCorrelationDrift)
	}
	return nil
}

func normalizeFirstErrorAttributionReference(reference FirstErrorAttributionReference) (FirstErrorAttributionReference, error) {
	reference.Version = normalizeToken(reference.Version)
	reference.ID = strings.TrimSpace(reference.ID)
	reference.CorpusItemID = strings.TrimSpace(reference.CorpusItemID)
	reference.BadcaseID = strings.TrimSpace(reference.BadcaseID)
	reference.RunID = strings.TrimSpace(reference.RunID)
	reference.StepID = strings.TrimSpace(reference.StepID)
	reference.ExperimentID = strings.TrimSpace(reference.ExperimentID)
	if reference.Version != FirstErrorAttributionVersionV1 || reference.ID == "" || reference.CorpusItemID == "" ||
		reference.BadcaseID == "" || reference.RunID == "" || reference.StepID == "" {
		return FirstErrorAttributionReference{}, fmt.Errorf("%s", ReasonFirstErrorSchemaDrift)
	}
	return reference, nil
}

func normalizeAttributionReferences(input []AttributionReference) ([]AttributionReference, error) {
	out := make([]AttributionReference, 0, len(input))
	seen := make(map[string]struct{}, len(input))
	for _, reference := range input {
		reference.Kind = normalizeToken(reference.Kind)
		reference.Owner = normalizeToken(reference.Owner)
		reference.ID = strings.TrimSpace(reference.ID)
		reference.Digest = strings.TrimSpace(reference.Digest)
		reference.Version = strings.TrimSpace(reference.Version)
		if reference.Kind == "" || !contains(attributionOwners, reference.Owner) || reference.ID == "" {
			return nil, fmt.Errorf("%s", ReasonFirstErrorSchemaDrift)
		}
		key := completeReferenceKey(reference)
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, reference)
	}
	sort.Slice(out, func(i, j int) bool { return completeReferenceKey(out[i]) < completeReferenceKey(out[j]) })
	return out, nil
}

func normalizeAttributionEvidence(input []AttributionEvidence) ([]AttributionEvidence, error) {
	out := make([]AttributionEvidence, 0, len(input))
	seen := make(map[string]AttributionReference, len(input))
	for _, evidence := range input {
		if evidence.Body != "" {
			return nil, fmt.Errorf("%s", ReasonFirstErrorPrivacyViolation)
		}
		references, err := normalizeAttributionReferences([]AttributionReference{evidence.Reference})
		if err != nil {
			return nil, err
		}
		evidence.Reference = references[0]
		key := stableReferenceKey(evidence.Reference)
		if previous, duplicate := seen[key]; duplicate {
			if previous.Digest != evidence.Reference.Digest || previous.Version != evidence.Reference.Version || previous.Kind != evidence.Reference.Kind {
				return nil, fmt.Errorf("%s", ReasonFirstErrorEvidenceConflict)
			}
			continue
		}
		seen[key] = evidence.Reference
		out = append(out, evidence)
	}
	sort.Slice(out, func(i, j int) bool {
		return completeReferenceKey(out[i].Reference) < completeReferenceKey(out[j].Reference)
	})
	return out, nil
}

func cloneFirstErrorAttributionInput(in FirstErrorAttribution) FirstErrorAttribution {
	clone := in
	if in.FirstError.Ordinal != nil {
		ordinal := *in.FirstError.Ordinal
		clone.FirstError.Ordinal = &ordinal
	}
	clone.Cause.Secondary = append([]RankedCause(nil), in.Cause.Secondary...)
	clone.Boundary.AcceptableActions = append([]AttributionReference(nil), in.Boundary.AcceptableActions...)
	clone.Boundary.ForbiddenActions = append([]AttributionReference(nil), in.Boundary.ForbiddenActions...)
	clone.Boundary.RequiredEvidence = append([]AttributionReference(nil), in.Boundary.RequiredEvidence...)
	clone.Boundary.SafetyConstraints = append([]AttributionReference(nil), in.Boundary.SafetyConstraints...)
	clone.Evidence = append([]AttributionEvidence(nil), in.Evidence...)
	return clone
}

func containsSatisfyingReference(evidence []AttributionEvidence, required AttributionReference) bool {
	for _, candidate := range evidence {
		if candidate.Reference.Kind != required.Kind || candidate.Reference.Owner != required.Owner || candidate.Reference.ID != required.ID {
			continue
		}
		if required.Digest != "" && candidate.Reference.Digest != required.Digest {
			continue
		}
		if required.Version != "" && candidate.Reference.Version != required.Version {
			continue
		}
		return true
	}
	return false
}

func hasReferenceOverlap(left, right []AttributionReference) bool {
	seen := make(map[string]struct{}, len(left))
	for _, reference := range left {
		seen[actionReferenceKey(reference)] = struct{}{}
	}
	for _, reference := range right {
		if _, conflict := seen[actionReferenceKey(reference)]; conflict {
			return true
		}
	}
	return false
}

func equalOrdinal(left, right *int) bool {
	return left != nil && right != nil && *left == *right
}

func equalAttributionCorrelation(left, right AttributionCorrelation) bool {
	return left.CorpusItemID == right.CorpusItemID &&
		left.BadcaseID == right.BadcaseID &&
		left.RunID == right.RunID &&
		left.ExperimentID == right.ExperimentID
}

func isRunStreamPair(left, right string) bool {
	return (left == "run" && right == "stream") || (left == "stream" && right == "run")
}

func validCauseCode(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for _, r := range value {
		if unicode.IsLower(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.' {
			continue
		}
		return false
	}
	return true
}

func normalizeToken(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func stableReferenceKey(reference AttributionReference) string {
	return reference.Owner + "\x00" + reference.ID
}

func actionReferenceKey(reference AttributionReference) string {
	return reference.Kind + "\x00" + reference.Owner + "\x00" + reference.ID
}

func completeReferenceKey(reference AttributionReference) string {
	return reference.Owner + "\x00" + reference.ID + "\x00" + reference.Kind + "\x00" + reference.Digest + "\x00" + reference.Version
}

func stringSet(values ...string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

func contains(values map[string]struct{}, value string) bool {
	_, ok := values[value]
	return ok
}
