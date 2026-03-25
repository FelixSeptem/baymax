package integration

import (
	"context"
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

func TestRuntimeReadinessAdmissionContractBlockedDenyRunStreamEquivalentAndNoSideEffects(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime-a44-blocked.yaml")
	writeRuntimeReadinessAdmissionConfig(t, cfgPath, true, runtimeconfig.ReadinessAdmissionDegradedPolicyAllowAndRecord)

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
		FilePath:  cfgPath,
		EnvPrefix: "BAYMAX_A44_TEST",
	})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	model := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	dispatcher := event.NewDispatcher(event.NewRuntimeRecorder(mgr))
	comp, err := composer.NewBuilder(model).
		WithRuntimeManager(mgr).
		WithEventHandler(dispatcherHandler{dispatcher: dispatcher}).
		Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}
	mgr.SetReadinessComponentSnapshot(runtimeconfig.RuntimeReadinessComponentSnapshot{
		Recovery: runtimeconfig.RuntimeReadinessComponentState{
			Enabled:           true,
			ConfiguredBackend: "file",
			EffectiveBackend:  "memory",
			ActivationError:   "permission denied",
		},
	})

	before, err := comp.SchedulerStats(context.Background())
	if err != nil {
		t.Fatalf("scheduler stats before deny failed: %v", err)
	}
	mailboxBefore := len(mgr.RecentMailbox(10))

	runRes, runErr := comp.Run(context.Background(), types.RunRequest{
		RunID: "run-a44-integration-blocked-run",
		Input: "blocked-run",
	}, nil)
	if runErr == nil {
		t.Fatal("run should be denied by readiness admission")
	}
	assertAdmissionContractDeniedResult(t, runRes, runtimeconfig.ReadinessAdmissionCodeBlocked)

	streamRes, streamErr := comp.Stream(context.Background(), types.RunRequest{
		RunID: "run-a44-integration-blocked-stream",
		Input: "blocked-stream",
	}, nil)
	if streamErr == nil {
		t.Fatal("stream should be denied by readiness admission")
	}
	assertAdmissionContractDeniedResult(t, streamRes, runtimeconfig.ReadinessAdmissionCodeBlocked)

	after, err := comp.SchedulerStats(context.Background())
	if err != nil {
		t.Fatalf("scheduler stats after deny failed: %v", err)
	}
	if before.QueueTotal != after.QueueTotal || before.ClaimTotal != after.ClaimTotal || before.ReclaimTotal != after.ReclaimTotal {
		t.Fatalf("deny path should be side-effect free, before=%#v after=%#v", before, after)
	}
	if len(mgr.RecentMailbox(10)) != mailboxBefore {
		t.Fatalf("deny path should not mutate mailbox diagnostics: before=%d after=%d", mailboxBefore, len(mgr.RecentMailbox(10)))
	}

	assertAdmissionRunRecord(t, mgr, "run-a44-integration-blocked-run", func(rec mapRunRecord) {
		if rec.RuntimeReadinessAdmissionTotal != 1 ||
			rec.RuntimeReadinessAdmissionBlockedTotal != 1 ||
			rec.RuntimeReadinessAdmissionDegradedAllowTotal != 0 ||
			rec.RuntimeReadinessAdmissionBypassTotal != 0 ||
			rec.RuntimeReadinessAdmissionMode != runtimeconfig.ReadinessAdmissionModeFailFast ||
			rec.RuntimeReadinessAdmissionPrimaryCode != runtimeconfig.ReadinessCodeRecoveryActivationError {
			t.Fatalf("blocked admission run record mismatch: %#v", rec)
		}
	})
	assertAdmissionRunRecord(t, mgr, "run-a44-integration-blocked-stream", func(rec mapRunRecord) {
		if rec.RuntimeReadinessAdmissionTotal != 1 ||
			rec.RuntimeReadinessAdmissionBlockedTotal != 1 ||
			rec.RuntimeReadinessAdmissionDegradedAllowTotal != 0 ||
			rec.RuntimeReadinessAdmissionBypassTotal != 0 ||
			rec.RuntimeReadinessAdmissionMode != runtimeconfig.ReadinessAdmissionModeFailFast ||
			rec.RuntimeReadinessAdmissionPrimaryCode != runtimeconfig.ReadinessCodeRecoveryActivationError {
			t.Fatalf("blocked stream admission run record mismatch: %#v", rec)
		}
	})
}

