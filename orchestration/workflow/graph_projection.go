package workflow

import "github.com/FelixSeptem/baymax/orchestration/graph"

// ProjectExecutionGraph returns the validated, normalized topology for this
// workflow definition without dispatching or restoring any step.
func (e *Engine) ProjectExecutionGraph(def Definition) (graph.Compiled, error) {
	if e == nil {
		e = New()
	}
	def = cloneDefinitionForTopologyProjection(def)
	def = normalizeDefinition(def)
	compiled, _, err := e.compileDefinition(def)
	if err != nil {
		return graph.Compiled{}, err
	}
	def = normalizeDefinition(compiled)
	if violations := ValidateDefinition(def); len(violations) > 0 {
		return graph.Compiled{}, violations
	}
	nodes := make([]graph.Node, 0, len(def.Steps))
	for _, step := range def.Steps {
		nodes = append(nodes, graph.Node{ID: step.StepID, DependsOn: append([]string(nil), step.DependsOn...)})
	}
	return graph.Compile(graph.Definition{GraphID: def.WorkflowID, Version: "v1", Nodes: nodes})
}

func cloneDefinitionForTopologyProjection(def Definition) Definition {
	def.Steps = append([]Step(nil), def.Steps...)
	if len(def.Subgraphs) == 0 {
		return def
	}
	subgraphs := make(map[string]Subgraph, len(def.Subgraphs))
	for name, subgraph := range def.Subgraphs {
		subgraph.Steps = append([]Step(nil), subgraph.Steps...)
		subgraphs[name] = subgraph
	}
	def.Subgraphs = subgraphs
	return def
}
