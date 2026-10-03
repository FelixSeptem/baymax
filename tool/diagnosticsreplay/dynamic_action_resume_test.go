package diagnosticsreplay

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestReplayDynamicActionResumeFixtureIsDeterministicAndOpaque(t *testing.T) {
	raw, err := os.ReadFile("testdata/dynamic_action_resume.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToLower(string(raw)), "business_payload") || strings.Contains(strings.ToLower(string(raw)), "secret") {
		t.Fatal("fixture contains business payload")
	}
	first, err := ReplayDynamicActionResumeJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ReplayDynamicActionResumeJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) || len(first.Cases) != 7 {
		t.Fatalf("replay mismatch/case count: %#v %#v", first, second)
	}
	if first.Cases[0].TokenDigest == "opaque-token-1" {
		t.Fatal("token was not normalized to a digest")
	}
}

func TestReplayDynamicActionResumeRejectsInvalidFixture(t *testing.T) {
	for _, raw := range []string{`{"version":"dynamic_action_resume.v0","cases":[]}`, `{"version":"dynamic_action_resume.v1","cases":[{"case_id":"x"}]}`} {
		if _, err := ReplayDynamicActionResumeJSON([]byte(raw)); err == nil {
			t.Fatalf("invalid fixture accepted: %s", raw)
		}
	}
}
