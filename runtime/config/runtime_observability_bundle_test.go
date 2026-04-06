package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestManagerGenerateDiagnosticsBundleManifestAndRedaction(t *testing.T) {
	outputDir := filepath.ToSlash(filepath.Join(t.TempDir(), "bundles"))
	cfgPath := filepath.Join(t.TempDir(), "runtime-diagnostics-bundle-success.yaml")
	writeDiagnosticsBundleRuntimeConfig(
		t,
		cfgPath,
		outputDir,
		16,
		[]string{
			RuntimeDiagnosticsBundleSectionTimeline,
			RuntimeDiagnosticsBundleSectionDiagnostics,
			RuntimeDiagnosticsBundleSectionEffectiveConfig,
			RuntimeDiagnosticsBundleSectionReplayHints,
			RuntimeDiagnosticsBundleSectionGateFingerprint,
		},
	)

	mgr, err := NewManager(ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX_DIAGNOSTICS_BUNDLE_TEST"})
	if err != nil {
		t.Fatalf("new runtime manager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	result, err := mgr.GenerateDiagnosticsBundle(context.Background(), DiagnosticsBundleGenerateRequest{
		RunID:           "run-a55-bundle-success",
		GeneratedAt:     time.Date(2026, 3, 31, 8, 0, 0, 0, time.UTC),
		GateFingerprint: RuntimeDiagnosticsGateFingerprintObservabilityBundleV1,
		RunFinishedPayload: map[string]any{
			"client_secret": "raw-secret",
			"status":        "success",
		},
	})
	if err != nil {
		t.Fatalf("GenerateDiagnosticsBundle failed: %v", err)
	}
	if result.Status != RuntimeDiagnosticsBundleStatusSuccess || result.Total != 1 {
		t.Fatalf("unexpected bundle result: %#v", result)
	}
	if strings.TrimSpace(result.BundleDir) == "" || strings.TrimSpace(result.ManifestPath) == "" {
		t.Fatalf("bundle paths should not be empty: %#v", result)
	}

	rawManifest, err := os.ReadFile(result.ManifestPath)
	if err != nil {
		t.Fatalf("read manifest failed: %v", err)
	}
	var manifest DiagnosticsBundleManifest
	if err := json.Unmarshal(rawManifest, &manifest); err != nil {
		t.Fatalf("unmarshal manifest failed: %v", err)
	}
	if manifest.SchemaVersion != RuntimeDiagnosticsBundleSchemaVersionV1 {
		t.Fatalf("manifest schema_version = %q, want %q", manifest.SchemaVersion, RuntimeDiagnosticsBundleSchemaVersionV1)
	}
	if manifest.Metadata.RedactionStatus != "redacted" ||
		manifest.Metadata.ReplayHintSchema != RuntimeDiagnosticsReplayHintSchemaV1 ||
		manifest.Metadata.GateFingerprint != RuntimeDiagnosticsGateFingerprintObservabilityBundleV1 {
		t.Fatalf("manifest metadata mismatch: %#v", manifest.Metadata)
	}
	if len(manifest.Sections) != 5 {
		t.Fatalf("manifest sections len=%d, want 5", len(manifest.Sections))
	}
	for _, required := range []string{
		RuntimeDiagnosticsBundleSectionTimeline,
		RuntimeDiagnosticsBundleSectionDiagnostics,
		RuntimeDiagnosticsBundleSectionEffectiveConfig,
		RuntimeDiagnosticsBundleSectionReplayHints,
		RuntimeDiagnosticsBundleSectionGateFingerprint,
	} {
		if !manifestHasSection(manifest, required) {
			t.Fatalf("manifest missing section %q: %#v", required, manifest.Sections)
		}
	}

	diagSection := filepath.Join(result.BundleDir, RuntimeDiagnosticsBundleSectionDiagnostics+".json")
	rawDiagnostics, err := os.ReadFile(diagSection)
	if err != nil {
		t.Fatalf("read diagnostics section failed: %v", err)
	}
	text := string(rawDiagnostics)
	if strings.Contains(text, "raw-secret") {
		t.Fatalf("diagnostics section should be redacted, got: %s", text)
	}
	if !strings.Contains(text, "***") {
		t.Fatalf("diagnostics section should contain redaction marker, got: %s", text)
	}
}

func TestManagerGenerateDiagnosticsBundleOutputUnavailable(t *testing.T) {
	tmp := t.TempDir()
	blocked := filepath.Join(tmp, "blocked")
	if err := os.WriteFile(blocked, []byte("x"), 0o600); err != nil {
		t.Fatalf("write blocked marker failed: %v", err)
	}
	outputDir := filepath.ToSlash(filepath.Join(blocked, "bundles"))
	cfgPath := filepath.Join(tmp, "runtime-diagnostics-bundle-output-unavailable.yaml")
	writeDiagnosticsBundleRuntimeConfig(
		t,
		cfgPath,
		outputDir,
		16,
		[]string{
			RuntimeDiagnosticsBundleSectionTimeline,
			RuntimeDiagnosticsBundleSectionDiagnostics,
			RuntimeDiagnosticsBundleSectionEffectiveConfig,
			RuntimeDiagnosticsBundleSectionReplayHints,
			RuntimeDiagnosticsBundleSectionGateFingerprint,
		},
	)

	mgr, err := NewManager(ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX_DIAGNOSTICS_BUNDLE_TEST"})
	if err != nil {
		t.Fatalf("new runtime manager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	result, err := mgr.GenerateDiagnosticsBundle(context.Background(), DiagnosticsBundleGenerateRequest{RunID: "run-a55-bundle-output"})
	if err == nil {
		t.Fatal("expected output_unavailable error, got nil")
	}
	if code := DiagnosticsBundleErrorCode(err); code != RuntimeDiagnosticsBundleReasonOutputUnavailable {
		t.Fatalf("error code = %q, want %q (err=%v)", code, RuntimeDiagnosticsBundleReasonOutputUnavailable, err)
	}
	if result.Status != RuntimeDiagnosticsBundleStatusFailed || result.ReasonCode != RuntimeDiagnosticsBundleReasonOutputUnavailable {
		t.Fatalf("unexpected bundle failure result: %#v", result)
	}
}

func TestManagerGenerateDiagnosticsBundleMaxSizeExceeded(t *testing.T) {
	outputDir := filepath.ToSlash(filepath.Join(t.TempDir(), "bundles"))
	cfgPath := filepath.Join(t.TempDir(), "runtime-diagnostics-bundle-max-size.yaml")
	writeDiagnosticsBundleRuntimeConfig(
		t,
		cfgPath,
		outputDir,
		1,
		[]string{
			RuntimeDiagnosticsBundleSectionTimeline,
			RuntimeDiagnosticsBundleSectionDiagnostics,
			RuntimeDiagnosticsBundleSectionEffectiveConfig,
			RuntimeDiagnosticsBundleSectionReplayHints,
			RuntimeDiagnosticsBundleSectionGateFingerprint,
		},
	)

	mgr, err := NewManager(ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX_DIAGNOSTICS_BUNDLE_TEST"})
	if err != nil {
		t.Fatalf("new runtime manager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	huge := strings.Repeat("x", 2*1024*1024)
	result, err := mgr.GenerateDiagnosticsBundle(context.Background(), DiagnosticsBundleGenerateRequest{
		RunID: "run-a55-bundle-max-size",
		RunFinishedPayload: map[string]any{
			"huge_payload": huge,
		},
	})
	if err == nil {
		t.Fatal("expected max_size_exceeded error, got nil")
	}
	if code := DiagnosticsBundleErrorCode(err); code != RuntimeDiagnosticsBundleReasonMaxSizeExceeded {
		t.Fatalf("error code = %q, want %q (err=%v)", code, RuntimeDiagnosticsBundleReasonMaxSizeExceeded, err)
	}
	if result.Status != RuntimeDiagnosticsBundleStatusFailed || result.ReasonCode != RuntimeDiagnosticsBundleReasonMaxSizeExceeded {
		t.Fatalf("unexpected bundle failure result: %#v", result)
	}
}

func TestManagerGenerateDiagnosticsBundleMissingRequiredSection(t *testing.T) {
	outputDir := filepath.ToSlash(filepath.Join(t.TempDir(), "bundles"))
	cfgPath := filepath.Join(t.TempDir(), "runtime-diagnostics-bundle-missing-section.yaml")
	writeDiagnosticsBundleRuntimeConfig(
		t,
		cfgPath,
		outputDir,
		16,
		[]string{
			RuntimeDiagnosticsBundleSectionTimeline,
			RuntimeDiagnosticsBundleSectionDiagnostics,
		},
	)

	mgr, err := NewManager(ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX_DIAGNOSTICS_BUNDLE_TEST"})
	if err != nil {
		t.Fatalf("new runtime manager failed: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	result, err := mgr.GenerateDiagnosticsBundle(context.Background(), DiagnosticsBundleGenerateRequest{RunID: "run-a55-bundle-missing"})
	if err == nil {
		t.Fatal("expected section_missing error, got nil")
	}
	if code := DiagnosticsBundleErrorCode(err); code != RuntimeDiagnosticsBundleReasonSectionMissing {
		t.Fatalf("error code = %q, want %q (err=%v)", code, RuntimeDiagnosticsBundleReasonSectionMissing, err)
	}
	if result.Status != RuntimeDiagnosticsBundleStatusFailed || result.ReasonCode != RuntimeDiagnosticsBundleReasonSectionMissing {
		t.Fatalf("unexpected bundle failure result: %#v", result)
	}
}

func writeDiagnosticsBundleRuntimeConfig(t *testing.T, path, outputDir string, maxSizeMB int, includeSections []string) {
	t.Helper()
	sections := make([]string, 0, len(includeSections))
	for _, section := range includeSections {
		sections = append(sections, "        - "+section)
	}
	content := "" +
		"mcp:\n" +
		"  active_profile: default\n" +
		"  profiles:\n" +
		"    default:\n" +
		"      call_timeout: 2s\n" +
		"      retry: 0\n" +
		"      backoff: 10ms\n" +
		"      queue_size: 16\n" +
		"      backpressure: block\n" +
		"      read_pool_size: 2\n" +
		"      write_pool_size: 1\n" +
		"runtime:\n" +
		"  diagnostics:\n" +
		"    bundle:\n" +
		"      enabled: true\n" +
		"      output_dir: " + outputDir + "\n" +
		"      max_size_mb: " + fmt.Sprintf("%d", maxSizeMB) + "\n" +
		"      include_sections:\n" + strings.Join(sections, "\n") + "\n" +
		"security:\n" +
		"  redaction:\n" +
		"    enabled: true\n" +
		"    strategy: keyword\n" +
		"    keywords: [secret, token, api_key, apikey, password]\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write runtime config failed: %v", err)
	}
}

func manifestHasSection(manifest DiagnosticsBundleManifest, section string) bool {
	section = strings.TrimSpace(section)
	for _, item := range manifest.Sections {
		if strings.TrimSpace(item.Name) == section {
			return true
		}
	}
	return false
}
