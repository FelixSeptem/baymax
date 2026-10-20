package evalcontract

import (
	"fmt"
	"math"
	"path"
	"regexp"
	"sort"
	"strings"
)

const (
	CandidateExperimentVersionV1 = "candidate_experiment.v1"
	CandidateComparisonV1        = "candidate_comparison.v1"

	CandidateEvidenceAvailable          = "available"
	CandidateEvidenceUnavailable        = "unavailable"
	CandidateEvidenceInsufficient       = "insufficient-evidence"
	CandidateDecisionPending            = "pending"
	CandidateDecisionSelected           = "selected"
	CandidateDecisionRejected           = "rejected"
	CandidateOutcomeCompleted           = "completed"
	CandidateOutcomeIncomplete          = "incomplete"
	CandidateOutcomeFailed              = "failed"
	CandidateSelectionBaseline          = "baseline"
	CandidateScoringUnscored            = "unscored"
	CandidateComparisonReviewOnly       = "review_only"
	CandidateNetworkPolicyOffline       = "offline"
	CandidateScoringDataIsolated        = "isolated"
	CandidateScoringDataUnavailable     = "unavailable"
	CandidateScoringDataSelectionHidden = "selection_hidden"
)

const (
	ReasonCandidateExperimentMalformed      = "candidate_experiment_malformed"
	ReasonCandidateIdentityDrift            = "candidate_experiment_identity_drift"
	ReasonCandidateScopeRejected            = "candidate_asset_scope_rejected"
	ReasonCandidateIsolationUnavailable     = "candidate_isolation_unavailable"
	ReasonCandidateEvidenceUnavailable      = "candidate_evidence_unavailable"
	ReasonCandidateSelectionDrift           = "candidate_selection_drift"
	ReasonCandidateScoringOrderDrift        = "candidate_scoring_order_drift"
	ReasonCandidateCostLedgerDrift          = "candidate_cost_ledger_drift"
	ReasonCandidateComparisonNonActionable  = "candidate_comparison_non_actionable"
	ReasonCandidateDuplicateRunConflict     = "candidate_run_conflict"
	ReasonCandidateComparisonVersionDrift   = "candidate_comparison_version_drift"
	ReasonCandidateNetworkPolicyUnsupported = "candidate_network_policy_unsupported"
)

type CandidateBudget struct {
	MaxAttempts       int   `json:"max_attempts"`
	MaxCostMilli      int64 `json:"max_cost_milli,omitempty"`
	MaxDurationMillis int64 `json:"max_duration_millis,omitempty"`
}

type CandidateAsset struct {
	Identity string `json:"identity"`
	Path     string `json:"path"`
	Version  string `json:"version"`
	Digest   string `json:"digest"`
	Bytes    int64  `json:"bytes,omitempty"`
	MaxBytes int64  `json:"max_bytes,omitempty"`
}

type CandidateSurface struct {
	AllowedPaths  []string `json:"allowed_paths"`
	MaxAssetBytes int64    `json:"max_asset_bytes"`
	StaticChecks  []string `json:"static_checks,omitempty"`
}

type CandidateExperiment struct {
	Version           string           `json:"version"`
	ID                string           `json:"id"`
	CorpusVersion     string           `json:"corpus_version"`
	CorpusItemID      string           `json:"corpus_item_id"`
	BadcaseID         string           `json:"badcase_id,omitempty"`
	RubricVersion     string           `json:"rubric_version"`
	RubricDigest      string           `json:"rubric_digest"`
	Model             string           `json:"model"`
	AuthorityDigest   string           `json:"authority_digest"`
	DataDigest        string           `json:"data_digest"`
	Budget            CandidateBudget  `json:"budget"`
	NetworkPolicy     string           `json:"network_policy"`
	ScoringDataPolicy string           `json:"scoring_data_policy"`
	ExecutionMode     string           `json:"execution_mode"`
	Baseline          CandidateAsset   `json:"baseline"`
	Surface           CandidateSurface `json:"surface"`
	Candidates        []CandidateAsset `json:"candidates,omitempty"`
}

