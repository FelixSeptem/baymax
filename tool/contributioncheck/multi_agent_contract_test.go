package contributioncheck

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestMultiAgentSharedContractSnapshotPass(t *testing.T) {
	root := repoRoot(t)
	a2aTimelineSpec := strings.Join([]string{
		mustReadChangeSpec(t, root, "a2a-minimal-interoperability", filepath.Join("specs", "action-timeline-events", "spec.md")),
		mustReadChangeSpec(t, root, "harden-a2a-delivery-and-card-version-negotiation-a4", filepath.Join("specs", "action-timeline-events", "spec.md")),
		mustReadChangeSpec(t, root, "compose-teams-workflow-with-a2a-remote-execution-a5", filepath.Join("specs", "action-timeline-events", "spec.md")),
		mustReadChangeSpec(t, root, "introduce-async-agent-reporting-contract-a12", filepath.Join("specs", "action-timeline-events", "spec.md")),
		mustReadChangeSpec(t, root, "close-a12-a13-tail-contract-and-compatibility-governance-a14", filepath.Join("specs", "action-timeline-events", "spec.md")),
	}, "\n")
	a2aCoreSpec := strings.Join([]string{
		mustReadChangeSpec(t, root, "a2a-minimal-interoperability", filepath.Join("specs", "a2a-minimal-interoperability", "spec.md")),
		mustReadChangeSpec(t, root, "harden-a2a-delivery-and-card-version-negotiation-a4", filepath.Join("specs", "a2a-delivery-and-version-negotiation", "spec.md")),
		mustReadChangeSpec(t, root, "compose-teams-workflow-with-a2a-remote-execution-a5", filepath.Join("specs", "a2a-minimal-interoperability", "spec.md")),
		mustReadChangeSpec(t, root, "introduce-async-agent-reporting-contract-a12", filepath.Join("specs", "a2a-minimal-interoperability", "spec.md")),
		mustReadChangeSpec(t, root, "introduce-async-agent-reporting-contract-a12", filepath.Join("specs", "multi-agent-async-reporting", "spec.md")),
	}, "\n")
	a2aRuntimeConfigSpec := strings.Join([]string{
		mustReadChangeSpec(t, root, "a2a-minimal-interoperability", filepath.Join("specs", "runtime-config-and-diagnostics-api", "spec.md")),
		mustReadChangeSpec(t, root, "harden-a2a-delivery-and-card-version-negotiation-a4", filepath.Join("specs", "runtime-config-and-diagnostics-api", "spec.md")),
		mustReadChangeSpec(t, root, "compose-teams-workflow-with-a2a-remote-execution-a5", filepath.Join("specs", "runtime-config-and-diagnostics-api", "spec.md")),
		mustReadChangeSpec(t, root, "introduce-async-agent-reporting-contract-a12", filepath.Join("specs", "runtime-config-and-diagnostics-api", "spec.md")),
		mustReadChangeSpec(t, root, "close-a12-a13-tail-contract-and-compatibility-governance-a14", filepath.Join("specs", "runtime-config-and-diagnostics-api", "spec.md")),
	}, "\n")
	a2aBoundarySpec := strings.Join([]string{
		mustReadChangeSpec(t, root, "a2a-minimal-interoperability", filepath.Join("specs", "runtime-module-boundaries", "spec.md")),
		mustReadChangeSpec(t, root, "harden-a2a-delivery-and-card-version-negotiation-a4", filepath.Join("specs", "runtime-module-boundaries", "spec.md")),
		mustReadChangeSpec(t, root, "compose-teams-workflow-with-a2a-remote-execution-a5", filepath.Join("specs", "runtime-module-boundaries", "spec.md")),
		mustReadChangeSpec(t, root, "introduce-async-agent-reporting-contract-a12", filepath.Join("specs", "a2a-minimal-interoperability", "spec.md")),
	}, "\n")
	teamsTimelineSpec := strings.Join([]string{
		mustReadChangeSpec(t, root, "teams-runtime-baseline", filepath.Join("specs", "action-timeline-events", "spec.md")),
		mustReadChangeSpec(t, root, "compose-teams-workflow-with-a2a-remote-execution-a5", filepath.Join("specs", "action-timeline-events", "spec.md")),
	}, "\n")
	workflowTimelineSpec := strings.Join([]string{
		mustReadChangeSpec(t, root, "workflow-dsl-baseline", filepath.Join("specs", "action-timeline-events", "spec.md")),
		mustReadChangeSpec(t, root, "compose-teams-workflow-with-a2a-remote-execution-a5", filepath.Join("specs", "action-timeline-events", "spec.md")),
	}, "\n")
	schedulerTimelineSpec := strings.Join([]string{
		mustReadChangeSpec(t, root, "distributed-subagent-scheduler-baseline-a6", filepath.Join("specs", "action-timeline-events", "spec.md")),
		mustReadChangeSpec(t, root, "close-a5-a6-tail-contract-and-governance-a7", filepath.Join("specs", "action-timeline-events", "spec.md")),
		mustReadChangeSpec(t, root, "introduce-lib-first-agent-composer-with-scheduler-bridge-a8", filepath.Join("specs", "action-timeline-events", "spec.md")),
		mustReadChangeSpec(t, root, "harden-composed-session-recovery-and-deterministic-replay-a9", filepath.Join("specs", "action-timeline-events", "spec.md")),
		mustReadChangeSpec(t, root, "introduce-scheduler-qos-fairness-and-deadletter-governance-a10", filepath.Join("specs", "action-timeline-events", "spec.md")),
		mustReadChangeSpec(t, root, "introduce-delayed-dispatch-not-before-contract-a13", filepath.Join("specs", "action-timeline-events", "spec.md")),
		mustReadChangeSpec(t, root, "close-a12-a13-tail-contract-and-compatibility-governance-a14", filepath.Join("specs", "action-timeline-events", "spec.md")),
	}, "\n")
	teamsRuntimeConfigSpec := strings.Join([]string{
		mustReadChangeSpec(t, root, "teams-runtime-baseline", filepath.Join("specs", "runtime-config-and-diagnostics-api", "spec.md")),
		mustReadChangeSpec(t, root, "compose-teams-workflow-with-a2a-remote-execution-a5", filepath.Join("specs", "runtime-config-and-diagnostics-api", "spec.md")),
	}, "\n")
	workflowRuntimeConfigSpec := strings.Join([]string{
		mustReadChangeSpec(t, root, "workflow-dsl-baseline", filepath.Join("specs", "runtime-config-and-diagnostics-api", "spec.md")),
		mustReadChangeSpec(t, root, "compose-teams-workflow-with-a2a-remote-execution-a5", filepath.Join("specs", "runtime-config-and-diagnostics-api", "spec.md")),
	}, "\n")
	schedulerRuntimeConfigSpec := strings.Join([]string{
		mustReadChangeSpec(t, root, "distributed-subagent-scheduler-baseline-a6", filepath.Join("specs", "runtime-config-and-diagnostics-api", "spec.md")),
		mustReadChangeSpec(t, root, "close-a5-a6-tail-contract-and-governance-a7", filepath.Join("specs", "runtime-config-and-diagnostics-api", "spec.md")),
		mustReadChangeSpec(t, root, "introduce-lib-first-agent-composer-with-scheduler-bridge-a8", filepath.Join("specs", "runtime-config-and-diagnostics-api", "spec.md")),
		mustReadChangeSpec(t, root, "harden-composed-session-recovery-and-deterministic-replay-a9", filepath.Join("specs", "runtime-config-and-diagnostics-api", "spec.md")),
		mustReadChangeSpec(t, root, "introduce-scheduler-qos-fairness-and-deadletter-governance-a10", filepath.Join("specs", "runtime-config-and-diagnostics-api", "spec.md")),
		mustReadChangeSpec(t, root, "introduce-delayed-dispatch-not-before-contract-a13", filepath.Join("specs", "runtime-config-and-diagnostics-api", "spec.md")),
		mustReadChangeSpec(t, root, "close-a12-a13-tail-contract-and-compatibility-governance-a14", filepath.Join("specs", "runtime-config-and-diagnostics-api", "spec.md")),
	}, "\n")
	teamsBoundarySpec := strings.Join([]string{
		mustReadChangeSpec(t, root, "teams-runtime-baseline", filepath.Join("specs", "runtime-module-boundaries", "spec.md")),
		mustReadChangeSpec(t, root, "compose-teams-workflow-with-a2a-remote-execution-a5", filepath.Join("specs", "runtime-module-boundaries", "spec.md")),
	}, "\n")
	workflowBoundarySpec := strings.Join([]string{
		mustReadChangeSpec(t, root, "workflow-dsl-baseline", filepath.Join("specs", "runtime-module-boundaries", "spec.md")),
		mustReadChangeSpec(t, root, "compose-teams-workflow-with-a2a-remote-execution-a5", filepath.Join("specs", "runtime-module-boundaries", "spec.md")),
	}, "\n")
	schedulerBoundarySpec := strings.Join([]string{
		mustReadChangeSpec(t, root, "distributed-subagent-scheduler-baseline-a6", filepath.Join("specs", "runtime-module-boundaries", "spec.md")),
		mustReadChangeSpec(t, root, "close-a5-a6-tail-contract-and-governance-a7", filepath.Join("specs", "runtime-module-boundaries", "spec.md")),
		mustReadChangeSpec(t, root, "introduce-lib-first-agent-composer-with-scheduler-bridge-a8", filepath.Join("specs", "runtime-module-boundaries", "spec.md")),
		mustReadChangeSpec(t, root, "harden-composed-session-recovery-and-deterministic-replay-a9", filepath.Join("specs", "runtime-module-boundaries", "spec.md")),
		mustReadChangeSpec(t, root, "introduce-scheduler-qos-fairness-and-deadletter-governance-a10", filepath.Join("specs", "runtime-module-boundaries", "spec.md")),
	}, "\n")
	composerCoreSpec := strings.Join([]string{
		mustReadChangeSpec(t, root, "introduce-lib-first-agent-composer-with-scheduler-bridge-a8", filepath.Join("specs", "multi-agent-lib-first-composer", "spec.md")),
		mustReadChangeSpec(t, root, "introduce-lib-first-agent-composer-with-scheduler-bridge-a8", filepath.Join("specs", "multi-agent-composed-orchestration", "spec.md")),
		mustReadChangeSpec(t, root, "harden-composed-session-recovery-and-deterministic-replay-a9", filepath.Join("specs", "multi-agent-composed-orchestration", "spec.md")),
		mustReadChangeSpec(t, root, "harden-composed-session-recovery-and-deterministic-replay-a9", filepath.Join("specs", "multi-agent-session-recovery", "spec.md")),
		mustReadChangeSpec(t, root, "introduce-async-agent-reporting-contract-a12", filepath.Join("specs", "multi-agent-composed-orchestration", "spec.md")),
		mustReadChangeSpec(t, root, "introduce-async-agent-reporting-contract-a12", filepath.Join("specs", "multi-agent-lib-first-composer", "spec.md")),
		mustReadChangeSpec(t, root, "introduce-delayed-dispatch-not-before-contract-a13", filepath.Join("specs", "multi-agent-lib-first-composer", "spec.md")),
	}, "\n")
	composerGateSpec := strings.Join([]string{
		mustReadChangeSpec(t, root, "introduce-lib-first-agent-composer-with-scheduler-bridge-a8", filepath.Join("specs", "go-quality-gate", "spec.md")),
		mustReadChangeSpec(t, root, "harden-composed-session-recovery-and-deterministic-replay-a9", filepath.Join("specs", "go-quality-gate", "spec.md")),
		mustReadChangeSpec(t, root, "introduce-scheduler-qos-fairness-and-deadletter-governance-a10", filepath.Join("specs", "go-quality-gate", "spec.md")),
		mustReadChangeSpec(t, root, "introduce-async-agent-reporting-contract-a12", filepath.Join("specs", "go-quality-gate", "spec.md")),
		mustReadChangeSpec(t, root, "introduce-delayed-dispatch-not-before-contract-a13", filepath.Join("specs", "go-quality-gate", "spec.md")),
		mustReadChangeSpec(t, root, "close-a12-a13-tail-contract-and-compatibility-governance-a14", filepath.Join("specs", "go-quality-gate", "spec.md")),
	}, "\n")

	snapshot := MultiAgentContractSnapshot{
		IdentifierDoc:              mustRead(t, filepath.Join(root, "docs", "multi-agent-identifier-model.md")),
		RuntimeConfigDoc:           mustRead(t, filepath.Join(root, "docs", "runtime-config-diagnostics.md")),
		V1AcceptanceDoc:            mustRead(t, filepath.Join(root, "docs", "v1-acceptance.md")),
		ComposerCoreSpec:           composerCoreSpec,
		ComposerGateSpec:           composerGateSpec,
		TeamsTimelineSpec:          teamsTimelineSpec,
		WorkflowTimelineSpec:       workflowTimelineSpec,
		A2ATimelineSpec:            a2aTimelineSpec,
		SchedulerTimelineSpec:      schedulerTimelineSpec,
		A2ACoreSpec:                a2aCoreSpec,
		TeamsRuntimeConfigSpec:     teamsRuntimeConfigSpec,
		WorkflowRuntimeConfigSpec:  workflowRuntimeConfigSpec,
		A2ARuntimeConfigSpec:       a2aRuntimeConfigSpec,
		SchedulerRuntimeConfigSpec: schedulerRuntimeConfigSpec,
		TeamsBoundarySpec:          teamsBoundarySpec,
		WorkflowBoundarySpec:       workflowBoundarySpec,
		A2ABoundarySpec:            a2aBoundarySpec,
		SchedulerBoundarySpec:      schedulerBoundarySpec,
	}

	violations := ValidateMultiAgentSharedContractSnapshot(snapshot)
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %+v", violations)
	}
}

