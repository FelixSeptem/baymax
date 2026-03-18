package teams

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

type timelineCollector struct {
	events []types.Event
}

func (c *timelineCollector) OnEvent(_ context.Context, ev types.Event) {
	c.events = append(c.events, ev)
}

func TestRunSerialLifecycleWithFailFast(t *testing.T) {
	engine := New()
	plan := Plan{
		TeamID:        "team-lifecycle",
		Strategy:      StrategySerial,
		FailurePolicy: FailurePolicyFailFast,
		Tasks: []Task{
			{
				TaskID:  "t-1",
				AgentID: "a-1",
				Role:    RoleLeader,
				Runner: TaskRunnerFunc(func(ctx context.Context, task Task) (TaskResult, error) {
					return TaskResult{Output: "ok"}, nil
				}),
			},
			{
				TaskID:  "t-2",
				AgentID: "a-2",
				Role:    RoleWorker,
				Runner: TaskRunnerFunc(func(ctx context.Context, task Task) (TaskResult, error) {
					return TaskResult{}, errors.New("boom")
				}),
			},
			{
				TaskID:  "t-3",
				AgentID: "a-3",
				Role:    RoleCoordinator,
				Runner: TaskRunnerFunc(func(ctx context.Context, task Task) (TaskResult, error) {
					return TaskResult{Output: "should-not-run"}, nil
				}),
			},
		},
	}

	res, err := engine.Run(context.Background(), plan)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(res.Tasks) != 3 {
		t.Fatalf("task count = %d, want 3", len(res.Tasks))
	}
	if res.Tasks[0].Status != TaskStatusSucceeded {
		t.Fatalf("task-1 status = %q, want succeeded", res.Tasks[0].Status)
	}
	if res.Tasks[1].Status != TaskStatusFailed {
		t.Fatalf("task-2 status = %q, want failed", res.Tasks[1].Status)
	}
	if res.Tasks[2].Status != TaskStatusSkipped || res.Tasks[2].Reason != "policy.fail_fast" {
		t.Fatalf("task-3 status/reason mismatch: %#v", res.Tasks[2])
	}
}

