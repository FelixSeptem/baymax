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
		SandboxDriftBackendProfile,
		SandboxDriftCapabilityClaim,
		SandboxDriftManifestCompat,
		SandboxDriftSessionLifecycle,
		SandboxDriftSessionModeCompat,
		SandboxDriftReasonTaxonomy,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("canonical drift classes mismatch: got=%#v want=%#v", got, want)
	}
}
