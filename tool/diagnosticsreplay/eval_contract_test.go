package diagnosticsreplay

import "testing"

func TestEvalContractFixtureSuccessAndLegacyShape(t *testing.T) {
	f, err := ParseEvalContractFixtureJSON([]byte(`{"version":"EVALUATION_CONTRACT.V1","cases":[{"name":"ok","expected":{"corpus_version":"evaluation_corpus.v1","rubric_digest":"r","badcase_observed_digest":"b","aggregate_digest":"a","approval_status":"approved"},"observed":{"corpus_version":"evaluation_corpus.v1","rubric_digest":"r","badcase_observed_digest":"b","aggregate_digest":"a","approval_status":"approved","reviewer_id":"u"}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := EvaluateEvalContractFixture(f); err != nil {
		t.Fatal(err)
	}
	legacy, err := ParseEvalContractFixtureJSON([]byte(`{"version":"evaluation_contract.v1","cases":[{"name":"legacy","expected":{},"observed":{}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := EvaluateEvalContractFixture(legacy); err != nil {
		t.Fatal(err)
	}
}

func TestEvalContractFixtureDriftClassification(t *testing.T) {
	cases := []struct{ code, field string }{{ReasonEvalCorpusDrift, "corpus_version"}, {ReasonEvalMetricRubricDrift, "rubric_digest"}, {ReasonEvalBadcaseDrift, "badcase_observed_digest"}, {ReasonEvalAggregateConflict, "aggregate_digest"}}
	for _, tc := range cases {
		raw := `{"version":"evaluation_contract.v1","cases":[{"name":"drift","expected":{"` + tc.field + `":"x"},"observed":{"` + tc.field + `":"y"}}]}`
		f, err := ParseEvalContractFixtureJSON([]byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		if err := EvaluateEvalContractFixture(f); err == nil || err.Error()[:len(tc.code)] != tc.code {
			t.Fatalf("got %v want %s", err, tc.code)
		}
	}
}

func TestEvalContractApprovalMissing(t *testing.T) {
	f, err := ParseEvalContractFixtureJSON([]byte(`{"version":"evaluation_contract.v1","cases":[{"name":"approval","expected":{"approval_status":"approved"},"observed":{"approval_status":"approved"}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := EvaluateEvalContractFixture(f); err == nil || err.Error()[:len(ReasonEvalApprovalMissing)] != ReasonEvalApprovalMissing {
		t.Fatalf("got %v", err)
	}
}
