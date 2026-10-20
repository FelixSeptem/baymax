package evalcontract

import (
	"encoding/json"
	"math"
	"testing"
)

func TestBuildCandidateComparisonRejectsAxisOverflow(t *testing.T) {
	for _, axis := range []string{"quality-overflow", "quality-underflow", "latency-overflow"} {
		t.Run(axis, func(t *testing.T) {
			input := comparisonBoundaryInput()
			input.Runs = append(input.Runs, input.Runs[0])
			input.Runs[1].Attempt = 2
			switch axis {
			case "quality-overflow":
				input.Runs[0].Quality, input.Runs[1].Quality = int64Ptr(math.MaxInt64), int64Ptr(1)
			case "quality-underflow":
				input.Runs[0].Quality, input.Runs[1].Quality = int64Ptr(math.MinInt64), int64Ptr(-1)
			case "latency-overflow":
				input.Runs[0].LatencyMS, input.Runs[1].LatencyMS = math.MaxInt64, 1
			}
			if _, err := BuildCandidateComparison(input); err == nil || err.Error() != ReasonCandidateEvidenceUnavailable {
				t.Fatalf("overflow accepted: %v", err)
			}
		})
	}
	input := comparisonBoundaryInput()
	input.Runs[0].Quality, input.Runs[0].LatencyMS = int64Ptr(math.MaxInt64), math.MaxInt64
	if _, err := BuildCandidateComparison(input); err != nil {
		t.Fatalf("valid boundary rejected: %v", err)
	}
}

func TestBuildCandidateComparisonRejectsContradictoryAttemptStatus(t *testing.T) {
	for _, status := range []string{CandidateOutcomeFailed, CandidateOutcomeIncomplete} {
		input := comparisonBoundaryInput()
		input.CostLedger = []CandidateCostEntry{{CandidateID: "baseline", Attempt: 1, Status: status}}
		if _, err := BuildCandidateComparison(input); err == nil || err.Error() != ReasonCandidateCostLedgerDrift {
			t.Fatalf("contradictory %s status accepted: %v", status, err)
		}
	}
}

func comparisonBoundaryInput() CandidateComparisonInput {
	in := CandidateComparisonInput{
		Version: CandidateComparisonV1, ReviewOnly: true, NetworkVerified: true,
		NetworkEvidenceRef: "evidence:egress", ScoringIsolationEvidenceRef: "evidence:scoring-isolation",
		ScoringDataStatus: CandidateScoringDataIsolated,
		Experiment: CandidateExperiment{
			ID: "boundary", CorpusVersion: CorpusVersionV1, CorpusItemID: "item", RubricVersion: "1", RubricDigest: "rubric",
			Model: "model", AuthorityDigest: "authority", DataDigest: "data", Budget: CandidateBudget{MaxAttempts: 3},
			NetworkPolicy: "offline", ScoringDataPolicy: CandidateScoringDataSelectionHidden, ExecutionMode: "offline",
			Baseline:   CandidateAsset{Identity: "baseline", Path: "baseline/base.md", Version: "1", Digest: "base"},
			Surface:    CandidateSurface{AllowedPaths: []string{"z", "candidate"}, MaxAssetBytes: 10},
			Candidates: []CandidateAsset{{Identity: "candidate", Path: "candidate/main.md", Version: "1", Digest: "candidate", MaxBytes: 10}},
		},
		Runs:      []CandidateRun{{CandidateID: "baseline", Identity: "baseline", Digest: "base", Status: CandidateOutcomeCompleted, Reliable: true, ContractOK: true}},
		Selection: CandidateSelection{CandidateID: "baseline", Sequence: 10, Reason: "baseline retained"},
		Score:     CandidateScore{Status: CandidateScoringUnscored},
	}
	_, in.ExperimentDigest, _ = NormalizeCandidateExperiment(in.Experiment)
	return in
}