func TestValidateMultiAgentSharedContractDetectsViolations(t *testing.T) {
	snapshot := MultiAgentContractSnapshot{
		IdentifierDoc:              "no mapping and no namespace",
		RuntimeConfigDoc:           "runtime doc missing composed reasons and summary fields",
		V1AcceptanceDoc:            "acceptance doc missing composed markers",
		ComposerCoreSpec:           "manual assembly only",
		ComposerGateSpec:           "quality gate missing composer suite",
		TeamsTimelineSpec:          "collect without namespace",
		WorkflowTimelineSpec:       "retry without namespace",
		A2ATimelineSpec:            "remote peer identifier and callback-retry only",
		SchedulerTimelineSpec:      "scheduler reason without namespace",
		A2ACoreSpec:                "submitted only",
		TeamsRuntimeConfigSpec:     "teams config",
		WorkflowRuntimeConfigSpec:  "workflow config",
		A2ARuntimeConfigSpec:       "a2a config with `a2a_peer` and `a2aDeliveryMode`",
		SchedulerRuntimeConfigSpec: "scheduler config",
		TeamsBoundarySpec:          "no gate",
		WorkflowBoundarySpec:       "no gate",
		A2ABoundarySpec:            "no gate",
		SchedulerBoundarySpec:      "no gate",
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
		"missing_composer_entrypoint_contract",
		"missing_reason_namespace_contract",
		"missing_reason_team_dispatch",
		"missing_reason_team_dispatch_remote",
		"missing_reason_team_collect_remote",
		"missing_reason_workflow_schedule",
		"missing_reason_workflow_dispatch_a2a",
		"missing_reason_a2a_submit",
		"missing_reason_a2a_sse_subscribe",
		"missing_reason_a2a_version_mismatch",
		"missing_reason_scheduler_enqueue",
		"missing_reason_scheduler_delayed_enqueue",
		"missing_reason_scheduler_delayed_wait",
		"missing_reason_scheduler_delayed_ready",
		"missing_reason_scheduler_claim",
		"missing_reason_scheduler_heartbeat",
		"missing_reason_scheduler_lease_expired",
		"missing_reason_scheduler_requeue",
		"missing_reason_scheduler_qos_claim",
		"missing_reason_scheduler_fairness_yield",
		"missing_reason_scheduler_retry_backoff",
		"missing_reason_scheduler_dead_letter",
		"missing_reason_subagent_spawn",
		"missing_reason_subagent_join",
		"missing_reason_subagent_budget_reject",
		"missing_reason_recovery_restore",
		"missing_reason_recovery_replay",
		"missing_reason_recovery_conflict",
		"missing_scheduler_timeline_field_run_id",
		"missing_scheduler_timeline_field_task_id",
		"missing_scheduler_timeline_field_attempt_id",
		"missing_peer_id_canonical_naming",
		"missing_identifier_field_workflow_id",
		"missing_identifier_field_team_id",
		"missing_identifier_field_step_id",
		"missing_identifier_field_task_id",
		"missing_identifier_field_attempt_id",
		"missing_identifier_field_agent_id",
		"missing_identifier_field_peer_id",
		"missing_identifier_summary_field_team_remote_task_total",
		"missing_identifier_summary_field_team_remote_task_failed",
		"missing_identifier_summary_field_workflow_remote_step_total",
		"missing_identifier_summary_field_workflow_remote_step_failed",
		"missing_identifier_summary_field_scheduler_queue_total",
		"missing_identifier_summary_field_scheduler_claim_total",
		"missing_identifier_summary_field_scheduler_reclaim_total",
		"missing_identifier_summary_field_scheduler_qos_mode",
		"missing_identifier_summary_field_scheduler_priority_claim_total",
		"missing_identifier_summary_field_scheduler_fairness_yield_total",
		"missing_identifier_summary_field_scheduler_retry_backoff_total",
		"missing_identifier_summary_field_scheduler_dead_letter_total",
		"missing_identifier_summary_field_scheduler_delayed_task_total",
		"missing_identifier_summary_field_scheduler_delayed_claim_total",
		"missing_identifier_summary_field_scheduler_delayed_wait_ms_p95",
		"missing_identifier_summary_field_subagent_child_total",
		"missing_identifier_summary_field_subagent_child_failed",
		"missing_identifier_summary_field_subagent_budget_reject_total",
		"missing_identifier_summary_field_recovery_enabled",
		"missing_identifier_summary_field_recovery_recovered",
		"missing_identifier_summary_field_recovery_replay_total",
		"missing_identifier_summary_field_recovery_conflict",
		"missing_identifier_summary_field_recovery_conflict_code",
		"missing_identifier_summary_field_recovery_fallback_used",
		"missing_identifier_summary_field_recovery_fallback_reason",
		"missing_runtime_doc_reason_team_dispatch_remote",
		"missing_runtime_doc_reason_team_collect_remote",
		"missing_runtime_doc_reason_workflow_dispatch_a2a",
		"missing_runtime_doc_reason_scheduler_enqueue",
		"missing_runtime_doc_reason_scheduler_delayed_enqueue",
		"missing_runtime_doc_reason_scheduler_delayed_wait",
		"missing_runtime_doc_reason_scheduler_delayed_ready",
		"missing_runtime_doc_reason_scheduler_claim",
		"missing_runtime_doc_reason_scheduler_heartbeat",
		"missing_runtime_doc_reason_scheduler_lease_expired",
		"missing_runtime_doc_reason_scheduler_requeue",
		"missing_runtime_doc_reason_scheduler_qos_claim",
		"missing_runtime_doc_reason_scheduler_fairness_yield",
		"missing_runtime_doc_reason_scheduler_retry_backoff",
		"missing_runtime_doc_reason_scheduler_dead_letter",
		"missing_runtime_doc_reason_subagent_spawn",
		"missing_runtime_doc_reason_subagent_join",
		"missing_runtime_doc_reason_subagent_budget_reject",
		"missing_runtime_doc_reason_recovery_restore",
		"missing_runtime_doc_reason_recovery_replay",
		"missing_runtime_doc_reason_recovery_conflict",
		"missing_runtime_doc_field_team_remote_task_total",
		"missing_runtime_doc_field_team_remote_task_failed",
		"missing_runtime_doc_field_workflow_remote_step_total",
		"missing_runtime_doc_field_workflow_remote_step_failed",
		"missing_runtime_doc_field_composer_managed",
		"missing_runtime_doc_field_scheduler_backend_fallback",
		"missing_runtime_doc_field_scheduler_backend_fallback_reason",
		"missing_runtime_doc_field_scheduler_backend",
		"missing_runtime_doc_field_scheduler_queue_total",
		"missing_runtime_doc_field_scheduler_claim_total",
		"missing_runtime_doc_field_scheduler_reclaim_total",
		"missing_runtime_doc_field_scheduler_qos_mode",
		"missing_runtime_doc_field_scheduler_priority_claim_total",
		"missing_runtime_doc_field_scheduler_fairness_yield_total",
		"missing_runtime_doc_field_scheduler_retry_backoff_total",
		"missing_runtime_doc_field_scheduler_dead_letter_total",
		"missing_runtime_doc_field_scheduler_delayed_task_total",
		"missing_runtime_doc_field_scheduler_delayed_claim_total",
		"missing_runtime_doc_field_scheduler_delayed_wait_ms_p95",
		"missing_runtime_doc_field_subagent_child_total",
		"missing_runtime_doc_field_subagent_child_failed",
		"missing_runtime_doc_field_subagent_budget_reject_total",
		"missing_runtime_doc_field_recovery_enabled",
		"missing_runtime_doc_field_recovery_recovered",
		"missing_runtime_doc_field_recovery_replay_total",
		"missing_runtime_doc_field_recovery_conflict",
		"missing_runtime_doc_field_recovery_conflict_code",
		"missing_runtime_doc_field_recovery_fallback_used",
		"missing_runtime_doc_field_recovery_fallback_reason",
		"missing_runtime_doc_compatibility_window_title",
		"missing_runtime_doc_compatibility_window_rule",
		"missing_runtime_doc_compatibility_window_default_rule",
		"missing_runtime_doc_compatibility_window_ignore_unknown_rule",
		"missing_runtime_doc_compatibility_window_stable_existing_rule",
		"missing_runtime_doc_env_mapping_baymax_teams_remote_enabled",
		"missing_runtime_doc_env_mapping_baymax_teams_remote_require_peer_id",
		"missing_runtime_doc_env_mapping_baymax_workflow_remote_enabled",
		"missing_runtime_doc_env_mapping_baymax_workflow_remote_default_retry_max_attempts",
		"missing_runtime_doc_env_mapping_baymax_scheduler_enabled",
		"missing_runtime_doc_env_mapping_baymax_scheduler_backend",
		"missing_runtime_doc_env_mapping_baymax_scheduler_lease_timeout",
		"missing_runtime_doc_env_mapping_baymax_scheduler_heartbeat_interval",
		"missing_runtime_doc_env_mapping_baymax_scheduler_qos_mode",
		"missing_runtime_doc_env_mapping_baymax_scheduler_qos_fairness_max_consecutive_claims_per_priority",
		"missing_runtime_doc_env_mapping_baymax_scheduler_dlq_enabled",
		"missing_runtime_doc_env_mapping_baymax_scheduler_retry_backoff_enabled",
		"missing_runtime_doc_env_mapping_baymax_scheduler_retry_backoff_initial",
		"missing_runtime_doc_env_mapping_baymax_scheduler_retry_backoff_max",
		"missing_runtime_doc_env_mapping_baymax_scheduler_retry_backoff_multiplier",
		"missing_runtime_doc_env_mapping_baymax_scheduler_retry_backoff_jitter_ratio",
		"missing_runtime_doc_env_mapping_baymax_subagent_max_depth",
		"missing_runtime_doc_env_mapping_baymax_subagent_max_active_children",
		"missing_runtime_doc_env_mapping_baymax_subagent_child_timeout_budget",
		"missing_scheduler_runtime_spec_field_composer_managed",
		"missing_scheduler_runtime_spec_field_scheduler_backend_fallback",
		"missing_scheduler_runtime_spec_field_scheduler_backend_fallback_reason",
		"missing_scheduler_runtime_spec_field_scheduler_qos_mode",
		"missing_scheduler_runtime_spec_field_scheduler_priority_claim_total",
		"missing_scheduler_runtime_spec_field_scheduler_fairness_yield_total",
		"missing_scheduler_runtime_spec_field_scheduler_retry_backoff_total",
		"missing_scheduler_runtime_spec_field_scheduler_dead_letter_total",
		"missing_scheduler_runtime_spec_field_scheduler_delayed_task_total",
		"missing_scheduler_runtime_spec_field_scheduler_delayed_claim_total",
		"missing_scheduler_runtime_spec_field_scheduler_delayed_wait_ms_p95",
		"missing_scheduler_runtime_spec_field_recovery_enabled",
		"missing_scheduler_runtime_spec_field_recovery_recovered",
		"missing_scheduler_runtime_spec_field_recovery_replay_total",
		"missing_scheduler_runtime_spec_field_recovery_conflict",
		"missing_scheduler_runtime_spec_field_recovery_conflict_code",
		"missing_scheduler_runtime_spec_field_recovery_fallback_used",
		"missing_scheduler_runtime_spec_field_recovery_fallback_reason",
		"missing_v1_acceptance_marker_teams_remote_*",
		"missing_v1_acceptance_marker_workflow_remote_*",
		"missing_v1_acceptance_marker_team_remote_task_total",
		"missing_v1_acceptance_marker_workflow_remote_step_total",
		"missing_v1_acceptance_marker_scheduler_*",
		"missing_v1_acceptance_marker_scheduler_qos_*",
		"missing_v1_acceptance_marker_scheduler_dlq_*",
		"missing_v1_acceptance_marker_subagent_*",
		"missing_v1_acceptance_marker_scheduler_queue_total",
		"missing_v1_acceptance_marker_scheduler_qos_mode",
		"missing_v1_acceptance_marker_subagent_child_total",
		"missing_v1_acceptance_marker_recovery_*",
		"missing_v1_acceptance_marker_recovery_enabled",
		"missing_v1_acceptance_compatibility_window_marker",
		"missing_v1_acceptance_a12_a13_compatibility_marker",
		"missing_a2a_timeline_field_delivery_mode",
		"missing_a2a_summary_field_a2a_delivery_mode",
		"non_snake_case_a2a_field_detected",
		"deprecated_a2a_peer_field_detected",
		"missing_domain_scoped_config_namespaces",
		"missing_teams_remote_config_contract",
		"missing_workflow_remote_config_contract",
		"missing_composed_summary_contract",
		"missing_blocking_shared_contract_gate",
		"missing_composer_boundary_contract",
		"missing_composer_gate_contract",
		"missing_single_blocking_gate_marker",
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

func mustReadChangeSpec(t *testing.T, root, changeName, relPath string) string {
	t.Helper()
	active := filepath.Join(root, "openspec", "changes", changeName, relPath)
	if _, err := os.Stat(active); err == nil {
		return mustRead(t, active)
	}

	archiveRoot := filepath.Join(root, "openspec", "changes", "archive")
	dirs, err := os.ReadDir(archiveRoot)
	if err != nil {
		t.Fatalf("read archive root: %v", err)
	}
	prefix := "-" + changeName
	candidates := make([]string, 0)
	for _, entry := range dirs {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, prefix) {
			candidates = append(candidates, filepath.Join(archiveRoot, name, relPath))
		}
	}
	if len(candidates) == 0 {
		t.Fatalf("change %q not found in active or archive", changeName)
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i] > candidates[j] })
	return mustRead(t, candidates[0])
}