func TestRuntimeReadinessAdmissionContractDegradedPolicyMappingAndBypassCompatibility(t *testing.T) {
	allowCfg := filepath.Join(t.TempDir(), "runtime-a44-degraded-allow.yaml")
	writeRuntimeReadinessAdmissionConfig(t, allowCfg, true, runtimeconfig.ReadinessAdmissionDegradedPolicyAllowAndRecord)
	allowMgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
		FilePath:  allowCfg,
		EnvPrefix: "BAYMAX_A44_TEST",
	})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	defer func() { _ = allowMgr.Close() }()
	allowModel := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	allowDispatcher := event.NewDispatcher(event.NewRuntimeRecorder(allowMgr))
	allowComp, err := composer.NewBuilder(allowModel).
		WithRuntimeManager(allowMgr).
		WithEventHandler(dispatcherHandler{dispatcher: allowDispatcher}).
		Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}
	allowMgr.SetReadinessComponentSnapshot(runtimeconfig.RuntimeReadinessComponentSnapshot{
		Scheduler: runtimeconfig.RuntimeReadinessComponentState{
			Enabled:           true,
			ConfiguredBackend: "file",
			EffectiveBackend:  "memory",
			Fallback:          true,
			FallbackReason:    "scheduler.backend.file_init_failed",
		},
	})
	if runRes, runErr := allowComp.Run(context.Background(), types.RunRequest{
		RunID: "run-a44-integration-degraded-allow-run",
		Input: "allow-run",
	}, nil); runErr != nil || runRes.Error != nil {
		t.Fatalf("degraded allow run should succeed, err=%v result=%#v", runErr, runRes)
	}
	if streamRes, streamErr := allowComp.Stream(context.Background(), types.RunRequest{
		RunID: "run-a44-integration-degraded-allow-stream",
		Input: "allow-stream",
	}, nil); streamErr != nil || streamRes.Error != nil {
		t.Fatalf("degraded allow stream should succeed, err=%v result=%#v", streamErr, streamRes)
	}
	assertAdmissionRunRecord(t, allowMgr, "run-a44-integration-degraded-allow-run", func(rec mapRunRecord) {
		if rec.RuntimeReadinessAdmissionTotal != 1 ||
			rec.RuntimeReadinessAdmissionBlockedTotal != 0 ||
			rec.RuntimeReadinessAdmissionDegradedAllowTotal != 1 ||
			rec.RuntimeReadinessAdmissionBypassTotal != 0 ||
			rec.RuntimeReadinessAdmissionPrimaryCode != runtimeconfig.ReadinessCodeSchedulerFallback {
			t.Fatalf("degraded allow run record mismatch: %#v", rec)
		}
	})
	assertAdmissionRunRecord(t, allowMgr, "run-a44-integration-degraded-allow-stream", func(rec mapRunRecord) {
		if rec.RuntimeReadinessAdmissionTotal != 1 ||
			rec.RuntimeReadinessAdmissionBlockedTotal != 0 ||
			rec.RuntimeReadinessAdmissionDegradedAllowTotal != 1 ||
			rec.RuntimeReadinessAdmissionBypassTotal != 0 ||
			rec.RuntimeReadinessAdmissionPrimaryCode != runtimeconfig.ReadinessCodeSchedulerFallback {
			t.Fatalf("degraded allow stream run record mismatch: %#v", rec)
		}
	})

	bypassCfg := filepath.Join(t.TempDir(), "runtime-a44-bypass.yaml")
	writeRuntimeReadinessAdmissionConfig(t, bypassCfg, false, runtimeconfig.ReadinessAdmissionDegradedPolicyAllowAndRecord)
	bypassMgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
		FilePath:  bypassCfg,
		EnvPrefix: "BAYMAX_A44_TEST",
	})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	defer func() { _ = bypassMgr.Close() }()
	bypassModel := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	bypassDispatcher := event.NewDispatcher(event.NewRuntimeRecorder(bypassMgr))
	bypassComp, err := composer.NewBuilder(bypassModel).
		WithRuntimeManager(bypassMgr).
		WithEventHandler(dispatcherHandler{dispatcher: bypassDispatcher}).
		Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}
	if runRes, runErr := bypassComp.Run(context.Background(), types.RunRequest{
		RunID: "run-a44-integration-bypass-run",
		Input: "bypass-run",
	}, nil); runErr != nil || runRes.Error != nil {
		t.Fatalf("bypass run should succeed, err=%v result=%#v", runErr, runRes)
	}
	assertAdmissionRunRecord(t, bypassMgr, "run-a44-integration-bypass-run", func(rec mapRunRecord) {
		if rec.RuntimeReadinessAdmissionTotal != 0 ||
			rec.RuntimeReadinessAdmissionBlockedTotal != 0 ||
			rec.RuntimeReadinessAdmissionDegradedAllowTotal != 0 ||
			rec.RuntimeReadinessAdmissionBypassTotal != 1 ||
			rec.RuntimeReadinessAdmissionMode != runtimeconfig.ReadinessAdmissionModeFailFast {
			t.Fatalf("bypass run record mismatch: %#v", rec)
		}
	})
}

