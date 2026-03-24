package integration

import (
	"context"
	"encoding/json"
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

func TestRuntimeReadinessContractDeterministicAndComposerParity(t *testing.T) {
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX_A40_TEST"})
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
		Scheduler: runtimeconfig.RuntimeReadinessComponentState{
			Enabled:           true,
			ConfiguredBackend: "file",
			EffectiveBackend:  "memory",
			Fallback:          true,
			FallbackReason:    "scheduler.backend.file_init_failed",
		},
	})

	before, err := comp.SchedulerStats(context.Background())
	if err != nil {
		t.Fatalf("SchedulerStats before readiness failed: %v", err)
	}
	runtimeFirst := mgr.ReadinessPreflight()
	runtimeSecond := mgr.ReadinessPreflight()
	composerResult, err := comp.ReadinessPreflight()
	if err != nil {
		t.Fatalf("composer readiness failed: %v", err)
	}
	after, err := comp.SchedulerStats(context.Background())
	if err != nil {
		t.Fatalf("SchedulerStats after readiness failed: %v", err)
	}

	if runtimeFirst.Status != runtimeconfig.ReadinessStatusDegraded {
		t.Fatalf("runtime readiness status = %q, want degraded", runtimeFirst.Status)
	}
	assertReadinessCode(t, runtimeFirst.Findings, runtimeconfig.ReadinessCodeSchedulerFallback)
	if readinessContractFingerprint(runtimeFirst) != readinessContractFingerprint(runtimeSecond) {
		t.Fatalf("runtime readiness changed across equivalent snapshots")
	}
	if readinessContractFingerprint(runtimeFirst) != readinessContractFingerprint(composerResult) {
		t.Fatalf("composer/runtime readiness parity mismatch")
	}
	if before.QueueTotal != after.QueueTotal || before.ClaimTotal != after.ClaimTotal || before.ReclaimTotal != after.ReclaimTotal {
		t.Fatalf("composer readiness query should not mutate scheduler state: before=%#v after=%#v", before, after)
	}
}

func TestRuntimeReadinessContractFallbackVisibilityMapsToDegraded(t *testing.T) {
	tmp := t.TempDir()
	blocked := filepath.Join(tmp, "backend-blocked")
	if err := os.WriteFile(blocked, []byte("x"), 0o600); err != nil {
		t.Fatalf("write blocked marker: %v", err)
	}
	cfgPath := filepath.Join(tmp, "runtime-a40.yaml")
	writeRuntimeReadinessFallbackConfig(
		t,
		cfgPath,
		filepath.ToSlash(filepath.Join(blocked, "scheduler-state.json")),
		filepath.ToSlash(filepath.Join(blocked, "mailbox-state.json")),
		filepath.ToSlash(filepath.Join(blocked, "recovery-state")),
	)

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
		FilePath:  cfgPath,
		EnvPrefix: "BAYMAX_A40_TEST",
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

	runtimeResult := mgr.ReadinessPreflight()
	composerResult, err := comp.ReadinessPreflight()
	if err != nil {
		t.Fatalf("composer readiness failed: %v", err)
	}

	if runtimeResult.Status != runtimeconfig.ReadinessStatusDegraded {
		t.Fatalf("runtime readiness status = %q, want degraded", runtimeResult.Status)
	}
	assertReadinessCode(t, runtimeResult.Findings, runtimeconfig.ReadinessCodeSchedulerFallback)
	assertReadinessCode(t, runtimeResult.Findings, runtimeconfig.ReadinessCodeMailboxFallback)
	assertReadinessCode(t, runtimeResult.Findings, runtimeconfig.ReadinessCodeRecoveryFallback)
	if readinessContractFingerprint(runtimeResult) != readinessContractFingerprint(composerResult) {
		t.Fatalf("composer/runtime readiness parity mismatch under fallback config")
	}
}

func assertReadinessCode(t *testing.T, findings []runtimeconfig.ReadinessFinding, code string) {
	t.Helper()
	for i := range findings {
		if strings.TrimSpace(findings[i].Code) == strings.TrimSpace(code) {
			return
		}
	}
	t.Fatalf("expected readiness code=%q, got findings=%#v", code, findings)
}

func readinessContractFingerprint(result runtimeconfig.ReadinessResult) string {
	payload := struct {
		Status   runtimeconfig.ReadinessStatus    `json:"status"`
		Findings []runtimeconfig.ReadinessFinding `json:"findings"`
	}{
		Status:   result.Status,
		Findings: result.Findings,
	}
	blob, _ := json.Marshal(payload)
	return string(blob)
}

func writeRuntimeReadinessFallbackConfig(t *testing.T, path, schedulerPath, mailboxPath, recoveryPath string) {
	t.Helper()
	cfg := strings.Join([]string{
		"runtime:",
		"  readiness:",
		"    enabled: true",
		"    strict: false",
		"    remote_probe_enabled: false",
		"scheduler:",
		"  enabled: true",
		"  backend: file",
		"  path: " + schedulerPath,
		"  lease_timeout: 2s",
		"  heartbeat_interval: 500ms",
		"  queue_limit: 64",
		"  retry_max_attempts: 3",
		"mailbox:",
		"  enabled: true",
		"  backend: file",
		"  path: " + mailboxPath,
		"recovery:",
		"  enabled: true",
		"  backend: file",
		"  path: " + recoveryPath,
		"reload:",
		"  enabled: false",
		"  debounce: 20ms",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		t.Fatalf("write runtime config: %v", err)
	}
}
