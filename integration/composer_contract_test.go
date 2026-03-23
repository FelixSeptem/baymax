package integration

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/a2a"
	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/integration/fakes"
	"github.com/FelixSeptem/baymax/observability/event"
	"github.com/FelixSeptem/baymax/orchestration/composer"
	"github.com/FelixSeptem/baymax/orchestration/scheduler"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
	runtimediag "github.com/FelixSeptem/baymax/runtime/diagnostics"
)

type composerEventCollector struct {
	events []types.Event
}

func (c *composerEventCollector) OnEvent(_ context.Context, ev types.Event) {
	c.events = append(c.events, ev)
}

type dispatcherHandler struct {
	dispatcher *event.Dispatcher
}

func (h dispatcherHandler) OnEvent(ctx context.Context, ev types.Event) {
	if h.dispatcher == nil {
		return
	}
	h.dispatcher.Emit(ctx, ev)
}

type composerMailboxA2AClient struct{}

func (composerMailboxA2AClient) Submit(_ context.Context, req a2a.TaskRequest) (a2a.TaskRecord, error) {
	return a2a.TaskRecord{
		TaskID:     req.TaskID,
		WorkflowID: req.WorkflowID,
		TeamID:     req.TeamID,
		StepID:     req.StepID,
		AttemptID:  req.AttemptID,
		AgentID:    req.AgentID,
		PeerID:     req.PeerID,
		Status:     a2a.StatusSubmitted,
		UpdatedAt:  time.Now(),
	}, nil
}

func (composerMailboxA2AClient) WaitResult(
	_ context.Context,
	taskID string,
	_ time.Duration,
	_ func(context.Context, a2a.TaskRecord) error,
) (a2a.TaskRecord, error) {
	return a2a.TaskRecord{
		TaskID:    taskID,
		Status:    a2a.StatusSucceeded,
		Result:    map[string]any{"ok": true},
		UpdatedAt: time.Now(),
	}, nil
}

func TestComposerContractRunStreamSemanticEquivalence(t *testing.T) {
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX_A8_TEST"})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	model := fakes.NewModel([]fakes.ModelStep{
		{Response: types.ModelResponse{FinalAnswer: "composer-run"}},
	})
	model.SetStream([]types.ModelEvent{
		{Type: types.ModelEventTypeFinalAnswer, TextDelta: "composer-stream"},
	}, nil)

	collector := &composerEventCollector{}
	dispatcher := event.NewDispatcher(
		event.NewRuntimeRecorder(mgr),
		collector,
	)
	comp, err := composer.NewBuilder(model).
		WithRuntimeManager(mgr).
		WithEventHandler(dispatcherHandler{dispatcher: dispatcher}).
		Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}

	runReq := types.RunRequest{RunID: "run-a8-equivalence-run", Input: "ping-run"}
	streamReq := types.RunRequest{RunID: "run-a8-equivalence-stream", Input: "ping-stream"}
	if _, err := comp.Run(context.Background(), runReq, nil); err != nil {
		t.Fatalf("composer run failed: %v", err)
	}
	if _, err := comp.Stream(context.Background(), streamReq, nil); err != nil {
		t.Fatalf("composer stream failed: %v", err)
	}

	runRecord := findRunRecord(t, mgr.RecentRuns(10), runReq.RunID)
	streamRecord := findRunRecord(t, mgr.RecentRuns(10), streamReq.RunID)
	if runRecord.Status != streamRecord.Status {
		t.Fatalf("status mismatch run=%q stream=%q", runRecord.Status, streamRecord.Status)
	}
	if !runRecord.ComposerManaged || !streamRecord.ComposerManaged {
		t.Fatalf("composer marker mismatch run=%v stream=%v", runRecord.ComposerManaged, streamRecord.ComposerManaged)
	}
	if runRecord.SchedulerBackend != streamRecord.SchedulerBackend {
		t.Fatalf("scheduler backend mismatch run=%q stream=%q", runRecord.SchedulerBackend, streamRecord.SchedulerBackend)
	}
	if runRecord.SchedulerQueueTotal != streamRecord.SchedulerQueueTotal ||
		runRecord.SchedulerClaimTotal != streamRecord.SchedulerClaimTotal ||
		runRecord.SchedulerReclaimTotal != streamRecord.SchedulerReclaimTotal {
		t.Fatalf("scheduler aggregate mismatch run=%#v stream=%#v", runRecord, streamRecord)
	}
	if runRecord.SubagentChildTotal != streamRecord.SubagentChildTotal ||
		runRecord.SubagentChildFailed != streamRecord.SubagentChildFailed ||
		runRecord.SubagentBudgetRejectTotal != streamRecord.SubagentBudgetRejectTotal {
		t.Fatalf("subagent aggregate mismatch run=%#v stream=%#v", runRecord, streamRecord)
	}

	runFinishedCount := 0
	for _, ev := range collector.events {
		if ev.Type != "run.finished" {
			continue
		}
		runFinishedCount++
		if _, ok := ev.Payload["composer_managed"]; !ok {
			t.Fatalf("run.finished missing composer marker: %#v", ev.Payload)
		}
	}
	if runFinishedCount < 2 {
		t.Fatalf("run.finished count=%d, want >=2", runFinishedCount)
	}
}