func TestBuildCandidateComparisonBoundaryValidation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*CandidateComparisonInput)
		reason string
	}{
		{"missing isolation", func(in *CandidateComparisonInput) { in.ScoringDataStatus = "" }, ReasonCandidateIsolationUnavailable},
		{"missing admitted identity", func(in *CandidateComparisonInput) { in.ExperimentDigest = "" }, ReasonCandidateIdentityDrift},
		{"foreign isolation", func(in *CandidateComparisonInput) { in.ScoringDataStatus = "trusted" }, ReasonCandidateIsolationUnavailable},
		{"missing network evidence", func(in *CandidateComparisonInput) { in.NetworkEvidenceRef = "" }, ReasonCandidateIsolationUnavailable},
		{"missing scoring isolation evidence", func(in *CandidateComparisonInput) { in.ScoringIsolationEvidenceRef = "" }, ReasonCandidateIsolationUnavailable},
		{"actual selection order", func(in *CandidateComparisonInput) {
			in.Score = CandidateScore{Status: CandidateEvidenceUnavailable, Sequence: 9}
		}, ReasonCandidateScoringOrderDrift},
		{"missing committed selection", func(in *CandidateComparisonInput) { in.Selection = CandidateSelection{} }, ReasonCandidateSelectionDrift},
		{"unavailable synthetic zero", func(in *CandidateComparisonInput) {
			in.Score = CandidateScore{Status: CandidateEvidenceUnavailable, Sequence: 11, Value: int64Ptr(0)}
		}, ReasonCandidateEvidenceUnavailable},
		{"available missing value", func(in *CandidateComparisonInput) {
			in.Score = CandidateScore{Status: CandidateEvidenceAvailable, Sequence: 11}
		}, ReasonCandidateEvidenceUnavailable},
		{"foreign cost", func(in *CandidateComparisonInput) {
			in.CostLedger = []CandidateCostEntry{{CandidateID: "foreign", Attempt: 1, Status: CandidateOutcomeCompleted}}
		}, ReasonCandidateCostLedgerDrift},
		{"unknown cost status", func(in *CandidateComparisonInput) {
			in.CostLedger = []CandidateCostEntry{{CandidateID: "baseline", Attempt: 1, Status: "unknown"}}
		}, ReasonCandidateCostLedgerDrift},
		{"duplicate cost", func(in *CandidateComparisonInput) {
			entry := CandidateCostEntry{CandidateID: "baseline", Attempt: 1, Status: CandidateOutcomeCompleted}
			in.CostLedger = []CandidateCostEntry{entry, entry}
		}, ReasonCandidateCostLedgerDrift},
		{"unknown run status", func(in *CandidateComparisonInput) { in.Runs[0].Status = "unknown" }, ReasonCandidateScopeRejected},
		{"negative latency", func(in *CandidateComparisonInput) { in.Runs[0].LatencyMS = -1 }, ReasonCandidateScopeRejected},
		{"invalid candidate name", func(in *CandidateComparisonInput) { in.Experiment.Candidates[0].Identity = "invalid/name" }, ReasonCandidateScopeRejected},
		{"static validation bypass", func(in *CandidateComparisonInput) {
			in.Experiment.Surface.StaticChecks = []string{"lint"}
			_, in.ExperimentDigest, _ = NormalizeCandidateExperiment(in.Experiment)
			in.Runs = []CandidateRun{{CandidateID: "candidate", Identity: "candidate", Digest: "candidate", Status: CandidateOutcomeCompleted, Reliable: true, ContractOK: true, Quality: int64Ptr(10)}}
			in.Selection.CandidateID = "candidate"
		}, ReasonCandidateScopeRejected},
		{"cost overflow", func(in *CandidateComparisonInput) {
			in.CostLedger = []CandidateCostEntry{
				{CandidateID: "baseline", Attempt: 1, Status: CandidateOutcomeCompleted, CostMilli: 1<<63 - 1},
				{CandidateID: "baseline", Attempt: 2, Status: CandidateOutcomeCompleted, CostMilli: 1},
			}
		}, ReasonCandidateCostLedgerDrift},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := comparisonBoundaryInput()
			tc.mutate(&input)
			if _, err := BuildCandidateComparison(input); err == nil || err.Error() != tc.reason {
				t.Fatalf("got %v, want %s", err, tc.reason)
			}
		})
	}
}

func TestBuildCandidateComparisonMissingCostAndCommittedSelection(t *testing.T) {
	input := comparisonBoundaryInput()
	result, err := BuildCandidateComparison(input)
	if err != nil {
		t.Fatal(err)
	}
	if result.Cost.Status != CandidateEvidenceInsufficient || result.Cost.Value != nil {
		t.Fatalf("missing ledger must not imply zero cost: %#v", result.Cost)
	}
	if result.Selected.Sequence != 10 || result.Selected.Reason != input.Selection.Reason {
		t.Fatalf("committed selection changed: %#v", result.Selected)
	}
}

