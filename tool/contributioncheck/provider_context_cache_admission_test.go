package contributioncheck

import (
	"os"
	"testing"
)

func TestAdmitProviderContextCacheEvidenceRoutesOnlyConfirmedDriftReviewOnly(t *testing.T) {
	raw, err := os.ReadFile("../diagnosticsreplay/testdata/provider_context_cache_evidence.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	result, err := AdmitProviderContextCacheEvidence(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Cases) != 3 {
		t.Fatalf("cases = %d, want 3", len(result.Cases))
	}
	for _, item := range result.Cases {
		if item.Verdict == "drift-confirmed" {
			if item.OwnerRoute != "model/"+item.Provider || !item.ReviewOnly {
				t.Fatalf("confirmed drift route = %#v", item)
			}
			continue
		}
		if item.OwnerRoute != "" || item.ReviewOnly {
			t.Fatalf("non-drift case has mutation-like route = %#v", item)
		}
	}
}
