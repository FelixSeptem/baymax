package composer

import (
	"strings"

	"github.com/FelixSeptem/baymax/orchestration/graph"
)

// ProjectChildExecutionGraph projects only the child dispatch represented by
// req. It does not infer sibling nodes or parent fanout dependencies.
func ProjectChildExecutionGraph(req ChildDispatchRequest) (graph.Compiled, error) {
	graphID := strings.TrimSpace(req.GraphID)
	if graphID == "" {
		graphID = strings.TrimSpace(req.Task.DispatchBinding)
	}
	if graphID == "" {
		graphID = strings.TrimSpace(req.Task.TaskID)
	}
	version := strings.TrimSpace(req.GraphVersion)
	if version == "" {
		version = "v1"
	}
	nodeID := strings.TrimSpace(req.Task.TaskID)
	return graph.Compile(graph.Definition{
		GraphID: graphID,
		Version: version,
		Nodes:   []graph.Node{{ID: nodeID}},
	})
}
