package diagnosticsreplay

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestSourceAdapterReplayGoldenAndDrift(t *testing.T) {
	p := os.Getenv("BAYMAX_SOURCE_ADAPTER_FIXTURE")
	if p == "" {
		p = "testdata/source_adapter_lifecycle.golden.json"
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := ReplaySourceAdapterLifecycleJSON(raw); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"cursor", "outcome", "pending", "identity", "version", "unknown", "trailing", "bounds"} {
		t.Run(name, func(t *testing.T) {
			var f SourceAdapterLifecycleFixture
			if err := json.Unmarshal(raw, &f); err != nil {
				t.Fatal(err)
			}
			switch name {
			case "cursor":
				f.Steps[2].ExpectedCursor = "3"
			case "outcome":
				f.Steps[2].Outcome = "sent"
			case "pending":
				f.Steps[2].ExpectedPending = ""
			case "identity":
				f.Steps[3].Reference = "changed"
			case "version":
				f.Version = "v2"
			case "bounds":
				f.Steps[0].ExternalID = strings.Repeat("x", 257)
			}
			bad, err := json.Marshal(f)
			if err != nil {
				t.Fatal(err)
			}
			if name == "unknown" {
				bad = []byte(strings.Replace(string(bad), `"version":`, `"payload":"private","version":`, 1))
			}
			if name == "trailing" {
				bad = append(bad, []byte(" {}")...)
			}
			if err := ReplaySourceAdapterLifecycleJSON(bad); err == nil || !strings.Contains(err.Error(), "source_adapter.replay_drift") {
				t.Fatalf("drift=%v", err)
			}
		})
	}
}
