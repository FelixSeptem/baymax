package integration

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/integration/fakes"
	"github.com/FelixSeptem/baymax/observability/event"
	"github.com/FelixSeptem/baymax/orchestration/composer"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
	runtimediag "github.com/FelixSeptem/baymax/runtime/diagnostics"
)

func TestArbitrationVersionGovernanceContractRunStreamParitySupportedRequested(t *testing.T) {
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX_A50_TEST"})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	model := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	model.SetStream([]types.ModelEvent{{Type: types.ModelEventTypeFinalAnswer, TextDelta: "ok"}}, nil)
	dispatcher := event.NewDispatcher(event.NewRuntimeRecorder(mgr))
	comp, err := composer.NewBuilder(model).
		WithRuntimeManager(mgr).
		WithEventHandler(dispatcherHandler{dispatcher: dispatcher}).
		Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}

	if _, err := comp.Run(context.Background(), types.RunRequest{
		RunID:                  "run-a50-supported-run",
		Input:                  "run",
		ArbitrationRuleVersion: runtimeconfig.RuntimeArbitrationRuleVersionA48V1,
	}, nil); err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if _, err := comp.Stream(context.Background(), types.RunRequest{
		RunID:                  "run-a50-supported-stream",
		Input:                  "stream",
		ArbitrationRuleVersion: runtimeconfig.RuntimeArbitrationRuleVersionA48V1,
	}, nil); err != nil {
		t.Fatalf("stream failed: %v", err)
	}

	runRecord := findRunRecord(t, mgr.RecentRuns(20), "run-a50-supported-run")
	streamRecord := findRunRecord(t, mgr.RecentRuns(20), "run-a50-supported-stream")
	runSnapshot := arbitrationGovernanceSnapshotFromRunRecord(runRecord)
	streamSnapshot := arbitrationGovernanceSnapshotFromRunRecord(streamRecord)
	if !reflect.DeepEqual(runSnapshot, streamSnapshot) {
		t.Fatalf("run/stream governance parity mismatch run=%#v stream=%#v", runSnapshot, streamSnapshot)
	}
	if runSnapshot.RuleRequestedVersion != runtimeconfig.RuntimeArbitrationRuleVersionA48V1 ||
		runSnapshot.RuleEffectiveVersion != runtimeconfig.RuntimeArbitrationRuleVersionA48V1 ||
		runSnapshot.RuleVersionSource != runtimeconfig.RuntimeArbitrationVersionSourceRequested ||
		runSnapshot.RulePolicyAction != runtimeconfig.RuntimeArbitrationPolicyActionNone ||
		runSnapshot.RuleUnsupportedTotal != 0 ||
		runSnapshot.RuleMismatchTotal != 0 ||
		runSnapshot.RuleVersion != runtimeconfig.RuntimeArbitrationRuleVersionA48V1 {
		t.Fatalf("supported requested version governance mismatch: %#v", runSnapshot)
	}
}

