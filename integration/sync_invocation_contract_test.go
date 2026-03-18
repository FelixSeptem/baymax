package integration

import (
	"context"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/a2a"
	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/integration/fakes"
	"github.com/FelixSeptem/baymax/orchestration/composer"
	"github.com/FelixSeptem/baymax/orchestration/scheduler"
	"github.com/FelixSeptem/baymax/orchestration/teams"
	"github.com/FelixSeptem/baymax/orchestration/workflow"
)

func TestSyncInvocationContractWorkflowTeamsSchedulerComposerConsistency(t *testing.T) {
	timeline := &a2aTimelineCollector{}
	server := a2a.NewInMemoryServer(a2a.HandlerFunc(func(_ context.Context, req a2a.TaskRequest) (map[string]any, error) {
		return map[string]any{
			"ok":          true,
			"vote":        "yes",
			"workflow_id": req.WorkflowID,
			"team_id":     req.TeamID,
		}, nil
	}), timeline)
	client := a2a.NewClient(server, []a2a.AgentCard{
		{
			AgentID:                "agent-remote",
			PeerID:                 "peer-remote",
			SchemaVersion:          "a2a.v1.0",
			SupportedDeliveryModes: []string{a2a.DeliveryModeCallback},
		},
	}, a2a.DeterministicRouter{RequireAll: true}, a2a.ClientPolicy{
		Timeout:            400 * time.Millisecond,
		RequestMaxAttempts: 1,
	}, timeline)

	workflowEngine := workflow.New(
		workflow.WithStepAdapter(workflow.DispatchAdapter{
			A2A: workflow.NewA2AStepAdapter(client, workflow.A2AStepAdapterOptions{
				Method:       "workflow.delegate",
				PollInterval: 5 * time.Millisecond,
			}),
		}),
	)
	workflowRes, err := workflowEngine.Run(context.Background(), workflow.RunRequest{
		RunID: "run-a11-workflow",
		DSL: workflow.Definition{
			WorkflowID: "wf-a11",
			Steps: []workflow.Step{
				{
					StepID:  "remote-step",
					TaskID:  "remote-task",
					Kind:    workflow.StepKindA2A,
					TeamID:  "team-a11",
					AgentID: "agent-main",
					PeerID:  "peer-remote",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("workflow run failed: %v", err)
	}
	if workflowRes.WorkflowStatus != "succeeded" || workflowRes.WorkflowRemoteTotal != 1 || workflowRes.WorkflowRemoteFailed != 0 {
		t.Fatalf("workflow aggregate mismatch: %#v", workflowRes)
	}

	teamsEngine := teams.New()
	teamsRes, err := teamsEngine.Run(context.Background(), teams.Plan{
		RunID:      "run-a11-teams",
		TeamID:     "team-a11",
		WorkflowID: "wf-a11",
		StepID:     "remote-step",
		Strategy:   teams.StrategyVote,
		Tasks: []teams.Task{
			{
				TaskID:  "local",
				AgentID: "agent-local",
				Role:    teams.RoleLeader,
				Target:  teams.TaskTargetLocal,
				Runner: teams.TaskRunnerFunc(func(context.Context, teams.Task) (teams.TaskResult, error) {
					return teams.TaskResult{Vote: "yes", Output: "local"}, nil
				}),
			},
			{
				TaskID:  "remote",
				AgentID: "agent-main",
				Role:    teams.RoleWorker,
				Target:  teams.TaskTargetRemote,
				Remote: teams.RemoteTarget{
					PeerID: "peer-remote",
					Method: "team.delegate",
					Payload: map[string]any{
						"intent": "review",
					},
				},
				RemoteRunner: teams.NewA2ARemoteTaskRunner(client, teams.A2ARemoteRunnerOptions{
					PollInterval: 5 * time.Millisecond,
				}),
			},
		},
	})
	if err != nil {
		t.Fatalf("teams run failed: %v", err)
	}
	if teamsRes.TeamRemoteTotal != 1 || teamsRes.TeamRemoteFailed != 0 || teamsRes.WinnerVote != "yes" {
		t.Fatalf("teams aggregate mismatch: %#v", teamsRes)
	}

	s, err := scheduler.New(scheduler.NewMemoryStore(), scheduler.WithLeaseTimeout(500*time.Millisecond))
	if err != nil {
		t.Fatalf("new scheduler: %v", err)
	}
	if _, err := s.Enqueue(context.Background(), scheduler.Task{
		TaskID:     "task-a11-scheduler",
		RunID:      "run-a11-scheduler",
		WorkflowID: "wf-a11",
		TeamID:     "team-a11",
		StepID:     "remote-step",
		AgentID:    "agent-main",
		PeerID:     "peer-remote",
		Payload:    map[string]any{"q": "ping"},
	}); err != nil {
		t.Fatalf("scheduler enqueue failed: %v", err)
	}
	claimed, ok, err := s.Claim(context.Background(), "worker-a11")
	if err != nil || !ok {
		t.Fatalf("scheduler claim failed: ok=%v err=%v", ok, err)
	}
	exec, err := scheduler.ExecuteClaimWithA2A(context.Background(), client, claimed, 5*time.Millisecond)
	if err != nil {
		t.Fatalf("scheduler execute failed: %v", err)
	}
	if exec.Commit.Status != scheduler.TaskStateSucceeded || exec.Retryable {
		t.Fatalf("scheduler execution mismatch: %#v", exec)
	}

	model := fakes.NewModel([]fakes.ModelStep{{Response: types.ModelResponse{FinalAnswer: "ok"}}})
	comp, err := composer.NewBuilder(model).WithA2AClient(client).Build()
	if err != nil {
		t.Fatalf("new composer: %v", err)
	}
	dispatched, err := comp.DispatchChild(context.Background(), composer.ChildDispatchRequest{
		Task: scheduler.Task{
			TaskID:     "task-a11-composer",
			RunID:      "run-a11-composer",
			WorkflowID: "wf-a11",
			TeamID:     "team-a11",
			StepID:     "remote-step",
			AgentID:    "agent-main",
			PeerID:     "peer-remote",
			Payload:    map[string]any{"q": "composer"},
		},
		Target:               composer.ChildTargetA2A,
		ParentDepth:          0,
		ParentActiveChildren: 0,
		ChildTimeout:         300 * time.Millisecond,
		PollInterval:         5 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("composer dispatch child failed: %v", err)
	}
	if dispatched.Commit.Status != scheduler.TaskStateSucceeded || dispatched.Retryable {
		t.Fatalf("composer child execution mismatch: %#v", dispatched)
	}
}

func TestSyncInvocationContractRunStreamRemoteAggregateEquivalence(t *testing.T) {
	timeline := &a2aTimelineCollector{}
	server := a2a.NewInMemoryServer(a2a.HandlerFunc(func(_ context.Context, req a2a.TaskRequest) (map[string]any, error) {
		return map[string]any{"vote": "yes", "peer": req.PeerID}, nil
	}), timeline)
	client := a2a.NewClient(server, []a2a.AgentCard{
		{
			AgentID:                "agent-remote",
			PeerID:                 "peer-remote",
			SchemaVersion:          "a2a.v1.0",
			SupportedDeliveryModes: []string{a2a.DeliveryModeCallback},
		},
	}, a2a.DeterministicRouter{RequireAll: true}, a2a.ClientPolicy{
		Timeout:            300 * time.Millisecond,
		RequestMaxAttempts: 1,
	}, timeline)

	workflowReq := workflow.RunRequest{
		RunID: "run-a11-eq-workflow",
		DSL: workflow.Definition{
			WorkflowID: "wf-a11-eq",
			Steps: []workflow.Step{
				{
					StepID:  "remote",
					TaskID:  "task-remote",
					Kind:    workflow.StepKindA2A,
					TeamID:  "team-a11-eq",
					AgentID: "agent-main",
					PeerID:  "peer-remote",
				},
			},
		},
	}
	workflowEngine := workflow.New(
		workflow.WithStepAdapter(workflow.DispatchAdapter{
			A2A: workflow.NewA2AStepAdapter(client, workflow.A2AStepAdapterOptions{
				PollInterval: 5 * time.Millisecond,
			}),
		}),
	)
	workflowRun, runErr := workflowEngine.Run(context.Background(), workflowReq)
	workflowStream, streamErr := workflowEngine.Stream(context.Background(), workflowReq, func(workflow.StreamEvent) error { return nil })
	if runErr != nil || streamErr != nil {
		t.Fatalf("workflow run/stream errors mismatch run=%v stream=%v", runErr, streamErr)
	}
	if workflowRun.WorkflowRemoteTotal != workflowStream.WorkflowRemoteTotal || workflowRun.WorkflowRemoteFailed != workflowStream.WorkflowRemoteFailed {
		t.Fatalf("workflow remote aggregate mismatch run=%#v stream=%#v", workflowRun, workflowStream)
	}

	plan := teams.Plan{
		RunID:      "run-a11-eq-teams",
		TeamID:     "team-a11-eq",
		WorkflowID: "wf-a11-eq",
		StepID:     "remote",
		Strategy:   teams.StrategyVote,
		Tasks: []teams.Task{
			{
				TaskID:  "local",
				AgentID: "agent-local",
				Role:    teams.RoleLeader,
				Target:  teams.TaskTargetLocal,
				Runner: teams.TaskRunnerFunc(func(context.Context, teams.Task) (teams.TaskResult, error) {
					return teams.TaskResult{Vote: "yes"}, nil
				}),
			},
			{
				TaskID:  "remote",
				AgentID: "agent-main",
				Role:    teams.RoleWorker,
				Target:  teams.TaskTargetRemote,
				Remote: teams.RemoteTarget{
					PeerID: "peer-remote",
				},
				RemoteRunner: teams.NewA2ARemoteTaskRunner(client, teams.A2ARemoteRunnerOptions{
					PollInterval: 5 * time.Millisecond,
				}),
			},
		},
	}
	teamsEngine := teams.New()
	teamsRun, runErr := teamsEngine.Run(context.Background(), plan)
	teamsStream, streamErr := teamsEngine.Stream(context.Background(), plan, func(teams.StreamEvent) error { return nil })
	if runErr != nil || streamErr != nil {
		t.Fatalf("teams run/stream errors mismatch run=%v stream=%v", runErr, streamErr)
	}
	if teamsRun.TeamRemoteTotal != teamsStream.TeamRemoteTotal || teamsRun.TeamRemoteFailed != teamsStream.TeamRemoteFailed {
		t.Fatalf("teams remote aggregate mismatch run=%#v stream=%#v", teamsRun, teamsStream)
	}
}

type canceledTerminalA2AClient struct{}

func (canceledTerminalA2AClient) Submit(_ context.Context, req a2a.TaskRequest) (a2a.TaskRecord, error) {
	return a2a.TaskRecord{TaskID: req.TaskID, Status: a2a.StatusSubmitted}, nil
}

func (canceledTerminalA2AClient) WaitResult(
	_ context.Context,
	taskID string,
	_ time.Duration,
	_ func(context.Context, a2a.TaskRecord) error,
) (a2a.TaskRecord, error) {
	return a2a.TaskRecord{
		TaskID:        taskID,
		Status:        a2a.StatusCanceled,
		ErrorMessage:  "canceled by peer",
		A2AErrorLayer: string(a2a.ErrorLayerProtocol),
	}, nil
}

func TestSyncInvocationContractSchedulerCanceledTerminalMappingAndRetryable(t *testing.T) {
	s, err := scheduler.New(scheduler.NewMemoryStore(), scheduler.WithLeaseTimeout(300*time.Millisecond))
	if err != nil {
		t.Fatalf("new scheduler: %v", err)
	}
	if _, err := s.Enqueue(context.Background(), scheduler.Task{
		TaskID:  "task-a11-canceled",
		RunID:   "run-a11-canceled",
		AgentID: "agent-main",
		PeerID:  "peer-remote",
	}); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	claimed, ok, err := s.Claim(context.Background(), "worker-a11-canceled")
	if err != nil || !ok {
		t.Fatalf("claim failed: ok=%v err=%v", ok, err)
	}
	exec, err := scheduler.ExecuteClaimWithA2A(context.Background(), canceledTerminalA2AClient{}, claimed, 5*time.Millisecond)
	if err != nil {
		t.Fatalf("canceled terminal should not return execution error, got %v", err)
	}
	if exec.Commit.Status != scheduler.TaskStateFailed {
		t.Fatalf("commit status = %q, want failed", exec.Commit.Status)
	}
	if exec.Retryable {
		t.Fatalf("protocol canceled terminal should not be retryable: %#v", exec)
	}
}