func TestBuildCandidateComparisonPartiallyAccountedRuns(t *testing.T) {
	input := comparisonBoundaryInput()
	input.Runs = append(input.Runs, CandidateRun{CandidateID: "candidate", Identity: "candidate", Digest: "candidate", Status: CandidateOutcomeFailed})
	input.CostLedger = []CandidateCostEntry{{CandidateID: "baseline", Attempt: 1, Status: CandidateOutcomeCompleted}}
	result, err := BuildCandidateComparison(input)
	if err != nil {
		t.Fatal(err)
	}
	if result.Cost.Status != CandidateEvidenceInsufficient || result.Cost.Value != nil {
		t.Fatal("partial cost ledger was declared complete")
	}
}

func TestNormalizeCandidateExperimentDoesNotMutateInput(t *testing.T) {
	in := comparisonBoundaryInput().Experiment
	before, _ := json.Marshal(in)
	if _, _, err := NormalizeCandidateExperiment(in); err != nil {
		t.Fatal(err)
	}
	after, _ := json.Marshal(in)
	if string(before) != string(after) {
		t.Fatal("normalization mutated caller-owned slices")
	}
}

func TestCompareCandidateIdentityBindsBadcase(t *testing.T) {
	base := comparisonBoundaryInput().Experiment
	observed := base
	observed.BadcaseID = "foreign"
	comparison, err := CompareCandidateIdentity(base, observed)
	if err != nil {
		t.Fatal(err)
	}
	if comparison.Passed {
		t.Fatal("badcase drift was accepted")
	}
}

func TestCompareCandidateIdentityCoversConfiguration(t *testing.T) {
	for name, mutate := range map[string]func(*CandidateExperiment){
		"model":     func(in *CandidateExperiment) { in.Model = "different" },
		"authority": func(in *CandidateExperiment) { in.AuthorityDigest = "different" },
		"data":      func(in *CandidateExperiment) { in.DataDigest = "different" },
		"rubric":    func(in *CandidateExperiment) { in.RubricDigest = "different" },
		"budget":    func(in *CandidateExperiment) { in.Budget.MaxAttempts++ },
		"baseline":  func(in *CandidateExperiment) { in.Baseline.Digest = "different" },
		"candidate": func(in *CandidateExperiment) { in.Candidates[0].Digest = "different" },
		"surface":   func(in *CandidateExperiment) { in.Surface.MaxAssetBytes++ },
	} {
		t.Run(name, func(t *testing.T) {
			base := comparisonBoundaryInput().Experiment
			observed := comparisonBoundaryInput().Experiment
			mutate(&observed)
			result, err := CompareCandidateIdentity(base, observed)
			if err != nil || result.Passed || len(result.Drifts) != 1 {
				t.Fatalf("identity drift was not isolated: %#v %v", result, err)
			}
		})
	}
	base := comparisonBoundaryInput().Experiment
	observed := comparisonBoundaryInput().Experiment
	observed.NetworkPolicy = "online"
	if _, err := CompareCandidateIdentity(base, observed); err == nil || err.Error() != ReasonCandidateNetworkPolicyUnsupported {
		t.Fatalf("unsupported egress accepted: %v", err)
	}
}

func TestBuildCandidateComparisonRetainsAttemptEvidence(t *testing.T) {
	input := comparisonBoundaryInput()
	input.Runs[0].Attempt = 1
	repeated := input.Runs[0]
	repeated.Attempt = 2
	input.Runs = append(input.Runs, repeated)
	input.CostLedger = []CandidateCostEntry{
		{CandidateID: "baseline", Attempt: 1, Status: CandidateOutcomeCompleted, ModelCalls: 1, CostMilli: 2},
		{CandidateID: "baseline", Attempt: 2, Status: CandidateOutcomeCompleted, CacheHit: true},
	}
	result, err := BuildCandidateComparison(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Runs) != 2 || len(result.CostLedger) != 2 || result.NetworkEvidenceRef != input.NetworkEvidenceRef {
		t.Fatal("result lost attempt or isolation evidence")
	}
	input.Runs[1].Attempt = 1
	if _, err := BuildCandidateComparison(input); err == nil || err.Error() != ReasonCandidateDuplicateRunConflict {
		t.Fatalf("duplicate run evidence was merged: %v", err)
	}
}

