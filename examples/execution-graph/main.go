package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/FelixSeptem/baymax/orchestration/composer"
	"github.com/FelixSeptem/baymax/orchestration/scheduler"
	"github.com/FelixSeptem/baymax/orchestration/teams"
	"github.com/FelixSeptem/baymax/orchestration/workflow"
)

func main() {
	output, err := run()
	if err != nil {
		fmt.Println("execution_graph.error=true")
		return
	}
	fmt.Print(output)
}

func run() (string, error) {
	tasks := []teams.Task{
		{TaskID: "prepare", AgentID: "agent-a", Runner: noOpRunner()},
		{TaskID: "execute", AgentID: "agent-b", Runner: noOpRunner()},
	}
	workflowOwner := workflow.New()
	workflowSerial, err := workflowOwner.ProjectExecutionGraph(workflow.Definition{WorkflowID: "shared", Steps: []workflow.Step{
		{StepID: "prepare", Kind: workflow.StepKindRunner},
		{StepID: "execute", Kind: workflow.StepKindRunner, DependsOn: []string{"prepare"}},
	}})
	if err != nil {
		return "", err
	}
	teamSerial, err := teams.ProjectExecutionGraph(teams.Plan{TeamID: "shared", Strategy: teams.StrategySerial, Tasks: tasks})
	if err != nil {
		return "", err
	}
	workflowParallel, err := workflowOwner.ProjectExecutionGraph(workflow.Definition{WorkflowID: "shared", Steps: []workflow.Step{
		{StepID: "prepare", Kind: workflow.StepKindRunner},
		{StepID: "execute", Kind: workflow.StepKindRunner},
	}})
	if err != nil {
		return "", err
	}
	teamParallel, err := teams.ProjectExecutionGraph(teams.Plan{TeamID: "shared", Strategy: teams.StrategyParallel, Tasks: tasks})
	if err != nil {
		return "", err
	}
	workflowChild, err := workflowOwner.ProjectExecutionGraph(workflow.Definition{WorkflowID: "shared", Steps: []workflow.Step{{StepID: "child", Kind: workflow.StepKindRunner}}})
	if err != nil {
		return "", err
	}
	childRequest := composer.ChildDispatchRequest{Task: scheduler.Task{TaskID: "child"}, GraphID: "shared", GraphVersion: "v1", GraphDigest: "owner-plan-digest"}
	child, err := composer.ProjectChildExecutionGraph(childRequest)
	if err != nil {
		return "", err
	}

	var output strings.Builder
	fmt.Fprintf(&output, "execution_graph.workflow_team_serial_equal=%t\n", workflowSerial.Identity == teamSerial.Identity)
	fmt.Fprintf(&output, "execution_graph.workflow_team_parallel_equal=%t\n", workflowParallel.Identity == teamParallel.Identity)
	fmt.Fprintf(&output, "execution_graph.workflow_child_single_node_equal=%t\n", workflowChild.Identity == child.Identity)
	fmt.Fprintf(&output, "execution_graph.owner_digest_unchanged=%t\n", childRequest.GraphDigest != child.Identity.TopologyDigest)
	output.WriteString("execution_graph.no_execution_performed=true\n")
	return output.String(), nil
}

func noOpRunner() teams.TaskRunner {
	return teams.TaskRunnerFunc(func(context.Context, teams.Task) (teams.TaskResult, error) {
		return teams.TaskResult{}, nil
	})
}
