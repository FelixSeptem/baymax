package adapterconformance

import (
	"reflect"
	"testing"

	adaptermanifest "github.com/FelixSeptem/baymax/adapter/manifest"
)

func TestSandboxAdapterConformanceMainstreamBackendMatrixLinux(t *testing.T) {
	results := EvaluateMainstreamSandboxBackendMatrix("linux")
	executed := make([]string, 0)
	skipped := map[string]string{}
	for i := range results {
		if results[i].Executed {
			executed = append(executed, results[i].Backend)
			continue
		}
		skipped[results[i].Backend] = results[i].SkipClass
	}
	wantExecuted := []string{
		adaptermanifest.SandboxBackendLinuxBwrap,
		adaptermanifest.SandboxBackendLinuxNSJail,
		adaptermanifest.SandboxBackendOCIRuntime,
	}
	if !reflect.DeepEqual(executed, wantExecuted) {
		t.Fatalf("linux matrix executed mismatch: got=%#v want=%#v", executed, wantExecuted)
	}
	if skipped[adaptermanifest.SandboxBackendWindowsJob] != SandboxSkipBackendUnavailable {
		t.Fatalf("windows backend skip classification mismatch: %#v", skipped)
	}
}

func TestSandboxAdapterConformanceMainstreamBackendMatrixWindows(t *testing.T) {
	results := EvaluateMainstreamSandboxBackendMatrix("windows")
	var windowsExecuted bool
	for i := range results {
		item := results[i]
		if item.Backend == adaptermanifest.SandboxBackendWindowsJob {
			windowsExecuted = item.Executed
			if item.SkipClass != "" {
				t.Fatalf("windows backend should not be skipped: %#v", item)
			}
			continue
		}
		if item.Executed {
			t.Fatalf("linux backend must be skipped on windows host: %#v", item)
		}
		if item.SkipClass != SandboxSkipBackendUnavailable {
			t.Fatalf("unexpected skip class: %#v", item)
		}
	}
	if !windowsExecuted {
		t.Fatal("windows backend suite must execute on windows host")
	}
}

func TestSandboxAdapterConformanceCapabilityNegotiation(t *testing.T) {
	missingRequired := EvaluateSandboxCapabilityNegotiation(
		[]string{"sandbox.adapter.backend_profile_resolved", "sandbox.adapter.session.lifecycle"},
		[]string{"sandbox.adapter.lifecycle.crash_reconnect"},
		[]string{"sandbox.adapter.backend_profile_resolved"},
	)
	if missingRequired.Accepted {
		t.Fatalf("required missing must fail fast: %#v", missingRequired)
	}
	if missingRequired.DriftClass != SandboxDriftCapabilityClaim {
		t.Fatalf("required missing drift class mismatch: %#v", missingRequired)
	}
	if len(missingRequired.MissingRequired) != 1 ||
		missingRequired.MissingRequired[0] != "sandbox.adapter.session.lifecycle" {
		t.Fatalf("missing required mismatch: %#v", missingRequired)
	}

	optionalDowngrade := EvaluateSandboxCapabilityNegotiation(
		[]string{"sandbox.adapter.backend_profile_resolved"},
		[]string{"sandbox.adapter.lifecycle.crash_reconnect"},
		[]string{"sandbox.adapter.backend_profile_resolved"},
	)
	if !optionalDowngrade.Accepted {
		t.Fatalf("optional downgrade path should be accepted: %#v", optionalDowngrade)
	}
	if len(optionalDowngrade.DowngradedOptional) != 1 ||
		optionalDowngrade.DowngradedOptional[0] != "sandbox.adapter.lifecycle.crash_reconnect" {
		t.Fatalf("optional downgrade mismatch: %#v", optionalDowngrade)
	}
}

func TestSandboxAdapterConformanceSessionLifecycle(t *testing.T) {
	h := NewSandboxSessionLifecycleHarness()
	const key = "adapter-session"

	perSessionFirst := h.Open(adaptermanifest.SandboxSessionModePerSession, key)
	perSessionSecond := h.Open(adaptermanifest.SandboxSessionModePerSession, key)
	if perSessionFirst != perSessionSecond {
		t.Fatalf("per_session must reuse token: first=%q second=%q", perSessionFirst, perSessionSecond)
	}

	h.Crash(key)
	reconnected := h.Open(adaptermanifest.SandboxSessionModePerSession, key)
	if reconnected == perSessionFirst {
		t.Fatalf("crash/reconnect should rotate token: old=%q new=%q", perSessionFirst, reconnected)
	}

	perCallFirst := h.Open(adaptermanifest.SandboxSessionModePerCall, key)
	perCallSecond := h.Open(adaptermanifest.SandboxSessionModePerCall, key)
	if perCallFirst == perCallSecond {
		t.Fatalf("per_call must use isolated token: first=%q second=%q", perCallFirst, perCallSecond)
	}

	if !h.Close(key) {
		t.Fatal("first close should apply terminal side-effect")
	}
	if h.Close(key) {
		t.Fatal("second close must be idempotent without duplicate side-effect")
	}
}