type mapRunRecord struct {
	RunID                                       string
	RuntimeReadinessAdmissionTotal              int
	RuntimeReadinessAdmissionBlockedTotal       int
	RuntimeReadinessAdmissionDegradedAllowTotal int
	RuntimeReadinessAdmissionBypassTotal        int
	RuntimeReadinessAdmissionMode               string
	RuntimeReadinessAdmissionPrimaryCode        string
}

func assertAdmissionRunRecord(t *testing.T, mgr *runtimeconfig.Manager, runID string, assertFn func(rec mapRunRecord)) {
	t.Helper()
	items := mgr.RecentRuns(20)
	for i := range items {
		if strings.TrimSpace(items[i].RunID) != strings.TrimSpace(runID) {
			continue
		}
		rec := mapRunRecord{
			RunID:                                       items[i].RunID,
			RuntimeReadinessAdmissionTotal:              items[i].RuntimeReadinessAdmissionTotal,
			RuntimeReadinessAdmissionBlockedTotal:       items[i].RuntimeReadinessAdmissionBlockedTotal,
			RuntimeReadinessAdmissionDegradedAllowTotal: items[i].RuntimeReadinessAdmissionDegradedAllowTotal,
			RuntimeReadinessAdmissionBypassTotal:        items[i].RuntimeReadinessAdmissionBypassTotal,
			RuntimeReadinessAdmissionMode:               items[i].RuntimeReadinessAdmissionMode,
			RuntimeReadinessAdmissionPrimaryCode:        items[i].RuntimeReadinessAdmissionPrimaryCode,
		}
		assertFn(rec)
		return
	}
	t.Fatalf("run record %q not found in %#v", runID, items)
}

func assertAdmissionContractDeniedResult(t *testing.T, result types.RunResult, wantReasonCode string) {
	t.Helper()
	if result.Error == nil {
		t.Fatalf("run result missing classified error: %#v", result)
	}
	if result.Error.Class != types.ErrContext {
		t.Fatalf("error class = %q, want %q", result.Error.Class, types.ErrContext)
	}
	gotReasonCode, _ := result.Error.Details["reason_code"].(string)
	if strings.TrimSpace(gotReasonCode) != strings.TrimSpace(wantReasonCode) {
		t.Fatalf("reason_code = %q, want %q, details=%#v", gotReasonCode, wantReasonCode, result.Error.Details)
	}
}

func writeRuntimeReadinessAdmissionConfig(t *testing.T, path string, enabled bool, degradedPolicy string) {
	t.Helper()
	toggle := "false"
	if enabled {
		toggle = "true"
	}
	cfg := strings.Join([]string{
		"runtime:",
		"  readiness:",
		"    enabled: true",
		"    strict: false",
		"    remote_probe_enabled: false",
		"    admission:",
		"      enabled: " + toggle,
		"      mode: fail_fast",
		"      block_on: blocked_only",
		"      degraded_policy: " + strings.TrimSpace(degradedPolicy),
		"reload:",
		"  enabled: false",
		"  debounce: 20ms",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		t.Fatalf("write runtime config %q: %v", path, err)
	}
}
