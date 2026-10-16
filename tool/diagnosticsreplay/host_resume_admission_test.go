package diagnosticsreplay

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestHostResumeReplayGoldenAndDrift(t *testing.T) {
	path := os.Getenv("BAYMAX_HOST_RESUME_ADMISSION_FIXTURE")
	if path == "" {
		path = "testdata/host_resume_admission.golden.json"
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	results, err := ReplayHostResumeAdmissionFixtureJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 14 {
		t.Fatalf("cases=%d", len(results))
	}
	var f HostResumeAdmissionFixture
	if err = json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"resolution", "effect", "continuation", "iteration", "terminal", "parity", "binding", "privacy", "version", "duplicate", "unknown"} {
		t.Run(name, func(t *testing.T) {
			var x HostResumeAdmissionFixture
			if err := json.Unmarshal(raw, &x); err != nil {
				t.Fatal(err)
			}
			switch name {
			case "resolution":
				x.Cases[2].Resolutions = 1
			case "effect":
				x.Cases[2].Effects = 1
			case "continuation":
				x.Cases[2].ProviderCalls = 1
			case "iteration":
				x.Cases[2].IterationAfter++
			case "terminal":
				x.Cases[2].TerminalEvents = 1
			case "parity":
				x.Cases[1].Resolutions = 0
			case "binding":
				x.Cases[0].BindingMatches = false
			case "privacy":
				x.Cases[0].RunID = "body with spaces"
			case "version":
				x.Version = "unsupported"
			case "duplicate":
				x.Cases[7].Resolutions = 1
			case "unknown":
				x.Cases[13].ProviderCalls = 1
			}
			bad, _ := json.Marshal(x)
			if _, err := ReplayHostResumeAdmissionFixtureJSON(bad); err == nil || !strings.Contains(err.Error(), "host_resume.replay_drift") {
				t.Fatalf("drift err=%v", err)
			}
		})
	}
	for _, bad := range [][]byte{nil, []byte(`{"version":"host_resume_admission.v1","body":"secret","cases":[]}`), append(raw, []byte(` {}`)...)} {
		if _, err := ReplayHostResumeAdmissionFixtureJSON(bad); err == nil {
			t.Fatal("malformed fixture accepted")
		}
	}
}
