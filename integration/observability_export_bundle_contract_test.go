package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/integration/fakes"
	"github.com/FelixSeptem/baymax/observability/event"
	"github.com/FelixSeptem/baymax/orchestration/composer"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

func TestObservabilityExportBundleContractRunStreamSemanticEquivalenceSuccess(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime-observability-run-stream-success.yaml")
	outputDir := filepath.ToSlash(filepath.Join(t.TempDir(), "bundles"))
	writeObservabilityBundleRuntimeConfig(t, cfgPath, outputDir, true, "http://127.0.0.1:4318")

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX_OBSERVABILITY_BUNDLE_TEST"})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	model := fakes.NewModel([]fakes.ModelStep{
		{Response: types.ModelResponse{FinalAnswer: "ok"}},
	})
	model.SetStream([]types.ModelEvent{
		{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "ok"},
	}, nil)

	dispatcher := event.NewDispatcher(event.NewRuntimeRecorder(mgr))
	comp, err := composer.NewBuilder(model).
		WithRuntimeManager(mgr).
		WithEventHandler(dispatcherHandler{dispatcher: dispatcher}).
		Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}

	runReq := types.RunRequest{RunID: "run-observability-export-run", Input: "ping"}
	streamReq := types.RunRequest{RunID: "run-observability-export-stream", Input: "ping"}
	if _, err := comp.Run(context.Background(), runReq, nil); err != nil {
		t.Fatalf("composer run failed: %v", err)
	}
	if _, err := comp.Stream(context.Background(), streamReq, nil); err != nil {
		t.Fatalf("composer stream failed: %v", err)
	}

	runRecord := findRunRecord(t, mgr.RecentRuns(20), runReq.RunID)
	streamRecord := findRunRecord(t, mgr.RecentRuns(20), streamReq.RunID)
	if runRecord.ObservabilityExportProfile != runtimeconfig.RuntimeObservabilityExportProfileOTLP ||
		streamRecord.ObservabilityExportProfile != runtimeconfig.RuntimeObservabilityExportProfileOTLP {
		t.Fatalf("unexpected export profile run=%q stream=%q", runRecord.ObservabilityExportProfile, streamRecord.ObservabilityExportProfile)
	}
	if runRecord.ObservabilityExportStatus != streamRecord.ObservabilityExportStatus {
		t.Fatalf("run/stream export status mismatch run=%q stream=%q", runRecord.ObservabilityExportStatus, streamRecord.ObservabilityExportStatus)
	}
	if runRecord.DiagnosticsBundleTotal != 1 || streamRecord.DiagnosticsBundleTotal != 1 {
		t.Fatalf("bundle total mismatch run=%d stream=%d", runRecord.DiagnosticsBundleTotal, streamRecord.DiagnosticsBundleTotal)
	}
	if runRecord.DiagnosticsBundleLastStatus != runtimeconfig.RuntimeDiagnosticsBundleStatusSuccess ||
		streamRecord.DiagnosticsBundleLastStatus != runtimeconfig.RuntimeDiagnosticsBundleStatusSuccess {
		t.Fatalf("bundle status mismatch run=%q stream=%q", runRecord.DiagnosticsBundleLastStatus, streamRecord.DiagnosticsBundleLastStatus)
	}
	if runRecord.DiagnosticsBundleLastReasonCode != "" || streamRecord.DiagnosticsBundleLastReasonCode != "" {
		t.Fatalf("bundle reason should be empty on success run=%q stream=%q", runRecord.DiagnosticsBundleLastReasonCode, streamRecord.DiagnosticsBundleLastReasonCode)
	}
	if runRecord.DiagnosticsBundleLastSchemaVersion != runtimeconfig.RuntimeDiagnosticsBundleSchemaVersionV1 ||
		streamRecord.DiagnosticsBundleLastSchemaVersion != runtimeconfig.RuntimeDiagnosticsBundleSchemaVersionV1 {
		t.Fatalf("bundle schema mismatch run=%q stream=%q", runRecord.DiagnosticsBundleLastSchemaVersion, streamRecord.DiagnosticsBundleLastSchemaVersion)
	}
}

