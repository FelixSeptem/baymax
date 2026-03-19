package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/a2a"
	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/integration/fakes"
	"github.com/FelixSeptem/baymax/orchestration/composer"
	"github.com/FelixSeptem/baymax/orchestration/workflow"
)

func TestWorkflowGraphComposabilityA15ExpansionDeterminismAndCanonicalIDs(t *testing.T) {
	engine := workflow.New(
		workflow.WithGraphComposabilityEnabled(true),
		workflow.WithStepAdapter(workflow.DispatchAdapter{
			Runner: func(ctx context.Context, workflowID string, step workflow.Step, attempt int) (workflow.StepOutput, error) {
				return workflow.StepOutput{Payload: map[string]any{"ok": true}}, nil
			},
		}),
	)
	def := workflow.Definition{
		WorkflowID: "wf-a15-determinism",
		Subgraphs: map[string]workflow.Subgraph{
			"prepare": {
				Steps: []workflow.Step{
					{StepID: "fetch", Kind: workflow.StepKindRunner},
					{StepID: "validate", Kind: workflow.StepKindRunner, DependsOn: []string{"fetch"}},
				},
			},
		},
		Steps: []workflow.Step{
			{StepID: "prepare-node", UseSubgraph: "prepare", Alias: "prepare"},
			{StepID: "final", Kind: workflow.StepKindRunner, DependsOn: []string{"prepare-node"}},
		},
	}

	orderA, err := engine.Plan(def)
	if err != nil {
		t.Fatalf("first plan failed: %v", err)
	}
	orderB, err := engine.Plan(def)
	if err != nil {
		t.Fatalf("second plan failed: %v", err)
	}
	if strings.Join(orderA, ",") != strings.Join(orderB, ",") {
		t.Fatalf("expanded order is not deterministic: first=%v second=%v", orderA, orderB)
	}
	if got, want := strings.Join(orderA, ","), "prepare/fetch,prepare/validate,final"; got != want {
		t.Fatalf("expanded canonical ids mismatch: got=%q want=%q", got, want)
	}

	res, err := engine.Run(context.Background(), workflow.RunRequest{RunID: "run-a15-determinism", DSL: def})
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if res.WorkflowSubgraphExpansionTotal != 2 || res.WorkflowGraphCompileFailed {
		t.Fatalf("compile summary mismatch: %#v", res)
	}
}

