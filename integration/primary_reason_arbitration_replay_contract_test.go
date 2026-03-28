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
	tests := []struct {
		name          string
		versionFolder string
		expected      string
	}{
		{
			name:          "a49",
			versionFolder: "a49",
			expected:      diagnosticsreplay.ArbitrationFixtureVersionA49V1,
		},
		{
			name:          "a50",
			versionFolder: "a50",
			expected:      diagnosticsreplay.ArbitrationFixtureVersionA50V1,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			raw := mustReadArbitrationReplayFixture(t, tc.versionFolder, "success.json")
			out, err := diagnosticsreplay.EvaluateArbitrationFixtureJSON(raw)
			if err != nil {
				t.Fatalf("EvaluateArbitrationFixtureJSON success fixture failed: %v", err)
			}
			if strings.TrimSpace(out.Version) != tc.expected {
				t.Fatalf("fixture version=%q, want %q", out.Version, tc.expected)
			}
			if len(out.Cases) < 1 {
				t.Fatalf("normalized cases len=%d, want >= 1", len(out.Cases))
			}
			replayOut, err := diagnosticsreplay.EvaluateArbitrationFixtureJSON(raw)
			if err != nil {
				t.Fatalf("EvaluateArbitrationFixtureJSON replay failed: %v", err)
			}
			if !reflect.DeepEqual(out, replayOut) {
				t.Fatalf("replay output drift first=%#v replay=%#v", out, replayOut)
			}
		})
	}
}

func TestPrimaryReasonArbitrationReplayContractDriftGuardFailFast(t *testing.T) {
	tests := []struct {
		name       string
		versionDir string
		fixture    string
		wantCode   string
		messageHas string
	}{
		{
			name:       "precedence",
			versionDir: "a49",
			fixture:    "drift-precedence.json",
			wantCode:   diagnosticsreplay.ReasonCodePrecedenceDrift,
			messageHas: "precedence drift",
		},
		{
			name:       "tie-break",
			versionDir: "a49",
			fixture:    "drift-tie-break.json",
			wantCode:   diagnosticsreplay.ReasonCodeTieBreakDrift,
			messageHas: "tie-break drift",
		},
		{
			name:       "taxonomy",
			versionDir: "a49",
			fixture:    "drift-taxonomy.json",
			wantCode:   diagnosticsreplay.ReasonCodeTaxonomyDrift,
			messageHas: "non-canonical primary code",
		},
		{
			name:       "secondary-order",
			versionDir: "a49",
			fixture:    "drift-secondary-order.json",
			wantCode:   diagnosticsreplay.ReasonCodeSecondaryOrderDrift,
			messageHas: "secondary order drift",
		},
		{
			name:       "secondary-count",
			versionDir: "a49",
			fixture:    "drift-secondary-count.json",
			wantCode:   diagnosticsreplay.ReasonCodeSecondaryCountDrift,
			messageHas: "secondary count drift",
		},
		{
			name:       "hint-taxonomy",
			versionDir: "a49",
			fixture:    "drift-hint-taxonomy.json",
			wantCode:   diagnosticsreplay.ReasonCodeHintTaxonomyDrift,
			messageHas: "hint taxonomy drift",
		},
		{
			name:       "rule-version",
			versionDir: "a49",
			fixture:    "drift-rule-version.json",
			wantCode:   diagnosticsreplay.ReasonCodeRuleVersionDrift,
			messageHas: "rule version drift",
		},
		{
			name:       "a50-version-mismatch",
			versionDir: "a50",
			fixture:    "drift-version-mismatch.json",
			wantCode:   diagnosticsreplay.ReasonCodeVersionMismatch,
			messageHas: "version mismatch",
		},
		{
			name:       "a50-unsupported-version",
			versionDir: "a50",
			fixture:    "drift-unsupported-version.json",
			wantCode:   diagnosticsreplay.ReasonCodeUnsupportedVersion,
			messageHas: "unsupported version",
		},
		{
			name:       "a50-cross-version-semantic-drift",
			versionDir: "a50",
			fixture:    "drift-cross-version-semantic-drift.json",
			wantCode:   diagnosticsreplay.ReasonCodeCrossVersionSemanticDrift,
			messageHas: "cross-version semantic drift",
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := diagnosticsreplay.EvaluateArbitrationFixtureJSON(
				mustReadArbitrationReplayFixture(t, tc.versionDir, tc.fixture),
			)
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

func mustReadArbitrationReplayFixture(t *testing.T, versionDir, name string) []byte {
	t.Helper()
	path := filepath.Join(
		repoRootForArbitrationReplay(t),
		"integration",
		"testdata",
		"diagnostics-replay",
		versionDir,
		"v1",
		name,
	)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	return raw
}

func repoRootForArbitrationReplay(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve runtime caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), ".."))
}
