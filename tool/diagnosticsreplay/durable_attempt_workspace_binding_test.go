package diagnosticsreplay

import (
	"strings"
	"testing"
)

func TestParseDurableAttemptWorkspaceBindingFixture(t *testing.T) {
	raw := []byte(`{"version":"durable_attempt_workspace_binding.v1","cases":[{"name":"valid","task_id":"task-1","attempt_id":"attempt-1","attempt":1,"lease_generation":1,"workspace":{"workspace_id":"ws-1","change_set_id":"change-1","before_integrity":"before","after_integrity":"after","checkpoint_id":"cp-1"},"expected":"valid_binding"}]}`)
	fixture, err := ParseDurableAttemptWorkspaceBindingFixtureJSON(raw)
	if err != nil || len(fixture.Cases) != 1 || fixture.Cases[0].Workspace == nil {
		t.Fatalf("fixture=%#v err=%v", fixture, err)
	}
}

func TestParseDurableAttemptWorkspaceBindingFixtureRejectsSchemaAndBounds(t *testing.T) {
	cases := []struct{ name, raw string }{
		{"unknown version", `{"version":"durable_attempt_workspace_binding.v2","cases":[]}`},
		{"missing cases", `{"version":"durable_attempt_workspace_binding.v1","cases":[]}`},
		{"oversized raw", strings.Repeat("x", 2<<20+1)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseDurableAttemptWorkspaceBindingFixtureJSON([]byte(tc.raw))
			if err == nil || !strings.Contains(err.Error(), ReasonCodeDurableAttemptWorkspaceSchemaDrift) {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestParseDurableAttemptWorkspaceBindingFixtureRejectsImplicitRetry(t *testing.T) {
	raw := []byte(`{"version":"durable_attempt_workspace_binding.v1","cases":[{"name":"retry","task_id":"task-1","attempt_id":"attempt-2","attempt":2,"lease_generation":2,"retry":{"mode":"implicit"},"expected":"retry"}]}`)
	_, err := ParseDurableAttemptWorkspaceBindingFixtureJSON(raw)
	if err == nil || !strings.Contains(err.Error(), ReasonCodeDurableAttemptWorkspaceBindingDrift) {
		t.Fatalf("err=%v", err)
	}
}

func TestEvaluateDurableAttemptWorkspaceBindingClassifiesObservedDrift(t *testing.T) {
	raw := []byte(`{"version":"durable_attempt_workspace_binding.v1","cases":[{"name":"stale","task_id":"task-1","attempt_id":"attempt-2","attempt":2,"lease_generation":2,"expected":"accepted","observed":"stale"}]}`)
	_, err := EvaluateDurableAttemptWorkspaceBindingFixtureJSON(raw)
	if err == nil || !strings.Contains(err.Error(), ReasonCodeDurableAttemptWorkspaceStaleAttemptDrift) {
		t.Fatalf("err=%v", err)
	}
}