func TestWorkflowGraphComposabilityA15CompileFailFastMatrix(t *testing.T) {
	timeline := &a2aTimelineCollector{}
	server := a2a.NewInMemoryServer(a2a.HandlerFunc(func(ctx context.Context, req a2a.TaskRequest) (map[string]any, error) {
		return map[string]any{"ok": true}, nil
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

	overrideKind := workflow.StepKindTool
	invalid := workflow.Definition{
		WorkflowID: "wf-a15-fail-fast",
		Subgraphs: map[string]workflow.Subgraph{
			"remote": {
				Steps: []workflow.Step{
					{StepID: "delegate", Kind: workflow.StepKindA2A, AgentID: "agent-main", PeerID: "peer-remote"},
				},
			},
		},
		Steps: []workflow.Step{
			{
				StepID:      "remote-node",
				UseSubgraph: "remote",
				Alias:       "remote",
				Overrides: map[string]workflow.StepOverride{
					"delegate": {Kind: &overrideKind},
				},
			},
		},
	}
	engine := workflow.New(
		workflow.WithGraphComposabilityEnabled(true),
		workflow.WithTimelineEmitter(timeline),
		workflow.WithStepAdapter(workflow.DispatchAdapter{
			A2A: workflow.NewA2AStepAdapter(client, workflow.A2AStepAdapterOptions{PollInterval: 5 * time.Millisecond}),
		}),
	)

	if _, err := engine.Run(context.Background(), workflow.RunRequest{RunID: "run-a15-fail-fast", DSL: invalid}); err == nil {
		t.Fatal("run should fail fast on forbidden kind override")
	}
	if _, err := engine.Stream(context.Background(), workflow.RunRequest{RunID: "stream-a15-fail-fast", DSL: invalid}, func(workflow.StreamEvent) error { return nil }); err == nil {
		t.Fatal("stream should fail fast on forbidden kind override")
	}

	for _, ev := range timeline.snapshot() {
		if ev.Type != types.EventTypeActionTimeline {
			continue
		}
		reason, _ := ev.Payload["reason"].(string)
		if reason == workflow.ReasonDispatchA2A || reason == a2a.ReasonSubmit {
			t.Fatalf("compile boundary regression: unexpected dispatch reason=%q payload=%#v", reason, ev.Payload)
		}
	}
}

func TestWorkflowGraphComposabilityA15RunStreamEquivalenceAndResumeConsistency(t *testing.T) {
	store := workflow.NewMemoryCheckpointStore()
	failOnce := true
	engine := workflow.New(
		workflow.WithGraphComposabilityEnabled(true),
		workflow.WithCheckpointStore(store),
		workflow.WithStepAdapter(workflow.DispatchAdapter{
			Runner: func(ctx context.Context, workflowID string, step workflow.Step, attempt int) (workflow.StepOutput, error) {
				if step.StepID == "flow/process" && failOnce {
					failOnce = false
					return workflow.StepOutput{}, context.DeadlineExceeded
				}
				return workflow.StepOutput{Payload: map[string]any{"step": step.StepID}}, nil
			},
		}),
	)
	def := workflow.Definition{
		WorkflowID: "wf-a15-resume",
		Subgraphs: map[string]workflow.Subgraph{
			"flow": {
				Steps: []workflow.Step{
					{StepID: "prepare", Kind: workflow.StepKindRunner},
					{StepID: "process", Kind: workflow.StepKindRunner, DependsOn: []string{"prepare"}},
				},
			},
		},
		Steps: []workflow.Step{
			{StepID: "flow-node", UseSubgraph: "flow", Alias: "flow"},
			{StepID: "tail", Kind: workflow.StepKindRunner, DependsOn: []string{"flow-node"}},
		},
	}

	first, err := engine.Run(context.Background(), workflow.RunRequest{RunID: "run-a15-resume-1", DSL: def})
	if err != nil {
		t.Fatalf("first run failed: %v", err)
	}
	if first.WorkflowStatus != "failed" {
		t.Fatalf("first workflow status=%q, want failed", first.WorkflowStatus)
	}

	resume, err := engine.Run(context.Background(), workflow.RunRequest{RunID: "run-a15-resume-2", Resume: true, DSL: def})
	if err != nil {
		t.Fatalf("resume run failed: %v", err)
	}
	if resume.WorkflowStatus != "succeeded" || resume.WorkflowResumeCount != 1 {
		t.Fatalf("resume semantics mismatch: %#v", resume)
	}

	eqReq := workflow.RunRequest{RunID: "run-a15-eq", DSL: def}
	runRes, runErr := engine.Run(context.Background(), eqReq)
	streamRes, streamErr := engine.Stream(context.Background(), workflow.RunRequest{RunID: "stream-a15-eq", DSL: def}, func(workflow.StreamEvent) error { return nil })
	if runErr != nil || streamErr != nil {
		t.Fatalf("run/stream errors mismatch run=%v stream=%v", runErr, streamErr)
	}
	if runRes.WorkflowStatus != streamRes.WorkflowStatus {
		t.Fatalf("run/stream status mismatch run=%q stream=%q", runRes.WorkflowStatus, streamRes.WorkflowStatus)
	}
	if runRes.WorkflowStepTotal != streamRes.WorkflowStepTotal ||
		runRes.WorkflowStepFailed != streamRes.WorkflowStepFailed ||
		runRes.WorkflowSubgraphExpansionTotal != streamRes.WorkflowSubgraphExpansionTotal {
		t.Fatalf("run/stream aggregate mismatch run=%#v stream=%#v", runRes, streamRes)
	}
}

func TestWorkflowGraphComposabilityA15ComposerManagedRemoteStep(t *testing.T) {
	timeline := &a2aTimelineCollector{}
	server := a2a.NewInMemoryServer(a2a.HandlerFunc(func(ctx context.Context, req a2a.TaskRequest) (map[string]any, error) {
		return map[string]any{"ok": true}, nil
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

	wfEngine := workflow.New(
		workflow.WithGraphComposabilityEnabled(true),
		workflow.WithTimelineEmitter(timeline),
		workflow.WithStepAdapter(workflow.DispatchAdapter{
			A2A: workflow.NewA2AStepAdapter(client, workflow.A2AStepAdapterOptions{PollInterval: 5 * time.Millisecond}),
		}),
	)
	comp, err := composer.New(
		&fakes.Model{},
		composer.WithEventHandler(timeline),
		composer.WithWorkflow(wfEngine),
	)
	if err != nil {
		t.Fatalf("composer init failed: %v", err)
	}

	req := workflow.RunRequest{
		RunID: "run-a15-composer-managed",
		DSL: workflow.Definition{
			WorkflowID: "wf-a15-composer-managed",
			Subgraphs: map[string]workflow.Subgraph{
				"remote": {
					Steps: []workflow.Step{
						{StepID: "delegate", TaskID: "task-delegate", Kind: workflow.StepKindA2A, TeamID: "team-a15", AgentID: "agent-main", PeerID: "peer-remote"},
					},
				},
			},
			Steps: []workflow.Step{
				{StepID: "remote-node", UseSubgraph: "remote", Alias: "remote"},
			},
		},
	}
	runRes, runErr := comp.Workflow().Run(context.Background(), req)
	streamRes, streamErr := comp.Workflow().Stream(context.Background(), req, func(workflow.StreamEvent) error { return nil })
	if runErr != nil || streamErr != nil {
		t.Fatalf("composer-managed workflow run/stream errors mismatch run=%v stream=%v", runErr, streamErr)
	}
	if runRes.WorkflowStatus != "succeeded" || streamRes.WorkflowStatus != "succeeded" {
		t.Fatalf("composer-managed workflow status mismatch run=%#v stream=%#v", runRes, streamRes)
	}
	if runRes.WorkflowRemoteTotal != 1 || streamRes.WorkflowRemoteTotal != 1 {
		t.Fatalf("composer-managed remote aggregate mismatch run=%#v stream=%#v", runRes, streamRes)
	}
}
