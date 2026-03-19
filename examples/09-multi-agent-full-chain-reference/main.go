package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/FelixSeptem/baymax/a2a"
	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/orchestration/composer"
	"github.com/FelixSeptem/baymax/orchestration/scheduler"
	"github.com/FelixSeptem/baymax/orchestration/teams"
	"github.com/FelixSeptem/baymax/orchestration/workflow"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

type stubModel struct{}

func (stubModel) Generate(ctx context.Context, req types.ModelRequest) (types.ModelResponse, error) {
	_ = ctx
	_ = req
	return types.ModelResponse{FinalAnswer: "a20 full-chain local child-run completed"}, nil
}

func (stubModel) Stream(ctx context.Context, req types.ModelRequest, onEvent func(types.ModelEvent) error) error {
	_ = ctx
	_ = req
	return onEvent(types.ModelEvent{
		Type:      types.ModelEventTypeFinalAnswer,
		TextDelta: "a20 full-chain stream local child-run completed",
	})
}

type pathSummary struct {
	WorkflowRemoteTotal  int    `json:"workflow_remote_total"`
	WorkflowRemoteFailed int    `json:"workflow_remote_failed"`
	TeamsRemoteTotal     int    `json:"teams_remote_total"`
	TeamsRemoteFailed    int    `json:"teams_remote_failed"`
	TeamsWinnerVote      string `json:"teams_winner_vote"`
}

type correlationIDs struct {
	RunWorkflowRunID    string `json:"run_workflow_run_id"`
	RunTeamsRunID       string `json:"run_teams_run_id"`
	StreamWorkflowRunID string `json:"stream_workflow_run_id"`
	StreamTeamsRunID    string `json:"stream_teams_run_id"`
	AsyncTaskID         string `json:"async_task_id"`
	DelayedTaskID       string `json:"delayed_task_id"`
	RecoveryRunID       string `json:"recovery_run_id"`
}

type summary struct {
	Mode                  string         `json:"mode"`
	RunPath               pathSummary    `json:"run_path"`
	StreamPath            pathSummary    `json:"stream_path"`
	StreamEventCount      int            `json:"stream_event_count"`
	RunStreamAligned      bool           `json:"run_stream_aligned"`
	AsyncReportSucceeded  bool           `json:"async_report_succeeded"`
	DelayedDispatchClaim  bool           `json:"delayed_dispatch_claimed"`
	RecoveryReplayed      bool           `json:"recovery_replayed"`
	RecoveryReplayCommits int            `json:"recovery_replay_commits"`
	Correlation           correlationIDs `json:"correlation"`
}

const (
	runWorkflowRunID    = "a20-run-workflow"
	runTeamsRunID       = "a20-run-teams"
	streamWorkflowRunID = "a20-stream-workflow"
	streamTeamsRunID    = "a20-stream-teams"
	asyncTaskID         = "a20-async-task"
	delayedTaskID       = "a20-delayed-task"
	delayedRunID        = "a20-delayed-run"
	recoveryRunID       = "a20-recovery-run"
	recoveryTaskID      = "a20-recovery-task"
)

