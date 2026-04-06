package integration

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/a2a"
	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/integration/fakes"
	"github.com/FelixSeptem/baymax/observability/event"
	"github.com/FelixSeptem/baymax/orchestration/composer"
	"github.com/FelixSeptem/baymax/orchestration/scheduler"
	"github.com/FelixSeptem/baymax/orchestration/teams"
	"github.com/FelixSeptem/baymax/orchestration/workflow"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
	runtimediag "github.com/FelixSeptem/baymax/runtime/diagnostics"
)

type collabTimelineCollector struct {
	mu     sync.Mutex
	events []types.Event
}

func (c *collabTimelineCollector) OnEvent(_ context.Context, ev types.Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, ev)
}

func (c *collabTimelineCollector) snapshot() []types.Event {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]types.Event, len(c.events))
	copy(out, c.events)
	return out
}

func TestCollaborationPrimitivesSyncMode(t *testing.T) {
	mgr := newCollabManager(t, "BAYMAX_COLLAB_SYNC")
	defer func() { _ = mgr.Close() }()

	model := newCollabModel()
	dispatcher := event.NewDispatcher(event.NewRuntimeRecorder(mgr))
	client := newCollabA2AClient(t)
	comp, err := composer.NewBuilder(model).
		WithRuntimeManager(mgr).
		WithEventHandler(dispatcherHandler{dispatcher: dispatcher}).
		WithA2AClient(client).
		Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}

	runID := "run-a16-sync"
	if _, err := comp.DispatchChild(context.Background(), composer.ChildDispatchRequest{
		Task: scheduler.Task{
			TaskID:      "task-a16-sync-delegation",
			RunID:       runID,
			WorkflowID:  "wf-a16-sync",
			TeamID:      "team-a16-sync",
			StepID:      "step-a16-sync",
			AgentID:     "agent-a16-sync",
			PeerID:      "peer-a16-sync",
			MaxAttempts: 1,
			Payload: map[string]any{
				"collab_primitive": "delegation",
			},
		},
		Target: composer.ChildTargetA2A,
	}); err != nil {
		t.Fatalf("dispatch delegation child failed: %v", err)
	}

	if _, err := comp.DispatchChild(context.Background(), composer.ChildDispatchRequest{
		Task: scheduler.Task{
			TaskID: "task-a16-sync-handoff",
			RunID:  runID,
			Payload: map[string]any{
				"collab_primitive": "handoff",
			},
		},
		Target: composer.ChildTargetLocal,
		LocalRunner: composer.LocalChildRunnerFunc(func(context.Context, scheduler.Task) (map[string]any, error) {
			return map[string]any{"ok": true}, nil
		}),
	}); err != nil {
		t.Fatalf("dispatch handoff child failed: %v", err)
	}

	if _, err := comp.DispatchChild(context.Background(), composer.ChildDispatchRequest{
		Task: scheduler.Task{
			TaskID: "task-a16-sync-aggregation",
			RunID:  runID,
			Payload: map[string]any{
				"collab_primitive": "aggregation",
			},
		},
		Target: composer.ChildTargetLocal,
		LocalRunner: composer.LocalChildRunnerFunc(func(context.Context, scheduler.Task) (map[string]any, error) {
			return map[string]any{"ok": true}, nil
		}),
	}); err != nil {
		t.Fatalf("dispatch aggregation child failed: %v", err)
	}

	if _, err := comp.Run(context.Background(), types.RunRequest{RunID: runID, Input: "emit-finished"}, nil); err != nil {
		t.Fatalf("composer run failed: %v", err)
	}

	record := findRunRecord(t, mgr.RecentRuns(20), runID)
	if record.CollabHandoffTotal != 1 ||
		record.CollabDelegationTotal != 1 ||
		record.CollabAggregationTotal != 3 ||
		record.CollabAggregationStrategy != runtimeconfig.ComposerCollabAggregationAllSettled ||
		record.CollabFailFastTotal != 0 {
		t.Fatalf("unexpected collab summary: %#v", record)
	}
}

