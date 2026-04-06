package composer

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/integration/fakes"
	"github.com/FelixSeptem/baymax/observability/event"
	"github.com/FelixSeptem/baymax/orchestration/scheduler"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

type timelineCollector struct {
	events []types.Event
}

func (c *timelineCollector) OnEvent(_ context.Context, ev types.Event) {
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

func TestComposerSchedulerReloadAppliesOnNextAttemptOnly(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime.yaml")
	writeComposerRuntimeConfig(t, cfgPath, 250*time.Millisecond)

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
		FilePath:        cfgPath,
		EnvPrefix:       "BAYMAX_COMPOSER_SCHEDULER_TEST",
		EnableHotReload: true,
	})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	model := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	comp, err := NewBuilder(model).WithRuntimeManager(mgr).Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}

	ctx := context.Background()
	taskOne := "task-a8-reload-one"
	if _, err := comp.SpawnChild(ctx, ChildDispatchRequest{
		Task: scheduler.Task{TaskID: taskOne, RunID: "run-a8-reload"},
	}); err != nil {
		t.Fatalf("spawn task one: %v", err)
	}
	claimedOne, ok, err := comp.Scheduler().Claim(ctx, "worker-one")
	if err != nil || !ok {
		t.Fatalf("claim task one failed: ok=%v err=%v", ok, err)
	}
	leaseOne := claimedOne.Attempt.LeaseExpiresAt.Sub(claimedOne.Attempt.StartedAt)
	if leaseOne < 150*time.Millisecond || leaseOne > 450*time.Millisecond {
		t.Fatalf("attempt one lease timeout=%s, want around 250ms", leaseOne)
	}

	recordOneBefore, found, err := comp.Scheduler().Get(ctx, taskOne)
	if err != nil || !found {
		t.Fatalf("get task one before reload failed: found=%v err=%v", found, err)
	}
	attemptOneBefore := mustFindAttempt(t, recordOneBefore, claimedOne.Attempt.AttemptID)

	writeComposerRuntimeConfig(t, cfgPath, 950*time.Millisecond)
	waitFor(t, 4*time.Second, func() bool {
		return mgr.EffectiveConfig().Scheduler.LeaseTimeout >= 900*time.Millisecond
	}, "runtime manager lease_timeout reload to >=900ms")

	taskTwo := "task-a8-reload-two"
	if _, err := comp.SpawnChild(ctx, ChildDispatchRequest{
		Task: scheduler.Task{TaskID: taskTwo, RunID: "run-a8-reload"},
	}); err != nil {
		t.Fatalf("spawn task two: %v", err)
	}
	claimedTwo, ok, err := comp.Scheduler().Claim(ctx, "worker-two")
	if err != nil || !ok {
		t.Fatalf("claim task two failed: ok=%v err=%v", ok, err)
	}
	leaseTwo := claimedTwo.Attempt.LeaseExpiresAt.Sub(claimedTwo.Attempt.StartedAt)
	if leaseTwo < 700*time.Millisecond {
		t.Fatalf("attempt two lease timeout=%s, want >=700ms after reload", leaseTwo)
	}

	recordOneAfter, found, err := comp.Scheduler().Get(ctx, taskOne)
	if err != nil || !found {
		t.Fatalf("get task one after reload failed: found=%v err=%v", found, err)
	}
	attemptOneAfter := mustFindAttempt(t, recordOneAfter, claimedOne.Attempt.AttemptID)
	if !attemptOneBefore.LeaseExpiresAt.Equal(attemptOneAfter.LeaseExpiresAt) {
		t.Fatalf(
			"in-flight attempt lease should not change after reload: before=%s after=%s",
			attemptOneBefore.LeaseExpiresAt,
			attemptOneAfter.LeaseExpiresAt,
		)
	}
}

func TestComposerGuardrailFailFastEmitsBudgetRejectAndSummary(t *testing.T) {
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX_COMPOSER_SCHEDULER_TEST"})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	model := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	collector := &timelineCollector{}
	dispatcher := event.NewDispatcher(event.NewRuntimeRecorder(mgr), collector)
	comp, err := NewBuilder(model).
		WithRuntimeManager(mgr).
		WithEventHandler(dispatcherHandler{dispatcher: dispatcher}).
		Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}

	runID := "run-a8-budget-reject"
	_, err = comp.SpawnChild(context.Background(), ChildDispatchRequest{
		Task: scheduler.Task{
			TaskID: "task-a8-budget-reject",
			RunID:  runID,
		},
		ParentDepth:          4,
		ParentActiveChildren: 0,
		ChildTimeout:         50 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("spawn should fail for budget reject")
	}
	var budgetErr *scheduler.BudgetError
	if !errors.As(err, &budgetErr) {
		t.Fatalf("spawn should return budget error, got %T (%v)", err, err)
	}

	hasBudgetRejectReason := false
	for _, ev := range collector.events {
		if ev.Type != types.EventTypeActionTimeline {
			continue
		}
		reason, _ := ev.Payload["reason"].(string)
		if reason == scheduler.ReasonBudgetReject {
			hasBudgetRejectReason = true
			break
		}
	}
	if !hasBudgetRejectReason {
		t.Fatalf("expected timeline reason %q in %#v", scheduler.ReasonBudgetReject, collector.events)
	}

	if _, err := comp.Run(context.Background(), types.RunRequest{RunID: runID, Input: "emit-finished"}, nil); err != nil {
		t.Fatalf("composer run failed: %v", err)
	}
	runs := mgr.RecentRuns(5)
	found := false
	for _, rec := range runs {
		if strings.TrimSpace(rec.RunID) != runID {
			continue
		}
		found = true
		if rec.SubagentBudgetRejectTotal != 1 {
			t.Fatalf("subagent_budget_reject_total=%d, want 1", rec.SubagentBudgetRejectTotal)
		}
	}
	if !found {
		t.Fatalf("run summary for %q not found in %#v", runID, runs)
	}
}

