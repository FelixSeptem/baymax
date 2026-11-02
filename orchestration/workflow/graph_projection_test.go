package workflow

import (
	"reflect"
	"testing"
)

func TestProjectExecutionGraphPreservesWorkflowDependencies(t *testing.T) {
	engine := New()
	compiled, err := engine.ProjectExecutionGraph(Definition{WorkflowID: "shared", Steps: []Step{
		{StepID: "build", Kind: StepKindRunner},
		{StepID: "deploy", Kind: StepKindRunner, DependsOn: []string{"build"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if compiled.Identity.GraphID != "shared" || compiled.Identity.Version != "v1" {
		t.Fatalf("identity = %#v", compiled.Identity)
	}
	if len(compiled.Definition.Nodes) != 2 || compiled.Definition.Nodes[1].ID != "deploy" || len(compiled.Definition.Nodes[1].DependsOn) != 1 || compiled.Definition.Nodes[1].DependsOn[0] != "build" {
		t.Fatalf("projected topology = %#v", compiled.Definition)
	}
}

func TestProjectExecutionGraphExpandsConfiguredSubgraphs(t *testing.T) {
	engine := New(WithGraphComposabilityEnabled(true))
	compiled, err := engine.ProjectExecutionGraph(Definition{
		WorkflowID: "shared",
		Subgraphs:  map[string]Subgraph{"build": {Steps: []Step{{StepID: "compile", Kind: StepKindRunner}, {StepID: "test", Kind: StepKindRunner, DependsOn: []string{"compile"}}}}},
		Steps:      []Step{{StepID: "component", UseSubgraph: "build", Alias: "service"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(compiled.Definition.Nodes) != 2 || compiled.Definition.Nodes[1].ID != "service/test" {
		t.Fatalf("expanded topology = %#v", compiled.Definition)
	}
}

func TestProjectExecutionGraphRejectsWorkflowCycles(t *testing.T) {
	_, err := New().ProjectExecutionGraph(Definition{WorkflowID: "cycle", Steps: []Step{
		{StepID: "a", Kind: StepKindRunner, DependsOn: []string{"b"}},
		{StepID: "b", Kind: StepKindRunner, DependsOn: []string{"a"}},
	}})
	if err == nil {
		t.Fatal("expected workflow validation error")
	}
	if _, ok := err.(ValidationErrors); !ok {
		t.Fatalf("error type = %T, want workflow ValidationErrors: %v", err, err)
	}
}

func TestProjectExecutionGraphDigestDoesNotReplaceWorkflowDefinitionDigest(t *testing.T) {
	engine := New()
	first, err := engine.ProjectExecutionGraph(Definition{WorkflowID: "shared", Steps: []Step{{StepID: "run", Kind: StepKindRunner, Payload: map[string]any{"command": "one"}}}})
	if err != nil {
		t.Fatal(err)
	}
	second, err := engine.ProjectExecutionGraph(Definition{WorkflowID: "shared", Steps: []Step{{StepID: "run", Kind: StepKindRunner, Payload: map[string]any{"command": "two"}}}})
	if err != nil {
		t.Fatal(err)
	}
	if first.Identity.TopologyDigest != second.Identity.TopologyDigest {
		t.Fatal("payload-only change should not alter the topology digest")
	}
	if workflowDefinitionDigest(Definition{WorkflowID: "shared", Steps: []Step{{StepID: "run", Kind: StepKindRunner, Payload: map[string]any{"command": "one"}}}}) == workflowDefinitionDigest(Definition{WorkflowID: "shared", Steps: []Step{{StepID: "run", Kind: StepKindRunner, Payload: map[string]any{"command": "two"}}}}) {
		t.Fatal("workflow owner digest failed to retain execution payload")
	}
}

func TestProjectExecutionGraphDoesNotMutateCallerDefinition(t *testing.T) {
	definition := Definition{WorkflowID: " shared ", Steps: []Step{{
		StepID:    " run ",
		Kind:      StepKindRunner,
		DependsOn: []string{" dependency "},
	}}}
	before := definition
	before.Steps = append([]Step(nil), definition.Steps...)
	before.Steps[0].DependsOn = append([]string(nil), definition.Steps[0].DependsOn...)

	_, _ = New().ProjectExecutionGraph(definition)
	if !reflect.DeepEqual(definition, before) {
		t.Fatalf("projection mutated caller definition:\ngot:  %#v\nwant: %#v", definition, before)
	}
}
