package teams

import "github.com/FelixSeptem/baymax/orchestration/graph"

// ProjectExecutionGraph exposes the dependency topology implied by the team
// strategy. It does not dispatch tasks or replace the team plan digest.
func ProjectExecutionGraph(input Plan) (graph.Compiled, error) {
	input.Tasks = append([]Task(nil), input.Tasks...)
	plan, err := normalizePlan(input)
	if err != nil {
		return graph.Compiled{}, err
	}
	nodes := make([]graph.Node, 0, len(plan.Tasks))
	for i, task := range plan.Tasks {
		node := graph.Node{ID: task.TaskID}
		if plan.Strategy == StrategySerial && i > 0 {
			node.DependsOn = []string{plan.Tasks[i-1].TaskID}
		}
		nodes = append(nodes, node)
	}
	return graph.Compile(graph.Definition{GraphID: plan.TeamID, Version: "v1", Nodes: nodes})
}
