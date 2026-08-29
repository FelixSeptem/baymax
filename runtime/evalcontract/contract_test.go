package evalcontract

import "testing"

func TestNormalizeCorpusDeterministicAndSorted(t *testing.T) {
	c := Corpus{Version: "EVALUATION_CORPUS.V1", ID: " corpus ", Items: []CorpusItem{{ID: "b", Scenario: " second "}, {ID: "a", Scenario: "first"}}}
	n, d1, err := NormalizeCorpus(c)
	if err != nil {
		t.Fatal(err)
	}
	_, d2, err := NormalizeCorpus(Corpus{Version: CorpusVersionV1, ID: "corpus", Items: []CorpusItem{{ID: "a", Scenario: "first"}, {ID: "b", Scenario: "second"}}})
	if err != nil {
		t.Fatal(err)
	}
	if n.Items[0].ID != "a" || d1 != d2 {
		t.Fatalf("normalization mismatch: %#v %s %s", n, d1, d2)
	}
}

func TestNormalizeCorpusRejectsVersionAndDuplicate(t *testing.T) {
	if _, _, err := NormalizeCorpus(Corpus{Version: "evaluation_corpus.v2", ID: "x", Items: []CorpusItem{{ID: "a", Scenario: "s"}}}); err == nil || err.Error() != ReasonCorpusVersionDrift+": evaluation_corpus.v2" {
		t.Fatalf("unexpected version error: %v", err)
	}
	if _, _, err := NormalizeCorpus(Corpus{Version: CorpusVersionV1, ID: "x", Items: []CorpusItem{{ID: "a", Scenario: "s"}, {ID: "a", Scenario: "s"}}}); err == nil {
		t.Fatal("expected duplicate rejection")
	}
}

func TestBadcaseClassification(t *testing.T) {
	b, err := ClassifyBadcase(Badcase{ID: "b1", Category: "tool", ExpectedDigest: "want"}, "got", true)
	if err == nil || b.Status != BadcaseStatusDrifted {
		t.Fatalf("expected drift: %#v %v", b, err)
	}
	b, err = ClassifyBadcase(Badcase{ID: "b1", Category: "tool"}, "", false)
	if err == nil || b.Status != BadcaseStatusUnavailable {
		t.Fatalf("expected unavailable: %#v %v", b, err)
	}
	b, err = ClassifyBadcase(Badcase{ID: "b1", Category: "tool", ExpectedDigest: "same"}, "same", true)
	if err != nil || b.Status != BadcaseStatusReplayable {
		t.Fatalf("expected replayable: %#v %v", b, err)
	}
}

func TestCompareExperimentsIdempotentAndConflict(t *testing.T) {
	e := Experiment{ID: "e1", CorpusVersion: CorpusVersionV1, RunBatch: "batch-1", Rubric: Rubric{Name: "r", Version: "1"}, ExecutionMode: "distributed", Shards: []ShardMetric{{ShardID: "s1", ItemID: "i1", Digest: "d", Passed: 1, Total: 1}, {ShardID: "s1", ItemID: "i1", Digest: "d", Passed: 1, Total: 1}}}
	r, err := CompareExperiments(e)
	if err != nil || r.Passed != 1 || r.Total != 1 {
		t.Fatalf("unexpected aggregate: %#v %v", r, err)
	}
	e.Shards[1].Digest = "other"
	if _, err := CompareExperiments(e); err == nil || err.Error() != ReasonExperimentConflict {
		t.Fatalf("expected conflict: %v", err)
	}
}

func TestFeedbackRequiresApprovalContext(t *testing.T) {
	if err := ValidateFeedback(FeedbackRecommendation{Version: FeedbackVersionV1, ID: "f", ExperimentID: "e", Status: "approved"}); err == nil || err.Error() != ReasonApprovalMissing {
		t.Fatalf("expected approval missing: %v", err)
	}
	if err := ValidateFeedback(FeedbackRecommendation{ID: "f", ExperimentID: "e", ReviewerID: "u", DecisionContext: "review", Status: "approved"}); err != nil {
		t.Fatal(err)
	}
}