func TestCollaborationPrimitivesAsyncReportingMode(t *testing.T) {
	mgr := newCollabManager(t, "BAYMAX_COLLAB_ASYNC")
	defer func() { _ = mgr.Close() }()

	model := newCollabModel()
	dispatcher := event.NewDispatcher(event.NewRuntimeRecorder(mgr))
	client := newCollabA2AClient(t)
	comp, err := composer.NewBuilder(model).
		WithRuntimeManager(mgr).
		WithEventHandler(dispatcherHandler{dispatcher: dispatcher}).
		WithA2AClient(client).
		Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}

	runID := "run-a16-async"
	taskID := "task-a16-async-delegation"
	out, err := comp.DispatchChild(context.Background(), composer.ChildDispatchRequest{
		Task: scheduler.Task{
			TaskID:      taskID,
			RunID:       runID,
			WorkflowID:  "wf-a16-async",
			TeamID:      "team-a16-async",
			StepID:      "step-a16-async",
			AgentID:     "agent-a16-async",
			PeerID:      "peer-a16-async",
			MaxAttempts: 1,
			Payload: map[string]any{
				"collab_primitive": "delegation",
			},
		},
		Target: composer.ChildTargetA2A,
		Async:  true,
	})
	if err != nil {
		t.Fatalf("dispatch async child failed: %v", err)
	}
	if !out.AsyncAccepted || strings.TrimSpace(out.AsyncTaskID) == "" {
		t.Fatalf("async dispatch result mismatch: %#v", out)
	}

	waitForTaskTerminalState(t, comp.Scheduler(), taskID, scheduler.TaskStateSucceeded)

	if _, err := comp.Run(context.Background(), types.RunRequest{RunID: runID, Input: "emit-finished"}, nil); err != nil {
		t.Fatalf("composer run failed: %v", err)
	}

	record := findRunRecord(t, mgr.RecentRuns(20), runID)
	if record.CollabDelegationTotal != 1 ||
		record.CollabAggregationTotal != 1 ||
		record.CollabFailFastTotal != 0 {
		t.Fatalf("unexpected async collab summary: %#v", record)
	}
	if record.A2AAsyncReportTotal <= 0 {
		t.Fatalf("a2a async report total should be > 0, got %#v", record)
	}
}

func TestCollaborationPrimitivesDelayedDispatchMode(t *testing.T) {
	mgr := newCollabManager(t, "BAYMAX_COLLAB_DELAYED")
	defer func() { _ = mgr.Close() }()

	model := newCollabModel()
	dispatcher := event.NewDispatcher(event.NewRuntimeRecorder(mgr))
	comp, err := composer.NewBuilder(model).
		WithRuntimeManager(mgr).
		WithEventHandler(dispatcherHandler{dispatcher: dispatcher}).
		Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}

	runID := "run-a16-delayed"
	taskID := "task-a16-delayed"
	notBefore := time.Now().Add(100 * time.Millisecond)
	if _, err := comp.SpawnChild(context.Background(), composer.ChildDispatchRequest{
		Task: scheduler.Task{
			TaskID:    taskID,
			RunID:     runID,
			NotBefore: notBefore,
			Payload: map[string]any{
				"collab_primitive": "aggregation",
			},
		},
	}); err != nil {
		t.Fatalf("spawn delayed child failed: %v", err)
	}

	if _, ok, err := comp.Scheduler().Claim(context.Background(), "worker-a16-delayed"); err != nil || ok {
		t.Fatalf("delayed task should not be claimable before boundary: ok=%v err=%v", ok, err)
	}
	// keep timing margin deterministic in CI
	time.Sleep(120 * time.Millisecond)
	claimed, ok, err := comp.Scheduler().Claim(context.Background(), "worker-a16-delayed")
	if err != nil || !ok {
		t.Fatalf("delayed task should be claimable after boundary: ok=%v err=%v", ok, err)
	}
	if _, err := comp.CommitChildTerminal(context.Background(), scheduler.TerminalCommit{
		TaskID:      claimed.Record.Task.TaskID,
		AttemptID:   claimed.Attempt.AttemptID,
		Status:      scheduler.TaskStateSucceeded,
		Result:      map[string]any{"ok": true},
		CommittedAt: time.Now(),
	}); err != nil {
		t.Fatalf("commit delayed child failed: %v", err)
	}

	if _, err := comp.Run(context.Background(), types.RunRequest{RunID: runID, Input: "emit-finished"}, nil); err != nil {
		t.Fatalf("composer run failed: %v", err)
	}
	record := findRunRecord(t, mgr.RecentRuns(20), runID)
	if record.CollabAggregationTotal != 1 || record.SchedulerDelayedTaskTotal != 1 || record.SchedulerDelayedClaimTotal != 1 {
		t.Fatalf("unexpected delayed collab summary: %#v", record)
	}
}