func TestComposerSpawnChildRejectsUnsupportedOperationProfile(t *testing.T) {
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX_COMPOSER_TIMEOUT_RESOLUTION_TEST"})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	model := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	comp, err := NewBuilder(model).WithRuntimeManager(mgr).Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}

	taskID := "task-a41-invalid-profile"
	_, err = comp.SpawnChild(context.Background(), ChildDispatchRequest{
		Task: scheduler.Task{
			TaskID: taskID,
			RunID:  "run-a41-invalid-profile",
		},
		OperationProfile: "realtime",
	})
	if err == nil {
		t.Fatal("spawn should fail for unsupported operation profile")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "operation profile must be one of") {
		t.Fatalf("unsupported operation profile error mismatch: %v", err)
	}
	if _, found, getErr := comp.Scheduler().Get(context.Background(), taskID); getErr != nil {
		t.Fatalf("get task failed: %v", getErr)
	} else if found {
		t.Fatalf("unsupported profile should not enqueue task %q", taskID)
	}
}

func TestComposerSpawnChildTimeoutResolutionPrecedenceAndSummary(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime-timeout-resolution.yaml")
	writeComposerTimeoutResolutionRuntimeConfig(t, cfgPath)

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
		FilePath:  cfgPath,
		EnvPrefix: "BAYMAX_COMPOSER_TIMEOUT_RESOLUTION_TEST",
	})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	model := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	dispatcher := event.NewDispatcher(event.NewRuntimeRecorder(mgr))
	comp, err := NewBuilder(model).
		WithRuntimeManager(mgr).
		WithEventHandler(dispatcherHandler{dispatcher: dispatcher}).
		Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}

	runID := "run-a41-timeout-resolution"
	domainRecord, err := comp.SpawnChild(context.Background(), ChildDispatchRequest{
		Task: scheduler.Task{
			TaskID: "task-a41-domain-override",
			RunID:  runID,
		},
		OperationProfile:      runtimeconfig.OperationProfileInteractive,
		ParentRemainingBudget: 8 * time.Second,
	})
	if err != nil {
		t.Fatalf("spawn with domain override failed: %v", err)
	}
	domainMeta := domainRecord.Task.TimeoutResolution
	if domainMeta.Source != runtimeconfig.TimeoutResolutionSourceDomain {
		t.Fatalf("domain override source = %q, want %q", domainMeta.Source, runtimeconfig.TimeoutResolutionSourceDomain)
	}
	if domainMeta.ResolvedTimeout != 6*time.Second {
		t.Fatalf("domain override resolved timeout = %s, want 6s", domainMeta.ResolvedTimeout)
	}
	if domainMeta.ParentBudgetClamped {
		t.Fatalf("domain override should not be clamped: %#v", domainMeta)
	}

	requestRecord, err := comp.SpawnChild(context.Background(), ChildDispatchRequest{
		Task: scheduler.Task{
			TaskID: "task-a41-request-override",
			RunID:  runID,
		},
		OperationProfile:      runtimeconfig.OperationProfileInteractive,
		RequestTimeout:        1500 * time.Millisecond,
		ParentRemainingBudget: 1200 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("spawn with request override failed: %v", err)
	}
	requestMeta := requestRecord.Task.TimeoutResolution
	if requestMeta.EffectiveOperationProfile != runtimeconfig.OperationProfileInteractive {
		t.Fatalf(
			"effective operation profile = %q, want %q",
			requestMeta.EffectiveOperationProfile,
			runtimeconfig.OperationProfileInteractive,
		)
	}
	if requestMeta.Source != runtimeconfig.TimeoutResolutionSourceRequest {
		t.Fatalf("request override source = %q, want %q", requestMeta.Source, runtimeconfig.TimeoutResolutionSourceRequest)
	}
	if requestMeta.ResolvedTimeout != 1200*time.Millisecond {
		t.Fatalf("request override resolved timeout = %s, want 1200ms", requestMeta.ResolvedTimeout)
	}
	if !requestMeta.ParentBudgetClamped {
		t.Fatalf("request override should be parent-budget clamped: %#v", requestMeta)
	}
	if strings.TrimSpace(requestMeta.Trace) == "" {
		t.Fatalf("request override trace should not be empty: %#v", requestMeta)
	}

	page, err := comp.Scheduler().QueryTasks(context.Background(), scheduler.TaskBoardQueryRequest{
		TaskID: "task-a41-request-override",
	})
	if err != nil {
		t.Fatalf("query task board failed: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("task board items len = %d, want 1", len(page.Items))
	}
	if page.Items[0].Task.TimeoutResolution.Source != runtimeconfig.TimeoutResolutionSourceRequest {
		t.Fatalf("task board timeout resolution source mismatch: %#v", page.Items[0].Task.TimeoutResolution)
	}

	if _, err := comp.Run(context.Background(), types.RunRequest{
		RunID: runID,
		Input: "emit-finished",
	}, nil); err != nil {
		t.Fatalf("composer run failed: %v", err)
	}
	runs := mgr.RecentRuns(10)
	found := false
	for _, rec := range runs {
		if strings.TrimSpace(rec.RunID) != runID {
			continue
		}
		found = true
		if rec.EffectiveOperationProfile != runtimeconfig.OperationProfileInteractive {
			t.Fatalf("effective_operation_profile = %q, want %q", rec.EffectiveOperationProfile, runtimeconfig.OperationProfileInteractive)
		}
		if rec.TimeoutResolutionSource != runtimeconfig.TimeoutResolutionSourceRequest {
			t.Fatalf("timeout_resolution_source = %q, want %q", rec.TimeoutResolutionSource, runtimeconfig.TimeoutResolutionSourceRequest)
		}
		if strings.TrimSpace(rec.TimeoutResolutionTrace) == "" {
			t.Fatalf("timeout_resolution_trace should not be empty: %#v", rec)
		}
		if rec.TimeoutParentBudgetClampTotal != 1 {
			t.Fatalf("timeout_parent_budget_clamp_total = %d, want 1", rec.TimeoutParentBudgetClampTotal)
		}
		if rec.TimeoutParentBudgetRejectTotal != 0 {
			t.Fatalf("timeout_parent_budget_reject_total = %d, want 0", rec.TimeoutParentBudgetRejectTotal)
		}
	}
	if !found {
		t.Fatalf("run summary for %q not found in %#v", runID, runs)
	}
}

func TestComposerSpawnChildPassesNotBeforeThrough(t *testing.T) {
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX_COMPOSER_NOT_BEFORE_TEST"})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	model := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	comp, err := NewBuilder(model).WithRuntimeManager(mgr).Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}

	notBefore := time.Now().Add(100 * time.Millisecond)
	record, err := comp.SpawnChild(context.Background(), ChildDispatchRequest{
		Task: scheduler.Task{
			TaskID:    "task-a13-not-before",
			RunID:     "run-a13-not-before",
			NotBefore: notBefore,
		},
	})
	if err != nil {
		t.Fatalf("spawn child failed: %v", err)
	}
	if record.Task.NotBefore.IsZero() || !record.Task.NotBefore.Equal(notBefore.UTC()) {
		t.Fatalf("spawned task not_before mismatch: got=%s want=%s", record.Task.NotBefore, notBefore.UTC())
	}
	if _, ok, err := comp.Scheduler().Claim(context.Background(), "worker-a13"); err != nil || ok {
		t.Fatalf("child should not be claimable before not_before: ok=%v err=%v", ok, err)
	}
	time.Sleep(120 * time.Millisecond)
	if _, ok, err := comp.Scheduler().Claim(context.Background(), "worker-a13"); err != nil || !ok {
		t.Fatalf("child should be claimable after not_before: ok=%v err=%v", ok, err)
	}
}

func TestComposerReadinessPreflightRequiresRuntimeManager(t *testing.T) {
	model := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	comp, err := NewBuilder(model).Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}
	if _, err := comp.ReadinessPreflight(); err == nil {
		t.Fatal("ReadinessPreflight should fail when runtime manager is not configured")
	}
}

