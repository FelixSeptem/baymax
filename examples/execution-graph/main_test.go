package main

import (
	"strings"
	"testing"
)

func TestOfflineExampleEmitsContractMarkers(t *testing.T) {
	output, err := run()
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{
		"execution_graph.workflow_team_serial_equal=true",
		"execution_graph.workflow_team_parallel_equal=true",
		"execution_graph.workflow_child_single_node_equal=true",
		"execution_graph.owner_digest_unchanged=true",
		"execution_graph.no_execution_performed=true",
	} {
		if !strings.Contains(output, marker) {
			t.Fatalf("missing marker %q in output:\n%s", marker, output)
		}
	}
}