func TestComposerContractDelayedChildRunStreamEquivalence(t *testing.T) {
	exec := func(stream bool, runID, taskID string) (runtimediag.RunRecord, error) {
		mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX_A13_TEST"})
		if err != nil {
			return runtimediag.RunRecord{}, err
		}
		defer func() { _ = mgr.Close() }()

		model := fakes.NewModel([]fakes.ModelStep{
			{Response: types.ModelResponse{FinalAnswer: "ok"}},
		})
		model.SetStream([]types.ModelEvent{
			{Type: types.ModelEventTypeFinalAnswer, TextDelta: "ok"},
		}, nil)

		dispatcher := event.NewDispatcher(event.NewRuntimeRecorder(mgr))
		comp, err := composer.NewBuilder(model).
			WithRuntimeManager(mgr).
			WithEventHandler(dispatcherHandler{dispatcher: dispatcher}).
			Build()
		if err != nil {
			return runtimediag.RunRecord{}, err
		}

		notBefore := time.Now().Add(90 * time.Millisecond)
		if _, err := comp.SpawnChild(context.Background(), composer.ChildDispatchRequest{
			Task: scheduler.Task{
				TaskID:    taskID,
				RunID:     runID,
				NotBefore: notBefore,
			},
		}); err != nil {
			return runtimediag.RunRecord{}, err
		}
		if _, ok, err := comp.Scheduler().Claim(context.Background(), "worker-delayed-a13"); err != nil {
			return runtimediag.RunRecord{}, err
		} else if ok {
			return runtimediag.RunRecord{}, errors.New("task claimed before not_before boundary")
		}
		time.Sleep(110 * time.Millisecond)
		claimed, ok, err := comp.Scheduler().Claim(context.Background(), "worker-delayed-a13")
		if err != nil || !ok {
			if err != nil {
				return runtimediag.RunRecord{}, err
			}
			return runtimediag.RunRecord{}, errors.New("task not claimable after not_before boundary")
		}
		if _, err := comp.CommitChildTerminal(context.Background(), scheduler.TerminalCommit{
			TaskID:      claimed.Record.Task.TaskID,
			AttemptID:   claimed.Attempt.AttemptID,
			Status:      scheduler.TaskStateSucceeded,
			Result:      map[string]any{"ok": true},
			CommittedAt: time.Now(),
		}); err != nil {
			return runtimediag.RunRecord{}, err
		}

		req := types.RunRequest{RunID: runID, Input: "emit-finished"}
		if stream {
			if _, err := comp.Stream(context.Background(), req, nil); err != nil {
				return runtimediag.RunRecord{}, err
			}
		} else {
			if _, err := comp.Run(context.Background(), req, nil); err != nil {
				return runtimediag.RunRecord{}, err
			}
		}
		return findRunRecord(t, mgr.RecentRuns(10), runID), nil
	}

	runRecord, err := exec(false, "run-a13-delayed-run", "task-a13-delayed-run")
	if err != nil {
		t.Fatalf("run path failed: %v", err)
	}
	streamRecord, err := exec(true, "run-a13-delayed-stream", "task-a13-delayed-stream")
	if err != nil {
		t.Fatalf("stream path failed: %v", err)
	}
	if runRecord.Status != streamRecord.Status {
		t.Fatalf("status mismatch run=%q stream=%q", runRecord.Status, streamRecord.Status)
	}
	if runRecord.SchedulerDelayedTaskTotal != streamRecord.SchedulerDelayedTaskTotal ||
		runRecord.SchedulerDelayedClaimTotal != streamRecord.SchedulerDelayedClaimTotal {
		t.Fatalf("delayed totals mismatch run=%#v stream=%#v", runRecord, streamRecord)
	}
	diff := runRecord.SchedulerDelayedWaitMsP95 - streamRecord.SchedulerDelayedWaitMsP95
	if diff < 0 {
		diff = -diff
	}
	if diff > 30 {
		t.Fatalf("delayed wait p95 mismatch beyond tolerance run=%d stream=%d diff=%d", runRecord.SchedulerDelayedWaitMsP95, streamRecord.SchedulerDelayedWaitMsP95, diff)
	}
	if runRecord.SchedulerDelayedTaskTotal != 1 || runRecord.SchedulerDelayedClaimTotal != 1 {
		t.Fatalf("unexpected delayed counters in run summary: %#v", runRecord)
	}
	if runRecord.SchedulerDelayedWaitMsP95 <= 0 {
		t.Fatalf("scheduler_delayed_wait_ms_p95=%d, want > 0", runRecord.SchedulerDelayedWaitMsP95)
	}
}