func TestComposerReadinessPreflightPassthroughAndReadOnly(t *testing.T) {
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: "BAYMAX_COMPOSER_READINESS_PREFLIGHT_TEST"})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	model := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	comp, err := NewBuilder(model).WithRuntimeManager(mgr).Build()
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
	runtimeResult := mgr.ReadinessPreflight()
	composerResult, err := comp.ReadinessPreflight()
	if err != nil {
		t.Fatalf("composer readiness failed: %v", err)
	}
	after, err := comp.SchedulerStats(context.Background())
	if err != nil {
		t.Fatalf("SchedulerStats after readiness failed: %v", err)
	}

	if runtimeResult.Status != composerResult.Status {
		t.Fatalf("status mismatch runtime=%q composer=%q", runtimeResult.Status, composerResult.Status)
	}
	if readinessFingerprint(runtimeResult) != readinessFingerprint(composerResult) {
		t.Fatalf("readiness semantics mismatch runtime=%s composer=%s", readinessFingerprint(runtimeResult), readinessFingerprint(composerResult))
	}
	if before.QueueTotal != after.QueueTotal || before.ClaimTotal != after.ClaimTotal || before.ReclaimTotal != after.ReclaimTotal {
		t.Fatalf("readiness query should be read-only, before=%#v after=%#v", before, after)
	}
}