func TestCollaborationPrimitivesRunStreamEquivalence(t *testing.T) {
	exec := func(stream bool, runID string) (runtimediag.RunRecord, error) {
		mgr := newCollabManager(t, "BAYMAX_COLLAB_EQ")
		defer func() { _ = mgr.Close() }()

		model := newCollabModel()
		dispatcher := event.NewDispatcher(event.NewRuntimeRecorder(mgr))
		comp, err := composer.NewBuilder(model).
			WithRuntimeManager(mgr).
			WithEventHandler(dispatcherHandler{dispatcher: dispatcher}).
			Build()
		if err != nil {
			return runtimediag.RunRecord{}, err
		}

		if _, err := comp.DispatchChild(context.Background(), composer.ChildDispatchRequest{
			Task: scheduler.Task{
				TaskID: "task-a16-eq-success",
				RunID:  runID,
				Payload: map[string]any{
					"collab_primitive": "handoff",
				},
			},
			Target: composer.ChildTargetLocal,
			LocalRunner: composer.LocalChildRunnerFunc(func(context.Context, scheduler.Task) (map[string]any, error) {
				return map[string]any{"ok": true}, nil
			}),
		}); err != nil {
			return runtimediag.RunRecord{}, err
		}

		_, _ = comp.DispatchChild(context.Background(), composer.ChildDispatchRequest{
			Task: scheduler.Task{
				TaskID:  "task-a16-eq-failed",
				RunID:   runID,
				PeerID:  "peer-a16-eq",
				AgentID: "agent-a16-eq",
				Payload: map[string]any{
					"collab_primitive": "delegation",
				},
			},
			Target: composer.ChildTargetLocal,
			LocalRunner: composer.LocalChildRunnerFunc(func(context.Context, scheduler.Task) (map[string]any, error) {
				return nil, errors.New("forced failure")
			}),
		})

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
		return findRunRecord(t, mgr.RecentRuns(20), runID), nil
	}

	runRecord, err := exec(false, "run-a16-eq-run")
	if err != nil {
		t.Fatalf("run path failed: %v", err)
	}
	streamRecord, err := exec(true, "run-a16-eq-stream")
	if err != nil {
		t.Fatalf("stream path failed: %v", err)
	}
	if runRecord.Status != streamRecord.Status {
		t.Fatalf("status mismatch run=%q stream=%q", runRecord.Status, streamRecord.Status)
	}
	if runRecord.CollabHandoffTotal != streamRecord.CollabHandoffTotal ||
		runRecord.CollabDelegationTotal != streamRecord.CollabDelegationTotal ||
		runRecord.CollabAggregationTotal != streamRecord.CollabAggregationTotal ||
		runRecord.CollabFailFastTotal != streamRecord.CollabFailFastTotal ||
		runRecord.CollabAggregationStrategy != streamRecord.CollabAggregationStrategy {
		t.Fatalf("run/stream collab summary mismatch run=%#v stream=%#v", runRecord, streamRecord)
	}
}

