package config

import (
	"os"
	"path/filepath"
	"testing"

	modelcatalog "github.com/FelixSeptem/baymax/model/catalog"
)

func TestProviderCatalogConfigLoadsFromFileAndEnvironmentOverrides(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	if err := os.WriteFile(file, []byte("runtime:\n  provider_catalog:\n    enabled: true\n    version: file-v1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BAYMAX_RUNTIME_PROVIDER_CATALOG_VERSION", "env-v2")
	cfg, err := Load(LoadOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !cfg.Runtime.ProviderCatalog.Enabled || cfg.Runtime.ProviderCatalog.Version != "env-v2" {
		t.Fatalf("ProviderCatalog = %#v, want enabled env-v2", cfg.Runtime.ProviderCatalog)
	}
}

func TestManagerEvaluateProviderModelUsesActiveCatalogSnapshot(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	if err := os.WriteFile(file, []byte("runtime:\n  provider_catalog:\n    enabled: true\n    version: v1\n    strict: false\n    descriptors:\n      - provider: openai\n        model: gpt\n        context_window: 128\n        capabilities: [streaming]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer func() { _ = mgr.Close() }()

	admission, err := mgr.EvaluateProviderModel(modelcatalog.Request{
		Identity: modelcatalog.Identity{Provider: "openai", Model: "gpt"},
		Required: []string{"streaming"},
	}, map[string]modelcatalog.CredentialEvidence{"openai": {Provider: "openai", Status: modelcatalog.CredentialAvailable}})
	if err != nil {
		t.Fatalf("EvaluateProviderModel() error = %v", err)
	}
	if admission.Status != modelcatalog.StatusReady || admission.Catalog != "v1" {
		t.Fatalf("admission = %#v, want ready v1", admission)
	}
}

func TestManagerProviderModelReadinessProjectsBlockingFinding(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	if err := os.WriteFile(file, []byte("runtime:\n  provider_catalog:\n    enabled: true\n    version: v1\n    descriptors:\n      - provider: openai\n        model: gpt\n        context_window: 128\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer func() { _ = mgr.Close() }()
	result, err := mgr.ProviderModelReadiness(modelcatalog.Request{Identity: modelcatalog.Identity{Provider: "openai", Model: "missing"}}, nil)
	if err != nil {
		t.Fatalf("ProviderModelReadiness() error = %v", err)
	}
	if result.Status != ReadinessStatusBlocked || result.ProviderAdmission == nil {
		t.Fatalf("result = %#v, want blocked provider admission", result)
	}
	if len(result.Findings) == 0 || result.Findings[len(result.Findings)-1].Code != modelcatalog.ReasonUnknownModel {
		t.Fatalf("findings = %#v, want unknown model finding", result.Findings)
	}
}

func TestProviderCatalogConfigRejectsInvalidDescriptorAtStartup(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	if err := os.WriteFile(file, []byte("runtime:\n  provider_catalog:\n    enabled: true\n    version: v1\n    descriptors:\n      - provider: openai\n        model: gpt\n        context_window: 0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(LoadOptions{FilePath: file, EnvPrefix: "BAYMAX"}); err == nil {
		t.Fatal("Load() error = nil, want invalid provider catalog rejection")
	}
}

func TestManagerProviderCatalogReloadRetainsPrecedingSnapshotOnInvalidCandidate(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	valid := "runtime:\n  provider_catalog:\n    enabled: true\n    version: v1\n    descriptors:\n      - provider: openai\n        model: gpt\n        context_window: 128\n"
	if err := os.WriteFile(file, []byte(valid), 0o600); err != nil {
		t.Fatal(err)
	}
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer func() { _ = mgr.Close() }()

	if err := os.WriteFile(file, []byte("runtime:\n  provider_catalog:\n    enabled: true\n    version: v2\n    descriptors:\n      - provider: openai\n        model: gpt\n        context_window: 0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	mgr.reload()

	catalog, ok := mgr.ProviderCatalog()
	if !ok || catalog.Version() != "v1" {
		t.Fatalf("ProviderCatalog() = (%#v, %t), want retained v1 catalog", catalog, ok)
	}
	if got := mgr.EffectiveConfig().Runtime.ProviderCatalog.Version; got != "v1" {
		t.Fatalf("effective catalog version = %q, want retained v1", got)
	}
	reloads := mgr.RecentReloads(1)
	if len(reloads) != 1 || reloads[0].Success {
		t.Fatalf("RecentReloads() = %#v, want one failed reload", reloads)
	}
}

func TestManagerProviderCatalogReloadPublishesCompleteCandidateAtomically(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	if err := os.WriteFile(file, []byte("runtime:\n  provider_catalog:\n    enabled: true\n    version: v1\n    descriptors:\n      - provider: openai\n        model: gpt\n        context_window: 128\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer func() { _ = mgr.Close() }()

	if err := os.WriteFile(file, []byte("runtime:\n  provider_catalog:\n    enabled: true\n    version: v2\n    descriptors:\n      - provider: openai\n        model: gpt\n        context_window: 256\n      - provider: openai\n        model: vision\n        context_window: 128\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	mgr.reload()

	catalog, ok := mgr.ProviderCatalog()
	if !ok || catalog.Version() != "v2" {
		t.Fatalf("ProviderCatalog() = (%#v, %t), want v2 catalog", catalog, ok)
	}
	if _, found := catalog.Lookup(modelcatalog.Identity{Provider: "openai", Model: "vision"}); !found {
		t.Fatal("published catalog is missing a descriptor from the complete v2 candidate")
	}
}

func TestManagerProviderModelReadinessStrictEscalatesUnverifiedCredential(t *testing.T) {
	file := filepath.Join(t.TempDir(), "runtime.yaml")
	if err := os.WriteFile(file, []byte("runtime:\n  readiness:\n    strict: true\n  provider_catalog:\n    enabled: true\n    version: v1\n    descriptors:\n      - provider: openai\n        model: gpt\n        context_window: 128\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	mgr, err := NewManager(ManagerOptions{FilePath: file, EnvPrefix: "BAYMAX"})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer func() { _ = mgr.Close() }()
	result, err := mgr.ProviderModelReadiness(modelcatalog.Request{Identity: modelcatalog.Identity{Provider: "openai", Model: "gpt"}}, map[string]modelcatalog.CredentialEvidence{
		"openai": {Provider: "openai", Status: modelcatalog.CredentialUnverified},
	})
	if err != nil {
		t.Fatalf("ProviderModelReadiness() error = %v", err)
	}
	if result.Status != ReadinessStatusBlocked {
		t.Fatalf("status = %q, want blocked", result.Status)
	}
	if result.ProviderAdmission == nil || result.ProviderAdmission.Status != modelcatalog.StatusBlocked {
		t.Fatalf("provider admission = %#v, want blocked", result.ProviderAdmission)
	}
	if result.Summary().PrimaryCode != modelcatalog.ReasonCredentialUnverified {
		t.Fatalf("primary code = %q, want %q", result.Summary().PrimaryCode, modelcatalog.ReasonCredentialUnverified)
	}
}