func TestObservabilityExportBundleContractRunStreamBundleFailureTaxonomyEquivalent(t *testing.T) {
	tmp := t.TempDir()
	blocked := filepath.Join(tmp, "blocked")
	if err := os.WriteFile(blocked, []byte("x"), 0o600); err != nil {
		t.Fatalf("write blocked marker failed: %v", err)
	}
	cfgPath := filepath.Join(tmp, "runtime-observability-run-stream-bundle-failure.yaml")
	outputDir := filepath.ToSlash(filepath.Join(blocked, "bundles"))
	writeObservabilityBundleRuntimeConfig(t, cfgPath, outputDir, false, "")

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX_OBSERVABILITY_BUNDLE_TEST"})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	model := fakes.NewModel([]fakes.ModelStep{
		{Response: types.ModelResponse{FinalAnswer: "ok"}},
	})
	model.SetStream([]types.ModelEvent{
		{Type: types.ModelEventTypeOutputTextDelta, TextDelta: "ok"},
	}, nil)

	dispatcher := event.NewDispatcher(event.NewRuntimeRecorder(mgr))
	comp, err := composer.NewBuilder(model).
		WithRuntimeManager(mgr).
		WithEventHandler(dispatcherHandler{dispatcher: dispatcher}).
		Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}

	runReq := types.RunRequest{RunID: "run-observability-bundle-failure-run", Input: "ping"}
	streamReq := types.RunRequest{RunID: "run-observability-bundle-failure-stream", Input: "ping"}
	if _, err := comp.Run(context.Background(), runReq, nil); err != nil {
		t.Fatalf("composer run failed: %v", err)
	}
	if _, err := comp.Stream(context.Background(), streamReq, nil); err != nil {
		t.Fatalf("composer stream failed: %v", err)
	}

	runRecord := findRunRecord(t, mgr.RecentRuns(20), runReq.RunID)
	streamRecord := findRunRecord(t, mgr.RecentRuns(20), streamReq.RunID)
	if runRecord.DiagnosticsBundleLastStatus != runtimeconfig.RuntimeDiagnosticsBundleStatusFailed ||
		streamRecord.DiagnosticsBundleLastStatus != runtimeconfig.RuntimeDiagnosticsBundleStatusFailed {
		t.Fatalf("bundle status mismatch run=%q stream=%q", runRecord.DiagnosticsBundleLastStatus, streamRecord.DiagnosticsBundleLastStatus)
	}
	if runRecord.DiagnosticsBundleLastReasonCode != runtimeconfig.RuntimeDiagnosticsBundleReasonOutputUnavailable ||
		streamRecord.DiagnosticsBundleLastReasonCode != runtimeconfig.RuntimeDiagnosticsBundleReasonOutputUnavailable {
		t.Fatalf(
			"bundle reason mismatch run=%q stream=%q",
			runRecord.DiagnosticsBundleLastReasonCode,
			streamRecord.DiagnosticsBundleLastReasonCode,
		)
	}
	if runRecord.DiagnosticsBundleLastSchemaVersion != runtimeconfig.RuntimeDiagnosticsBundleSchemaVersionV1 ||
		streamRecord.DiagnosticsBundleLastSchemaVersion != runtimeconfig.RuntimeDiagnosticsBundleSchemaVersionV1 {
		t.Fatalf("bundle schema mismatch run=%q stream=%q", runRecord.DiagnosticsBundleLastSchemaVersion, streamRecord.DiagnosticsBundleLastSchemaVersion)
	}
}

func writeObservabilityBundleRuntimeConfig(t *testing.T, path, outputDir string, exportEnabled bool, endpoint string) {
	t.Helper()
	endpoint = strings.TrimSpace(endpoint)
	profile := runtimeconfig.RuntimeObservabilityExportProfileNone
	if exportEnabled {
		profile = runtimeconfig.RuntimeObservabilityExportProfileOTLP
	}
	content := fmt.Sprintf(`
mcp:
  active_profile: default
  profiles:
    default:
      call_timeout: 2s
      retry: 0
      backoff: 10ms
      queue_size: 16
      backpressure: block
      read_pool_size: 2
      write_pool_size: 1
runtime:
  observability:
    export:
      enabled: %t
      profile: %s
      endpoint: %q
      queue_capacity: 32
      on_error: degrade_and_record
  diagnostics:
    bundle:
      enabled: true
      output_dir: %q
      max_size_mb: 8
      include_sections:
        - timeline
        - diagnostics
        - effective_config
        - replay_hints
        - gate_fingerprint
`, exportEnabled, profile, endpoint, outputDir)
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("write runtime config failed: %v", err)
	}
}
