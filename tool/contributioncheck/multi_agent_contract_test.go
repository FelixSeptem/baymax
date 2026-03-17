package contributioncheck

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMultiAgentSharedContractSnapshotPass(t *testing.T) {
	root := repoRoot(t)
	snapshot := MultiAgentContractSnapshot{
		IdentifierDoc:             mustRead(t, filepath.Join(root, "docs", "multi-agent-identifier-model.md")),
		TeamsTimelineSpec:         mustRead(t, filepath.Join(root, "openspec", "changes", "teams-runtime-baseline", "specs", "action-timeline-events", "spec.md")),
		WorkflowTimelineSpec:      mustRead(t, filepath.Join(root, "openspec", "changes", "workflow-dsl-baseline", "specs", "action-timeline-events", "spec.md")),
		A2ATimelineSpec:           mustRead(t, filepath.Join(root, "openspec", "changes", "a2a-minimal-interoperability", "specs", "action-timeline-events", "spec.md")),
		A2ACoreSpec:               mustRead(t, filepath.Join(root, "openspec", "changes", "a2a-minimal-interoperability", "specs", "a2a-minimal-interoperability", "spec.md")),
		TeamsRuntimeConfigSpec:    mustRead(t, filepath.Join(root, "openspec", "changes", "teams-runtime-baseline", "specs", "runtime-config-and-diagnostics-api", "spec.md")),
		WorkflowRuntimeConfigSpec: mustRead(t, filepath.Join(root, "openspec", "changes", "workflow-dsl-baseline", "specs", "runtime-config-and-diagnostics-api", "spec.md")),
		A2ARuntimeConfigSpec:      mustRead(t, filepath.Join(root, "openspec", "changes", "a2a-minimal-interoperability", "specs", "runtime-config-and-diagnostics-api", "spec.md")),
		TeamsBoundarySpec:         mustRead(t, filepath.Join(root, "openspec", "changes", "teams-runtime-baseline", "specs", "runtime-module-boundaries", "spec.md")),
		WorkflowBoundarySpec:      mustRead(t, filepath.Join(root, "openspec", "changes", "workflow-dsl-baseline", "specs", "runtime-module-boundaries", "spec.md")),
		A2ABoundarySpec:           mustRead(t, filepath.Join(root, "openspec", "changes", "a2a-minimal-interoperability", "specs", "runtime-module-boundaries", "spec.md")),
	}

	violations := ValidateMultiAgentSharedContractSnapshot(snapshot)
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %+v", violations)
	}
}

func TestValidateMultiAgentSharedContractDetectsViolations(t *testing.T) {
	snapshot := MultiAgentContractSnapshot{
		IdentifierDoc:             "no mapping and no namespace",
		TeamsTimelineSpec:         "collect without namespace",
		WorkflowTimelineSpec:      "retry without namespace",
		A2ATimelineSpec:           "remote peer identifier and callback-retry",
		A2ACoreSpec:               "submitted only",
		TeamsRuntimeConfigSpec:    "teams config",
		WorkflowRuntimeConfigSpec: "workflow config",
		A2ARuntimeConfigSpec:      "a2a config with `a2a_peer`",
		TeamsBoundarySpec:         "no gate",
		WorkflowBoundarySpec:      "no gate",
		A2ABoundarySpec:           "no gate",
	}

	violations := ValidateMultiAgentSharedContractSnapshot(snapshot)
	if len(violations) == 0 {
		t.Fatal("expected violations, got none")
	}
	codes := make(map[string]struct{}, len(violations))
	for _, v := range violations {
		codes[v.Code] = struct{}{}
	}

	required := []string{
		"missing_status_mapping_a2a_submitted_pending",
		"missing_a2a_submitted_pending_alignment",
		"missing_reason_namespace_contract",
		"missing_reason_team_dispatch",
		"missing_reason_workflow_schedule",
		"missing_reason_a2a_submit",
		"missing_peer_id_canonical_naming",
		"deprecated_a2a_peer_field_detected",
		"missing_domain_scoped_config_namespaces",
		"missing_blocking_shared_contract_gate",
	}
	for _, code := range required {
		if _, ok := codes[code]; !ok {
			t.Fatalf("missing expected violation code %q, got %+v", code, violations)
		}
	}
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(raw)
}