func TestArbitrationVersionGovernanceContractRunStreamParityUnsupportedFailFast(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime-a50-admission.yaml")
	cfg := strings.Join([]string{
		"runtime:",
		"  readiness:",
		"    enabled: true",
		"    strict: false",
		"    remote_probe_enabled: false",
		"    admission:",
		"      enabled: true",
		"      mode: fail_fast",
		"      block_on: blocked_only",
		"      degraded_policy: allow_and_record",
		"  arbitration:",
		"    version:",
		"      enabled: true",
		"      default: a49.v1",
		"      compat_window: 1",
		"      on_unsupported: fail_fast",
		"      on_mismatch: fail_fast",
		"reload:",
		"  enabled: false",
		"  debounce: 20ms",
		"",
	}, "\n")
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
		FilePath:  cfgPath,
		EnvPrefix: "BAYMAX_A50_TEST",
	})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	model := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	model.SetStream([]types.ModelEvent{{Type: types.ModelEventTypeFinalAnswer, TextDelta: "ok"}}, nil)
	dispatcher := event.NewDispatcher(event.NewRuntimeRecorder(mgr))
	comp, err := composer.NewBuilder(model).
		WithRuntimeManager(mgr).
		WithEventHandler(dispatcherHandler{dispatcher: dispatcher}).
		Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}

	runRes, runErr := comp.Run(context.Background(), types.RunRequest{
		RunID:                  "run-a50-unsupported-run",
		Input:                  "run",
		ArbitrationRuleVersion: "a77.v9",
	}, nil)
	if runErr == nil {
		t.Fatal("run should fail-fast on unsupported arbitration version")
	}
	streamRes, streamErr := comp.Stream(context.Background(), types.RunRequest{
		RunID:                  "run-a50-unsupported-stream",
		Input:                  "stream",
		ArbitrationRuleVersion: "a77.v9",
	}, nil)
	if streamErr == nil {
		t.Fatal("stream should fail-fast on unsupported arbitration version")
	}
	assertA50AdmissionDenyDetails(t, runRes, runtimeconfig.ReadinessCodeArbitrationVersionUnsupported)
	assertA50AdmissionDenyDetails(t, streamRes, runtimeconfig.ReadinessCodeArbitrationVersionUnsupported)

	runRecord := findRunRecord(t, mgr.RecentRuns(20), "run-a50-unsupported-run")
	streamRecord := findRunRecord(t, mgr.RecentRuns(20), "run-a50-unsupported-stream")
	runSnapshot := arbitrationGovernanceSnapshotFromRunRecord(runRecord)
	streamSnapshot := arbitrationGovernanceSnapshotFromRunRecord(streamRecord)
	if !reflect.DeepEqual(runSnapshot, streamSnapshot) {
		t.Fatalf("run/stream unsupported governance parity mismatch run=%#v stream=%#v", runSnapshot, streamSnapshot)
	}
	if runSnapshot.PrimaryCode != runtimeconfig.ReadinessCodeArbitrationVersionUnsupported ||
		runSnapshot.PrimarySource != runtimeconfig.RuntimePrimarySourceArbitration ||
		runSnapshot.RuleRequestedVersion != "a77.v9" ||
		runSnapshot.RuleVersion != runtimeconfig.RuntimeArbitrationRuleVersionA49V1 ||
		runSnapshot.RuleEffectiveVersion != runtimeconfig.RuntimeArbitrationRuleVersionA49V1 ||
		runSnapshot.RuleVersionSource != runtimeconfig.RuntimeArbitrationVersionSourceRequested ||
		runSnapshot.RulePolicyAction != runtimeconfig.RuntimeArbitrationPolicyActionFailFastUnsupported ||
		runSnapshot.RuleUnsupportedTotal != 1 ||
		runSnapshot.RuleMismatchTotal != 0 {
		t.Fatalf("unsupported governance snapshot mismatch: %#v", runSnapshot)
	}
}

func TestArbitrationVersionGovernanceContractMemoryFileParity(t *testing.T) {
	runOnce := func(t *testing.T, backend string, runID string) runtimediag.RunRecord {
		t.Helper()
		cfgPath := filepath.Join(t.TempDir(), "runtime-"+backend+".yaml")
		cfg := strings.Join([]string{
			"runtime:",
			"  readiness:",
			"    enabled: true",
			"    strict: false",
			"    remote_probe_enabled: false",
			"scheduler:",
			"  enabled: true",
			"  backend: " + backend,
			"  path: " + filepath.ToSlash(filepath.Join(t.TempDir(), "scheduler-"+backend+".json")),
			"reload:",
			"  enabled: false",
			"  debounce: 20ms",
			"",
		}, "\n")
		if err := os.WriteFile(cfgPath, []byte(cfg), 0o600); err != nil {
			t.Fatalf("write config file: %v", err)
		}
		mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
			FilePath:  cfgPath,
			EnvPrefix: "BAYMAX_A50_TEST",
		})
		if err != nil {
			t.Fatalf("new runtime manager: %v", err)
		}
		t.Cleanup(func() { _ = mgr.Close() })
		model := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
		dispatcher := event.NewDispatcher(event.NewRuntimeRecorder(mgr))
		comp, err := composer.NewBuilder(model).
			WithRuntimeManager(mgr).
			WithEventHandler(dispatcherHandler{dispatcher: dispatcher}).
			Build()
		if err != nil {
			t.Fatalf("new composer: %v", err)
		}
		if _, err := comp.Run(context.Background(), types.RunRequest{
			RunID:                  runID,
			Input:                  "run",
			ArbitrationRuleVersion: runtimeconfig.RuntimeArbitrationRuleVersionA48V1,
		}, nil); err != nil {
			t.Fatalf("composer run failed: %v", err)
		}
		return findRunRecord(t, mgr.RecentRuns(20), runID)
	}

	memoryRecord := runOnce(t, "memory", "run-a50-memory")
	fileRecord := runOnce(t, "file", "run-a50-file")
	memorySnapshot := arbitrationGovernanceSnapshotFromRunRecord(memoryRecord)
	fileSnapshot := arbitrationGovernanceSnapshotFromRunRecord(fileRecord)
	if !reflect.DeepEqual(memorySnapshot, fileSnapshot) {
		t.Fatalf("memory/file governance parity mismatch memory=%#v file=%#v", memorySnapshot, fileSnapshot)
	}
}