func TestComposerContractSchedulerFallbackToMemory(t *testing.T) {
	tmpDir := t.TempDir()
	blockedDir := filepath.Join(tmpDir, "blocked-dir")
	if err := os.WriteFile(blockedDir, []byte("x"), 0o600); err != nil {
		t.Fatalf("write blocked dir marker: %v", err)
	}
	cfgPath := filepath.Join(tmpDir, "runtime.yaml")
	cfg := strings.Join([]string{
		"reload:",
		"  enabled: false",
		"scheduler:",
		"  enabled: true",
		"  backend: file",
		"  path: " + filepath.ToSlash(filepath.Join(blockedDir, "scheduler-state.json")),
		"  lease_timeout: 400ms",
		"  heartbeat_interval: 100ms",
		"  queue_limit: 32",
		"  retry_max_attempts: 3",
		"subagent:",
		"  max_depth: 4",
		"  max_active_children: 8",
		"  child_timeout_budget: 3s",
		"",
	}, "\n")
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
		FilePath:  cfgPath,
		EnvPrefix: "BAYMAX_A8_TEST",
	})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	model := fakes.NewModel([]fakes.ModelStep{
		{Response: types.ModelResponse{FinalAnswer: "ok"}},
	})
	model.SetStream([]types.ModelEvent{
		{Type: types.ModelEventTypeFinalAnswer, TextDelta: "ok"},
	}, nil)

	dispatcher := event.NewDispatcher(event.NewRuntimeRecorder(mgr))
	comp, err := composer.NewBuilder(model).
		WithRuntimeManager(mgr).
		WithEventHandler(dispatcherHandler{dispatcher: dispatcher}).
		Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}
	runID := "run-a8-fallback"
	if _, err := comp.Run(context.Background(), types.RunRequest{RunID: runID, Input: "fallback"}, nil); err != nil {
		t.Fatalf("composer run failed: %v", err)
	}

	record := findRunRecord(t, mgr.RecentRuns(10), runID)
	if !record.ComposerManaged {
		t.Fatalf("expected composer_managed=true, got %#v", record)
	}
	if record.SchedulerBackend != "memory" {
		t.Fatalf("scheduler backend=%q, want memory", record.SchedulerBackend)
	}
	if !record.SchedulerBackendFallback {
		t.Fatalf("scheduler fallback marker should be true: %#v", record)
	}
	if record.SchedulerBackendFallbackReason == "" {
		t.Fatalf("scheduler fallback reason should not be empty: %#v", record)
	}
}