func TestComposerReadinessAdmissionBlockedDenyRunAndStreamNoSideEffects(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime-readiness-admission.yaml")
	writeComposerReadinessAdmissionRuntimeConfig(t, cfgPath, runtimeconfig.ReadinessAdmissionDegradedPolicyAllowAndRecord)

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
		FilePath:  cfgPath,
		EnvPrefix: "BAYMAX_COMPOSER_READINESS_ADMISSION_TEST",
	})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	model := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	comp, err := NewBuilder(model).WithRuntimeManager(mgr).Build()
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
		RunID: "run-a44-blocked-run",
		Input: "blocked-run",
	}, nil)
	if runErr == nil {
		t.Fatal("run should be denied by readiness admission")
	}
	assertAdmissionDeniedResult(t, runRes, runtimeconfig.ReadinessAdmissionCodeBlocked)
	afterRun, err := comp.SchedulerStats(context.Background())
	if err != nil {
		t.Fatalf("scheduler stats after run deny failed: %v", err)
	}
	assertSchedulerStatsUnchanged(t, before, afterRun)

	streamRes, streamErr := comp.Stream(context.Background(), types.RunRequest{
		RunID: "run-a44-blocked-stream",
		Input: "blocked-stream",
	}, nil)
	if streamErr == nil {
		t.Fatal("stream should be denied by readiness admission")
	}
	assertAdmissionDeniedResult(t, streamRes, runtimeconfig.ReadinessAdmissionCodeBlocked)
	afterStream, err := comp.SchedulerStats(context.Background())
	if err != nil {
		t.Fatalf("scheduler stats after stream deny failed: %v", err)
	}
	assertSchedulerStatsUnchanged(t, before, afterStream)

	if len(mgr.RecentMailbox(10)) != mailboxBefore {
		t.Fatalf("deny path should not mutate mailbox diagnostics: before=%d after=%d", mailboxBefore, len(mgr.RecentMailbox(10)))
	}
}

func TestComposerReadinessAdmissionSandboxRequiredDenyRunAndStreamEquivalent(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime-sandbox-required-blocked.yaml")
	writeComposerSandboxRequiredRuntimeConfig(t, cfgPath)

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
		FilePath:  cfgPath,
		EnvPrefix: "BAYMAX_COMPOSER_SANDBOX_REQUIRED_TEST",
	})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	model := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	comp, err := NewBuilder(model).WithRuntimeManager(mgr).Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}

	before, err := comp.SchedulerStats(context.Background())
	if err != nil {
		t.Fatalf("scheduler stats before deny failed: %v", err)
	}
	mailboxBefore := len(mgr.RecentMailbox(10))

	runRes, runErr := comp.Run(context.Background(), types.RunRequest{
		RunID: "run-a51-blocked-run",
		Input: "blocked-run",
	}, nil)
	if runErr == nil {
		t.Fatal("run should be denied by sandbox-required readiness admission")
	}
	assertAdmissionDeniedResult(t, runRes, runtimeconfig.ReadinessAdmissionCodeBlocked)

	streamRes, streamErr := comp.Stream(context.Background(), types.RunRequest{
		RunID: "run-a51-blocked-stream",
		Input: "blocked-stream",
	}, nil)
	if streamErr == nil {
		t.Fatal("stream should be denied by sandbox-required readiness admission")
	}
	assertAdmissionDeniedResult(t, streamRes, runtimeconfig.ReadinessAdmissionCodeBlocked)

	runPrimaryCode, _ := runRes.Error.Details["readiness_primary_code"].(string)
	streamPrimaryCode, _ := streamRes.Error.Details["readiness_primary_code"].(string)
	if runPrimaryCode != runtimeconfig.ReadinessCodeSandboxRequiredUnavailable ||
		streamPrimaryCode != runtimeconfig.ReadinessCodeSandboxRequiredUnavailable {
		t.Fatalf("sandbox primary_code mismatch run=%q stream=%q", runPrimaryCode, streamPrimaryCode)
	}
	runPrimaryDomain, _ := runRes.Error.Details["readiness_primary_domain"].(string)
	streamPrimaryDomain, _ := streamRes.Error.Details["readiness_primary_domain"].(string)
	if runPrimaryDomain != runtimeconfig.ReadinessDomainRuntime ||
		streamPrimaryDomain != runtimeconfig.ReadinessDomainRuntime {
		t.Fatalf("sandbox primary_domain mismatch run=%q stream=%q", runPrimaryDomain, streamPrimaryDomain)
	}
	runPrimarySource, _ := runRes.Error.Details["readiness_primary_source"].(string)
	streamPrimarySource, _ := streamRes.Error.Details["readiness_primary_source"].(string)
	if runPrimarySource != runtimeconfig.RuntimePrimarySourceReadiness ||
		streamPrimarySource != runtimeconfig.RuntimePrimarySourceReadiness {
		t.Fatalf("sandbox primary_source mismatch run=%q stream=%q", runPrimarySource, streamPrimarySource)
	}
	runWinnerStage, _ := runRes.Error.Details["winner_stage"].(string)
	streamWinnerStage, _ := streamRes.Error.Details["winner_stage"].(string)
	if runWinnerStage != runtimeconfig.RuntimePolicyStageSandboxAction ||
		streamWinnerStage != runtimeconfig.RuntimePolicyStageSandboxAction {
		t.Fatalf("winner_stage mismatch run=%q stream=%q", runWinnerStage, streamWinnerStage)
	}
	runDenySource, _ := runRes.Error.Details["deny_source"].(string)
	streamDenySource, _ := streamRes.Error.Details["deny_source"].(string)
	if strings.TrimSpace(runDenySource) == "" || strings.TrimSpace(streamDenySource) == "" {
		t.Fatalf("deny_source should be populated run=%q stream=%q", runDenySource, streamDenySource)
	}
	runPath := policyDecisionPathFromDetail(runRes.Error.Details["policy_decision_path"])
	streamPath := policyDecisionPathFromDetail(streamRes.Error.Details["policy_decision_path"])
	if len(runPath) == 0 || len(streamPath) == 0 {
		t.Fatalf("policy_decision_path should be present run=%#v stream=%#v", runRes.Error.Details["policy_decision_path"], streamRes.Error.Details["policy_decision_path"])
	}
	if runPath[0].Stage != streamPath[0].Stage {
		t.Fatalf("policy_decision_path first stage mismatch run=%#v stream=%#v", runPath, streamPath)
	}

	after, err := comp.SchedulerStats(context.Background())
	if err != nil {
		t.Fatalf("scheduler stats after deny failed: %v", err)
	}
	assertSchedulerStatsUnchanged(t, before, after)
	if len(mgr.RecentMailbox(10)) != mailboxBefore {
		t.Fatalf("deny path should not mutate mailbox diagnostics: before=%d after=%d", mailboxBefore, len(mgr.RecentMailbox(10)))
	}
}

