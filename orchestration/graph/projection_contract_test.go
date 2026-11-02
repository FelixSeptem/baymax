package graph_test

import (
	"context"
	"testing"

	"github.com/FelixSeptem/baymax/orchestration/composer"
	"github.com/FelixSeptem/baymax/orchestration/graph"
	"github.com/FelixSeptem/baymax/orchestration/scheduler"
	"github.com/FelixSeptem/baymax/orchestration/teams"
	"github.com/FelixSeptem/baymax/orchestration/workflow"
)

func TestEquivalentTopologyMatchesAcrossOwners(t *testing.T) {
	teamTasks := []teams.Task{
		{TaskID: "first", AgentID: "agent-a", Target: teams.TaskTargetLocal, Runner: teams.TaskRunnerFunc(func(context.Context, teams.Task) (teams.TaskResult, error) { return teams.TaskResult{}, nil })},
		{TaskID: "second", AgentID: "agent-b", Target: teams.TaskTargetLocal, Runner: teams.TaskRunnerFunc(func(context.Context, teams.Task) (teams.TaskResult, error) { return teams.TaskResult{}, nil })},
	}

	t.Run("serial workflow and team", func(t *testing.T) {
		workflowGraph, err := workflow.New().ProjectExecutionGraph(workflow.Definition{WorkflowID: "shared", Steps: []workflow.Step{
			{StepID: "first", Kind: workflow.StepKindRunner},
			{StepID: "second", Kind: workflow.StepKindRunner, DependsOn: []string{"first"}},
		}})
		if err != nil {
			t.Fatal(err)
		}
		teamGraph, err := teams.ProjectExecutionGraph(teams.Plan{TeamID: "shared", Strategy: teams.StrategySerial, Tasks: teamTasks})
		if err != nil {
			t.Fatal(err)
		}
		assertSameTopology(t, workflowGraph, teamGraph)
	})

	t.Run("independent workflow and team", func(t *testing.T) {
		workflowGraph, err := workflow.New().ProjectExecutionGraph(workflow.Definition{WorkflowID: "shared", Steps: []workflow.Step{
			{StepID: "first", Kind: workflow.StepKindRunner},
			{StepID: "second", Kind: workflow.StepKindRunner},
		}})
		if err != nil {
			t.Fatal(err)
		}
		teamGraph, err := teams.ProjectExecutionGraph(teams.Plan{TeamID: "shared", Strategy: teams.StrategyParallel, Tasks: teamTasks})
		if err != nil {
			t.Fatal(err)
		}
		assertSameTopology(t, workflowGraph, teamGraph)
	})

	t.Run("single workflow node and child dispatch", func(t *testing.T) {
		workflowGraph, err := workflow.New().ProjectExecutionGraph(workflow.Definition{WorkflowID: "shared", Steps: []workflow.Step{{StepID: "child", Kind: workflow.StepKindRunner}}})
		if err != nil {
			t.Fatal(err)
		}
		childGraph, err := composer.ProjectChildExecutionGraph(composer.ChildDispatchRequest{
			Task: scheduler.Task{TaskID: "child"}, GraphID: "shared", GraphVersion: "v1",
		})
		if err != nil {
			t.Fatal(err)
		}
		assertSameTopology(t, workflowGraph, childGraph)
	})
}

func TestTopologyDriftChangesOnlyProjectionDigest(t *testing.T) {
	base, err := graph.Compile(graph.Definition{GraphID: "shared", Version: "v1", Nodes: []graph.Node{{ID: "first"}, {ID: "second"}}})
	if err != nil {
		t.Fatal(err)
	}
	changed, err := graph.Compile(graph.Definition{GraphID: "shared", Version: "v1", Nodes: []graph.Node{{ID: "first"}, {ID: "second", DependsOn: []string{"first"}}}})
	if err != nil {
		t.Fatal(err)
	}
	if base.Identity.TopologyDigest == changed.Identity.TopologyDigest {
		t.Fatal("dependency drift was not represented in topology digest")
	}
}

func assertSameTopology(t *testing.T, left, right graph.Compiled) {
	t.Helper()
	if left.Identity != right.Identity {
		t.Fatalf("topology identities differ:\nleft: %#v\nright: %#v", left.Identity, right.Identity)
	}
	if len(left.Order) != len(right.Order) {
		t.Fatalf("topological orders differ: %v != %v", left.Order, right.Order)
	}
	for i := range left.Order {
		if left.Order[i] != right.Order[i] {
			t.Fatalf("topological orders differ: %v != %v", left.Order, right.Order)
		}
	}
}