type CandidateRun struct {
	Attempt           int                         `json:"attempt,omitempty"`
	Retry             int                         `json:"retry,omitempty"`
	CandidateID       string                      `json:"candidate_id"`
	Identity          string                      `json:"identity"`
	Digest            string                      `json:"digest"`
	Status            string                      `json:"status"`
	Quality           *int64                      `json:"quality,omitempty"`
	Reliable          bool                        `json:"reliable"`
	ContractOK        bool                        `json:"contract_ok"`
	LatencyMS         int64                       `json:"latency_ms,omitempty"`
	StaticValidations []CandidateStaticValidation `json:"static_validations,omitempty"`
}

type CandidateCostEntry struct {
	CandidateID string `json:"candidate_id"`
	Attempt     int    `json:"attempt"`
	Retry       int    `json:"retry,omitempty"`
	CacheHit    bool   `json:"cache_hit,omitempty"`
	ModelCalls  int    `json:"model_calls"`
	CostMilli   int64  `json:"cost_milli"`
	Status      string `json:"status"`
}

type CandidateSelection struct {
	CandidateID string   `json:"candidate_id"`
	Reason      string   `json:"reason"`
	Evidence    []string `json:"evidence,omitempty"`
	Sequence    int      `json:"sequence"`
}

type CandidateScore struct {
	Status        string `json:"status"`
	Value         *int64 `json:"value,omitempty"`
	ScoringDigest string `json:"scoring_digest,omitempty"`
	Sequence      int    `json:"sequence"`
}

type CandidateComparisonInput struct {
	Version                     string               `json:"version"`
	Experiment                  CandidateExperiment  `json:"experiment"`
	ExperimentDigest            string               `json:"experiment_digest"`
	Runs                        []CandidateRun       `json:"runs"`
	CostLedger                  []CandidateCostEntry `json:"cost_ledger"`
	Selection                   CandidateSelection   `json:"selection"`
	Score                       CandidateScore       `json:"score"`
	NetworkVerified             bool                 `json:"network_verified"`
	ScoringDataStatus           string               `json:"scoring_data_status"`
	ReviewOnly                  bool                 `json:"review_only"`
	NetworkEvidenceRef          string               `json:"network_evidence_ref"`
	ScoringIsolationEvidenceRef string               `json:"scoring_isolation_evidence_ref"`
}

type CandidateComparisonResult struct {
	Version                     string               `json:"version"`
	ExperimentID                string               `json:"experiment_id"`
	ExperimentDigest            string               `json:"experiment_digest"`
	Selected                    CandidateSelection   `json:"selected"`
	Score                       CandidateScore       `json:"score"`
	Quality                     CandidateEvidence    `json:"quality"`
	Reliability                 CandidateEvidence    `json:"reliability"`
	Contract                    CandidateEvidence    `json:"contract"`
	Latency                     CandidateEvidence    `json:"latency"`
	Cost                        CandidateEvidence    `json:"cost"`
	AttemptTotal                int                  `json:"attempt_total"`
	RetryTotal                  int                  `json:"retry_total"`
	CacheHitTotal               int                  `json:"cache_hit_total"`
	IncompleteTotal             int                  `json:"incomplete_total"`
	ModelCallTotal              int                  `json:"model_call_total"`
	CostTotalMilli              int64                `json:"cost_total_milli"`
	RunCount                    int                  `json:"run_count"`
	ReviewOnly                  bool                 `json:"review_only"`
	Digest                      string               `json:"digest"`
	Runs                        []CandidateRun       `json:"runs"`
	CostLedger                  []CandidateCostEntry `json:"cost_ledger"`
	NetworkEvidenceRef          string               `json:"network_evidence_ref"`
	ScoringIsolationEvidenceRef string               `json:"scoring_isolation_evidence_ref"`
}

type CandidateEvidence struct {
	Status string `json:"status"`
	Value  *int64 `json:"value,omitempty"`
}

// AdmitCandidate validates a candidate against the normalized experiment
// surface before any probe is started. Baseline is an explicit, immutable
// candidate and is always admissible.
func AdmitCandidate(experiment CandidateExperiment, candidateID string) (CandidateExperiment, string, error) {
	return AdmitCandidateWithValidation(experiment, candidateID, nil)
}

type CandidateStaticValidation struct {
	Check  string `json:"check"`
	Passed bool   `json:"passed"`
}

