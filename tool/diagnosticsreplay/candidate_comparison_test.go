package diagnosticsreplay

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/FelixSeptem/baymax/runtime/evalcontract"
)

func candidateComparisonFixtureInput(t *testing.T) evalcontract.CandidateComparisonInput {
	t.Helper()
	experiment := evalcontract.CandidateExperiment{Version: evalcontract.CandidateExperimentVersionV1, ID: "exp-1", CorpusVersion: evalcontract.CorpusVersionV1, CorpusItemID: "item-1", RubricVersion: "rubric-v1", RubricDigest: "rubric-digest", Model: "model", AuthorityDigest: "authority", DataDigest: "data", Budget: evalcontract.CandidateBudget{MaxAttempts: 1}, NetworkPolicy: evalcontract.CandidateNetworkPolicyOffline, ScoringDataPolicy: evalcontract.CandidateScoringDataSelectionHidden, ExecutionMode: "offline", Baseline: evalcontract.CandidateAsset{Identity: "baseline", Path: "baseline/base.md", Version: "1", Digest: "base", MaxBytes: 10}, Surface: evalcontract.CandidateSurface{AllowedPaths: []string{"candidate"}, MaxAssetBytes: 10}, Candidates: []evalcontract.CandidateAsset{{Identity: "candidate", Path: "candidate/main.md", Version: "1", Digest: "candidate", MaxBytes: 10}}}
	_, digest, err := evalcontract.NormalizeCandidateExperiment(experiment)
	if err != nil {
		t.Fatal(err)
	}
	return evalcontract.CandidateComparisonInput{Version: evalcontract.CandidateComparisonV1, Experiment: experiment, ExperimentDigest: digest, NetworkVerified: true, NetworkEvidenceRef: "evidence:egress", ScoringIsolationEvidenceRef: "evidence:scoring-isolation", ScoringDataStatus: evalcontract.CandidateScoringDataIsolated, ReviewOnly: true, Runs: []evalcontract.CandidateRun{{CandidateID: "baseline", Identity: "baseline", Digest: "base", Status: evalcontract.CandidateOutcomeCompleted, Reliable: true, ContractOK: true}}, Selection: evalcontract.CandidateSelection{CandidateID: "baseline", Sequence: 1}, Score: evalcontract.CandidateScore{Status: evalcontract.CandidateScoringUnscored}}
}

func TestCandidateComparisonReplaySuccessAndIsolationFailure(t *testing.T) {
	input := candidateComparisonFixtureInput(t)
	if err := EvaluateCandidateComparisonFixture(CandidateComparisonFixture{Version: CandidateComparisonFixtureVersionV1, Input: input, Expected: CandidateComparisonExpectation{Status: "success", SelectedID: "baseline", ScoreStatus: evalcontract.CandidateScoringUnscored, ReviewOnly: true}}); err != nil {
		t.Fatal(err)
	}
	input.NetworkVerified = false
	if err := EvaluateCandidateComparisonFixture(CandidateComparisonFixture{Version: CandidateComparisonFixtureVersionV1, Input: input, Expected: CandidateComparisonExpectation{Status: "error", Reason: evalcontract.ReasonCandidateIsolationUnavailable}}); err != nil {
		t.Fatal(err)
	}
}

func TestCandidateComparisonReplayFixtureFile(t *testing.T) {
	for _, name := range []string{"success", "identity_drift", "scope_rejection", "isolation_unavailable", "scoring_unavailable", "incomplete_cost", "baseline_selected"} {
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("testdata", "candidate_comparison_"+name+".v1.json"))
			if err != nil {
				t.Fatal(err)
			}
			fixture, err := ParseCandidateComparisonFixtureJSON(raw)
			if err != nil {
				t.Fatal(err)
			}
			before, _ := json.Marshal(fixture)
			for i := 0; i < 2; i++ {
				if err := EvaluateCandidateComparisonFixture(fixture); err != nil {
					t.Fatal(err)
				}
			}
			after, _ := json.Marshal(fixture)
			if string(before) != string(after) {
				t.Fatal("replay mutated fixture")
			}
			if fixture.Expected.Status == "success" {
				first, err := evalcontract.BuildCandidateComparison(fixture.Input)
				if err != nil {
					t.Fatal(err)
				}
				second, err := evalcontract.BuildCandidateComparison(fixture.Input)
				if err != nil || !reflect.DeepEqual(first, second) || first.Digest == "" {
					t.Fatal("replay is not idempotent")
				}
			}
		})
	}
}

func TestCandidateComparisonReplayRejectsEmptyErrorClassification(t *testing.T) {
	input := candidateComparisonFixtureInput(t)
	input.NetworkVerified = false
	if err := EvaluateCandidateComparisonFixture(CandidateComparisonFixture{Version: CandidateComparisonFixtureVersionV1, Input: input, Expected: CandidateComparisonExpectation{Status: "error"}}); err == nil {
		t.Fatal("an error fixture without an exact reason must fail")
	}
}

func TestCandidateComparisonReplayRejectsOutcomeAndCostDrift(t *testing.T) {
	input := candidateComparisonFixtureInput(t)
	for _, extra := range []string{
		`"axes":{"quality":{"status":"available","value":99}}`,
		`"totals":{"attempts":99}`,
	} {
		rawInput, _ := json.Marshal(input)
		raw := `{"version":"candidate_comparison.v1","input":` + string(rawInput) + `,"expected":{"status":"success","selected_id":"baseline","score_status":"unscored","review_only":true,` + extra + `}}`
		fixture, err := ParseCandidateComparisonFixtureJSON([]byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		if err := EvaluateCandidateComparisonFixture(fixture); err == nil {
			t.Fatalf("accepted outcome drift: %s", extra)
		}
	}
}

func TestCandidateComparisonReplayRejectsDriftAndPreservesLegacy(t *testing.T) {
	input := candidateComparisonFixtureInput(t)
	fixture := CandidateComparisonFixture{Version: CandidateComparisonFixtureVersionV1, Input: input, Expected: CandidateComparisonExpectation{Status: "success", SelectedID: "foreign", ScoreStatus: evalcontract.CandidateScoringUnscored, ReviewOnly: true}}
	if err := EvaluateCandidateComparisonFixture(fixture); err == nil || err.Error() != evalcontract.ReasonCandidateComparisonNonActionable {
		t.Fatalf("expected gate failure, got %v", err)
	}
	legacy, err := ParseEvalContractFixtureJSON([]byte(`{"version":"evaluation_contract.v1","cases":[{"name":"legacy","expected":{},"observed":{}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := EvaluateEvalContractFixture(legacy); err != nil {
		t.Fatal(err)
	}
}

// The contract gate can target a supplied fixture to prove that drift exits nonzero.
func TestCandidateComparisonReplayGateFixture(t *testing.T) {
	fixturePath := os.Getenv("BAYMAX_CANDIDATE_COMPARISON_FIXTURE")
	if fixturePath == "" {
		return
	}
	raw, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := ParseCandidateComparisonFixtureJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := EvaluateCandidateComparisonFixture(fixture); err != nil {
		t.Fatal(err)
	}
}
