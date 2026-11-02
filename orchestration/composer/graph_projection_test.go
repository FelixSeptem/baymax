package composer

import (
	"testing"

	"github.com/FelixSeptem/baymax/orchestration/scheduler"
)

func TestProjectChildExecutionGraphPreservesScopeAndCallerIdentity(t *testing.T) {
	request := ChildDispatchRequest{
		Task:         scheduler.Task{TaskID: "child"},
		GraphID:      "parent-graph",
		GraphVersion: "v2",
		GraphDigest:  "owner-plan-digest",
	}
	compiled, err := ProjectChildExecutionGraph(request)
	if err != nil {
		t.Fatal(err)
	}
	if compiled.Identity.GraphID != request.GraphID || compiled.Identity.Version != request.GraphVersion {
		t.Fatalf("identity = %#v", compiled.Identity)
	}
	if compiled.Identity.TopologyDigest == request.GraphDigest {
		t.Fatal("topology digest was conflated with caller owner digest")
	}
	if len(compiled.Definition.Nodes) != 1 || compiled.Definition.Nodes[0].ID != "child" || len(compiled.Definition.Nodes[0].DependsOn) != 0 {
		t.Fatalf("child projection inferred parent topology: %#v", compiled.Definition)
	}
	if request.GraphDigest != "owner-plan-digest" {
		t.Fatalf("projection mutated caller request: %#v", request)
	}
}

func TestProjectChildExecutionGraphUsesExistingIdentityFallbacks(t *testing.T) {
	compiled, err := ProjectChildExecutionGraph(ChildDispatchRequest{Task: scheduler.Task{TaskID: "child", DispatchBinding: "binding"}})
	if err != nil {
		t.Fatal(err)
	}
	if compiled.Identity.GraphID != "binding" || compiled.Identity.Version != "v1" {
		t.Fatalf("fallback identity = %#v", compiled.Identity)
	}
}