type arbitrationGovernanceSnapshot struct {
	PrimaryCode          string
	PrimarySource        string
	RuleVersion          string
	RuleRequestedVersion string
	RuleEffectiveVersion string
	RuleVersionSource    string
	RulePolicyAction     string
	RuleUnsupportedTotal int
	RuleMismatchTotal    int
}

func arbitrationGovernanceSnapshotFromRunRecord(rec runtimediag.RunRecord) arbitrationGovernanceSnapshot {
	return arbitrationGovernanceSnapshot{
		PrimaryCode:          strings.TrimSpace(rec.RuntimePrimaryCode),
		PrimarySource:        strings.TrimSpace(rec.RuntimePrimarySource),
		RuleVersion:          strings.TrimSpace(rec.RuntimeArbitrationRuleVersion),
		RuleRequestedVersion: strings.TrimSpace(rec.RuntimeArbitrationRuleRequestedVersion),
		RuleEffectiveVersion: strings.TrimSpace(rec.RuntimeArbitrationRuleEffectiveVersion),
		RuleVersionSource:    strings.TrimSpace(rec.RuntimeArbitrationRuleVersionSource),
		RulePolicyAction:     strings.TrimSpace(rec.RuntimeArbitrationRulePolicyAction),
		RuleUnsupportedTotal: rec.RuntimeArbitrationRuleUnsupportedTotal,
		RuleMismatchTotal:    rec.RuntimeArbitrationRuleMismatchTotal,
	}
}

func assertA50AdmissionDenyDetails(t *testing.T, result types.RunResult, wantReasonCode string) {
	t.Helper()
	if result.Error == nil {
		t.Fatalf("run result missing classified error: %#v", result)
	}
	if result.Error.Class != types.ErrContext {
		t.Fatalf("error class = %q, want %q", result.Error.Class, types.ErrContext)
	}
	reasonCode, _ := result.Error.Details["reason_code"].(string)
	if strings.TrimSpace(reasonCode) != strings.TrimSpace(wantReasonCode) {
		t.Fatalf("reason_code=%q, want %q details=%#v", reasonCode, wantReasonCode, result.Error.Details)
	}
	requestedVersion, _ := result.Error.Details["readiness_arbitration_rule_requested_version"].(string)
	versionSource, _ := result.Error.Details["readiness_arbitration_rule_version_source"].(string)
	policyAction, _ := result.Error.Details["readiness_arbitration_rule_policy_action"].(string)
	if strings.TrimSpace(requestedVersion) == "" ||
		strings.TrimSpace(versionSource) != runtimeconfig.RuntimeArbitrationVersionSourceRequested ||
		strings.TrimSpace(policyAction) != runtimeconfig.RuntimeArbitrationPolicyActionFailFastUnsupported {
		t.Fatalf("deny details missing A50 governance explainability: %#v", result.Error.Details)
	}
}