func TestComposerReadinessAdmissionSandboxRolloutFrozenRunAndStreamEquivalent(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime-sandbox-rollout-frozen.yaml")
	writeComposerSandboxRolloutRuntimeConfig(t, cfgPath, runtimeconfig.SecuritySandboxRolloutPhaseFrozen, runtimeconfig.SecuritySandboxCapacityDegradedPolicyAllowAndRecord)

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
		FilePath:  cfgPath,
		EnvPrefix: "BAYMAX_COMPOSER_SANDBOX_ROLLOUT_TEST",
	})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	model := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	comp, err := NewBuilder(model).WithRuntimeManager(mgr).Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}

	before, err := comp.SchedulerStats(context.Background())
	if err != nil {
		t.Fatalf("scheduler stats before deny failed: %v", err)
	}
	mailboxBefore := len(mgr.RecentMailbox(10))

	runRes, runErr := comp.Run(context.Background(), types.RunRequest{
		RunID: "run-a52-frozen-run",
		Input: "frozen-run",
	}, nil)
	if runErr == nil {
		t.Fatal("run should be denied by sandbox rollout frozen admission")
	}
	assertAdmissionDeniedResult(t, runRes, runtimeconfig.ReadinessAdmissionCodeSandboxFrozen)

	streamRes, streamErr := comp.Stream(context.Background(), types.RunRequest{
		RunID: "run-a52-frozen-stream",
		Input: "frozen-stream",
	}, nil)
	if streamErr == nil {
		t.Fatal("stream should be denied by sandbox rollout frozen admission")
	}
	assertAdmissionDeniedResult(t, streamRes, runtimeconfig.ReadinessAdmissionCodeSandboxFrozen)

	runPrimaryCode, _ := runRes.Error.Details["readiness_primary_code"].(string)
	streamPrimaryCode, _ := streamRes.Error.Details["readiness_primary_code"].(string)
	if runPrimaryCode != runtimeconfig.ReadinessCodeSandboxRolloutFrozen ||
		streamPrimaryCode != runtimeconfig.ReadinessCodeSandboxRolloutFrozen {
		t.Fatalf("sandbox rollout frozen primary_code mismatch run=%q stream=%q", runPrimaryCode, streamPrimaryCode)
	}

	after, err := comp.SchedulerStats(context.Background())
	if err != nil {
		t.Fatalf("scheduler stats after deny failed: %v", err)
	}
	assertSchedulerStatsUnchanged(t, before, after)
	if len(mgr.RecentMailbox(10)) != mailboxBefore {
		t.Fatalf("deny path should not mutate mailbox diagnostics: before=%d after=%d", mailboxBefore, len(mgr.RecentMailbox(10)))
	}
}

