package workflow

import (
	"testing"
	"time"
)

func TestPreflightDefinitionNativeDefaultsAndValidation(t *testing.T) {
	raw := Definition{WorkflowID: " w ", Steps: []Step{{Step: " s "}}}
	e := New(WithDefaultStepTimeout(time.Second))
	got, err := e.PreflightDefinition(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Steps[0].StepID != "s" || got.Steps[0].Timeout != time.Second || got.Steps[0].Kind != StepKindRunner {
		t.Fatalf("defaults: %+v", got)
	}
	if raw.Steps[0].StepID != "" {
		t.Fatal("caller mutation")
	}
	bad := Definition{WorkflowID: "w", Steps: []Step{{StepID: "s", DependsOn: []string{"missing"}}}}
	_, a := e.PreflightDefinition(bad)
	_, b := e.Plan(bad)
	if a == nil || b == nil || a.Error() != b.Error() {
		t.Fatalf("validation drift: %v %v", a, b)
	}
}
