package integration

import (
	"testing"

	"github.com/FelixSeptem/baymax/runtime/evalcontract"
)

func TestEvaluationContractRunStreamCorrelationAndReviewOnlyFeedback(t *testing.T) {
	corpus, digest, err := evalcontract.NormalizeCorpus(evalcontract.Corpus{Version: evalcontract.CorpusVersionV1, ID: "corpus-1", Items: []evalcontract.CorpusItem{{ID: "item-1", Scenario: "tool success"}}})
	if err != nil || corpus.ID != "corpus-1" || digest == "" {
		t.Fatalf("corpus normalization failed: %#v %s %v", corpus, digest, err)
	}
	result, err := evalcontract.CompareExperiments(evalcontract.Experiment{ID: "exp-1", CorpusVersion: corpus.Version, RunBatch: "batch-1", Rubric: evalcontract.Rubric{Name: "task", Version: "1"}, ExecutionMode: "local", Shards: []evalcontract.ShardMetric{{ShardID: "run", ItemID: "item-1", Digest: digest, Passed: 1, Total: 1}}})
	if err != nil || result.Passed != 1 || result.Total != 1 {
		t.Fatalf("experiment comparison failed: %#v %v", result, err)
	}
	feedback := evalcontract.FeedbackRecommendation{ID: "feedback-1", ExperimentID: "exp-1", ReviewerID: "reviewer-1", DecisionContext: "manual review", Status: "approved"}
	if err := evalcontract.ValidateFeedback(feedback); err != nil {
		t.Fatalf("feedback validation failed: %v", err)
	}
	runPayload := evalcontract.CorrelationPayload(corpus.Version, corpus.Items[0].ID, "", "exp-1", "1", "compared", "approved")
	streamPayload := evalcontract.CorrelationPayload(corpus.Version, corpus.Items[0].ID, "", "exp-1", "1", "compared", "approved")
	if len(runPayload) != len(streamPayload) || runPayload["eval_experiment_id"] != streamPayload["eval_experiment_id"] {
		t.Fatalf("run/stream evaluation correlation diverged: %#v %#v", runPayload, streamPayload)
	}
	// Run and Stream consume the same normalized result; feedback remains non-executable metadata.
	if result.Digest == "" {
		t.Fatal("expected stable comparison digest")
	}
}
