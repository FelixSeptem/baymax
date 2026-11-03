package diagnosticsreplay

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestAttentionReplayGoldenAndDrift(t *testing.T) {
	path := os.Getenv("BAYMAX_ATTENTION_FIXTURE")
	if path == "" {
		path = "testdata/attention.golden.json"
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := ReplayAttentionJSON(raw); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"owner", "action", "evidence", "count", "truncated", "version", "unknown", "trailing", "oversize", "duplicate"} {
		t.Run(name, func(t *testing.T) {
			var f AttentionFixture
			if err := json.Unmarshal(raw, &f); err != nil {
				t.Fatal(err)
			}
			switch name {
			case "owner":
				f.Expected.Items[0].Owner = "wrong"
			case "action":
				f.Expected.Items[0].NextAction = "wrong"
			case "evidence":
				f.Request.Evidence[0].Reference = "changed"
			case "count":
				f.Expected.Total++
			case "truncated":
				f.Expected.Truncated = true
			case "version":
				f.Version = "future"
			}
			bad, err := json.Marshal(f)
			if err != nil {
				t.Fatal(err)
			}
			switch name {
			case "unknown":
				bad = []byte(strings.Replace(string(bad), `"version":`, `"private":true,"version":`, 1))
			case "trailing":
				bad = append(bad, []byte(" {}")...)
			case "oversize":
				bad = []byte(strings.Repeat(" ", (1<<20)+1))
			case "duplicate":
				bad = []byte(strings.Replace(string(bad), `"version":`, `"version":"attention.v1","version":`, 1))
			}
			if err := ReplayAttentionJSON(bad); err == nil || !strings.Contains(err.Error(), "attention.replay_drift") {
				t.Fatalf("drift=%v", err)
			}
		})
	}
}

func TestAttentionReplayRejectsUnicodeKeyAliases(t *testing.T) {
	raw := []byte(`{"version":"bad","verſion":"attention.v1","request":{},"expected":{"items":[],"total":0,"truncated":false}}`)
	if err := ReplayAttentionJSON(raw); err == nil || err.Error() != "attention.replay_drift" {
		t.Fatalf("Unicode alias bypass: %v", err)
	}
}
