package main

import "testing"

func TestHostResumeOfflineExample(t *testing.T) {
	for _, stream := range []bool{false, true} {
		markers, err := exercise(stream)
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{"authority_denied", "resume_accepted", "completed", "revision_drift", "duplicate_no_effect", "resume_in_progress", "effect_cas_rejected"} {
			found := false
			for _, got := range markers {
				if got == want {
					found = true
				}
			}
			if !found {
				t.Fatalf("stream=%v missing %s", stream, want)
			}
		}
	}
}
