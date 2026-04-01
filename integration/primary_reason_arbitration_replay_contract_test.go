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
		fixture       string
		expected      string
	}{
		{
			name:          "a49",
			versionFolder: "a49",
			fixture:       "success.json",
			expected:      diagnosticsreplay.ArbitrationFixtureVersionA49V1,
		},
		{
			name:          "a50",
			versionFolder: "a50",
			fixture:       "success.json",
			expected:      diagnosticsreplay.ArbitrationFixtureVersionA50V1,
		},
		{
			name:          "a51",
			versionFolder: "tool",
			fixture:       "a51_sandbox_success_input.json",
			expected:      diagnosticsreplay.ArbitrationFixtureVersionA51V1,
		},
		{
			name:          "a57-sandbox-egress",
			versionFolder: "tool",
			fixture:       "a57_sandbox_egress_success_input.json",
			expected:      diagnosticsreplay.ArbitrationFixtureVersionA57V1,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			raw := mustReadArbitrationReplayFixture(t, tc.versionFolder, tc.fixture)
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
		{
			name:       "a51-sandbox-policy-drift",
			versionDir: "tool",
			fixture:    "a51_sandbox_policy_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeSandboxPolicyDrift,
			messageHas: "sandbox policy drift",
		},
		{
			name:       "a51-sandbox-fallback-drift",
			versionDir: "tool",
			fixture:    "a51_sandbox_fallback_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeSandboxFallbackDrift,
			messageHas: "sandbox fallback drift",
		},
		{
			name:       "a51-sandbox-timeout-drift",
			versionDir: "tool",
			fixture:    "a51_sandbox_timeout_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeSandboxTimeoutDrift,
			messageHas: "sandbox timeout drift",
		},
		{
			name:       "a51-sandbox-capability-drift",
			versionDir: "tool",
			fixture:    "a51_sandbox_capability_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeSandboxCapabilityDrift,
			messageHas: "sandbox capability drift",
		},
		{
			name:       "a51-sandbox-resource-policy-drift",
			versionDir: "tool",
			fixture:    "a51_sandbox_resource_policy_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeSandboxResourcePolicyDrift,
			messageHas: "sandbox resource policy drift",
		},
		{
			name:       "a51-sandbox-session-lifecycle-drift",
			versionDir: "tool",
			fixture:    "a51_sandbox_session_lifecycle_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeSandboxSessionLifecycleDrift,
			messageHas: "sandbox session lifecycle drift",
		},
		{
			name:       "a57-sandbox-egress-action-drift",
			versionDir: "tool",
			fixture:    "a57_sandbox_egress_action_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeSandboxEgressActionDrift,
			messageHas: "sandbox egress action drift",
		},
		{
			name:       "a57-sandbox-egress-policy-source-drift",
			versionDir: "tool",
			fixture:    "a57_sandbox_egress_policy_source_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeSandboxEgressPolicySourceDrift,
			messageHas: "policy source drift",
		},
		{
			name:       "a57-sandbox-egress-violation-taxonomy-drift",
			versionDir: "tool",
			fixture:    "a57_sandbox_egress_violation_taxonomy_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeSandboxEgressViolationTaxonomyDrift,
			messageHas: "violation taxonomy drift",
		},
		{
			name:       "a57-adapter-allowlist-decision-drift",
			versionDir: "tool",
			fixture:    "a57_adapter_allowlist_decision_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeAdapterAllowlistDecisionDrift,
			messageHas: "allowlist decision drift",
		},
		{
			name:       "a57-adapter-allowlist-taxonomy-drift",
			versionDir: "tool",
			fixture:    "a57_adapter_allowlist_taxonomy_drift_input.json",
			wantCode:   diagnosticsreplay.ReasonCodeAdapterAllowlistTaxonomyDrift,
			messageHas: "allowlist taxonomy drift",
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

func TestReplayContractSandboxEgressAllowlistFixture(t *testing.T) {
	raw := mustReadArbitrationReplayFixture(t, "tool", "a57_sandbox_egress_success_input.json")
	out, err := diagnosticsreplay.EvaluateArbitrationFixtureJSON(raw)
	if err != nil {
		t.Fatalf("EvaluateArbitrationFixtureJSON success fixture failed: %v", err)
	}
	if strings.TrimSpace(out.Version) != diagnosticsreplay.ArbitrationFixtureVersionA57V1 {
		t.Fatalf("fixture version=%q, want %q", out.Version, diagnosticsreplay.ArbitrationFixtureVersionA57V1)
	}
	if len(out.Cases) == 0 {
		t.Fatal("normalized output cases should not be empty")
	}
	replayOut, err := diagnosticsreplay.EvaluateArbitrationFixtureJSON(raw)
	if err != nil {
		t.Fatalf("EvaluateArbitrationFixtureJSON replay failed: %v", err)
	}
	if !reflect.DeepEqual(out, replayOut) {
		t.Fatalf("replay output drift first=%#v replay=%#v", out, replayOut)
	}

	_, err = diagnosticsreplay.EvaluateArbitrationFixtureJSON(
		mustReadArbitrationReplayFixture(t, "tool", "a57_adapter_allowlist_taxonomy_drift_input.json"),
	)
	if err == nil {
		t.Fatal("taxonomy drift fixture should fail")
	}
	vErr, ok := err.(*diagnosticsreplay.ValidationError)
	if !ok {
		t.Fatalf("error type=%T, want *ValidationError", err)
	}
	if vErr.Code != diagnosticsreplay.ReasonCodeAdapterAllowlistTaxonomyDrift {
		t.Fatalf("error code=%q, want %q", vErr.Code, diagnosticsreplay.ReasonCodeAdapterAllowlistTaxonomyDrift)
	}
}

func mustReadArbitrationReplayFixture(t *testing.T, versionDir, name string) []byte {
	t.Helper()
	root := repoRootForArbitrationReplay(t)
	path := ""
	if strings.EqualFold(strings.TrimSpace(versionDir), "tool") {
		path = filepath.Join(
			root,
			"tool",
			"diagnosticsreplay",
			"testdata",
			name,
		)
	} else {
		path = filepath.Join(
			root,
			"integration",
			"testdata",
			"diagnostics-replay",
			versionDir,
			"v1",
			name,
		)
	}
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
