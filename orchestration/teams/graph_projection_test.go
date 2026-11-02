package teams

import (
	"context"
	"testing"
)

func TestProjectExecutionGraphMapsStrategyToDependencies(t *testing.T) {
	plan := Plan{
		TeamID: "shared",
		Tasks: []Task{
			{TaskID: "first", AgentID: "agent-a", Target: TaskTargetLocal, Runner: TaskRunnerFunc(func(context.Context, Task) (TaskResult, error) { return TaskResult{}, nil })},
			{TaskID: "second", AgentID: "agent-b", Target: TaskTargetLocal, Runner: TaskRunnerFunc(func(context.Context, Task) (TaskResult, error) { return TaskResult{}, nil })},
		},
	}
	serial, err := ProjectExecutionGraph(plan)
	if err != nil {
		t.Fatal(err)
	}
	if len(serial.Definition.Nodes) != 2 || len(serial.Definition.Nodes[1].DependsOn) != 1 || serial.Definition.Nodes[1].DependsOn[0] != "first" {
		t.Fatalf("serial topology = %#v", serial.Definition)
	}

	plan.Strategy = StrategyParallel
	parallel, err := ProjectExecutionGraph(plan)
	if err != nil {
		t.Fatal(err)
	}
	if len(parallel.Definition.Nodes[0].DependsOn) != 0 || len(parallel.Definition.Nodes[1].DependsOn) != 0 {
		t.Fatalf("parallel topology imposes an order: %#v", parallel.Definition)
	}
	if serial.Identity.TopologyDigest == parallel.Identity.TopologyDigest {
		t.Fatal("serial and parallel strategy topology digests should differ")
	}
}

func TestProjectExecutionGraphRejectsInvalidPlanBeforeProjection(t *testing.T) {
	if compiled, err := ProjectExecutionGraph(Plan{TeamID: "team", Tasks: []Task{{TaskID: "task"}}}); err == nil {
		t.Fatalf("invalid plan projection = %#v, want error", compiled)
	}
}