func TestComposerReadinessAdmissionSandboxCapacityThrottlePolicyParity(t *testing.T) {
	allowCfg := filepath.Join(t.TempDir(), "runtime-sandbox-rollout-throttle-allow.yaml")
	writeComposerSandboxRolloutRuntimeConfig(t, allowCfg, runtimeconfig.SecuritySandboxRolloutPhaseCanary, runtimeconfig.SecuritySandboxCapacityDegradedPolicyAllowAndRecord)
	allowMgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
		FilePath:  allowCfg,
		EnvPrefix: "BAYMAX_COMPOSER_SANDBOX_ROLLOUT_TEST",
	})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	defer func() { _ = allowMgr.Close() }()
	allowMgr.SetSandboxRolloutRuntimeState(runtimeconfig.SandboxRolloutRuntimeState{CapacityAction: runtimeconfig.SandboxCapacityActionThrottle})

	allowModel := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	allowComp, err := NewBuilder(allowModel).WithRuntimeManager(allowMgr).Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}

	allowRunRes, allowRunErr := allowComp.Run(context.Background(), types.RunRequest{
		RunID: "run-a52-throttle-allow-run",
		Input: "allow-run",
	}, nil)
	if allowRunErr != nil || allowRunRes.Error != nil {
		t.Fatalf("run should be allowed under throttle allow policy, err=%v result=%#v", allowRunErr, allowRunRes.Error)
	}
	allowStreamRes, allowStreamErr := allowComp.Stream(context.Background(), types.RunRequest{
		RunID: "run-a52-throttle-allow-stream",
		Input: "allow-stream",
	}, nil)
	if allowStreamErr != nil || allowStreamRes.Error != nil {
		t.Fatalf("stream should be allowed under throttle allow policy, err=%v result=%#v", allowStreamErr, allowStreamRes.Error)
	}

	denyCfg := filepath.Join(t.TempDir(), "runtime-sandbox-rollout-throttle-deny.yaml")
	writeComposerSandboxRolloutRuntimeConfig(t, denyCfg, runtimeconfig.SecuritySandboxRolloutPhaseCanary, runtimeconfig.SecuritySandboxCapacityDegradedPolicyFailFast)
	denyMgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
		FilePath:  denyCfg,
		EnvPrefix: "BAYMAX_COMPOSER_SANDBOX_ROLLOUT_TEST",
	})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	defer func() { _ = denyMgr.Close() }()
	denyMgr.SetSandboxRolloutRuntimeState(runtimeconfig.SandboxRolloutRuntimeState{CapacityAction: runtimeconfig.SandboxCapacityActionThrottle})

	denyModel := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	denyComp, err := NewBuilder(denyModel).WithRuntimeManager(denyMgr).Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}
	before, err := denyComp.SchedulerStats(context.Background())
	if err != nil {
		t.Fatalf("scheduler stats before deny failed: %v", err)
	}
	mailboxBefore := len(denyMgr.RecentMailbox(10))

	denyRunRes, denyRunErr := denyComp.Run(context.Background(), types.RunRequest{
		RunID: "run-a52-throttle-deny-run",
		Input: "deny-run",
	}, nil)
	if denyRunErr == nil {
		t.Fatal("run should be denied under throttle fail_fast policy")
	}
	assertAdmissionDeniedResult(t, denyRunRes, runtimeconfig.ReadinessAdmissionCodeSandboxThrottledDeny)

	denyStreamRes, denyStreamErr := denyComp.Stream(context.Background(), types.RunRequest{
		RunID: "run-a52-throttle-deny-stream",
		Input: "deny-stream",
	}, nil)
	if denyStreamErr == nil {
		t.Fatal("stream should be denied under throttle fail_fast policy")
	}
	assertAdmissionDeniedResult(t, denyStreamRes, runtimeconfig.ReadinessAdmissionCodeSandboxThrottledDeny)

	after, err := denyComp.SchedulerStats(context.Background())
	if err != nil {
		t.Fatalf("scheduler stats after deny failed: %v", err)
	}
	assertSchedulerStatsUnchanged(t, before, after)
	if len(denyMgr.RecentMailbox(10)) != mailboxBefore {
		t.Fatalf("deny path should not mutate mailbox diagnostics: before=%d after=%d", mailboxBefore, len(denyMgr.RecentMailbox(10)))
	}
}

