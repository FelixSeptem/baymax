package extension

import (
	"context"
	"testing"
)

func TestExtensionConformanceCannotBypassAdmissionBoundary(t *testing.T) {
	d := Descriptor{Name: "lint", Kind: KindHook, Version: "1.0.0", Compat: ">=0.1.0 <1.0.0", Digest: "sha256:abc", Source: SourceProjectExplicit, Required: []string{"sandbox.exec"}}
	decision, err := Admit(d, AdmissionInput{RuntimeVersion: "0.2.0", AvailableCapabilities: []string{"tool.read"}})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Outcome != AdmissionDeny || decision.Activated {
		t.Fatalf("required capability boundary bypassed: %#v", decision)
	}
	result := Execute(context.Background(), ExecutionOptions{FailurePolicy: FailurePolicyDeny}, func(context.Context) (any, error) {
		return "must not run", nil
	})
	if result.Outcome != ExecutionSucceeded {
		t.Fatalf("executor should remain independent until admission callback is invoked: %#v", result)
	}
}
