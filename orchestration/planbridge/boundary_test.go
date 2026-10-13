package planbridge

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/orchestration/teams"
	"github.com/FelixSeptem/baymax/orchestration/workflow"
)

func TestHostExecutionPlanArraySnapshotDetached(t *testing.T) {
	value := [1]map[string]any{{"value": []string{"original"}}}
	copy, err := snapshot(value, 65536, 8)
	if err != nil {
		t.Fatal(err)
	}
	value[0]["value"].([]string)[0] = "mutated"
	if copy[0]["value"].([]string)[0] != "original" {
		t.Fatal("array payload aliases caller")
	}
}

func TestHostExecutionPlanAllLimitOverrides(t *testing.T) {
	d, b, _ := directFixture(t)
	for i := 0; i < 6; i++ {
		for _, n := range []int{-1, reflect.ValueOf(defaultLimits).Field(i).Interface().(int) + 1} {
			l := Limits{}
			reflect.ValueOf(&l).Elem().Field(i).SetInt(int64(n))
			_, err := Compile(d, b, l)
			codeOf(t, err, InvalidPlan)
		}
	}
}

func TestHostExecutionPlanCombinedStructuralLimit(t *testing.T) {
	d, b, r := directFixture(t)
	d.RequiredCapabilities = []string{"source_timeline"}
	b.Direct.Request.Messages = make([]types.Message, 127)
	if _, err := Describe(d, b, Limits{}); err != nil {
		t.Fatal(err)
	}
	b.Direct.Request.Messages = append(b.Direct.Request.Messages, types.Message{})
	_, err := Describe(d, b, Limits{})
	codeOf(t, err, LimitExceeded)
	if r.calls != 0 {
		t.Fatal("rejection activated source")
	}
}

func TestHostExecutionPlanDependencyLimitAndSemanticOrder(t *testing.T) {
	d, b, _ := directFixture(t)
	d.Shape = ShapeWorkflow
	b.Direct = nil
	steps := make([]workflow.Step, 48)
	for i := range steps {
		steps[i] = workflow.Step{StepID: fmt.Sprintf("s%02d", i)}
	}
	// 1024 distinct edges in an acyclic native graph, beneath all other bounds.
	edges := 0
	for i := range steps {
		for j := 0; j < i && edges < 1024; j++ {
			steps[i].DependsOn = append(steps[i].DependsOn, steps[j].StepID)
			edges++
		}
	}
	b.Workflow = &WorkflowBinding{TerminalOnly: true, Request: workflow.RunRequest{RunID: "run", DSL: workflow.Definition{WorkflowID: "w", Steps: steps}}, Adapter: workflow.DispatchAdapter{Runner: func(context.Context, string, workflow.Step, int) (workflow.StepOutput, error) {
		t.Fatal("preflight activated")
		return workflow.StepOutput{}, nil
	}}}
	d, err := Describe(d, b, Limits{})
	if err != nil {
		t.Fatal(err)
	}
	p, err := Compile(d, b, Limits{})
	if err != nil {
		t.Fatal(err)
	}
	b.Workflow.Request.DSL.Steps[2].DependsOn[0], b.Workflow.Request.DSL.Steps[2].DependsOn[1] = b.Workflow.Request.DSL.Steps[2].DependsOn[1], b.Workflow.Request.DSL.Steps[2].DependsOn[0]
	_, err = Compile(d, b, Limits{})
	codeOf(t, err, BindingMismatch)
	next, err := Describe(d, b, Limits{})
	if err != nil {
		t.Fatal(err)
	}
	q, err := Compile(next, b, Limits{})
	if err != nil || q.Identity().Digest == p.Identity().Digest {
		t.Fatalf("dependency order digest: %v", err)
	}
	b.Workflow.Request.DSL.Steps[47].DependsOn = append(b.Workflow.Request.DSL.Steps[47].DependsOn, "s46")
	_, err = Describe(d, b, Limits{})
	codeOf(t, err, LimitExceeded)
}

func TestHostExecutionPlanTeamOrderBudgetAndFactMismatch(t *testing.T) {
	d, b, _ := directFixture(t)
	d.Shape = ShapeTeam
	b.Direct = nil
	callback := teams.TaskRunnerFunc(func(context.Context, teams.Task) (teams.TaskResult, error) {
		t.Fatal("preflight activated")
		return teams.TaskResult{}, nil
	})
	b.Team = &TeamBinding{TerminalOnly: true, Plan: teams.Plan{RunID: "run", TeamID: "t", Strategy: teams.StrategyParallel, Tasks: []teams.Task{{TaskID: "a", AgentID: "worker", Runner: callback}, {TaskID: "b", AgentID: "worker", Runner: callback}}}}
	d, err := Describe(d, b, Limits{})
	if err != nil {
		t.Fatal(err)
	}
	p, err := Compile(d, b, Limits{})
	if err != nil {
		t.Fatal(err)
	}
	b.Team.Plan.Tasks[0], b.Team.Plan.Tasks[1] = b.Team.Plan.Tasks[1], b.Team.Plan.Tasks[0]
	_, err = Compile(d, b, Limits{})
	codeOf(t, err, BindingMismatch)
	for _, timeout := range []time.Duration{time.Second, 2 * time.Second} {
		b.Team.Plan.TaskTimeout = timeout
		next, e := Describe(d, b, Limits{})
		if e != nil {
			t.Fatal(e)
		}
		q, e := Compile(next, b, Limits{})
		if e != nil || q.Identity().Digest == p.Identity().Digest {
			t.Fatalf("budget digest: %v", e)
		}
	}
	// Canonical output carries reference facts, not callback payloads.
	var facts map[string]any
	if err = json.Unmarshal(p.Canonical(), &facts); err != nil {
		t.Fatal(err)
	}
	if facts["schema_version"] != SchemaVersion {
		t.Fatal("version missing")
	}
}
