package teams

import (
	"context"
	"reflect"
	"testing"
)

func TestNormalizePlanPureNativeParity(t *testing.T) {
	run := TaskRunnerFunc(func(context.Context, Task) (TaskResult, error) { return TaskResult{}, nil })
	raw := Plan{TeamID: " team ", Tasks: []Task{{TaskID: " t ", AgentID: " a ", Runner: run}}}
	got, err := NormalizePlan(raw)
	if err != nil {
		t.Fatal(err)
	}
	if raw.Tasks[0].TaskID != " t " {
		t.Fatal("normalization mutated caller")
	}
	want, err := normalizePlan(Plan{TeamID: raw.TeamID, Tasks: append([]Task(nil), raw.Tasks...)})
	if err != nil || !reflect.DeepEqual(got.Tasks[0].TaskID, want.Tasks[0].TaskID) || got.TaskTimeout != want.TaskTimeout || got.Strategy != want.Strategy {
		t.Fatalf("native defaults drift: %+v", got)
	}
	for _, bad := range []Plan{{}, {TeamID: "t", Tasks: []Task{{TaskID: "a", AgentID: "a", Runner: run}, {TaskID: "a", AgentID: "a", Runner: run}}}, {TeamID: "t", Strategy: "bogus", Tasks: raw.Tasks}} {
		_, a := NormalizePlan(bad)
		_, b := New().Run(context.Background(), bad)
		if a == nil || b == nil || a.Error() != b.Error() {
			t.Fatalf("validation differs: %v %v", a, b)
		}
	}
}
