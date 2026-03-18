package integration

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/a2a"
	"github.com/FelixSeptem/baymax/core/types"
	stdiomcp "github.com/FelixSeptem/baymax/mcp/stdio"
	"github.com/FelixSeptem/baymax/orchestration/teams"
	"github.com/FelixSeptem/baymax/orchestration/workflow"
)

func TestWorkflowA2ARemoteStepRunStreamContract(t *testing.T) {
	timeline := &a2aTimelineCollector{}
	server := a2a.NewInMemoryServer(a2a.HandlerFunc(func(ctx context.Context, req a2a.TaskRequest) (map[string]any, error) {
		return map[string]any{
			"status":      "ok",
			"workflow_id": req.WorkflowID,
			"team_id":     req.TeamID,
			"step_id":     req.StepID,
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
		Timeout:            300 * time.Millisecond,
		RequestMaxAttempts: 1,
	}, timeline)

	adapter := workflow.DispatchAdapter{
		A2A: func(ctx context.Context, workflowID string, step workflow.Step, attempt int) (workflow.StepOutput, error) {
			req := a2a.TaskRequest{
				TaskID:     fmt.Sprintf("%s-attempt-%d-%d", step.TaskID, attempt, time.Now().UnixNano()),
				WorkflowID: workflowID,
				TeamID:     step.TeamID,
				StepID:     step.StepID,
				AgentID:    step.AgentID,
				PeerID:     step.PeerID,
				Method:     "workflow.delegate",
				Payload:    step.Payload,
			}
			submitted, err := client.Submit(ctx, req)
			if err != nil {
				return workflow.StepOutput{}, err
			}
			record, err := client.WaitResult(ctx, submitted.TaskID, 5*time.Millisecond, nil)
			if err != nil {
				return workflow.StepOutput{}, err
			}
			if record.Status != a2a.StatusSucceeded {
				return workflow.StepOutput{}, fmt.Errorf("a2a task status %q", record.Status)
			}
			return workflow.StepOutput{Payload: record.Result}, nil
		},
	}
	engine := workflow.New(
		workflow.WithStepAdapter(adapter),
		workflow.WithTimelineEmitter(timeline),
	)

	req := workflow.RunRequest{
		RunID: "run-wf-a5",
		DSL: workflow.Definition{
			WorkflowID: "wf-a5",
			Steps: []workflow.Step{
				{
					StepID:  "step-remote",
					TaskID:  "task-remote",
					Kind:    workflow.StepKindA2A,
					TeamID:  "team-a5",
					AgentID: "agent-main",
					PeerID:  "peer-remote",
					Payload: map[string]any{"query": "ping"},
				},
			},
		},
	}

	runRes, runErr := engine.Run(context.Background(), req)
	streamRes, streamErr := engine.Stream(context.Background(), req, func(ev workflow.StreamEvent) error { return nil })
	if runErr != nil || streamErr != nil {
		t.Fatalf("run/stream errors mismatch run=%v stream=%v", runErr, streamErr)
	}
	if runRes.WorkflowStatus != "succeeded" || streamRes.WorkflowStatus != "succeeded" {
		t.Fatalf("workflow status mismatch run=%#v stream=%#v", runRes, streamRes)
	}
	if runRes.WorkflowRemoteTotal != 1 || runRes.WorkflowRemoteFailed != 0 {
		t.Fatalf("workflow remote aggregate mismatch: %#v", runRes)
	}
	if runRes.WorkflowRemoteTotal != streamRes.WorkflowRemoteTotal || runRes.WorkflowRemoteFailed != streamRes.WorkflowRemoteFailed {
		t.Fatalf("workflow remote run/stream mismatch run=%#v stream=%#v", runRes, streamRes)
	}

	seenWorkflowDispatch := false
	seenA2ASubmit := false
	for _, ev := range timeline.snapshot() {
		if ev.Type != types.EventTypeActionTimeline {
			continue
		}
		reason, _ := ev.Payload["reason"].(string)
		switch reason {
		case workflow.ReasonDispatchA2A:
			seenWorkflowDispatch = true
			if ev.Payload["workflow_id"] != "wf-a5" ||
				ev.Payload["team_id"] != "team-a5" ||
				ev.Payload["step_id"] != "step-remote" ||
				ev.Payload["task_id"] != "task-remote" ||
				ev.Payload["agent_id"] != "agent-main" ||
				ev.Payload["peer_id"] != "peer-remote" {
				t.Fatalf("workflow.dispatch_a2a metadata mismatch: %#v", ev.Payload)
			}
		case a2a.ReasonSubmit:
			seenA2ASubmit = true
		}
	}
	if !seenWorkflowDispatch {
		t.Fatalf("missing reason %q", workflow.ReasonDispatchA2A)
	}
	if !seenA2ASubmit {
		t.Fatalf("missing reason %q", a2a.ReasonSubmit)
	}
}

func TestTeamsMixedLocalRemoteRunStreamContract(t *testing.T) {
	timeline := &a2aTimelineCollector{}
	server := a2a.NewInMemoryServer(a2a.HandlerFunc(func(ctx context.Context, req a2a.TaskRequest) (map[string]any, error) {
		return map[string]any{"vote": "yes", "team_id": req.TeamID}, nil
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

	engine := teams.New(teams.WithTimelineEmitter(timeline))
	plan := teams.Plan{
		RunID:      "run-team-a5",
		TeamID:     "team-a5",
		WorkflowID: "wf-a5",
		StepID:     "step-team",
		Strategy:   teams.StrategyVote,
		Tasks: []teams.Task{
			{
				TaskID:  "local-task",
				AgentID: "agent-local",
				Role:    teams.RoleLeader,
				Target:  teams.TaskTargetLocal,
				Runner: teams.TaskRunnerFunc(func(ctx context.Context, task teams.Task) (teams.TaskResult, error) {
					return teams.TaskResult{Vote: "yes", Output: "local"}, nil
				}),
			},
			{
				TaskID:  "remote-task",
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
				RemoteRunner: teams.RemoteTaskRunnerFunc(func(ctx context.Context, plan teams.Plan, task teams.Task) (teams.TaskResult, error) {
					req := a2a.TaskRequest{
						TaskID:     fmt.Sprintf("%s-%d", task.TaskID, time.Now().UnixNano()),
						WorkflowID: plan.WorkflowID,
						TeamID:     plan.TeamID,
						StepID:     plan.StepID,
						AgentID:    task.AgentID,
						PeerID:     task.Remote.PeerID,
						Method:     task.Remote.Method,
						Payload:    task.Remote.Payload,
					}
					submitted, err := client.Submit(ctx, req)
					if err != nil {
						return teams.TaskResult{}, err
					}
					record, err := client.WaitResult(ctx, submitted.TaskID, 5*time.Millisecond, nil)
					if err != nil {
						return teams.TaskResult{}, err
					}
					if record.Status != a2a.StatusSucceeded {
						return teams.TaskResult{}, fmt.Errorf("remote status %q", record.Status)
					}
					return teams.TaskResult{Vote: "yes", Output: record.Result}, nil
				}),
			},
		},
	}

	runRes, runErr := engine.Run(context.Background(), plan)
	streamRes, streamErr := engine.Stream(context.Background(), plan, func(ev teams.StreamEvent) error { return nil })
	if runErr != nil || streamErr != nil {
		t.Fatalf("run/stream errors mismatch run=%v stream=%v", runErr, streamErr)
	}
	if runRes.WinnerVote != "yes" || streamRes.WinnerVote != "yes" {
		t.Fatalf("winner vote mismatch run=%#v stream=%#v", runRes, streamRes)
	}
	if runRes.TeamRemoteTotal != 1 || runRes.TeamRemoteFailed != 0 {
		t.Fatalf("teams remote aggregate mismatch: %#v", runRes)
	}
	if runRes.TeamRemoteTotal != streamRes.TeamRemoteTotal || runRes.TeamRemoteFailed != streamRes.TeamRemoteFailed {
		t.Fatalf("teams remote run/stream mismatch run=%#v stream=%#v", runRes, streamRes)
	}

	seenDispatchRemote := false
	seenCollectRemote := false
	for _, ev := range timeline.snapshot() {
		if ev.Type != types.EventTypeActionTimeline {
			continue
		}
		reason, _ := ev.Payload["reason"].(string)
		switch reason {
		case teams.ReasonDispatchRemote:
			seenDispatchRemote = true
			if ev.Payload["workflow_id"] != "wf-a5" ||
				ev.Payload["step_id"] != "step-team" ||
				ev.Payload["team_id"] != "team-a5" ||
				ev.Payload["task_id"] != "remote-task" ||
				ev.Payload["agent_id"] != "agent-main" ||
				ev.Payload["peer_id"] != "peer-remote" {
				t.Fatalf("team.dispatch_remote metadata mismatch: %#v", ev.Payload)
			}
		case teams.ReasonCollectRemote:
			seenCollectRemote = true
		}
	}
	if !seenDispatchRemote || !seenCollectRemote {
		t.Fatalf("missing remote reasons dispatch=%v collect=%v", seenDispatchRemote, seenCollectRemote)
	}
}

func TestComposedA2AAndMCPBoundaryRegression(t *testing.T) {
	timeline := &a2aTimelineCollector{}
	stdioClient := stdiomcp.NewClient(&contractSTDIOTransport{
		callFn: func(ctx context.Context, name string, args map[string]any) (stdiomcp.Response, error) {
			return stdiomcp.Response{Content: "mcp-ok"}, nil
		},
	}, stdiomcp.Config{
		CallTimeout: 300 * time.Millisecond,
		Retry:       0,
	})

	server := a2a.NewInMemoryServer(a2a.HandlerFunc(func(ctx context.Context, req a2a.TaskRequest) (map[string]any, error) {
		return map[string]any{"remote": "ok"}, nil
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

	engine := workflow.New(
		workflow.WithTimelineEmitter(timeline),
		workflow.WithStepAdapter(workflow.DispatchAdapter{
			MCP: func(ctx context.Context, workflowID string, step workflow.Step, attempt int) (workflow.StepOutput, error) {
				resp, err := stdioClient.CallTool(ctx, "tool", map[string]any{"request": "ping"})
				if err != nil {
					return workflow.StepOutput{}, err
				}
				return workflow.StepOutput{Payload: map[string]any{"mcp": resp.Content}}, nil
			},
			A2A: func(ctx context.Context, workflowID string, step workflow.Step, attempt int) (workflow.StepOutput, error) {
				submitted, err := client.Submit(ctx, a2a.TaskRequest{
					TaskID:     fmt.Sprintf("%s-%d", step.TaskID, time.Now().UnixNano()),
					WorkflowID: workflowID,
					TeamID:     step.TeamID,
					StepID:     step.StepID,
					AgentID:    step.AgentID,
					PeerID:     step.PeerID,
				})
				if err != nil {
					return workflow.StepOutput{}, err
				}
				record, err := client.WaitResult(ctx, submitted.TaskID, 5*time.Millisecond, nil)
				if err != nil {
					return workflow.StepOutput{}, err
				}
				return workflow.StepOutput{Payload: record.Result}, nil
			},
		}),
	)

	res, err := engine.Run(context.Background(), workflow.RunRequest{
		RunID: "run-boundary-a5",
		DSL: workflow.Definition{
			WorkflowID: "wf-boundary-a5",
			Steps: []workflow.Step{
				{StepID: "mcp-local", Kind: workflow.StepKindMCP},
				{
					StepID:    "a2a-remote",
					TaskID:    "task-a2a-remote",
					Kind:      workflow.StepKindA2A,
					DependsOn: []string{"mcp-local"},
					TeamID:    "team-boundary",
					AgentID:   "agent-main",
					PeerID:    "peer-remote",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("workflow run failed: %v", err)
	}
	if res.WorkflowStatus != "succeeded" || res.WorkflowStepFailed != 0 || res.WorkflowRemoteTotal != 1 {
		t.Fatalf("workflow result mismatch: %#v", res)
	}

	hasWorkflowDispatchA2A := false
	for _, ev := range timeline.snapshot() {
		if ev.Type != types.EventTypeActionTimeline {
			continue
		}
		reason, _ := ev.Payload["reason"].(string)
		if reason == workflow.ReasonDispatchA2A {
			hasWorkflowDispatchA2A = true
		}
		if strings.HasPrefix(reason, "a2a.") && ev.Payload["peer_id"] == "" {
			t.Fatalf("a2a timeline event must keep peer_id correlation: %#v", ev.Payload)
		}
	}
	if !hasWorkflowDispatchA2A {
		t.Fatalf("missing reason %q", workflow.ReasonDispatchA2A)
	}
}