func TestComposerReadinessAdmissionSandboxRolloutTimelineReasonParity(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime-sandbox-rollout-timeline-frozen.yaml")
	writeComposerSandboxRolloutRuntimeConfig(t, cfgPath, runtimeconfig.SecuritySandboxRolloutPhaseFrozen, runtimeconfig.SecuritySandboxCapacityDegradedPolicyAllowAndRecord)
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
		FilePath:  cfgPath,
		EnvPrefix: "BAYMAX_COMPOSER_SANDBOX_ROLLOUT_TEST",
	})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	model := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	comp, err := NewBuilder(model).WithRuntimeManager(mgr).Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}
	runCollector := &timelineCollector{}
	streamCollector := &timelineCollector{}

	_, runErr := comp.Run(context.Background(), types.RunRequest{
		RunID: "run-a52-timeline-frozen-run",
		Input: "frozen-run",
	}, runCollector)
	if runErr == nil {
		t.Fatal("run should be denied under frozen rollout")
	}
	_, streamErr := comp.Stream(context.Background(), types.RunRequest{
		RunID: "run-a52-timeline-frozen-stream",
		Input: "frozen-stream",
	}, streamCollector)
	if streamErr == nil {
		t.Fatal("stream should be denied under frozen rollout")
	}
	if !hasTimelineReason(runCollector.events, "sandbox.rollout.phase_frozen") {
		t.Fatalf("run timeline missing reason sandbox.rollout.phase_frozen: %#v", runCollector.events)
	}
	if !hasTimelineReason(streamCollector.events, "sandbox.rollout.phase_frozen") {
		t.Fatalf("stream timeline missing reason sandbox.rollout.phase_frozen: %#v", streamCollector.events)
	}

	throttleCfg := filepath.Join(t.TempDir(), "runtime-sandbox-rollout-timeline-throttle.yaml")
	writeComposerSandboxRolloutRuntimeConfig(t, throttleCfg, runtimeconfig.SecuritySandboxRolloutPhaseCanary, runtimeconfig.SecuritySandboxCapacityDegradedPolicyAllowAndRecord)
	throttleMgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
		FilePath:  throttleCfg,
		EnvPrefix: "BAYMAX_COMPOSER_SANDBOX_ROLLOUT_TEST",
	})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	defer func() { _ = throttleMgr.Close() }()
	throttleMgr.SetSandboxRolloutRuntimeState(runtimeconfig.SandboxRolloutRuntimeState{CapacityAction: runtimeconfig.SandboxCapacityActionThrottle})

	throttleModel := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	throttleComp, err := NewBuilder(throttleModel).WithRuntimeManager(throttleMgr).Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}
	throttleRunCollector := &timelineCollector{}
	throttleStreamCollector := &timelineCollector{}
	if _, err := throttleComp.Run(context.Background(), types.RunRequest{
		RunID: "run-a52-timeline-throttle-run",
		Input: "throttle-run",
	}, throttleRunCollector); err != nil {
		t.Fatalf("run should be allowed under throttle allow policy: %v", err)
	}
	if _, err := throttleComp.Stream(context.Background(), types.RunRequest{
		RunID: "run-a52-timeline-throttle-stream",
		Input: "throttle-stream",
	}, throttleStreamCollector); err != nil {
		t.Fatalf("stream should be allowed under throttle allow policy: %v", err)
	}
	if !hasTimelineReason(throttleRunCollector.events, "sandbox.rollout.capacity_throttle") {
		t.Fatalf("run timeline missing reason sandbox.rollout.capacity_throttle: %#v", throttleRunCollector.events)
	}
	if !hasTimelineReason(throttleStreamCollector.events, "sandbox.rollout.capacity_throttle") {
		t.Fatalf("stream timeline missing reason sandbox.rollout.capacity_throttle: %#v", throttleStreamCollector.events)
	}
}

func TestComposerReadinessAdmissionDegradedPolicyAllowRunAndStreamEquivalent(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime-readiness-admission-allow.yaml")
	writeComposerReadinessAdmissionRuntimeConfig(t, cfgPath, runtimeconfig.ReadinessAdmissionDegradedPolicyAllowAndRecord)

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
		FilePath:  cfgPath,
		EnvPrefix: "BAYMAX_COMPOSER_READINESS_ADMISSION_TEST",
	})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	model := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	comp, err := NewBuilder(model).WithRuntimeManager(mgr).Build()
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

	runRes, runErr := comp.Run(context.Background(), types.RunRequest{
		RunID: "run-a44-degraded-allow-run",
		Input: "allow-run",
	}, nil)
	if runErr != nil {
		t.Fatalf("run should be allowed under degraded allow policy, err=%v result=%#v", runErr, runRes)
	}
	if runRes.Error != nil {
		t.Fatalf("run result should not contain error under degraded allow policy: %#v", runRes.Error)
	}

	streamRes, streamErr := comp.Stream(context.Background(), types.RunRequest{
		RunID: "run-a44-degraded-allow-stream",
		Input: "allow-stream",
	}, nil)
	if streamErr != nil {
		t.Fatalf("stream should be allowed under degraded allow policy, err=%v result=%#v", streamErr, streamRes)
	}
	if streamRes.Error != nil {
		t.Fatalf("stream result should not contain error under degraded allow policy: %#v", streamRes.Error)
	}
}

func TestComposerReadinessAdmissionDegradedFailFastDeny(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime-readiness-admission-deny.yaml")
	writeComposerReadinessAdmissionRuntimeConfig(t, cfgPath, runtimeconfig.ReadinessAdmissionDegradedPolicyFailFast)

	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{
		FilePath:  cfgPath,
		EnvPrefix: "BAYMAX_COMPOSER_READINESS_ADMISSION_TEST",
	})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	model := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	comp, err := NewBuilder(model).WithRuntimeManager(mgr).Build()
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

	runRes, runErr := comp.Run(context.Background(), types.RunRequest{
		RunID: "run-a44-degraded-deny-run",
		Input: "deny-run",
	}, nil)
	if runErr == nil {
		t.Fatal("run should be denied under degraded fail_fast policy")
	}
	assertAdmissionDeniedResult(t, runRes, runtimeconfig.ReadinessAdmissionCodeDegradedDeny)

	streamRes, streamErr := comp.Stream(context.Background(), types.RunRequest{
		RunID: "run-a44-degraded-deny-stream",
		Input: "deny-stream",
	}, nil)
	if streamErr == nil {
		t.Fatal("stream should be denied under degraded fail_fast policy")
	}
	assertAdmissionDeniedResult(t, streamRes, runtimeconfig.ReadinessAdmissionCodeDegradedDeny)
}

