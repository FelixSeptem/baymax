package integration

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/integration/fakes"
	"github.com/FelixSeptem/baymax/observability/event"
	"github.com/FelixSeptem/baymax/orchestration/composer"
	"github.com/FelixSeptem/baymax/orchestration/scheduler"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
	runtimediag "github.com/FelixSeptem/baymax/runtime/diagnostics"
)

func TestTimeoutResolutionContractValidationPrecedenceClampAndTaskBoard(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime-a41-memory.yaml")
	writeTimeoutResolutionContractConfig(t, cfgPath, "memory", "")

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
		FilePath:  cfgPath,
		EnvPrefix: "BAYMAX_A41_TEST",
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

	runID := "run-a41-contract-validation"
	_, err = comp.SpawnChild(context.Background(), composer.ChildDispatchRequest{
		Task: scheduler.Task{
			TaskID: "task-a41-invalid-profile",
			RunID:  runID,
		},
		OperationProfile: "realtime",
	})
	if err == nil {
		t.Fatal("unsupported operation profile should fail fast")
	}

	domainRecord, err := comp.SpawnChild(context.Background(), composer.ChildDispatchRequest{
		Task: scheduler.Task{
			TaskID: "task-a41-domain",
			RunID:  runID,
		},
		OperationProfile:      runtimeconfig.OperationProfileInteractive,
		ParentRemainingBudget: 8 * time.Second,
	})
	if err != nil {
		t.Fatalf("spawn with domain override failed: %v", err)
	}
	if domainRecord.Task.TimeoutResolution.Source != runtimeconfig.TimeoutResolutionSourceDomain {
		t.Fatalf("domain override source = %q, want %q", domainRecord.Task.TimeoutResolution.Source, runtimeconfig.TimeoutResolutionSourceDomain)
	}
	if domainRecord.Task.TimeoutResolution.ResolvedTimeout != 6*time.Second {
		t.Fatalf("domain override resolved timeout = %s, want 6s", domainRecord.Task.TimeoutResolution.ResolvedTimeout)
	}
	if domainRecord.Task.TimeoutResolution.ParentBudgetClamped {
		t.Fatalf("domain override should not be clamped: %#v", domainRecord.Task.TimeoutResolution)
	}

	requestRecord, err := comp.SpawnChild(context.Background(), composer.ChildDispatchRequest{
		Task: scheduler.Task{
			TaskID: "task-a41-request",
			RunID:  runID,
		},
		OperationProfile:      runtimeconfig.OperationProfileInteractive,
		RequestTimeout:        1500 * time.Millisecond,
		ParentRemainingBudget: 1200 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("spawn with request override failed: %v", err)
	}
	if requestRecord.Task.TimeoutResolution.Source != runtimeconfig.TimeoutResolutionSourceRequest {
		t.Fatalf("request override source = %q, want %q", requestRecord.Task.TimeoutResolution.Source, runtimeconfig.TimeoutResolutionSourceRequest)
	}
	if requestRecord.Task.TimeoutResolution.ResolvedTimeout != 1200*time.Millisecond {
		t.Fatalf("request override resolved timeout = %s, want 1200ms", requestRecord.Task.TimeoutResolution.ResolvedTimeout)
	}
	if !requestRecord.Task.TimeoutResolution.ParentBudgetClamped {
		t.Fatalf("request override should be clamped: %#v", requestRecord.Task.TimeoutResolution)
	}

	taskPage, err := comp.Scheduler().QueryTasks(context.Background(), scheduler.TaskBoardQueryRequest{TaskID: "task-a41-request"})
	if err != nil {
		t.Fatalf("task board query failed: %v", err)
	}
	if len(taskPage.Items) != 1 {
		t.Fatalf("task board items len = %d, want 1", len(taskPage.Items))
	}
	tbMeta := taskPage.Items[0].Task.TimeoutResolution
	if tbMeta.Source != runtimeconfig.TimeoutResolutionSourceRequest ||
		tbMeta.ResolvedTimeout != 1200*time.Millisecond ||
		!tbMeta.ParentBudgetClamped {
		t.Fatalf("task board timeout metadata mismatch: %#v", tbMeta)
	}

	if _, err := comp.Run(context.Background(), types.RunRequest{RunID: runID, Input: "emit-finished"}, nil); err != nil {
		t.Fatalf("composer run failed: %v", err)
	}
	run := findRunRecord(t, mgr.RecentRuns(10), runID)
	if run.EffectiveOperationProfile != runtimeconfig.OperationProfileInteractive ||
		run.TimeoutResolutionSource != runtimeconfig.TimeoutResolutionSourceRequest ||
		strings.TrimSpace(run.TimeoutResolutionTrace) == "" ||
		run.TimeoutParentBudgetClampTotal != 1 ||
		run.TimeoutParentBudgetRejectTotal != 0 {
		t.Fatalf("run summary timeout fields mismatch: %#v", run)
	}

	runPage, err := mgr.QueryRuns(runtimediag.UnifiedRunQueryRequest{RunID: runID})
	if err != nil {
		t.Fatalf("query runs failed: %v", err)
	}
	if len(runPage.Items) != 1 {
		t.Fatalf("query runs items len = %d, want 1", len(runPage.Items))
	}
	if runPage.Items[0].TimeoutResolutionSource != runtimeconfig.TimeoutResolutionSourceRequest ||
		runPage.Items[0].TimeoutParentBudgetClampTotal != 1 {
		t.Fatalf("query runs timeout summary mismatch: %#v", runPage.Items[0])
	}
}

func TestTimeoutResolutionContractParentBudgetExhaustedRejectClassification(t *testing.T) {
	s, err := scheduler.New(scheduler.NewMemoryStore(), scheduler.WithLeaseTimeout(500*time.Millisecond))
	if err != nil {
		t.Fatalf("new scheduler: %v", err)
	}
	_, err = s.SpawnChild(context.Background(), scheduler.SpawnRequest{
		Task:                  scheduler.Task{TaskID: "task-a41-parent-exhausted", RunID: "run-a41-parent-exhausted"},
		ParentDepth:           0,
		ParentActiveChildren:  0,
		ParentRemainingBudget: 0,
		ChildTimeout:          time.Second,
	})
	if err == nil {
		t.Fatal("expected parent budget exhausted reject")
	}
	var budgetErr *scheduler.BudgetError
	if !errors.As(err, &budgetErr) {
		t.Fatalf("budget reject type mismatch: %v", err)
	}
	if budgetErr.Code != scheduler.BudgetRejectParentBudgetExhausted {
		t.Fatalf("budget reject code = %q, want %q", budgetErr.Code, scheduler.BudgetRejectParentBudgetExhausted)
	}
}

func TestTimeoutResolutionContractRunStreamAndMemoryFileParity(t *testing.T) {
	memRun, err := executeTimeoutResolutionContract(t, false, "memory")
	if err != nil {
		t.Fatalf("memory run path failed: %v", err)
	}
	memStream, err := executeTimeoutResolutionContract(t, true, "memory")
	if err != nil {
		t.Fatalf("memory stream path failed: %v", err)
	}
	fileRun, err := executeTimeoutResolutionContract(t, false, "file")
	if err != nil {
		t.Fatalf("file run path failed: %v", err)
	}
	fileStream, err := executeTimeoutResolutionContract(t, true, "file")
	if err != nil {
		t.Fatalf("file stream path failed: %v", err)
	}

	assertTimeoutResolutionContractSummaryEqual(t, memRun, memStream, "memory run/stream")
	assertTimeoutResolutionContractSummaryEqual(t, fileRun, fileStream, "file run/stream")
	assertTimeoutResolutionContractSummaryEqual(t, memRun, fileRun, "memory/file parity")
}

func TestTimeoutResolutionContractReplayIdempotency(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime-a41-replay.yaml")
	writeTimeoutResolutionContractConfig(t, cfgPath, "memory", "")

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
		FilePath:  cfgPath,
		EnvPrefix: "BAYMAX_A41_TEST",
	})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	model := fakes.NewModel([]fakes.ModelStep{
		{Response: types.ModelResponse{FinalAnswer: "ok-1"}},
		{Response: types.ModelResponse{FinalAnswer: "ok-2"}},
	})
	model.SetStream([]types.ModelEvent{{Type: types.ModelEventTypeFinalAnswer, TextDelta: "ok"}}, nil)
	dispatcher := event.NewDispatcher(event.NewRuntimeRecorder(mgr))
	comp, err := composer.NewBuilder(model).
		WithRuntimeManager(mgr).
		WithEventHandler(dispatcherHandler{dispatcher: dispatcher}).
		Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}

	runID := "run-a41-replay"
	if _, err := comp.SpawnChild(context.Background(), composer.ChildDispatchRequest{
		Task: scheduler.Task{
			TaskID: "task-a41-replay",
			RunID:  runID,
		},
		OperationProfile:      runtimeconfig.OperationProfileInteractive,
		RequestTimeout:        1400 * time.Millisecond,
		ParentRemainingBudget: 1100 * time.Millisecond,
	}); err != nil {
		t.Fatalf("spawn child failed: %v", err)
	}

	if _, err := comp.Run(context.Background(), types.RunRequest{RunID: runID, Input: "emit-finished-1"}, nil); err != nil {
		t.Fatalf("run #1 failed: %v", err)
	}
	if _, err := comp.Run(context.Background(), types.RunRequest{RunID: runID, Input: "emit-finished-2"}, nil); err != nil {
		t.Fatalf("run #2 failed: %v", err)
	}

	page, err := mgr.QueryRuns(runtimediag.UnifiedRunQueryRequest{RunID: runID})
	if err != nil {
		t.Fatalf("query runs failed: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("query items len = %d, want 1", len(page.Items))
	}
	got := page.Items[0]
	if got.TimeoutParentBudgetClampTotal != 1 ||
		got.TimeoutParentBudgetRejectTotal != 0 ||
		got.TimeoutResolutionSource != runtimeconfig.TimeoutResolutionSourceRequest {
		t.Fatalf("replay-idempotent timeout counters mismatch: %#v", got)
	}
}