func TestComposerContractTakeoverReplayIdempotency(t *testing.T) {
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX_A8_TEST"})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	model := fakes.NewModel([]fakes.ModelStep{
		{Response: types.ModelResponse{FinalAnswer: "ok"}},
	})
	model.SetStream([]types.ModelEvent{
		{Type: types.ModelEventTypeFinalAnswer, TextDelta: "ok"},
	}, nil)
	dispatcher := event.NewDispatcher(event.NewRuntimeRecorder(mgr))
	comp, err := composer.NewBuilder(model).
		WithRuntimeManager(mgr).
		WithEventHandler(dispatcherHandler{dispatcher: dispatcher}).
		Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}

	childReq := composer.ChildDispatchRequest{
		Task: scheduler.Task{
			TaskID: "task-a8-idempotent",
			RunID:  "run-a8-idempotent",
		},
		Target:               composer.ChildTargetLocal,
		ParentDepth:          0,
		ParentActiveChildren: 0,
		ChildTimeout:         300 * time.Millisecond,
		LocalRunner: composer.LocalChildRunnerFunc(func(ctx context.Context, task scheduler.Task) (map[string]any, error) {
			return map[string]any{"task_id": task.TaskID, "ok": true}, nil
		}),
	}

	out, err := comp.DispatchChild(context.Background(), childReq)
	if err != nil {
		t.Fatalf("dispatch child failed: %v", err)
	}
	if out.CommitMeta.Duplicate {
		t.Fatalf("first commit should not be duplicate: %#v", out.CommitMeta)
	}

	dup, err := comp.CommitChildTerminal(context.Background(), out.Commit)
	if err != nil {
		t.Fatalf("duplicate terminal commit failed: %v", err)
	}
	if !dup.Duplicate {
		t.Fatalf("second commit should be duplicate: %#v", dup)
	}

	if _, err := comp.Run(context.Background(), types.RunRequest{
		RunID: out.Record.Task.RunID,
		Input: "emit-finished",
	}, nil); err != nil {
		t.Fatalf("composer run failed: %v", err)
	}
	record := findRunRecord(t, mgr.RecentRuns(10), out.Record.Task.RunID)
	if record.SubagentChildTotal != 1 || record.SubagentChildFailed != 0 {
		t.Fatalf("subagent aggregate should not inflate under replay: %#v", record)
	}
}

func TestComposerContractMailboxRuntimeWiringEnabledMatrix(t *testing.T) {
	tmp := t.TempDir()
	disabledCfg := filepath.Join(tmp, "runtime-disabled.yaml")
	disabledBlocked := filepath.Join(tmp, "mailbox-disabled-blocked")
	if err := os.WriteFile(disabledBlocked, []byte("x"), 0o600); err != nil {
		t.Fatalf("write disabled blocked marker: %v", err)
	}
	writeComposerMailboxRuntimeConfig(
		t,
		disabledCfg,
		false,
		"file",
		filepath.ToSlash(filepath.Join(disabledBlocked, "mailbox-state.json")),
	)
	disabledRunID := "run-a35-mailbox-disabled"
	disabledRecords, _ := executeComposerMailboxRuntimeWiring(t, disabledCfg, false, disabledRunID, "task-a35-mailbox-disabled")
	assertMailboxBackendState(t, disabledRecords, "memory", "disabled", false, "")

	enabledCfg := filepath.Join(tmp, "runtime-enabled-memory.yaml")
	writeComposerMailboxRuntimeConfig(t, enabledCfg, true, "memory", filepath.ToSlash(filepath.Join(tmp, "mailbox-memory-state.json")))
	enabledRunID := "run-a35-mailbox-enabled-memory"
	enabledRecords, _ := executeComposerMailboxRuntimeWiring(t, enabledCfg, false, enabledRunID, "task-a35-mailbox-enabled-memory")
	assertMailboxBackendState(t, enabledRecords, "memory", "memory", false, "")
}

func TestComposerContractMailboxRuntimeWiringFileFallbackAndDiagnostics(t *testing.T) {
	tmp := t.TempDir()
	blocked := filepath.Join(tmp, "mailbox-blocked")
	if err := os.WriteFile(blocked, []byte("x"), 0o600); err != nil {
		t.Fatalf("write blocked marker: %v", err)
	}
	cfgPath := filepath.Join(tmp, "runtime-fallback.yaml")
	writeComposerMailboxRuntimeConfig(
		t,
		cfgPath,
		true,
		"file",
		filepath.ToSlash(filepath.Join(blocked, "mailbox-state.json")),
	)

	records, agg := executeComposerMailboxRuntimeWiring(
		t,
		cfgPath,
		false,
		"run-a35-mailbox-file-fallback",
		"task-a35-mailbox-file-fallback",
	)
	assertMailboxBackendState(
		t,
		records,
		"memory",
		"file",
		true,
		"mailbox.backend.file_init_failed",
	)
	if got := agg.ReasonCodeTotals["mailbox.backend.file_init_failed"]; got == 0 {
		t.Fatalf("fallback reason aggregate missing, got=%d agg=%#v", got, agg)
	}
}