func writeComposerRuntimeConfig(t *testing.T, path string, leaseTimeout time.Duration) {
	t.Helper()
	cfg := strings.Join([]string{
		"reload:",
		"  enabled: true",
		"  debounce: 15ms",
		"scheduler:",
		"  enabled: true",
		"  backend: memory",
		"  lease_timeout: " + leaseTimeout.String(),
		"  heartbeat_interval: 100ms",
		"  queue_limit: 64",
		"  retry_max_attempts: 3",
		"subagent:",
		"  max_depth: 4",
		"  max_active_children: 8",
		"  child_timeout_budget: 3s",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		t.Fatalf("write runtime config %q: %v", path, err)
	}
}

func writeComposerTimeoutResolutionRuntimeConfig(t *testing.T, path string) {
	t.Helper()
	cfg := strings.Join([]string{
		"reload:",
		"  enabled: false",
		"scheduler:",
		"  enabled: true",
		"  backend: memory",
		"  lease_timeout: 300ms",
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
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		t.Fatalf("write runtime config %q: %v", path, err)
	}
}

func writeComposerReadinessAdmissionRuntimeConfig(t *testing.T, path, degradedPolicy string) {
	t.Helper()
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

func writeComposerSandboxRequiredRuntimeConfig(t *testing.T, path string) {
	t.Helper()
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
		"security:",
		"  sandbox:",
		"    enabled: true",
		"    required: true",
		"    mode: enforce",
		"    policy:",
		"      default_action: sandbox",
		"      profile: default",
		"      fallback_action: deny",
		"reload:",
		"  enabled: false",
		"  debounce: 20ms",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		t.Fatalf("write runtime config %q: %v", path, err)
	}
}

func writeComposerSandboxRolloutRuntimeConfig(t *testing.T, path, phase, degradedPolicy string) {
	t.Helper()
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
		"security:",
		"  sandbox:",
		"    enabled: true",
		"    required: false",
		"    mode: observe",
		"    policy:",
		"      default_action: host",
		"      profile: default",
		"      fallback_action: allow_and_record",
		"    rollout:",
		"      phase: " + strings.TrimSpace(phase),
		"    capacity:",
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

func waitFor(t *testing.T, timeout time.Duration, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %s", msg)
}

func mustFindAttempt(t *testing.T, record scheduler.TaskRecord, attemptID string) scheduler.Attempt {
	t.Helper()
	for _, attempt := range record.Attempts {
		if strings.TrimSpace(attempt.AttemptID) == strings.TrimSpace(attemptID) {
			return attempt
		}
	}
	t.Fatalf("attempt %q not found in %#v", attemptID, record.Attempts)
	return scheduler.Attempt{}
}

func readinessFingerprint(result runtimeconfig.ReadinessResult) string {
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

func assertAdmissionDeniedResult(t *testing.T, result types.RunResult, wantReasonCode string) {
	t.Helper()
	if result.Error == nil {
		t.Fatalf("run result error = nil, want denied classified error; result=%#v", result)
	}
	if result.Error.Class != types.ErrContext {
		t.Fatalf("error class = %q, want %q", result.Error.Class, types.ErrContext)
	}
	gotReasonCode, _ := result.Error.Details["reason_code"].(string)
	if strings.TrimSpace(gotReasonCode) != strings.TrimSpace(wantReasonCode) {
		t.Fatalf("reason_code = %q, want %q (details=%#v)", gotReasonCode, wantReasonCode, result.Error.Details)
	}
}

func assertSchedulerStatsUnchanged(t *testing.T, before, after scheduler.Stats) {
	t.Helper()
	if before.QueueTotal != after.QueueTotal ||
		before.ClaimTotal != after.ClaimTotal ||
		before.ReclaimTotal != after.ReclaimTotal {
		t.Fatalf("scheduler stats changed on deny path: before=%#v after=%#v", before, after)
	}
}

func policyDecisionPathFromDetail(raw any) []runtimeconfig.RuntimePolicyCandidate {
	switch value := raw.(type) {
	case []runtimeconfig.RuntimePolicyCandidate:
		out := make([]runtimeconfig.RuntimePolicyCandidate, len(value))
		copy(out, value)
		return out
	case []any:
		out := make([]runtimeconfig.RuntimePolicyCandidate, 0, len(value))
		for i := range value {
			switch item := value[i].(type) {
			case runtimeconfig.RuntimePolicyCandidate:
				out = append(out, item)
			case map[string]any:
				candidate := runtimeconfig.RuntimePolicyCandidate{}
				if stage, ok := item["stage"].(string); ok {
					candidate.Stage = strings.TrimSpace(stage)
				}
				if code, ok := item["code"].(string); ok {
					candidate.Code = strings.TrimSpace(code)
				}
				if source, ok := item["source"].(string); ok {
					candidate.Source = strings.TrimSpace(source)
				}
				if decision, ok := item["decision"].(string); ok {
					candidate.Decision = strings.TrimSpace(decision)
				}
				if strings.TrimSpace(candidate.Stage) != "" {
					out = append(out, candidate)
				}
			}
		}
		if len(out) == 0 {
			return nil
		}
		return out
	default:
		return nil
	}
}

func hasTimelineReason(events []types.Event, reason string) bool {
	want := strings.TrimSpace(reason)
	if want == "" {
		return false
	}
	for i := range events {
		ev := events[i]
		if ev.Type != types.EventTypeActionTimeline {
			continue
		}
		got, _ := ev.Payload["reason"].(string)
		if strings.TrimSpace(got) == want {
			return true
		}
	}
	return false
}