func TestCollaborationPrimitivesReplayRecoveryConsistency(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "runtime-collab.yaml")
	recoveryPath := filepath.Join(t.TempDir(), "recovery-collab")
	writeCollabRecoveryRuntimeConfig(t, cfgPath, recoveryPath)

	mgr1, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX_COLLAB_RECOVERY"})
	if err != nil {
		t.Fatalf("new runtime manager #1: %v", err)
	}
	defer func() { _ = mgr1.Close() }()

	model := newCollabModel()
	dispatcher1 := event.NewDispatcher(event.NewRuntimeRecorder(mgr1))
	comp1, err := composer.NewBuilder(model).
		WithRuntimeManager(mgr1).
		WithEventHandler(dispatcherHandler{dispatcher: dispatcher1}).
		Build()
	if err != nil {
		t.Fatalf("new composer #1: %v", err)
	}

	runID := "run-a16-recovery"
	out, err := comp1.DispatchChild(context.Background(), composer.ChildDispatchRequest{
		Task: scheduler.Task{
			TaskID:  "task-a16-recovery",
			RunID:   runID,
			PeerID:  "peer-a16-recovery",
			AgentID: "agent-a16-recovery",
			Payload: map[string]any{
				"collab_primitive": "delegation",
			},
		},
		Target: composer.ChildTargetLocal,
		LocalRunner: composer.LocalChildRunnerFunc(func(context.Context, scheduler.Task) (map[string]any, error) {
			return map[string]any{"ok": true}, nil
		}),
	})
	if err != nil {
		t.Fatalf("dispatch child failed: %v", err)
	}
	dup, err := comp1.CommitChildTerminal(context.Background(), out.Commit)
	if err != nil {
		t.Fatalf("duplicate commit failed: %v", err)
	}
	if !dup.Duplicate {
		t.Fatalf("expected duplicate commit marker, got %#v", dup)
	}
	if _, err := comp1.Run(context.Background(), types.RunRequest{RunID: runID, Input: "emit-finished"}, nil); err != nil {
		t.Fatalf("composer #1 run failed: %v", err)
	}
	before := findRunRecord(t, mgr1.RecentRuns(20), runID)

	mgr2, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX_COLLAB_RECOVERY"})
	if err != nil {
		t.Fatalf("new runtime manager #2: %v", err)
	}
	defer func() { _ = mgr2.Close() }()
	dispatcher2 := event.NewDispatcher(event.NewRuntimeRecorder(mgr2))
	comp2, err := composer.NewBuilder(model).
		WithRuntimeManager(mgr2).
		WithEventHandler(dispatcherHandler{dispatcher: dispatcher2}).
		WithSchedulerStore(scheduler.NewMemoryStore()).
		Build()
	if err != nil {
		t.Fatalf("new composer #2: %v", err)
	}
	if _, err := comp2.Recover(context.Background(), composer.RecoverRequest{RunID: runID}); err != nil {
		t.Fatalf("recover failed: %v", err)
	}
	if _, err := comp2.Run(context.Background(), types.RunRequest{RunID: runID, Input: "emit-finished"}, nil); err != nil {
		t.Fatalf("composer #2 run failed: %v", err)
	}
	after := findRunRecord(t, mgr2.RecentRuns(20), runID)

	if before.CollabDelegationTotal != 1 || before.CollabAggregationTotal != 1 || before.CollabFailFastTotal != 0 {
		t.Fatalf("unexpected pre-recovery collab summary: %#v", before)
	}
	if after.CollabDelegationTotal != before.CollabDelegationTotal ||
		after.CollabAggregationTotal != before.CollabAggregationTotal ||
		after.CollabFailFastTotal != before.CollabFailFastTotal ||
		after.CollabAggregationStrategy != before.CollabAggregationStrategy {
		t.Fatalf("recovery collab summary drift before=%#v after=%#v", before, after)
	}
}