func AdmitCandidateWithValidation(experiment CandidateExperiment, candidateID string, validations []CandidateStaticValidation) (CandidateExperiment, string, error) {
	normalized, digest, err := NormalizeCandidateExperiment(experiment)
	if err != nil {
		return CandidateExperiment{}, "", err
	}
	candidateID = strings.TrimSpace(candidateID)
	if !candidateIdentityPattern.MatchString(candidateID) {
		return CandidateExperiment{}, "", fmt.Errorf("%s", ReasonCandidateScopeRejected)
	}
	if candidateID != normalized.Baseline.Identity {
		if err := validateCandidateStaticChecks(normalized.Surface.StaticChecks, validations); err != nil {
			return CandidateExperiment{}, "", err
		}
	}
	if candidateID == normalized.Baseline.Identity {
		return normalized, digest, nil
	}
	for _, candidate := range normalized.Candidates {
		if candidate.Identity == candidateID {
			return normalized, digest, nil
		}
	}
	return CandidateExperiment{}, "", fmt.Errorf("%s", ReasonCandidateScopeRejected)
}

var candidateIdentityPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,127}$`)

func validateCandidateStaticChecks(required []string, observed []CandidateStaticValidation) error {
	checks := make(map[string]bool, len(observed))
	for _, validation := range observed {
		name := strings.TrimSpace(validation.Check)
		if name == "" {
			return fmt.Errorf("%s", ReasonCandidateScopeRejected)
		}
		if _, duplicate := checks[name]; duplicate {
			return fmt.Errorf("%s", ReasonCandidateScopeRejected)
		}
		checks[name] = validation.Passed
	}
	for _, check := range required {
		passed, ok := checks[check]
		if !ok || !passed {
			return fmt.Errorf("%s", ReasonCandidateScopeRejected)
		}
	}
	return nil
}

// SelectCandidate commits selection from probe evidence only. Final scoring
// data is intentionally absent from this API and must be supplied later.
func SelectCandidate(experiment CandidateExperiment, runs []CandidateRun) (CandidateSelection, error) {
	normalized, _, err := NormalizeCandidateExperiment(experiment)
	if err != nil {
		return CandidateSelection{}, err
	}
	allowed := map[string]string{normalized.Baseline.Identity: normalized.Baseline.Digest}
	for _, candidate := range normalized.Candidates {
		allowed[candidate.Identity] = candidate.Digest
	}
	selection := CandidateSelection{CandidateID: normalized.Baseline.Identity, Reason: CandidateDecisionSelected, Evidence: []string{"baseline-eligible"}, Sequence: 1}
	best := int64(math.MinInt64)
	seenRuns := map[string]bool{}
	for _, run := range runs {
		key := candidateAttemptKey(run.CandidateID, run.Attempt, run.Retry)
		if seenRuns[key] {
			return CandidateSelection{}, fmt.Errorf("%s", ReasonCandidateDuplicateRunConflict)
		}
		seenRuns[key] = true
		expectedDigest, ok := allowed[run.CandidateID]
		if !ok || run.Identity != run.CandidateID || run.Digest != expectedDigest || !candidateOutcomeValid(run.Status) || run.LatencyMS < 0 || run.Attempt < 0 || run.Retry < 0 {
			return CandidateSelection{}, fmt.Errorf("%s", ReasonCandidateScopeRejected)
		}
		if run.CandidateID != normalized.Baseline.Identity {
			if err := validateCandidateStaticChecks(normalized.Surface.StaticChecks, run.StaticValidations); err != nil {
				return CandidateSelection{}, err
			}
		}
		if run.Status != CandidateOutcomeCompleted || !run.Reliable || !run.ContractOK || run.Quality == nil {
			continue
		}
		if *run.Quality > best || (*run.Quality == best && (run.CandidateID == normalized.Baseline.Identity || (selection.CandidateID != normalized.Baseline.Identity && run.CandidateID < selection.CandidateID))) {
			best = *run.Quality
			selection = CandidateSelection{CandidateID: run.CandidateID, Reason: CandidateDecisionSelected, Evidence: []string{"probe-quality", "reliable", "contract-ok"}, Sequence: 1}
		}
	}
	return selection, nil
}

// BuildCandidateComparison performs the read-only, post-selection reduction
// into separate outcome axes and totals. It never treats missing scoring data
// as a numeric zero.
func BuildCandidateComparison(input CandidateComparisonInput) (CandidateComparisonResult, error) {
	if strings.TrimSpace(input.Version) == "" {
		input.Version = CandidateComparisonV1
	}
	if input.Version != CandidateComparisonV1 || !input.ReviewOnly {
		return CandidateComparisonResult{}, fmt.Errorf("%s", ReasonCandidateComparisonVersionDrift)
	}
	experiment, digest, err := NormalizeCandidateExperiment(input.Experiment)
	if err != nil {
		return CandidateComparisonResult{}, err
	}
	if input.ExperimentDigest != digest {
		return CandidateComparisonResult{}, fmt.Errorf("%s", ReasonCandidateIdentityDrift)
	}
	if !input.NetworkVerified || input.ScoringDataStatus != CandidateScoringDataIsolated || strings.TrimSpace(input.NetworkEvidenceRef) == "" || strings.TrimSpace(input.ScoringIsolationEvidenceRef) == "" {
		return CandidateComparisonResult{}, fmt.Errorf("%s", ReasonCandidateIsolationUnavailable)
	}
	selection, err := SelectCandidate(experiment, input.Runs)
	if err != nil {
		return CandidateComparisonResult{}, err
	}
	if (input.Selection.CandidateID != experiment.Baseline.Identity && input.Selection.CandidateID != selection.CandidateID) || input.Selection.Sequence <= 0 {
		return CandidateComparisonResult{}, fmt.Errorf("%s", ReasonCandidateSelectionDrift)
	}
	selection = input.Selection
	if input.Score.Status != "" && input.Score.Status != CandidateScoringUnscored && input.Score.Sequence <= selection.Sequence {
		return CandidateComparisonResult{}, fmt.Errorf("%s", ReasonCandidateScoringOrderDrift)
	}
	score := input.Score
	if score.Status == "" {
		score.Status = CandidateScoringUnscored
	}
	if score.Status != CandidateScoringUnscored && score.Status != CandidateEvidenceAvailable && score.Status != CandidateEvidenceUnavailable && score.Status != CandidateEvidenceInsufficient {
		return CandidateComparisonResult{}, fmt.Errorf("%s", ReasonCandidateEvidenceUnavailable)
	}
	if (score.Status == CandidateEvidenceAvailable && (score.Value == nil || score.ScoringDigest == "")) || (score.Status != CandidateEvidenceAvailable && score.Value != nil) {
		return CandidateComparisonResult{}, fmt.Errorf("%s", ReasonCandidateEvidenceUnavailable)
	}
	result := CandidateComparisonResult{Version: CandidateComparisonV1, ExperimentID: experiment.ID, ExperimentDigest: digest, Selected: selection, Score: score, ReviewOnly: true, RunCount: len(input.Runs)}
	result.Runs = append([]CandidateRun(nil), input.Runs...)
	result.CostLedger = append([]CandidateCostEntry(nil), input.CostLedger...)
	result.NetworkEvidenceRef = input.NetworkEvidenceRef
	result.ScoringIsolationEvidenceRef = input.ScoringIsolationEvidenceRef
	result.Quality, err = evidenceForRuns(input.Runs, func(run CandidateRun) bool { return run.Quality != nil }, func(run CandidateRun) int64 { return *run.Quality })
	if err != nil {
		return CandidateComparisonResult{}, err
	}
	result.Reliability, _ = evidenceForRuns(input.Runs, func(run CandidateRun) bool { return true }, func(run CandidateRun) int64 {
		if run.Reliable {
			return 1
		}
		return 0
	})
	result.Contract, _ = evidenceForRuns(input.Runs, func(run CandidateRun) bool { return true }, func(run CandidateRun) int64 {
		if run.ContractOK {
			return 1
		}
		return 0
	})
	result.Latency, err = evidenceForRuns(input.Runs, func(run CandidateRun) bool { return run.LatencyMS > 0 }, func(run CandidateRun) int64 { return run.LatencyMS })
	if err != nil {
		return CandidateComparisonResult{}, err
	}
	seenAttempts := map[string]bool{}
	accounted := map[string]string{}
	allowed := map[string]bool{experiment.Baseline.Identity: true}
	for _, candidate := range experiment.Candidates {
		allowed[candidate.Identity] = true
	}
	for _, entry := range input.CostLedger {
		if !allowed[entry.CandidateID] || entry.Attempt <= 0 || entry.Retry < 0 || entry.ModelCalls < 0 || entry.CostMilli < 0 || !candidateOutcomeValid(entry.Status) {
			return CandidateComparisonResult{}, fmt.Errorf("%s", ReasonCandidateCostLedgerDrift)
		}
		key := candidateAttemptKey(entry.CandidateID, entry.Attempt, entry.Retry)
		if seenAttempts[key] {
			return CandidateComparisonResult{}, fmt.Errorf("%s", ReasonCandidateCostLedgerDrift)
		}
		seenAttempts[key] = true
		accounted[key] = entry.Status
		if result.ModelCallTotal > math.MaxInt-entry.ModelCalls || result.CostTotalMilli > math.MaxInt64-entry.CostMilli {
			return CandidateComparisonResult{}, fmt.Errorf("%s", ReasonCandidateCostLedgerDrift)
		}
		result.AttemptTotal++
		if entry.Retry > 0 {
			result.RetryTotal++
		}
		if entry.CacheHit {
			result.CacheHitTotal++
		}
		if entry.Status != CandidateOutcomeCompleted {
			result.IncompleteTotal++
		}
		result.ModelCallTotal += entry.ModelCalls
		result.CostTotalMilli += entry.CostMilli
	}
	result.Cost = CandidateEvidence{Status: CandidateEvidenceInsufficient}
	completeCost := len(input.CostLedger) > 0
	for _, run := range input.Runs {
		status, exists := accounted[candidateAttemptKey(run.CandidateID, run.Attempt, run.Retry)]
		if exists && status != run.Status {
			return CandidateComparisonResult{}, fmt.Errorf("%s", ReasonCandidateCostLedgerDrift)
		}
		completeCost = completeCost && exists
	}
	if completeCost {
		result.Cost = CandidateEvidence{Status: CandidateEvidenceAvailable, Value: int64Ptr(result.CostTotalMilli)}
	}
	result.Digest, err = digestValue(result)
	return result, err
}

func evidenceForRuns(runs []CandidateRun, available func(CandidateRun) bool, value func(CandidateRun) int64) (CandidateEvidence, error) {
	if len(runs) == 0 {
		return CandidateEvidence{Status: CandidateEvidenceInsufficient}, nil
	}
	var total int64
	for _, run := range runs {
		if !available(run) {
			return CandidateEvidence{Status: CandidateEvidenceInsufficient}, nil
		}
		next := value(run)
		if (next > 0 && total > math.MaxInt64-next) || (next < 0 && total < math.MinInt64-next) {
			return CandidateEvidence{}, fmt.Errorf("%s", ReasonCandidateEvidenceUnavailable)
		}
		total += next
	}
	return CandidateEvidence{Status: CandidateEvidenceAvailable, Value: int64Ptr(total)}, nil
}

func int64Ptr(value int64) *int64 { return &value }

func candidateOutcomeValid(status string) bool {
	return status == CandidateOutcomeCompleted || status == CandidateOutcomeIncomplete || status == CandidateOutcomeFailed
}

func candidateAttemptKey(candidateID string, attempt, retry int) string {
	if attempt == 0 {
		attempt = 1
	}
	return fmt.Sprintf("%s/%d/%d", candidateID, attempt, retry)
}

type CandidateIdentityDrift struct {
	Field    string `json:"field"`
	Expected string `json:"expected"`
	Observed string `json:"observed"`
}

type CandidateIdentityComparison struct {
	Version      string                   `json:"version"`
	ExperimentID string                   `json:"experiment_id"`
	Baseline     string                   `json:"baseline_digest"`
	Passed       bool                     `json:"passed"`
	Drifts       []CandidateIdentityDrift `json:"drifts,omitempty"`
}

// CompareCandidateIdentity checks that two manifests using the same logical
// experiment ID still refer to the same admitted execution inputs.
func CompareCandidateIdentity(expected, observed CandidateExperiment) (CandidateIdentityComparison, error) {
	base, _, err := NormalizeCandidateExperiment(expected)
	if err != nil {
		return CandidateIdentityComparison{}, err
	}
	got, _, err := NormalizeCandidateExperiment(observed)
	if err != nil {
		return CandidateIdentityComparison{}, err
	}
	result := CandidateIdentityComparison{Version: CandidateComparisonV1, ExperimentID: base.ID, Baseline: base.Baseline.Digest}
	if base.ID != got.ID {
		result.Drifts = append(result.Drifts, CandidateIdentityDrift{Field: "id", Expected: base.ID, Observed: got.ID})
	}
	compare := func(field, left, right string) {
		if left != right {
			result.Drifts = append(result.Drifts, CandidateIdentityDrift{Field: field, Expected: left, Observed: right})
		}
	}
	compare("corpus_version", base.CorpusVersion, got.CorpusVersion)
	compare("corpus_item_id", base.CorpusItemID, got.CorpusItemID)
	compare("badcase_id", base.BadcaseID, got.BadcaseID)
	compare("rubric_version", base.RubricVersion, got.RubricVersion)
	compare("rubric_digest", base.RubricDigest, got.RubricDigest)
	compare("model", base.Model, got.Model)
	compare("authority_digest", base.AuthorityDigest, got.AuthorityDigest)
	compare("data_digest", base.DataDigest, got.DataDigest)
	compare("network_policy", base.NetworkPolicy, got.NetworkPolicy)
	compare("execution_mode", base.ExecutionMode, got.ExecutionMode)
	compare("scoring_data_policy", base.ScoringDataPolicy, got.ScoringDataPolicy)
	if base.Budget != got.Budget {
		result.Drifts = append(result.Drifts, CandidateIdentityDrift{Field: "budget", Expected: fmt.Sprintf("%+v", base.Budget), Observed: fmt.Sprintf("%+v", got.Budget)})
	}
	compareAsset := func(field string, left, right CandidateAsset) {
		leftDigest, _ := digestValue(left)
		rightDigest, _ := digestValue(right)
		compare(field, leftDigest, rightDigest)
	}
	compareAsset("baseline", base.Baseline, got.Baseline)
	leftSurface, _ := digestValue(base.Surface)
	rightSurface, _ := digestValue(got.Surface)
	compare("candidate_surface", leftSurface, rightSurface)
	if len(base.Candidates) != len(got.Candidates) {
		result.Drifts = append(result.Drifts, CandidateIdentityDrift{Field: "candidates", Expected: fmt.Sprint(len(base.Candidates)), Observed: fmt.Sprint(len(got.Candidates))})
	} else {
		for i := range base.Candidates {
			compareAsset("candidate["+fmt.Sprint(i)+"]", base.Candidates[i], got.Candidates[i])
		}
	}
	result.Passed = len(result.Drifts) == 0
	return result, nil
}

func NormalizeCandidateExperiment(in CandidateExperiment) (CandidateExperiment, string, error) {
	// Normalization is pure: sorting must not mutate the caller's backing arrays.
	in.Candidates = append([]CandidateAsset(nil), in.Candidates...)
	in.Surface.AllowedPaths = append([]string(nil), in.Surface.AllowedPaths...)
	in.Surface.StaticChecks = append([]string(nil), in.Surface.StaticChecks...)
	in.Version = strings.ToLower(strings.TrimSpace(in.Version))
	if in.Version == "" {
		in.Version = CandidateExperimentVersionV1
	}
	in.ID = strings.TrimSpace(in.ID)
	in.CorpusVersion = strings.TrimSpace(in.CorpusVersion)
	in.CorpusItemID = strings.TrimSpace(in.CorpusItemID)
	in.BadcaseID = strings.TrimSpace(in.BadcaseID)
	in.RubricVersion = strings.TrimSpace(in.RubricVersion)
	in.RubricDigest = strings.TrimSpace(in.RubricDigest)
	in.Model = strings.TrimSpace(in.Model)
	in.AuthorityDigest = strings.TrimSpace(in.AuthorityDigest)
	in.DataDigest = strings.TrimSpace(in.DataDigest)
	in.NetworkPolicy = strings.ToLower(strings.TrimSpace(in.NetworkPolicy))
	in.ScoringDataPolicy = strings.ToLower(strings.TrimSpace(in.ScoringDataPolicy))
	in.ExecutionMode = strings.ToLower(strings.TrimSpace(in.ExecutionMode))
	if in.Version != CandidateExperimentVersionV1 || in.ID == "" || in.CorpusVersion != CorpusVersionV1 ||
		in.CorpusItemID == "" || in.RubricVersion == "" || in.RubricDigest == "" || in.Model == "" ||
		in.AuthorityDigest == "" || in.DataDigest == "" || in.Budget.MaxAttempts <= 0 ||
		in.Budget.MaxCostMilli < 0 || in.Budget.MaxDurationMillis < 0 || in.NetworkPolicy == "" ||
		in.ScoringDataPolicy != CandidateScoringDataSelectionHidden || in.ExecutionMode != "offline" {
		return CandidateExperiment{}, "", fmt.Errorf("%s", ReasonCandidateExperimentMalformed)
	}
	if in.NetworkPolicy != CandidateNetworkPolicyOffline {
		return CandidateExperiment{}, "", fmt.Errorf("%s", ReasonCandidateNetworkPolicyUnsupported)
	}
	var err error
	if in.Baseline, err = normalizeCandidateAsset(in.Baseline); err != nil {
		return CandidateExperiment{}, "", err
	}
	if len(in.Surface.AllowedPaths) == 0 || in.Surface.MaxAssetBytes <= 0 {
		return CandidateExperiment{}, "", fmt.Errorf("%s", ReasonCandidateExperimentMalformed)
	}
	for i := range in.Surface.AllowedPaths {
		in.Surface.AllowedPaths[i], err = normalizeCandidatePath(in.Surface.AllowedPaths[i])
		if err != nil {
			return CandidateExperiment{}, "", err
		}
	}
	sort.Strings(in.Surface.AllowedPaths)
	in.Surface.AllowedPaths = deduplicateCandidateStrings(in.Surface.AllowedPaths)
	for i := range in.Surface.StaticChecks {
		in.Surface.StaticChecks[i] = strings.TrimSpace(in.Surface.StaticChecks[i])
		if in.Surface.StaticChecks[i] == "" {
			return CandidateExperiment{}, "", fmt.Errorf("%s", ReasonCandidateExperimentMalformed)
		}
	}
	sort.Strings(in.Surface.StaticChecks)
	in.Surface.StaticChecks = deduplicateCandidateStrings(in.Surface.StaticChecks)
	for i := range in.Candidates {
		if in.Candidates[i], err = normalizeCandidateAsset(in.Candidates[i]); err != nil {
			return CandidateExperiment{}, "", err
		}
		candidate := in.Candidates[i]
		if candidate.MaxBytes <= 0 || candidate.MaxBytes > in.Surface.MaxAssetBytes || candidate.Bytes > candidate.MaxBytes || !candidatePathAllowed(candidate.Path, in.Surface.AllowedPaths) {
			return CandidateExperiment{}, "", fmt.Errorf("%s", ReasonCandidateScopeRejected)
		}
	}
	sort.Slice(in.Candidates, func(i, j int) bool { return in.Candidates[i].Identity < in.Candidates[j].Identity })
	seen := map[string]bool{in.Baseline.Identity: true}
	for _, candidate := range in.Candidates {
		if seen[candidate.Identity] {
			return CandidateExperiment{}, "", fmt.Errorf("%s", ReasonCandidateExperimentMalformed)
		}
		seen[candidate.Identity] = true
	}
	digest, err := digestValue(in)
	return in, digest, err
}

func normalizeCandidateAsset(in CandidateAsset) (CandidateAsset, error) {
	in.Identity = strings.TrimSpace(in.Identity)
	in.Version = strings.TrimSpace(in.Version)
	in.Digest = strings.TrimSpace(in.Digest)
	if !candidateIdentityPattern.MatchString(in.Identity) {
		return CandidateAsset{}, fmt.Errorf("%s", ReasonCandidateScopeRejected)
	}
	var err error
	if in.Path, err = normalizeCandidatePath(in.Path); err != nil {
		return CandidateAsset{}, err
	}
	if in.Identity == "" || in.Version == "" || in.Digest == "" || in.Bytes < 0 || in.MaxBytes < 0 || (in.MaxBytes > 0 && in.Bytes > in.MaxBytes) {
		return CandidateAsset{}, fmt.Errorf("%s", ReasonCandidateExperimentMalformed)
	}
	return in, nil
}

func candidatePathAllowed(candidate string, allowed []string) bool {
	for _, root := range allowed {
		if candidate == root || strings.HasPrefix(candidate, root+"/") {
			return true
		}
	}
	return false
}

func normalizeCandidatePath(value string) (string, error) {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	clean := path.Clean(value)
	if value == "" || clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/") || strings.Contains(clean, ":") {
		return "", fmt.Errorf("%s", ReasonCandidateExperimentMalformed)
	}
	return clean, nil
}

func deduplicateCandidateStrings(values []string) []string {
	if len(values) < 2 {
		return values
	}
	result := values[:1]
	for _, value := range values[1:] {
		if value != result[len(result)-1] {
			result = append(result, value)
		}
	}
	return result
}