func TestSandboxAdapterConformanceCanonicalDriftClasses(t *testing.T) {
	got := CanonicalSandboxDriftClasses()
	want := []string{
		SandboxDriftAllowlistActivation,
		SandboxDriftAllowlistTaxonomy,
		SandboxDriftBackendProfile,
		SandboxDriftCapabilityClaim,
		SandboxDriftEgressPolicyDecision,
		SandboxDriftEgressSelectorPrecedence,
		SandboxDriftManifestCompat,
		SandboxDriftSessionLifecycle,
		SandboxDriftSessionModeCompat,
		SandboxDriftReasonTaxonomy,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("canonical drift classes mismatch: got=%#v want=%#v", got, want)
	}
}

func TestSandboxAdapterConformanceEgressPolicyMatrix(t *testing.T) {
	matrix := MainstreamSandboxEgressPolicyMatrix()
	if len(matrix) == 0 {
		t.Fatal("egress policy matrix should not be empty")
	}
	coverage := map[string]map[string]bool{}
	for i := range matrix {
		tc := matrix[i]
		if coverage[tc.CaseID] == nil {
			coverage[tc.CaseID] = map[string]bool{}
		}
		got, err := EvaluateSandboxEgressPolicyCase(tc)
		if err != nil {
			t.Fatalf("evaluate egress case %s/%s failed: %v", tc.CaseID, tc.Backend, err)
		}
		drift := classifySandboxEgressMatrixDrift(tc, got)
		if drift != "" {
			t.Fatalf("egress matrix drift case=%s backend=%s drift=%s got=%#v", tc.CaseID, tc.Backend, drift, got)
		}
		coverage[tc.CaseID][tc.Backend] = true
	}
	requiredCases := []string{
		"sandbox-egress-deny-matrix",
		"sandbox-egress-allow-matrix",
		"sandbox-egress-allow-and-record-matrix",
		"sandbox-egress-selector-override-precedence",
	}
	backends := MainstreamSandboxBackendMatrix()
	for _, caseID := range requiredCases {
		for _, backend := range backends {
			if !coverage[caseID][backend.Backend] {
				t.Fatalf("missing egress matrix coverage case=%s backend=%s", caseID, backend.Backend)
			}
		}
	}
}

func TestSandboxAdapterConformanceEgressSelectorOverridePrecedence(t *testing.T) {
	caseDef := SandboxEgressPolicyCase{
		CaseID:               "sandbox-egress-selector-override-precedence",
		Backend:              adaptermanifest.SandboxBackendLinuxNSJail,
		ProfileID:            adaptermanifest.SandboxBackendLinuxNSJail,
		NamespaceTool:        "local+shell",
		Host:                 "api.example.com",
		DefaultAction:        "allow",
		OnViolation:          "deny",
		ByTool:               map[string]string{"local+shell": "deny"},
		Allowlist:            []string{"api.example.com"},
		ExpectedAction:       "deny",
		ExpectedPolicySource: "by_tool",
	}
	got, err := EvaluateSandboxEgressPolicyCase(caseDef)
	if err != nil {
		t.Fatalf("evaluate selector override precedence failed: %v", err)
	}
	if got.Action != "deny" || got.PolicySource != "by_tool" || got.ReasonCode != "sandbox.egress_deny" {
		t.Fatalf("selector override precedence mismatch: %#v", got)
	}
}

func TestSandboxAdapterConformanceAllowlistActivationMatrix(t *testing.T) {
	matrix := MainstreamAdapterAllowlistActivationMatrix()
	if len(matrix) == 0 {
		t.Fatal("allowlist activation matrix should not be empty")
	}
	coverage := map[string]map[string]bool{}
	for i := range matrix {
		tc := matrix[i]
		if coverage[tc.CaseID] == nil {
			coverage[tc.CaseID] = map[string]bool{}
		}
		got, err := EvaluateAdapterAllowlistActivationCase(tc)
		if err != nil {
			t.Fatalf("evaluate allowlist case %s/%s failed: %v", tc.CaseID, tc.Backend, err)
		}
		drift := classifyAllowlistMatrixDrift(tc.ExpectedContractCode, got)
		if drift != "" {
			t.Fatalf("allowlist matrix drift case=%s backend=%s drift=%s got=%#v", tc.CaseID, tc.Backend, drift, got)
		}
		coverage[tc.CaseID][tc.Backend] = true
	}
	requiredCases := []string{
		"adapter-allowlist-missing-entry-enforce",
		"adapter-allowlist-signature-invalid-enforce",
		"adapter-allowlist-allowed-path-enforce",
		"adapter-allowlist-policy-conflict",
	}
	backends := MainstreamSandboxBackendMatrix()
	for _, caseID := range requiredCases {
		for _, backend := range backends {
			if !coverage[caseID][backend.Backend] {
				t.Fatalf("missing allowlist matrix coverage case=%s backend=%s", caseID, backend.Backend)
			}
		}
	}
}

func TestSandboxAdapterConformanceAllowlistTaxonomyDriftClassification(t *testing.T) {
	drift := classifyAllowlistMatrixDrift(
		adaptermanifest.CodeAllowlistMissingEntry,
		AdapterAllowlistActivationResult{Accepted: false, ContractCode: adaptermanifest.CodeInvalidField},
	)
	if drift != SandboxDriftAllowlistTaxonomy {
		t.Fatalf("allowlist taxonomy drift classification mismatch: %q", drift)
	}
}