type timeoutResolutionContractSummary struct {
	status      string
	profile     string
	source      string
	clampTotal  int
	rejectTotal int
	taskSource  string
	taskTimeout time.Duration
	taskClamped bool
}

func executeTimeoutResolutionContract(
	t *testing.T,
	stream bool,
	backend string,
) (timeoutResolutionContractSummary, error) {
	t.Helper()

	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "runtime-a41-"+backend+".yaml")
	backendPath := ""
	if backend == "file" {
		backendPath = filepath.ToSlash(filepath.Join(tmp, "scheduler-state.json"))
	}
	writeTimeoutResolutionContractConfig(t, cfgPath, backend, backendPath)

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
		FilePath:  cfgPath,
		EnvPrefix: "BAYMAX_A41_TEST",
	})
	if err != nil {
		return timeoutResolutionContractSummary{}, err
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
		return timeoutResolutionContractSummary{}, err
	}

	runID := "run-a41-" + backend
	taskID := "task-a41-" + backend
	if stream {
		runID += "-stream"
		taskID += "-stream"
	} else {
		runID += "-run"
		taskID += "-run"
	}

	if _, err := comp.SpawnChild(context.Background(), composer.ChildDispatchRequest{
		Task: scheduler.Task{
			TaskID: taskID,
			RunID:  runID,
		},
		OperationProfile:      runtimeconfig.OperationProfileInteractive,
		RequestTimeout:        1500 * time.Millisecond,
		ParentRemainingBudget: 1200 * time.Millisecond,
	}); err != nil {
		return timeoutResolutionContractSummary{}, err
	}

	req := types.RunRequest{RunID: runID, Input: "emit-finished"}
	if stream {
		if _, err := comp.Stream(context.Background(), req, nil); err != nil {
			return timeoutResolutionContractSummary{}, err
		}
	} else {
		if _, err := comp.Run(context.Background(), req, nil); err != nil {
			return timeoutResolutionContractSummary{}, err
		}
	}

	run := findRunRecord(t, mgr.RecentRuns(10), runID)
	page, err := comp.Scheduler().QueryTasks(context.Background(), scheduler.TaskBoardQueryRequest{TaskID: taskID})
	if err != nil {
		return timeoutResolutionContractSummary{}, err
	}
	if len(page.Items) != 1 {
		return timeoutResolutionContractSummary{}, errors.New("task board query should return exactly one item")
	}
	meta := page.Items[0].Task.TimeoutResolution
	return timeoutResolutionContractSummary{
		status:      run.Status,
		profile:     run.EffectiveOperationProfile,
		source:      run.TimeoutResolutionSource,
		clampTotal:  run.TimeoutParentBudgetClampTotal,
		rejectTotal: run.TimeoutParentBudgetRejectTotal,
		taskSource:  meta.Source,
		taskTimeout: meta.ResolvedTimeout,
		taskClamped: meta.ParentBudgetClamped,
	}, nil
}