func TestComposerContractMailboxRuntimeWiringRunStreamMemoryFileParity(t *testing.T) {
	tmp := t.TempDir()
	memoryCfg := filepath.Join(tmp, "runtime-memory.yaml")
	fileCfg := filepath.Join(tmp, "runtime-file.yaml")
	writeComposerMailboxRuntimeConfig(t, memoryCfg, true, "memory", filepath.ToSlash(filepath.Join(tmp, "mailbox-memory.json")))
	writeComposerMailboxRuntimeConfig(t, fileCfg, true, "file", filepath.ToSlash(filepath.Join(tmp, "mailbox-file.json")))

	memoryRunRecords, memoryRunAgg := executeComposerMailboxRuntimeWiring(
		t,
		memoryCfg,
		false,
		"run-a35-mailbox-memory-run",
		"task-a35-mailbox-memory-run",
	)
	memoryStreamRecords, memoryStreamAgg := executeComposerMailboxRuntimeWiring(
		t,
		memoryCfg,
		true,
		"run-a35-mailbox-memory-stream",
		"task-a35-mailbox-memory-stream",
	)
	fileRunRecords, fileRunAgg := executeComposerMailboxRuntimeWiring(
		t,
		fileCfg,
		false,
		"run-a35-mailbox-file-run",
		"task-a35-mailbox-file-run",
	)
	fileStreamRecords, fileStreamAgg := executeComposerMailboxRuntimeWiring(
		t,
		fileCfg,
		true,
		"run-a35-mailbox-file-stream",
		"task-a35-mailbox-file-stream",
	)

	assertMailboxBackendState(t, memoryRunRecords, "memory", "memory", false, "")
	assertMailboxBackendState(t, memoryStreamRecords, "memory", "memory", false, "")
	assertMailboxBackendState(t, fileRunRecords, "file", "file", false, "")
	assertMailboxBackendState(t, fileStreamRecords, "file", "file", false, "")

	assertMailboxAggregateShapeEqual(t, memoryRunAgg, memoryStreamAgg, "memory run/stream")
	assertMailboxAggregateShapeEqual(t, fileRunAgg, fileStreamAgg, "file run/stream")
	assertMailboxAggregateShapeEqual(t, memoryRunAgg, fileRunAgg, "memory/file parity")
}

func executeComposerMailboxRuntimeWiring(
	t *testing.T,
	cfgPath string,
	stream bool,
	runID string,
	taskID string,
) ([]runtimediag.MailboxRecord, runtimediag.MailboxAggregate) {
	t.Helper()
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
		FilePath:  cfgPath,
		EnvPrefix: "BAYMAX_A35_TEST",
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
		WithA2AClient(composerMailboxA2AClient{}).
		Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}

	if stream {
		if _, err := comp.Stream(context.Background(), types.RunRequest{
			RunID: runID,
			Input: "mailbox-stream",
		}, nil); err != nil {
			t.Fatalf("composer stream failed: %v", err)
		}
	} else {
		if _, err := comp.Run(context.Background(), types.RunRequest{
			RunID: runID,
			Input: "mailbox-run",
		}, nil); err != nil {
			t.Fatalf("composer run failed: %v", err)
		}
	}

	out, err := comp.DispatchChild(context.Background(), composer.ChildDispatchRequest{
		Task: scheduler.Task{
			TaskID:     taskID,
			RunID:      runID,
			WorkflowID: runID,
			TeamID:     "team-a35",
			StepID:     "step-a35",
			AgentID:    "agent-a35",
			PeerID:     "peer-a35",
			Payload:    map[string]any{"mode": "a35"},
		},
		Target:       composer.ChildTargetA2A,
		ChildTimeout: 500 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("dispatch child failed: %v", err)
	}
	if out.Commit.Status != scheduler.TaskStateSucceeded {
		t.Fatalf("dispatch commit status=%q, want succeeded", out.Commit.Status)
	}

	page, err := mgr.QueryMailbox(runtimediag.MailboxQueryRequest{
		RunID: runID,
	})
	if err != nil {
		t.Fatalf("QueryMailbox failed: %v", err)
	}
	if len(page.Items) < 2 {
		t.Fatalf("mailbox records len=%d, want >=2", len(page.Items))
	}
	agg := mgr.MailboxAggregates(runtimediag.MailboxAggregateRequest{
		RunID: runID,
	})
	if agg.TotalRecords < 2 || agg.TotalMessages < 2 {
		t.Fatalf("mailbox aggregate mismatch: %#v", agg)
	}
	return page.Items, agg
}

