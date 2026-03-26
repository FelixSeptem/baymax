package integration

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/FelixSeptem/baymax/tool/diagnosticsreplay"
)

func TestReadinessTimeoutHealthReplayContractCompositeFixtureSuite(t *testing.T) {
	raw := mustReadA47ReplayFixture(t, "composite-success.json")
	out, err := diagnosticsreplay.EvaluateCompositeFixtureJSON(raw)
	if err != nil {
		t.Fatalf("EvaluateCompositeFixtureJSON success fixture failed: %v", err)
	}
	if strings.TrimSpace(out.Version) != diagnosticsreplay.CompositeFixtureVersionA47V1 {
		t.Fatalf("fixture version = %q, want %q", out.Version, diagnosticsreplay.CompositeFixtureVersionA47V1)
	}
	if len(out.Cases) < 3 {
		t.Fatalf("normalized cases len = %d, want >= 3", len(out.Cases))
	}

	replayOut, err := diagnosticsreplay.EvaluateCompositeFixtureJSON(raw)
	if err != nil {
		t.Fatalf("EvaluateCompositeFixtureJSON replay failed: %v", err)
	}
	if strings.TrimSpace(replayOut.Version) != strings.TrimSpace(out.Version) || len(replayOut.Cases) != len(out.Cases) {
		t.Fatalf("replay output drift: first=%#v replay=%#v", out, replayOut)
	}
	for i := range out.Cases {
		if !reflect.DeepEqual(out.Cases[i], replayOut.Cases[i]) {
			t.Fatalf("replay idempotency drift at case=%d first=%#v replay=%#v", i, out.Cases[i], replayOut.Cases[i])
		}
	}
}

func TestReadinessTimeoutHealthReplayContractDriftGuardFailFast(t *testing.T) {
	tests := []struct {
		name           string
		fixture        string
		messageSnippet string
	}{
		{name: "taxonomy", fixture: "drift-taxonomy.json", messageSnippet: "non-canonical readiness code"},
		{name: "source", fixture: "drift-source.json", messageSnippet: "non-canonical timeout source"},
		{name: "state", fixture: "drift-state.json", messageSnippet: "non-canonical adapter circuit state"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			raw := mustReadA47ReplayFixture(t, tc.fixture)
			_, err := diagnosticsreplay.EvaluateCompositeFixtureJSON(raw)
			if err == nil {
				t.Fatalf("fixture %s should fail fast", tc.fixture)
			}
			vErr, ok := err.(*diagnosticsreplay.ValidationError)
			if !ok {
				t.Fatalf("error type = %T, want *ValidationError", err)
			}
			if vErr.Code != diagnosticsreplay.ReasonCodeSemanticDrift {
				t.Fatalf("error code = %q, want %q", vErr.Code, diagnosticsreplay.ReasonCodeSemanticDrift)
			}
			if !strings.Contains(strings.ToLower(vErr.Message), strings.ToLower(tc.messageSnippet)) {
				t.Fatalf("error message = %q, want contains %q", vErr.Message, tc.messageSnippet)
			}
		})
	}
}

func mustReadA47ReplayFixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join(repoRootForA47Replay(t), "integration", "testdata", "diagnostics-replay", "a47", "v1", name)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	return raw
}

func repoRootForA47Replay(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve runtime caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), ".."))
}