func main() {
	mode := flag.String("mode", "both", "execution mode: run|stream|both")
	flag.Parse()

	switch strings.TrimSpace(strings.ToLower(*mode)) {
	case "run", "stream", "both":
	default:
		fmt.Fprintf(os.Stderr, "invalid mode %q; expected run|stream|both\n", *mode)
		os.Exit(2)
	}

	ctx := context.Background()
	client := newInMemoryClient()
	result, err := execute(ctx, client, strings.TrimSpace(strings.ToLower(*mode)))
	if err != nil {
		fmt.Fprintf(os.Stderr, "[a20] full-chain example failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("CHECKPOINT async_report_succeeded=%t\n", result.AsyncReportSucceeded)
	fmt.Printf("CHECKPOINT delayed_dispatch_claimed=%t\n", result.DelayedDispatchClaim)
	fmt.Printf("CHECKPOINT recovery_replayed=%t commits=%d\n", result.RecoveryReplayed, result.RecoveryReplayCommits)
	fmt.Printf(
		"CHECKPOINT correlation run_workflow_run_id=%s run_teams_run_id=%s stream_workflow_run_id=%s stream_teams_run_id=%s async_task_id=%s delayed_task_id=%s recovery_run_id=%s\n",
		result.Correlation.RunWorkflowRunID,
		result.Correlation.RunTeamsRunID,
		result.Correlation.StreamWorkflowRunID,
		result.Correlation.StreamTeamsRunID,
		result.Correlation.AsyncTaskID,
		result.Correlation.DelayedTaskID,
		result.Correlation.RecoveryRunID,
	)
	if result.Mode == "both" {
		fmt.Printf("CHECKPOINT run_stream_aligned=%t\n", result.RunStreamAligned)
	}
	fmt.Printf(
		"A20_RUN_TERMINAL workflow_remote_total=%d workflow_remote_failed=%d teams_remote_total=%d teams_remote_failed=%d winner_vote=%s\n",
		result.RunPath.WorkflowRemoteTotal,
		result.RunPath.WorkflowRemoteFailed,
		result.RunPath.TeamsRemoteTotal,
		result.RunPath.TeamsRemoteFailed,
		result.RunPath.TeamsWinnerVote,
	)
	fmt.Printf(
		"A20_STREAM_TERMINAL workflow_remote_total=%d workflow_remote_failed=%d teams_remote_total=%d teams_remote_failed=%d winner_vote=%s stream_events=%d\n",
		result.StreamPath.WorkflowRemoteTotal,
		result.StreamPath.WorkflowRemoteFailed,
		result.StreamPath.TeamsRemoteTotal,
		result.StreamPath.TeamsRemoteFailed,
		result.StreamPath.TeamsWinnerVote,
		result.StreamEventCount,
	)

	payload, err := json.Marshal(result)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[a20] marshal summary failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("A20_TERMINAL_SUMMARY=%s\n", payload)
	fmt.Println("A20_SUCCESS")
}

func execute(ctx context.Context, client *a2a.Client, mode string) (summary, error) {
	out := summary{
		Mode: mode,
		Correlation: correlationIDs{
			RunWorkflowRunID:    runWorkflowRunID,
			RunTeamsRunID:       runTeamsRunID,
			StreamWorkflowRunID: streamWorkflowRunID,
			StreamTeamsRunID:    streamTeamsRunID,
			AsyncTaskID:         asyncTaskID,
			DelayedTaskID:       delayedTaskID,
			RecoveryRunID:       recoveryRunID,
		},
	}
	var err error

	if mode == "run" || mode == "both" {
		out.RunPath, err = executeRunPath(ctx, client)
		if err != nil {
			return summary{}, err
		}
	}

	if mode == "stream" || mode == "both" {
		out.StreamPath, out.StreamEventCount, err = executeStreamPath(ctx, client)
		if err != nil {
			return summary{}, err
		}
	}

	if mode == "run" {
		out.StreamPath = out.RunPath
	}
	if mode == "stream" {
		out.RunPath = out.StreamPath
	}
	out.RunStreamAligned = out.RunPath == out.StreamPath

	out.AsyncReportSucceeded, err = executeAsyncCheckpoint(ctx, client)
	if err != nil {
		return summary{}, err
	}

	out.DelayedDispatchClaim, err = executeDelayedCheckpoint(ctx)
	if err != nil {
		return summary{}, err
	}

	out.RecoveryReplayCommits, err = executeRecoveryCheckpoint(ctx)
	if err != nil {
		return summary{}, err
	}
	out.RecoveryReplayed = out.RecoveryReplayCommits > 0

	if !out.AsyncReportSucceeded {
		return summary{}, fmt.Errorf("async reporting checkpoint failed")
	}
	if !out.DelayedDispatchClaim {
		return summary{}, fmt.Errorf("delayed dispatch checkpoint failed")
	}
	if !out.RecoveryReplayed {
		return summary{}, fmt.Errorf("recovery checkpoint failed")
	}
	if mode == "both" && !out.RunStreamAligned {
		return summary{}, fmt.Errorf("run/stream semantic alignment checkpoint failed")
	}

	return out, nil
}

func executeRunPath(ctx context.Context, client *a2a.Client) (pathSummary, error) {
	workflowEngine := workflow.New(
		workflow.WithStepAdapter(workflow.DispatchAdapter{
			A2A: workflow.NewA2AStepAdapter(client, workflow.A2AStepAdapterOptions{
				Method:       "workflow.delegate",
				PollInterval: 5 * time.Millisecond,
			}),
		}),
	)
	workflowRes, err := workflowEngine.Run(ctx, workflow.RunRequest{
		RunID: runWorkflowRunID,
		DSL: workflow.Definition{
			WorkflowID: "a20-workflow",
			Steps: []workflow.Step{
				{
					StepID:  "remote-step-run",
					TaskID:  "remote-task-run",
					Kind:    workflow.StepKindA2A,
					TeamID:  "a20-team",
					AgentID: "agent-main",
					PeerID:  "peer-remote",
				},
			},
		},
	})
	if err != nil {
		return pathSummary{}, fmt.Errorf("workflow run path failed: %w", err)
	}

	teamsEngine := teams.New()
	teamsRes, err := teamsEngine.Run(ctx, teams.Plan{
		RunID:      runTeamsRunID,
		TeamID:     "a20-team",
		WorkflowID: "a20-workflow",
		StepID:     "remote-step-run",
		Strategy:   teams.StrategyVote,
		Tasks: []teams.Task{
			{
				TaskID:  "local-run",
				AgentID: "agent-local",
				Role:    teams.RoleLeader,
				Target:  teams.TaskTargetLocal,
				Runner: teams.TaskRunnerFunc(func(context.Context, teams.Task) (teams.TaskResult, error) {
					return teams.TaskResult{Vote: "yes", Output: "local"}, nil
				}),
			},
			{
				TaskID:  "remote-run",
				AgentID: "agent-main",
				Role:    teams.RoleWorker,
				Target:  teams.TaskTargetRemote,
				Remote: teams.RemoteTarget{
					PeerID: "peer-remote",
					Method: "team.delegate",
				},
				RemoteRunner: teams.NewA2ARemoteTaskRunner(client, teams.A2ARemoteRunnerOptions{
					PollInterval: 5 * time.Millisecond,
				}),
			},
		},
	})
	if err != nil {
		return pathSummary{}, fmt.Errorf("teams run path failed: %w", err)
	}

	return pathSummary{
		WorkflowRemoteTotal:  workflowRes.WorkflowRemoteTotal,
		WorkflowRemoteFailed: workflowRes.WorkflowRemoteFailed,
		TeamsRemoteTotal:     teamsRes.TeamRemoteTotal,
		TeamsRemoteFailed:    teamsRes.TeamRemoteFailed,
		TeamsWinnerVote:      teamsRes.WinnerVote,
	}, nil
}

func executeStreamPath(ctx context.Context, client *a2a.Client) (pathSummary, int, error) {
	streamEvents := 0
	workflowEngine := workflow.New(
		workflow.WithStepAdapter(workflow.DispatchAdapter{
			A2A: workflow.NewA2AStepAdapter(client, workflow.A2AStepAdapterOptions{
				Method:       "workflow.delegate",
				PollInterval: 5 * time.Millisecond,
			}),
		}),
	)
	workflowRes, err := workflowEngine.Stream(ctx, workflow.RunRequest{
		RunID: streamWorkflowRunID,
		DSL: workflow.Definition{
			WorkflowID: "a20-workflow",
			Steps: []workflow.Step{
				{
					StepID:  "remote-step-stream",
					TaskID:  "remote-task-stream",
					Kind:    workflow.StepKindA2A,
					TeamID:  "a20-team",
					AgentID: "agent-main",
					PeerID:  "peer-remote",
				},
			},
		},
	}, func(workflow.StreamEvent) error {
		streamEvents++
		return nil
	})
	if err != nil {
		return pathSummary{}, 0, fmt.Errorf("workflow stream path failed: %w", err)
	}

	teamsEngine := teams.New()
	teamsRes, err := teamsEngine.Stream(ctx, teams.Plan{
		RunID:      streamTeamsRunID,
		TeamID:     "a20-team",
		WorkflowID: "a20-workflow",
		StepID:     "remote-step-stream",
		Strategy:   teams.StrategyVote,
		Tasks: []teams.Task{
			{
				TaskID:  "local-stream",
				AgentID: "agent-local",
				Role:    teams.RoleLeader,
				Target:  teams.TaskTargetLocal,
				Runner: teams.TaskRunnerFunc(func(context.Context, teams.Task) (teams.TaskResult, error) {
					return teams.TaskResult{Vote: "yes", Output: "local"}, nil
				}),
			},
			{
				TaskID:  "remote-stream",
				AgentID: "agent-main",
				Role:    teams.RoleWorker,
				Target:  teams.TaskTargetRemote,
				Remote: teams.RemoteTarget{
					PeerID: "peer-remote",
					Method: "team.delegate",
				},
				RemoteRunner: teams.NewA2ARemoteTaskRunner(client, teams.A2ARemoteRunnerOptions{
					PollInterval: 5 * time.Millisecond,
				}),
			},
		},
	}, func(teams.StreamEvent) error {
		streamEvents++
		return nil
	})
	if err != nil {
		return pathSummary{}, 0, fmt.Errorf("teams stream path failed: %w", err)
	}

	return pathSummary{
		WorkflowRemoteTotal:  workflowRes.WorkflowRemoteTotal,
		WorkflowRemoteFailed: workflowRes.WorkflowRemoteFailed,
		TeamsRemoteTotal:     teamsRes.TeamRemoteTotal,
		TeamsRemoteFailed:    teamsRes.TeamRemoteFailed,
		TeamsWinnerVote:      teamsRes.WinnerVote,
	}, streamEvents, nil
}

func executeAsyncCheckpoint(ctx context.Context, client *a2a.Client) (bool, error) {
	sink := a2a.NewChannelReportSink(2)
	ack, err := client.SubmitAsync(ctx, a2a.TaskRequest{
		TaskID:  asyncTaskID,
		AgentID: "agent-main",
		PeerID:  "peer-remote",
		Method:  "a20.async",
	}, sink)
	if err != nil {
		return false, fmt.Errorf("async submit failed: %w", err)
	}

	select {
	case report := <-sink.Channel():
		return report.TaskID == ack.TaskID && report.Status == a2a.StatusSucceeded, nil
	case <-time.After(2 * time.Second):
		return false, fmt.Errorf("async report checkpoint timeout")
	}
}

func executeDelayedCheckpoint(ctx context.Context) (bool, error) {
	store := scheduler.NewMemoryStore()
	s, err := scheduler.New(store, scheduler.WithLeaseTimeout(500*time.Millisecond))
	if err != nil {
		return false, fmt.Errorf("new scheduler failed: %w", err)
	}

	notBefore := time.Now().Add(400 * time.Millisecond)
	if _, err := s.Enqueue(ctx, scheduler.Task{
		TaskID:    delayedTaskID,
		RunID:     delayedRunID,
		NotBefore: notBefore,
	}); err != nil {
		return false, fmt.Errorf("enqueue delayed task failed: %w", err)
	}
	if _, ok, err := s.Claim(ctx, "a20-delayed-worker"); err != nil {
		return false, fmt.Errorf("early claim failed: %w", err)
	} else if ok {
		return false, nil
	}

	time.Sleep(450 * time.Millisecond)
	claimed, ok, err := s.Claim(ctx, "a20-delayed-worker")
	if err != nil {
		return false, fmt.Errorf("delayed claim failed: %w", err)
	}
	if !ok {
		return false, nil
	}
	if _, err := s.Complete(ctx, scheduler.TerminalCommit{
		TaskID:      claimed.Record.Task.TaskID,
		AttemptID:   claimed.Attempt.AttemptID,
		Status:      scheduler.TaskStateSucceeded,
		CommittedAt: time.Now(),
		Result:      map[string]any{"ok": true},
	}); err != nil {
		return false, fmt.Errorf("delayed completion failed: %w", err)
	}
	return true, nil
}

func executeRecoveryCheckpoint(ctx context.Context) (int, error) {
	root := filepath.Join(os.TempDir(), "baymax-a20")
	if err := os.MkdirAll(root, 0o755); err != nil {
		return 0, fmt.Errorf("create temp root failed: %w", err)
	}
	cfgPath := filepath.Join(root, "runtime-a20.yaml")
	recoveryPath := filepath.Join(root, "recovery")
	if err := writeRecoveryRuntimeConfig(cfgPath, recoveryPath); err != nil {
		return 0, err
	}

	mgr1, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX_A20"})
	if err != nil {
		return 0, fmt.Errorf("new runtime manager #1 failed: %w", err)
	}
	comp1, err := composer.NewBuilder(stubModel{}).WithRuntimeManager(mgr1).Build()
	if err != nil {
		_ = mgr1.Close()
		return 0, fmt.Errorf("new composer #1 failed: %w", err)
	}

	_, err = comp1.DispatchChild(ctx, composer.ChildDispatchRequest{
		Task: scheduler.Task{
			TaskID: recoveryTaskID,
			RunID:  recoveryRunID,
		},
		Target:               composer.ChildTargetLocal,
		ParentDepth:          0,
		ParentActiveChildren: 0,
		ChildTimeout:         500 * time.Millisecond,
		LocalRunner: composer.LocalChildRunnerFunc(func(context.Context, scheduler.Task) (map[string]any, error) {
			return map[string]any{"ok": true}, nil
		}),
	})
	if err != nil {
		_ = mgr1.Close()
		return 0, fmt.Errorf("dispatch recovery child failed: %w", err)
	}
	_ = mgr1.Close()

	mgr2, err := runtimeconfig.NewManager(runtimeconfig.ManagerOptions{FilePath: cfgPath, EnvPrefix: "BAYMAX_A20"})
	if err != nil {
		return 0, fmt.Errorf("new runtime manager #2 failed: %w", err)
	}
	defer func() { _ = mgr2.Close() }()

	comp2, err := composer.NewBuilder(stubModel{}).
		WithRuntimeManager(mgr2).
		WithSchedulerStore(scheduler.NewMemoryStore()).
		Build()
	if err != nil {
		return 0, fmt.Errorf("new composer #2 failed: %w", err)
	}
	result, err := comp2.Recover(ctx, composer.RecoverRequest{RunID: recoveryRunID})
	if err != nil {
		return 0, fmt.Errorf("recover failed: %w", err)
	}
	return result.ReplayedTerminalCommits, nil
}

func writeRecoveryRuntimeConfig(path, recoveryPath string) error {
	content := strings.Join([]string{
		"reload:",
		"  enabled: false",
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
		"  resume_boundary: next_attempt_only",
		"  inflight_policy: no_rewind",
		"  timeout_reentry_policy: single_reentry_then_fail",
		"  timeout_reentry_max_per_task: 1",
		"subagent:",
		"  max_depth: 4",
		"  max_active_children: 8",
		"  child_timeout_budget: 5s",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return fmt.Errorf("write recovery config failed: %w", err)
	}
	return nil
}

func newInMemoryClient() *a2a.Client {
	server := a2a.NewInMemoryServer(a2a.HandlerFunc(func(_ context.Context, req a2a.TaskRequest) (map[string]any, error) {
		return map[string]any{
			"ok":          true,
			"vote":        "yes",
			"task_id":     req.TaskID,
			"workflow_id": req.WorkflowID,
			"team_id":     req.TeamID,
			"step_id":     req.StepID,
		}, nil
	}), nil)
	return a2a.NewClient(server, []a2a.AgentCard{
		{
			AgentID:                "agent-remote",
			PeerID:                 "peer-remote",
			SchemaVersion:          "a2a.v1.0",
			SupportedDeliveryModes: []string{a2a.DeliveryModeCallback},
			Capabilities:           []string{"workflow.delegate", "team.delegate"},
		},
	}, a2a.DeterministicRouter{RequireAll: true}, a2a.ClientPolicy{
		Timeout:            400 * time.Millisecond,
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