func TestBuildCandidateComparisonCountsRetryAttempts(t *testing.T) {
	input := comparisonBoundaryInput()
	for retry := 0; retry <= 2; retry++ {
		input.CostLedger = append(input.CostLedger, CandidateCostEntry{CandidateID: "baseline", Attempt: 1, Retry: retry, Status: CandidateOutcomeCompleted})
	}
	result, err := BuildCandidateComparison(input)
	if err != nil {
		t.Fatal(err)
	}
	if result.AttemptTotal != 3 || result.RetryTotal != 2 {
		t.Fatalf("retry ordinal counted as retry quantity: %#v", result)
	}
}

func TestBuildCandidateComparisonAllowsConservativeBaselineSelection(t *testing.T) {
	input := comparisonBoundaryInput()
	input.Runs = append(input.Runs, CandidateRun{CandidateID: "candidate", Identity: "candidate", Digest: "candidate", Status: CandidateOutcomeCompleted, Reliable: true, ContractOK: true, Quality: int64Ptr(10)})
	result, err := BuildCandidateComparison(input)
	if err != nil || result.Selected.CandidateID != "baseline" {
		t.Fatalf("baseline became ineligible: %v", err)
	}
}

func TestBuildCandidateComparisonZeroQualityIsAvailable(t *testing.T) {
	input := comparisonBoundaryInput()
	input.Runs[0].Quality = int64Ptr(0)
	result, err := BuildCandidateComparison(input)
	if err != nil {
		t.Fatal(err)
	}
	if result.Quality.Status != CandidateEvidenceAvailable || result.Quality.Value == nil || *result.Quality.Value != 0 {
		t.Fatal("valid zero quality became missing evidence")
	}
}