func TestCollaborationPrimitivesTimelineReasonAndCorrelation(t *testing.T) {
	collector := &collabTimelineCollector{}

	teamEngine := teams.New(teams.WithTimelineEmitter(collector))
	_, err := teamEngine.Run(context.Background(), teams.Plan{
		RunID:      "run-a16-team",
		TeamID:     "team-a16",
		WorkflowID: "wf-a16",
		StepID:     "step-a16",
		Tasks: []teams.Task{
			{
				TaskID:          "task-a16-team",
				AgentID:         "agent-a16-team",
				CollabPrimitive: "handoff",
				Runner: teams.TaskRunnerFunc(func(context.Context, teams.Task) (teams.TaskResult, error) {
					return teams.TaskResult{Output: map[string]any{"ok": true}}, nil
				}),
			},
		},
	})
	if err != nil {
		t.Fatalf("teams run failed: %v", err)
	}

	wfEngine := workflow.New(
		workflow.WithTimelineEmitter(collector),
		workflow.WithStepAdapter(workflow.DispatchAdapter{
			Runner: func(context.Context, string, workflow.Step, int) (workflow.StepOutput, error) {
				return workflow.StepOutput{Payload: map[string]any{"ok": true}}, nil
			},
		}),
	)
	_, err = wfEngine.Run(context.Background(), workflow.RunRequest{
		RunID: "run-a16-workflow",
		DSL: workflow.Definition{
			WorkflowID: "wf-a16",
			Steps: []workflow.Step{
				{
					StepID:          "step-a16-wf",
					TaskID:          "task-a16-wf",
					Kind:            workflow.StepKindRunner,
					CollabPrimitive: "delegation",
					TeamID:          "team-a16",
					AgentID:         "agent-a16-wf",
					PeerID:          "peer-a16-wf",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("workflow run failed: %v", err)
	}

	events := collector.snapshot()
	seenTeamHandoff := false
	seenWorkflowDelegation := false
	for _, ev := range events {
		if ev.Type != types.EventTypeActionTimeline {
			continue
		}
		reason, _ := ev.Payload["reason"].(string)
		if strings.HasPrefix(reason, "collab.") {
			t.Fatalf("reason should not use collab.* namespace: %#v", ev.Payload)
		}
		switch reason {
		case "team.handoff":
			seenTeamHandoff = true
			if _, ok := ev.Payload["team_id"]; !ok {
				t.Fatalf("team.handoff missing team_id: %#v", ev.Payload)
			}
			if _, ok := ev.Payload["task_id"]; !ok {
				t.Fatalf("team.handoff missing task_id: %#v", ev.Payload)
			}
			if _, ok := ev.Payload["agent_id"]; !ok {
				t.Fatalf("team.handoff missing agent_id: %#v", ev.Payload)
			}
		case "workflow.delegation":
			seenWorkflowDelegation = true
			if _, ok := ev.Payload["workflow_id"]; !ok {
				t.Fatalf("workflow.delegation missing workflow_id: %#v", ev.Payload)
			}
			if _, ok := ev.Payload["step_id"]; !ok {
				t.Fatalf("workflow.delegation missing step_id: %#v", ev.Payload)
			}
			if _, ok := ev.Payload["task_id"]; !ok {
				t.Fatalf("workflow.delegation missing task_id: %#v", ev.Payload)
			}
			if _, ok := ev.Payload["agent_id"]; !ok {
				t.Fatalf("workflow.delegation missing agent_id: %#v", ev.Payload)
			}
			if _, ok := ev.Payload["peer_id"]; !ok {
				t.Fatalf("workflow.delegation missing peer_id: %#v", ev.Payload)
			}
		}
	}
	if !seenTeamHandoff {
		t.Fatal("missing team.handoff timeline reason")
	}
	if !seenWorkflowDelegation {
		t.Fatal("missing workflow.delegation timeline reason")
	}
}

func newCollabManager(t *testing.T, envPrefix string) *runtimeconfig.Manager {
	t.Helper()
	t.Setenv(envPrefix+"_COMPOSER_COLLAB_ENABLED", "true")
	mgr, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{EnvPrefix: envPrefix})
	if err != nil {
		t.Fatalf("new runtime manager: %v", err)
	}
	return mgr
}

func newCollabModel() *fakes.Model {
	model := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	model.SetStream([]types.ModelEvent{{Type: types.ModelEventTypeFinalAnswer, TextDelta: "ok"}}, nil)
	return model
}

func newCollabA2AClient(t *testing.T) *a2a.Client {
	t.Helper()
	server := a2a.NewInMemoryServer(a2a.HandlerFunc(func(context.Context, a2a.TaskRequest) (map[string]any, error) {
		return map[string]any{"ok": true}, nil
	}), nil)
	return a2a.NewClient(server, nil, nil, a2a.ClientPolicy{
		Timeout:            300 * time.Millisecond,
		RequestMaxAttempts: 1,
		AsyncReporting: a2a.AsyncReportingPolicy{
			Enabled: true,
			Retry: a2a.AsyncReportingRetryPolicy{
				MaxAttempts:    2,
				BackoffInitial: time.Millisecond,
				BackoffMax:     2 * time.Millisecond,
			},
			JitterRatio: 0,
		},
	}, nil)
}

func waitForTaskTerminalState(t *testing.T, s *scheduler.Scheduler, taskID string, expected scheduler.TaskState) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		record, ok, err := s.Get(context.Background(), taskID)
		if err == nil && ok {
			if record.State == expected {
				return
			}
			if record.State == scheduler.TaskStateFailed || record.State == scheduler.TaskStateDeadLetter {
				t.Fatalf("task %s reached unexpected terminal state %q", taskID, record.State)
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for task %s to reach state %q", taskID, expected)
}

func writeCollabRecoveryRuntimeConfig(t *testing.T, path, recoveryPath string) {
	t.Helper()
	content := strings.Join([]string{
		"reload:",
		"  enabled: false",
		"composer:",
		"  collab:",
		"    enabled: true",
		"    default_aggregation: all_settled",
		"    failure_policy: fail_fast",
		"    retry:",
		"      enabled: false",
		"scheduler:",
		"  enabled: true",
		"  backend: memory",
		"  lease_timeout: 2s",
		"  heartbeat_interval: 400ms",
		"  queue_limit: 1024",
		"  retry_max_attempts: 3",
		"recovery:",
		"  enabled: true",
		"  backend: file",
		"  path: " + filepath.ToSlash(recoveryPath),
		"  conflict_policy: fail_fast",
		"subagent:",
		"  max_depth: 4",
		"  max_active_children: 8",
		"  child_timeout_budget: 5s",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write recovery runtime config: %v", err)
	}
}
