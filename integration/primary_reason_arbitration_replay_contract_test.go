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

func TestPrimaryReasonArbitrationReplayContractFixtureSuite(t *testing.T) {
	raw := mustReadA48ReplayFixture(t, "success.json")
	out, err := diagnosticsreplay.EvaluateArbitrationFixtureJSON(raw)
	if err != nil {
		t.Fatalf("EvaluateArbitrationFixtureJSON success fixture failed: %v", err)
	}
	if strings.TrimSpace(out.Version) != diagnosticsreplay.ArbitrationFixtureVersionA48V1 {
		t.Fatalf("fixture version=%q, want %q", out.Version, diagnosticsreplay.ArbitrationFixtureVersionA48V1)
	}
	if len(out.Cases) < 2 {
		t.Fatalf("normalized cases len=%d, want >= 2", len(out.Cases))
	}

	replayOut, err := diagnosticsreplay.EvaluateArbitrationFixtureJSON(raw)
	if err != nil {
		t.Fatalf("EvaluateArbitrationFixtureJSON replay failed: %v", err)
	}
	if !reflect.DeepEqual(out, replayOut) {
		t.Fatalf("replay output drift first=%#v replay=%#v", out, replayOut)
	}
}

func TestPrimaryReasonArbitrationReplayContractDriftGuardFailFast(t *testing.T) {
	tests := []struct {
		name       string
		fixture    string
		wantCode   string
		messageHas string
	}{
		{
			name:       "precedence",
			fixture:    "drift-precedence.json",
			wantCode:   diagnosticsreplay.ReasonCodePrecedenceDrift,
			messageHas: "precedence drift",
		},
		{
			name:       "tie-break",
			fixture:    "drift-tie-break.json",
			wantCode:   diagnosticsreplay.ReasonCodeTieBreakDrift,
			messageHas: "tie-break drift",
		},
		{
			name:       "taxonomy",
			fixture:    "drift-taxonomy.json",
			wantCode:   diagnosticsreplay.ReasonCodeTaxonomyDrift,
			messageHas: "non-canonical primary code",
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := diagnosticsreplay.EvaluateArbitrationFixtureJSON(mustReadA48ReplayFixture(t, tc.fixture))
			if err == nil {
				t.Fatalf("fixture %q should fail", tc.fixture)
			}
			vErr, ok := err.(*diagnosticsreplay.ValidationError)
			if !ok {
				t.Fatalf("error type=%T, want *ValidationError", err)
			}
			if vErr.Code != tc.wantCode {
				t.Fatalf("error code=%q, want %q", vErr.Code, tc.wantCode)
			}
			if !strings.Contains(strings.ToLower(vErr.Message), strings.ToLower(tc.messageHas)) {
				t.Fatalf("error message=%q, want contains %q", vErr.Message, tc.messageHas)
			}
		})
	}
}

func mustReadA48ReplayFixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join(repoRootForA48Replay(t), "integration", "testdata", "diagnostics-replay", "a48", "v1", name)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	return raw
}

func repoRootForA48Replay(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve runtime caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), ".."))
}