func TestNormalizeCandidateExperimentDeterministicAndSorted(t *testing.T) {
	manifest := CandidateExperiment{
		Version:           CandidateExperimentVersionV1,
		ID:                " exp-1 ",
		CorpusVersion:     CorpusVersionV1,
		CorpusItemID:      "badcase-1",
		BadcaseID:         "badcase-1",
		RubricVersion:     "rubric-v1",
		RubricDigest:      "rubric-digest",
		Model:             "model-v1",
		AuthorityDigest:   "authority-digest",
		DataDigest:        "data-digest",
		Budget:            CandidateBudget{MaxAttempts: 3, MaxCostMilli: 100},
		NetworkPolicy:     "offline",
		ScoringDataPolicy: CandidateScoringDataSelectionHidden,
		ExecutionMode:     "offline",
		Baseline:          CandidateAsset{Identity: "baseline", Path: "baseline/skill.md", Version: "1", Digest: "base-digest", MaxBytes: 1000},
		Surface:           CandidateSurface{AllowedPaths: []string{"skills", "prompt"}, MaxAssetBytes: 1000},
		Candidates: []CandidateAsset{
			{Identity: "z", Path: "skills/z.md", Version: "1", Digest: "z-digest", MaxBytes: 1000},
			{Identity: "a", Path: "prompt/a.md", Version: "1", Digest: "a-digest", MaxBytes: 1000},
		},
	}

	normalized, digest, err := NormalizeCandidateExperiment(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if normalized.ID != "exp-1" || normalized.Candidates[0].Identity != "a" {
		t.Fatalf("unexpected normalization: %#v", normalized)
	}
	_, digest2, err := NormalizeCandidateExperiment(CandidateExperiment{
		Version:           CandidateExperimentVersionV1,
		ID:                "exp-1",
		CorpusVersion:     CorpusVersionV1,
		CorpusItemID:      "badcase-1",
		BadcaseID:         "badcase-1",
		RubricVersion:     "rubric-v1",
		RubricDigest:      "rubric-digest",
		Model:             "model-v1",
		AuthorityDigest:   "authority-digest",
		DataDigest:        "data-digest",
		Budget:            CandidateBudget{MaxAttempts: 3, MaxCostMilli: 100},
		NetworkPolicy:     "offline",
		ScoringDataPolicy: CandidateScoringDataSelectionHidden,
		ExecutionMode:     "offline",
		Baseline:          CandidateAsset{Identity: "baseline", Path: "baseline/skill.md", Version: "1", Digest: "base-digest", MaxBytes: 1000},
		Surface:           CandidateSurface{AllowedPaths: []string{"skills", "prompt"}, MaxAssetBytes: 1000},
		Candidates: []CandidateAsset{
			{Identity: "a", Path: "prompt/a.md", Version: "1", Digest: "a-digest", MaxBytes: 1000},
			{Identity: "z", Path: "skills/z.md", Version: "1", Digest: "z-digest", MaxBytes: 1000},
		},
	})
	if err != nil || digest == "" || digest != digest2 {
		t.Fatalf("digest is not stable: %q %q %v", digest, digest2, err)
	}
}

func TestNormalizeCandidateExperimentRejectsMissingIdentityAndInvalidBudget(t *testing.T) {
	_, _, err := NormalizeCandidateExperiment(CandidateExperiment{Version: CandidateExperimentVersionV1})
	if err == nil || err.Error() != ReasonCandidateExperimentMalformed {
		t.Fatalf("expected malformed candidate experiment, got %v", err)
	}
	_, _, err = NormalizeCandidateExperiment(CandidateExperiment{
		Version:       CandidateExperimentVersionV1,
		ID:            "exp-1",
		CorpusVersion: CorpusVersionV1,
		CorpusItemID:  "item-1",
		RubricVersion: "rubric-v1",
		Model:         "model",
		DataDigest:    "data",
		Budget:        CandidateBudget{MaxAttempts: -1},
		NetworkPolicy: "offline",
		ExecutionMode: "offline",
	})
	if err == nil || err.Error() != ReasonCandidateExperimentMalformed {
		t.Fatalf("expected invalid budget rejection, got %v", err)
	}
}

func TestNormalizeCandidateExperimentRejectsPathEscapeDuplicateIdentityAndAssetLimit(t *testing.T) {
	base := CandidateExperiment{
		Version: CandidateExperimentVersionV1, ID: "exp-1", CorpusVersion: CorpusVersionV1,
		CorpusItemID: "item-1", RubricVersion: "rubric-v1", RubricDigest: "rubric-digest",
		Model: "model", AuthorityDigest: "authority", DataDigest: "data",
		Budget: CandidateBudget{MaxAttempts: 1}, NetworkPolicy: CandidateNetworkPolicyOffline,
		ScoringDataPolicy: CandidateScoringDataSelectionHidden, ExecutionMode: "offline",
		Baseline:   CandidateAsset{Identity: "baseline", Path: "baseline/base.md", Version: "1", Digest: "base", MaxBytes: 10},
		Surface:    CandidateSurface{AllowedPaths: []string{"candidate"}, MaxAssetBytes: 10, StaticChecks: []string{"lint", "schema"}},
		Candidates: []CandidateAsset{{Identity: "candidate", Path: "candidate/main.md", Version: "1", Digest: "candidate", Bytes: 10, MaxBytes: 10}},
	}
	for name, mutate := range map[string]func(*CandidateExperiment){
		"path escape":                func(in *CandidateExperiment) { in.Candidates[0].Path = "../outside.md" },
		"duplicate identity":         func(in *CandidateExperiment) { in.Candidates[0].Identity = "baseline" },
		"asset exceeds declared max": func(in *CandidateExperiment) { in.Candidates[0].Bytes = 11 },
	} {
		t.Run(name, func(t *testing.T) {
			in := base
			in.Candidates = append([]CandidateAsset(nil), base.Candidates...)
			mutate(&in)
			if _, _, err := NormalizeCandidateExperiment(in); err == nil {
				t.Fatal("expected candidate experiment rejection")
			}
		})
	}
}

func TestCompareCandidateIdentityClassifiesDriftWithoutMerging(t *testing.T) {
	base := CandidateExperiment{
		Version: CandidateExperimentVersionV1, ID: "exp-1", CorpusVersion: CorpusVersionV1,
		CorpusItemID: "item-1", RubricVersion: "rubric-v1", RubricDigest: "rubric-digest",
		Model: "model-v1", AuthorityDigest: "authority", DataDigest: "data",
		Budget: CandidateBudget{MaxAttempts: 2, MaxCostMilli: 10}, NetworkPolicy: CandidateNetworkPolicyOffline,
		ScoringDataPolicy: CandidateScoringDataSelectionHidden, ExecutionMode: "offline",
		Baseline:   CandidateAsset{Identity: "baseline", Path: "baseline/base.md", Version: "1", Digest: "base", MaxBytes: 10},
		Surface:    CandidateSurface{AllowedPaths: []string{"candidate"}, MaxAssetBytes: 10, StaticChecks: []string{"lint", "schema"}},
		Candidates: []CandidateAsset{{Identity: "candidate", Path: "candidate/main.md", Version: "1", Digest: "candidate", MaxBytes: 10}},
	}
	observed := base
	observed.Model = "model-v2"
	observed.AuthorityDigest = "authority-v2"
	comparison, err := CompareCandidateIdentity(base, observed)
	if err != nil {
		t.Fatal(err)
	}
	if comparison.Passed || len(comparison.Drifts) != 2 || comparison.Drifts[0].Field != "model" {
		t.Fatalf("unexpected identity comparison: %#v", comparison)
	}
}

func TestAdmitCandidateAndSelectKeepsBaselineEligible(t *testing.T) {
	manifest := CandidateExperiment{
		Version: CandidateExperimentVersionV1, ID: "exp-1", CorpusVersion: CorpusVersionV1,
		CorpusItemID: "item-1", RubricVersion: "rubric-v1", RubricDigest: "rubric-digest",
		Model: "model", AuthorityDigest: "authority", DataDigest: "data", Budget: CandidateBudget{MaxAttempts: 1},
		NetworkPolicy: CandidateNetworkPolicyOffline, ScoringDataPolicy: CandidateScoringDataSelectionHidden, ExecutionMode: "offline",
		Baseline:   CandidateAsset{Identity: "baseline", Path: "baseline/base.md", Version: "1", Digest: "base", MaxBytes: 10},
		Surface:    CandidateSurface{AllowedPaths: []string{"candidate"}, MaxAssetBytes: 10, StaticChecks: []string{"lint", "schema"}},
		Candidates: []CandidateAsset{{Identity: "candidate", Path: "candidate/main.md", Version: "1", Digest: "candidate", MaxBytes: 10}},
	}
	if _, _, err := AdmitCandidate(manifest, "baseline"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := AdmitCandidate(manifest, "candidate"); err == nil {
		t.Fatal("candidate must not be admitted without static-check evidence")
	}
	if _, _, err := AdmitCandidateWithValidation(manifest, "candidate", []CandidateStaticValidation{{Check: "lint", Passed: true}, {Check: "schema", Passed: true}}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := AdmitCandidateWithValidation(manifest, "candidate", []CandidateStaticValidation{{Check: "lint", Passed: true}, {Check: "schema", Passed: false}}); err == nil {
		t.Fatal("candidate must fail a declared static check")
	}
	if _, _, err := AdmitCandidate(manifest, "outside"); err == nil || err.Error() != ReasonCandidateScopeRejected {
		t.Fatalf("expected scope rejection, got %v", err)
	}
	selection, err := SelectCandidate(manifest, []CandidateRun{{CandidateID: "candidate", Identity: "candidate", Digest: "candidate", Status: CandidateOutcomeCompleted, Reliable: true, ContractOK: true, Quality: int64Ptr(5), StaticValidations: []CandidateStaticValidation{{Check: "lint", Passed: true}, {Check: "schema", Passed: true}}}})
	if err != nil || selection.CandidateID != "candidate" {
		t.Fatalf("unexpected selection: %#v %v", selection, err)
	}
	selection, err = SelectCandidate(manifest, nil)
	if err != nil || selection.CandidateID != "baseline" {
		t.Fatalf("baseline must remain eligible: %#v %v", selection, err)
	}
}

func TestBuildCandidateComparisonSeparatesSelectionScoringAndCost(t *testing.T) {
	experiment := CandidateExperiment{
		Version: CandidateExperimentVersionV1, ID: "exp-1", CorpusVersion: CorpusVersionV1,
		CorpusItemID: "item-1", RubricVersion: "rubric-v1", RubricDigest: "rubric-digest",
		Model: "model", AuthorityDigest: "authority", DataDigest: "data", Budget: CandidateBudget{MaxAttempts: 2},
		NetworkPolicy: CandidateNetworkPolicyOffline, ScoringDataPolicy: CandidateScoringDataSelectionHidden, ExecutionMode: "offline",
		Baseline:   CandidateAsset{Identity: "baseline", Path: "baseline/base.md", Version: "1", Digest: "base", MaxBytes: 10},
		Surface:    CandidateSurface{AllowedPaths: []string{"candidate"}, MaxAssetBytes: 10},
		Candidates: []CandidateAsset{{Identity: "candidate", Path: "candidate/main.md", Version: "1", Digest: "candidate", MaxBytes: 10}},
	}
	_, digest, err := NormalizeCandidateExperiment(experiment)
	if err != nil {
		t.Fatal(err)
	}
	result, err := BuildCandidateComparison(CandidateComparisonInput{
		Version: CandidateComparisonV1, Experiment: experiment, ExperimentDigest: digest, NetworkVerified: true,
		NetworkEvidenceRef: "evidence:egress", ScoringIsolationEvidenceRef: "evidence:scoring-isolation",
		ScoringDataStatus: CandidateScoringDataIsolated, ReviewOnly: true,
		Runs:       []CandidateRun{{CandidateID: "candidate", Identity: "candidate", Digest: "candidate", Status: CandidateOutcomeCompleted, Reliable: true, ContractOK: true, Quality: int64Ptr(9), LatencyMS: 12}},
		Selection:  CandidateSelection{CandidateID: "candidate", Sequence: 1},
		Score:      CandidateScore{Status: CandidateEvidenceUnavailable, Sequence: 2},
		CostLedger: []CandidateCostEntry{{CandidateID: "candidate", Attempt: 1, ModelCalls: 2, CostMilli: 7, Status: CandidateOutcomeCompleted, CacheHit: true}, {CandidateID: "candidate", Attempt: 1, Retry: 1, Status: CandidateOutcomeIncomplete}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Selected.CandidateID != "candidate" || result.Score.Status != CandidateEvidenceUnavailable || result.CostTotalMilli != 7 || result.ModelCallTotal != 2 || result.AttemptTotal != 2 || result.RetryTotal != 1 || result.CacheHitTotal != 1 {
		t.Fatalf("unexpected comparison: %#v", result)
	}
	if result.Score.Value != nil {
		t.Fatal("unavailable score must not have synthetic value")
	}
}

func TestBuildCandidateComparisonRejectsIsolationAndScoringOrderDrift(t *testing.T) {
	experiment := CandidateExperiment{
		Version: CandidateExperimentVersionV1, ID: "exp-1", CorpusVersion: CorpusVersionV1, CorpusItemID: "item-1", RubricVersion: "rubric", RubricDigest: "rubric-digest", Model: "model", AuthorityDigest: "authority", DataDigest: "data", Budget: CandidateBudget{MaxAttempts: 1}, NetworkPolicy: CandidateNetworkPolicyOffline, ScoringDataPolicy: CandidateScoringDataSelectionHidden, ExecutionMode: "offline", Baseline: CandidateAsset{Identity: "baseline", Path: "base/base.md", Version: "1", Digest: "base", MaxBytes: 10}, Surface: CandidateSurface{AllowedPaths: []string{"candidate"}, MaxAssetBytes: 10}, Candidates: []CandidateAsset{{Identity: "candidate", Path: "candidate/main.md", Version: "1", Digest: "candidate", MaxBytes: 10}},
	}
	_, digest, _ := NormalizeCandidateExperiment(experiment)
	input := CandidateComparisonInput{Version: CandidateComparisonV1, Experiment: experiment, ExperimentDigest: digest, ReviewOnly: true, Runs: []CandidateRun{{CandidateID: "baseline", Identity: "baseline", Digest: "base", Status: CandidateOutcomeCompleted, Reliable: true, ContractOK: true}}, Selection: CandidateSelection{CandidateID: "baseline", Sequence: 1}, Score: CandidateScore{Status: CandidateEvidenceAvailable, Sequence: 1}}
	input.NetworkVerified = false
	if _, err := BuildCandidateComparison(input); err == nil || err.Error() != ReasonCandidateIsolationUnavailable {
		t.Fatalf("expected isolation rejection, got %v", err)
	}
	input.NetworkVerified = true
	input.NetworkEvidenceRef = "evidence:egress"
	input.ScoringIsolationEvidenceRef = "evidence:scoring-isolation"
	input.ScoringDataStatus = CandidateScoringDataIsolated
	if _, err := BuildCandidateComparison(input); err == nil || err.Error() != ReasonCandidateScoringOrderDrift {
		t.Fatalf("expected scoring order rejection, got %v", err)
	}
}