func assertTimeoutResolutionContractSummaryEqual(
	t *testing.T,
	left timeoutResolutionContractSummary,
	right timeoutResolutionContractSummary,
	label string,
) {
	t.Helper()
	if left.status != right.status {
		t.Fatalf("%s status mismatch: left=%#v right=%#v", label, left, right)
	}
	if left.profile != right.profile ||
		left.source != right.source ||
		left.clampTotal != right.clampTotal ||
		left.rejectTotal != right.rejectTotal {
		t.Fatalf("%s timeout summary mismatch: left=%#v right=%#v", label, left, right)
	}
	if left.taskSource != right.taskSource ||
		left.taskTimeout != right.taskTimeout ||
		left.taskClamped != right.taskClamped {
		t.Fatalf("%s task board timeout metadata mismatch: left=%#v right=%#v", label, left, right)
	}
}

func writeTimeoutResolutionContractConfig(t *testing.T, path, schedulerBackend, schedulerPath string) {
	t.Helper()
	if strings.TrimSpace(schedulerBackend) == "" {
		schedulerBackend = "memory"
	}
	if strings.TrimSpace(schedulerPath) == "" {
		schedulerPath = filepath.ToSlash(filepath.Join(filepath.Dir(path), "scheduler-state.json"))
	}
	content := strings.Join([]string{
		"reload:",
		"  enabled: false",
		"scheduler:",
		"  enabled: true",
		"  backend: " + schedulerBackend,
		"  path: " + schedulerPath,
		"  lease_timeout: 500ms",
		"  heartbeat_interval: 100ms",
		"  queue_limit: 64",
		"  retry_max_attempts: 3",
		"subagent:",
		"  max_depth: 4",
		"  max_active_children: 8",
		"  child_timeout_budget: 6s",
		"runtime:",
		"  operation_profiles:",
		"    default_profile: legacy",
		"    legacy:",
		"      timeout: 3s",
		"    interactive:",
		"      timeout: 9s",
		"    background:",
		"      timeout: 30s",
		"    batch:",
		"      timeout: 2m",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write timeout resolution config failed: %v", err)
	}
}