func assertMailboxBackendState(
	t *testing.T,
	records []runtimediag.MailboxRecord,
	expectedBackend string,
	expectedConfigured string,
	expectedFallback bool,
	expectedFallbackReason string,
) {
	t.Helper()
	for _, rec := range records {
		if rec.Backend != expectedBackend {
			t.Fatalf("mailbox backend=%q, want %q, rec=%#v", rec.Backend, expectedBackend, rec)
		}
		if rec.ConfiguredBackend != expectedConfigured {
			t.Fatalf("mailbox configured_backend=%q, want %q, rec=%#v", rec.ConfiguredBackend, expectedConfigured, rec)
		}
		if rec.BackendFallback != expectedFallback {
			t.Fatalf("mailbox backend_fallback=%v, want %v, rec=%#v", rec.BackendFallback, expectedFallback, rec)
		}
		if strings.TrimSpace(rec.BackendFallbackReason) != strings.TrimSpace(expectedFallbackReason) {
			t.Fatalf(
				"mailbox backend_fallback_reason=%q, want %q, rec=%#v",
				rec.BackendFallbackReason,
				expectedFallbackReason,
				rec,
			)
		}
	}
}

func assertMailboxAggregateShapeEqual(
	t *testing.T,
	left runtimediag.MailboxAggregate,
	right runtimediag.MailboxAggregate,
	label string,
) {
	t.Helper()
	if left.TotalRecords != right.TotalRecords || left.TotalMessages != right.TotalMessages {
		t.Fatalf("%s mailbox totals mismatch: left=%#v right=%#v", label, left, right)
	}
	if !reflect.DeepEqual(left.ByKind, right.ByKind) {
		t.Fatalf("%s mailbox by_kind mismatch: left=%#v right=%#v", label, left.ByKind, right.ByKind)
	}
	if !reflect.DeepEqual(left.ByState, right.ByState) {
		t.Fatalf("%s mailbox by_state mismatch: left=%#v right=%#v", label, left.ByState, right.ByState)
	}
	if left.RetryTotal != right.RetryTotal ||
		left.DeadLetterTotal != right.DeadLetterTotal ||
		left.ExpiredTotal != right.ExpiredTotal {
		t.Fatalf("%s mailbox aggregate counters mismatch: left=%#v right=%#v", label, left, right)
	}
}

func writeComposerMailboxRuntimeConfig(
	t *testing.T,
	path string,
	enabled bool,
	backend string,
	mailboxPath string,
) {
	t.Helper()
	content := strings.Join([]string{
		"reload:",
		"  enabled: false",
		"scheduler:",
		"  enabled: true",
		"  backend: memory",
		"  lease_timeout: 500ms",
		"  heartbeat_interval: 100ms",
		"  queue_limit: 64",
		"  retry_max_attempts: 3",
		"mailbox:",
		"  enabled: " + strings.ToLower(strconv.FormatBool(enabled)),
		"  backend: " + backend,
		"  path: " + mailboxPath,
		"  retry:",
		"    max_attempts: 3",
		"    backoff_initial: 50ms",
		"    backoff_max: 500ms",
		"    jitter_ratio: 0.2",
		"  ttl: 15m",
		"  dlq:",
		"    enabled: false",
		"  query:",
		"    page_size_default: 50",
		"    page_size_max: 200",
		"subagent:",
		"  max_depth: 4",
		"  max_active_children: 8",
		"  child_timeout_budget: 3s",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write mailbox runtime config: %v", err)
	}
}

func findRunRecord(t *testing.T, records []runtimediag.RunRecord, runID string) runtimediag.RunRecord {
	t.Helper()
	for _, rec := range records {
		if strings.TrimSpace(rec.RunID) == strings.TrimSpace(runID) {
			return rec
		}
	}
	t.Fatalf("run record %q not found in %#v", runID, records)
	return runtimediag.RunRecord{}
}