func TestParallelCancellationPropagation(t *testing.T) {
	engine := New()
	plan := Plan{
		TeamID:             "team-cancel",
		Strategy:           StrategyParallel,
		ParallelMaxWorkers: 2,
		Tasks: []Task{
			{
				TaskID:  "t-1",
				AgentID: "a-1",
				Role:    RoleWorker,
				Runner: TaskRunnerFunc(func(ctx context.Context, task Task) (TaskResult, error) {
					<-ctx.Done()
					return TaskResult{}, ctx.Err()
				}),
			},
			{
				TaskID:  "t-2",
				AgentID: "a-2",
				Role:    RoleWorker,
				Runner: TaskRunnerFunc(func(ctx context.Context, task Task) (TaskResult, error) {
					<-ctx.Done()
					return TaskResult{}, ctx.Err()
				}),
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()
	res, err := engine.Run(ctx, plan)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run error = %v, want context.Canceled", err)
	}
	for _, task := range res.Tasks {
		if task.Status != TaskStatusCanceled {
			t.Fatalf("task status = %q, want canceled; task=%#v", task.Status, task)
		}
		if task.Reason != "cancel.propagated" {
			t.Fatalf("task reason = %q, want cancel.propagated", task.Reason)
		}
	}
}

func TestParallelMixedLocalAndRemoteExecution(t *testing.T) {
	engine := New()
	plan := Plan{
		TeamID:             "team-mixed",
		Strategy:           StrategyParallel,
		ParallelMaxWorkers: 2,
		Tasks: []Task{
			{
				TaskID:  "local-1",
				AgentID: "agent-local",
				Role:    RoleWorker,
				Target:  TaskTargetLocal,
				Runner: TaskRunnerFunc(func(ctx context.Context, task Task) (TaskResult, error) {
					return TaskResult{Output: "local-ok"}, nil
				}),
			},
			{
				TaskID:  "remote-1",
				AgentID: "agent-remote",
				Role:    RoleWorker,
				Target:  TaskTargetRemote,
				Remote: RemoteTarget{
					PeerID: "peer-1",
					Method: "delegate",
				},
				RemoteRunner: RemoteTaskRunnerFunc(func(ctx context.Context, plan Plan, task Task) (TaskResult, error) {
					return TaskResult{Output: "remote-ok"}, nil
				}),
			},
		},
	}

	res, err := engine.Run(context.Background(), plan)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(res.Tasks) != 2 {
		t.Fatalf("task count = %d, want 2", len(res.Tasks))
	}
	if res.TeamRemoteTotal != 1 || res.TeamRemoteFailed != 0 {
		t.Fatalf("remote aggregate mismatch: %#v", res)
	}
	byID := map[string]TaskRecord{}
	for _, item := range res.Tasks {
		byID[item.TaskID] = item
	}
	if byID["local-1"].Status != TaskStatusSucceeded || byID["local-1"].Target != TaskTargetLocal {
		t.Fatalf("local task mismatch: %#v", byID["local-1"])
	}
	if byID["remote-1"].Status != TaskStatusSucceeded || byID["remote-1"].Target != TaskTargetRemote || byID["remote-1"].PeerID != "peer-1" {
		t.Fatalf("remote task mismatch: %#v", byID["remote-1"])
	}
}

func TestMixedCancellationConvergence(t *testing.T) {
	engine := New()
	plan := Plan{
		TeamID:             "team-mixed-cancel",
		Strategy:           StrategyParallel,
		ParallelMaxWorkers: 2,
		Tasks: []Task{
			{
				TaskID:  "local-cancel",
				AgentID: "agent-local",
				Role:    RoleWorker,
				Target:  TaskTargetLocal,
				Runner: TaskRunnerFunc(func(ctx context.Context, task Task) (TaskResult, error) {
					<-ctx.Done()
					return TaskResult{}, ctx.Err()
				}),
			},
			{
				TaskID:  "remote-cancel",
				AgentID: "agent-remote",
				Role:    RoleWorker,
				Target:  TaskTargetRemote,
				Remote: RemoteTarget{
					PeerID: "peer-cancel",
				},
				RemoteRunner: RemoteTaskRunnerFunc(func(ctx context.Context, plan Plan, task Task) (TaskResult, error) {
					<-ctx.Done()
					return TaskResult{}, ctx.Err()
				}),
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()
	res, err := engine.Run(ctx, plan)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run error = %v, want context.Canceled", err)
	}
	for _, task := range res.Tasks {
		if task.Status != TaskStatusCanceled {
			t.Fatalf("task status = %q, want canceled; task=%#v", task.Status, task)
		}
		if task.Reason != "cancel.propagated" {
			t.Fatalf("task reason = %q, want cancel.propagated", task.Reason)
		}
	}
}

func TestVoteDeterministicTieBreak(t *testing.T) {
	engine := New()
	buildPlan := func() Plan {
		return Plan{
			TeamID:       "team-vote",
			Strategy:     StrategyVote,
			VoteTieBreak: VoteTieBreakHighestPriority,
			Tasks: []Task{
				{
					TaskID:   "task-a",
					AgentID:  "agent-a",
					Role:     RoleWorker,
					Priority: 1,
					Runner: TaskRunnerFunc(func(ctx context.Context, task Task) (TaskResult, error) {
						return TaskResult{Vote: "approve"}, nil
					}),
				},
				{
					TaskID:   "task-b",
					AgentID:  "agent-b",
					Role:     RoleWorker,
					Priority: 5,
					Runner: TaskRunnerFunc(func(ctx context.Context, task Task) (TaskResult, error) {
						return TaskResult{Vote: "reject"}, nil
					}),
				},
			},
		}
	}

	first, err := engine.Run(context.Background(), buildPlan())
	if err != nil {
		t.Fatalf("first vote run failed: %v", err)
	}
	second, err := engine.Run(context.Background(), buildPlan())
	if err != nil {
		t.Fatalf("second vote run failed: %v", err)
	}
	if first.WinnerVote != "reject" {
		t.Fatalf("winner_vote = %q, want reject", first.WinnerVote)
	}
	if second.WinnerVote != first.WinnerVote {
		t.Fatalf("vote winner should be deterministic, first=%q second=%q", first.WinnerVote, second.WinnerVote)
	}
}

func TestRunAndStreamSemanticEquivalence(t *testing.T) {
	engine := New()
	plan := Plan{
		TeamID:       "team-equivalence",
		Strategy:     StrategyVote,
		VoteTieBreak: VoteTieBreakFirstTaskID,
		Tasks: []Task{
			{
				TaskID:  "t-1",
				AgentID: "a-1",
				Role:    RoleLeader,
				Runner: TaskRunnerFunc(func(ctx context.Context, task Task) (TaskResult, error) {
					return TaskResult{Output: "r1", Vote: "yes"}, nil
				}),
			},
			{
				TaskID:  "t-2",
				AgentID: "a-2",
				Role:    RoleWorker,
				Runner: TaskRunnerFunc(func(ctx context.Context, task Task) (TaskResult, error) {
					return TaskResult{Output: "r2", Vote: "yes"}, nil
				}),
			},
		},
	}

	runRes, runErr := engine.Run(context.Background(), plan)
	if runErr != nil {
		t.Fatalf("Run returned error: %v", runErr)
	}

	streamEvents := 0
	streamRes, streamErr := engine.Stream(context.Background(), plan, func(ev StreamEvent) error {
		streamEvents++
		return nil
	})
	if streamErr != nil {
		t.Fatalf("Stream returned error: %v", streamErr)
	}
	if streamEvents == 0 {
		t.Fatal("Stream should emit events")
	}
	if runRes.WinnerVote != streamRes.WinnerVote {
		t.Fatalf("winner mismatch run=%q stream=%q", runRes.WinnerVote, streamRes.WinnerVote)
	}
	if runRes.TeamTaskTotal != streamRes.TeamTaskTotal || runRes.TeamTaskFailed != streamRes.TeamTaskFailed || runRes.TeamTaskCanceled != streamRes.TeamTaskCanceled {
		t.Fatalf("aggregate mismatch run=%#v stream=%#v", runRes, streamRes)
	}
	for i := range runRes.Tasks {
		if runRes.Tasks[i].TaskID != streamRes.Tasks[i].TaskID || runRes.Tasks[i].Status != streamRes.Tasks[i].Status {
			t.Fatalf("task[%d] mismatch run=%#v stream=%#v", i, runRes.Tasks[i], streamRes.Tasks[i])
		}
	}
}

func TestMixedRunAndStreamSemanticEquivalence(t *testing.T) {
	engine := New()
	plan := Plan{
		RunID:        "run-mixed-eq",
		TeamID:       "team-mixed-equivalence",
		WorkflowID:   "wf-mixed",
		StepID:       "wf-step-mixed",
		Strategy:     StrategyVote,
		VoteTieBreak: VoteTieBreakFirstTaskID,
		Tasks: []Task{
			{
				TaskID:  "local-1",
				AgentID: "agent-local",
				Role:    RoleLeader,
				Target:  TaskTargetLocal,
				Runner: TaskRunnerFunc(func(ctx context.Context, task Task) (TaskResult, error) {
					return TaskResult{Vote: "yes", Output: "local"}, nil
				}),
			},
			{
				TaskID:  "remote-1",
				AgentID: "agent-remote",
				Role:    RoleWorker,
				Target:  TaskTargetRemote,
				Remote: RemoteTarget{
					PeerID: "peer-eq",
				},
				RemoteRunner: RemoteTaskRunnerFunc(func(ctx context.Context, plan Plan, task Task) (TaskResult, error) {
					return TaskResult{Vote: "yes", Output: "remote"}, nil
				}),
			},
		},
	}

	runRes, runErr := engine.Run(context.Background(), plan)
	if runErr != nil {
		t.Fatalf("Run returned error: %v", runErr)
	}
	streamEvents := 0
	streamRes, streamErr := engine.Stream(context.Background(), plan, func(ev StreamEvent) error {
		streamEvents++
		return nil
	})
	if streamErr != nil {
		t.Fatalf("Stream returned error: %v", streamErr)
	}
	if streamEvents == 0 {
		t.Fatal("Stream should emit events")
	}
	if runRes.WinnerVote != streamRes.WinnerVote {
		t.Fatalf("winner mismatch run=%q stream=%q", runRes.WinnerVote, streamRes.WinnerVote)
	}
	if runRes.TeamTaskTotal != streamRes.TeamTaskTotal ||
		runRes.TeamTaskFailed != streamRes.TeamTaskFailed ||
		runRes.TeamTaskCanceled != streamRes.TeamTaskCanceled ||
		runRes.TeamRemoteTotal != streamRes.TeamRemoteTotal ||
		runRes.TeamRemoteFailed != streamRes.TeamRemoteFailed {
		t.Fatalf("aggregate mismatch run=%#v stream=%#v", runRes, streamRes)
	}
}

func TestTimelineEventsContainTeamMetadataAndReasons(t *testing.T) {
	collector := &timelineCollector{}
	engine := New(WithTimelineEmitter(collector))
	plan := Plan{
		RunID:    "run-team",
		TeamID:   "team-metadata",
		Strategy: StrategySerial,
		Tasks: []Task{
			{
				TaskID:  "t-1",
				AgentID: "a-1",
				Role:    RoleWorker,
				Runner: TaskRunnerFunc(func(ctx context.Context, task Task) (TaskResult, error) {
					return TaskResult{Output: "ok"}, nil
				}),
			},
		},
	}

	if _, err := engine.Run(context.Background(), plan); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(collector.events) < 3 {
		t.Fatalf("timeline event count = %d, want >= 3", len(collector.events))
	}

	reasons := map[string]bool{}
	for _, ev := range collector.events {
		if ev.Type != types.EventTypeActionTimeline {
			continue
		}
		reason, _ := ev.Payload["reason"].(string)
		reasons[reason] = true
		if !strings.HasPrefix(reason, "team.") {
			t.Fatalf("timeline reason namespace mismatch: %q", reason)
		}
		teamID, _ := ev.Payload["team_id"].(string)
		if teamID != "team-metadata" {
			t.Fatalf("team_id = %q, want team-metadata", teamID)
		}
		if reason == ReasonDispatch || reason == ReasonCollect {
			if ev.Payload["task_id"] != "t-1" || ev.Payload["agent_id"] != "a-1" {
				t.Fatalf("dispatch/collect metadata mismatch: %#v", ev.Payload)
			}
		}
	}
	for _, reason := range []string{ReasonDispatch, ReasonCollect, ReasonResolve} {
		if !reasons[reason] {
			t.Fatalf("missing timeline reason %q", reason)
		}
	}
}

func TestTimelineEventsContainRemoteReasonsAndCorrelation(t *testing.T) {
	collector := &timelineCollector{}
	engine := New(WithTimelineEmitter(collector))
	plan := Plan{
		RunID:      "run-team-remote",
		TeamID:     "team-remote",
		WorkflowID: "wf-team-remote",
		StepID:     "wf-step-remote",
		Strategy:   StrategySerial,
		Tasks: []Task{
			{
				TaskID:  "t-remote",
				AgentID: "a-remote",
				Role:    RoleWorker,
				Target:  TaskTargetRemote,
				Remote: RemoteTarget{
					PeerID: "peer-remote",
				},
				RemoteRunner: RemoteTaskRunnerFunc(func(ctx context.Context, plan Plan, task Task) (TaskResult, error) {
					return TaskResult{Output: "ok"}, nil
				}),
			},
		},
	}

	if _, err := engine.Run(context.Background(), plan); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	reasons := map[string]bool{}
	for _, ev := range collector.events {
		if ev.Type != types.EventTypeActionTimeline {
			continue
		}
		reason, _ := ev.Payload["reason"].(string)
		reasons[reason] = true
		if reason == ReasonDispatchRemote || reason == ReasonCollectRemote {
			if ev.Payload["team_id"] != "team-remote" ||
				ev.Payload["workflow_id"] != "wf-team-remote" ||
				ev.Payload["step_id"] != "wf-step-remote" ||
				ev.Payload["task_id"] != "t-remote" ||
				ev.Payload["agent_id"] != "a-remote" ||
				ev.Payload["peer_id"] != "peer-remote" {
				t.Fatalf("remote timeline metadata mismatch: %#v", ev.Payload)
			}
		}
	}
	for _, reason := range []string{ReasonDispatchRemote, ReasonCollectRemote, ReasonResolve} {
		if !reasons[reason] {
			t.Fatalf("missing timeline reason %q", reason)
		}
	}
}
