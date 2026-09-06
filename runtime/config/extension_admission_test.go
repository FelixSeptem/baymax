package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/FelixSeptem/baymax/extension"
)

func extensionDescriptorForAdmission() extension.Descriptor {
	return extension.Descriptor{
		Name: "lint", Kind: extension.KindHook, Version: "1.0.0",
		Compat: ">=0.1.0 <1.0.0", Digest: "sha256:abc",
		Source: extension.SourceProjectExplicit,
	}
}

func TestEvaluateExtensionAdmissionBlocksBeforeActivation(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	if err := os.WriteFile(file, []byte("runtime:\n  readiness:\n    enabled: true\n    admission:\n      enabled: true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = mgr.Close() }()
	mgr.SetReadinessComponentSnapshot(RuntimeReadinessComponentSnapshot{
		Scheduler: RuntimeReadinessComponentState{ActivationError: "scheduler unavailable"},
	})
	called := false
	decision, err := ActivateExtension(mgr, extensionDescriptorForAdmission(), extension.AdmissionInput{
		RuntimeVersion: "0.2.0",
	}, func() error { called = true; return nil })
	if err != nil {
		t.Fatal(err)
	}
	if decision.Outcome != ExtensionAdmissionBlocked || decision.Activated || called {
		t.Fatalf("decision=%#v called=%v, want blocked and side-effect free", decision, called)
	}
}

func TestActivateExtensionAllowsDegradedAdmissionAndRecordsMarker(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	if err := os.WriteFile(file, []byte("runtime:\n  readiness:\n    enabled: true\n    admission:\n      enabled: true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = mgr.Close() }()
	mgr.SetReadinessComponentSnapshot(RuntimeReadinessComponentSnapshot{
		Scheduler: RuntimeReadinessComponentState{Fallback: true, FallbackReason: "memory fallback"},
	})
	called := false
	decision, err := ActivateExtension(mgr, extensionDescriptorForAdmission(), extension.AdmissionInput{
		RuntimeVersion: "0.2.0",
	}, func() error { called = true; return nil })
	if err != nil {
		t.Fatal(err)
	}
	if decision.Outcome != ExtensionAdmissionDegraded || !decision.Activated || !called {
		t.Fatalf("decision=%#v called=%v, want degraded allow with activation", decision, called)
	}
}

func TestActivateExtensionDegradedFailFastDenies(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	if err := os.WriteFile(file, []byte("runtime:\n  readiness:\n    enabled: true\n    admission:\n      enabled: true\n      degraded_policy: fail_fast\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = mgr.Close() }()
	mgr.SetReadinessComponentSnapshot(RuntimeReadinessComponentSnapshot{
		Scheduler: RuntimeReadinessComponentState{Fallback: true, FallbackReason: "memory fallback"},
	})
	called := false
	decision, err := ActivateExtension(mgr, extensionDescriptorForAdmission(), extension.AdmissionInput{
		RuntimeVersion: "0.2.0",
	}, func() error { called = true; return nil })
	if err != nil {
		t.Fatal(err)
	}
	if decision.Outcome != ExtensionAdmissionBlocked || decision.Activated || called {
		t.Fatalf("decision=%#v called=%v, want fail-fast deny", decision, called)
	}
}

func TestExtensionAdmissionRunStreamProjectionIsEquivalent(t *testing.T) {
	mgr, err := NewManager(ManagerOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = mgr.Close() }()
	d := extensionDescriptorForAdmission()
	input := extension.AdmissionInput{RuntimeVersion: "0.2.0"}
	runDecision, err := EvaluateExtensionAdmission(mgr, d, input)
	if err != nil {
		t.Fatal(err)
	}
	streamDecision, err := EvaluateExtensionAdmission(mgr, d, input)
	if err != nil {
		t.Fatal(err)
	}
	if runDecision != streamDecision {
		t.Fatalf("run=%#v stream=%#v, want equivalent decisions", runDecision, streamDecision)
	}
}
