package evalcontract

import "testing"

func TestCompareExperimentsPreservesAdditiveContinuityAssociation(t *testing.T) {
	e := Experiment{ID: "e1", CorpusVersion: CorpusVersionV1, RunBatch: "b1", Rubric: Rubric{Name: "r", Version: "1"}, ExecutionMode: "local", Continuity: &ContinuityComparison{Version: ContinuityComparisonVersionV1, ID: "cmp-1", Passed: true}}
	r, err := CompareExperiments(e)
	if err != nil {
		t.Fatal(err)
	}
	if r.ContinuityID != "cmp-1" {
		t.Fatalf("continuity association lost: %#v", r)
	}
}
