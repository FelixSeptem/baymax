package contributioncheck

import "strings"

type MultiAgentContractSnapshot struct {
	IdentifierDoc              string
	RuntimeConfigDoc           string
	MainlineContractIndexDoc   string
	V1AcceptanceDoc            string
	ComposerCoreSpec           string
	ComposerGateSpec           string
	UnifiedQuerySpec           string
	TeamsTimelineSpec          string
	WorkflowTimelineSpec       string
	A2ATimelineSpec            string
	SchedulerTimelineSpec      string
	A2ACoreSpec                string
	TeamsRuntimeConfigSpec     string
	WorkflowRuntimeConfigSpec  string
	A2ARuntimeConfigSpec       string
	SchedulerRuntimeConfigSpec string
	TeamsBoundarySpec          string
	WorkflowBoundarySpec       string
	A2ABoundarySpec            string
	SchedulerBoundarySpec      string
}

func ValidateMultiAgentSharedContractSnapshot(snapshot MultiAgentContractSnapshot) []Violation {
	violations := make([]Violation, 0)

	if !strings.Contains(snapshot.IdentifierDoc, "| a2a | `submitted` | `pending` |") {
		violations = append(violations, Violation{
			Code:    "missing_status_mapping_a2a_submitted_pending",
			Message: "identifier model must include mapping a2a submitted -> pending",
		})
	}
	if !strings.Contains(snapshot.A2ACoreSpec, "submitted") || !strings.Contains(snapshot.A2ACoreSpec, "pending") {
		violations = append(violations, Violation{
			Code:    "missing_a2a_submitted_pending_alignment",
			Message: "a2a lifecycle spec must align submitted with pending semantic layer",
		})
	}
	if !strings.Contains(snapshot.ComposerCoreSpec, "`orchestration/composer`") ||
		!strings.Contains(strings.ToLower(snapshot.ComposerCoreSpec), "run and stream") {
		violations = append(violations, Violation{
			Code:    "missing_composer_entrypoint_contract",
			Message: "composer spec must define orchestration/composer Run and Stream entrypoint contract",
		})
	}

	if !strings.Contains(snapshot.IdentifierDoc, "`team.*`") ||
		!strings.Contains(snapshot.IdentifierDoc, "`workflow.*`") ||
		!strings.Contains(snapshot.IdentifierDoc, "`a2a.*`") ||
		!strings.Contains(snapshot.IdentifierDoc, "`scheduler.*`") ||
		!strings.Contains(snapshot.IdentifierDoc, "`subagent.*`") {
		violations = append(violations, Violation{
			Code:    "missing_reason_namespace_contract",
			Message: "identifier model must define team/workflow/a2a/scheduler/subagent reason namespaces",
		})
	}

	requiredReasons := map[string]string{
		"team.dispatch":             snapshot.TeamsTimelineSpec,
		"team.collect":              snapshot.TeamsTimelineSpec,
		"team.resolve":              snapshot.TeamsTimelineSpec,
		"team.dispatch_remote":      snapshot.TeamsTimelineSpec,
		"team.collect_remote":       snapshot.TeamsTimelineSpec,
		"team.handoff":              snapshot.TeamsTimelineSpec,
		"team.delegation":           snapshot.TeamsTimelineSpec,
		"team.aggregation":          snapshot.TeamsTimelineSpec,
		"workflow.schedule":         snapshot.WorkflowTimelineSpec,
		"workflow.retry":            snapshot.WorkflowTimelineSpec,
		"workflow.resume":           snapshot.WorkflowTimelineSpec,
		"workflow.dispatch_a2a":     snapshot.WorkflowTimelineSpec,
		"workflow.handoff":          snapshot.WorkflowTimelineSpec,
		"workflow.delegation":       snapshot.WorkflowTimelineSpec,
		"workflow.aggregation":      snapshot.WorkflowTimelineSpec,
		"a2a.submit":                snapshot.A2ATimelineSpec,
		"a2a.status_poll":           snapshot.A2ATimelineSpec,
		"a2a.callback_retry":        snapshot.A2ATimelineSpec,
		"a2a.resolve":               snapshot.A2ATimelineSpec,
		"a2a.sse_subscribe":         snapshot.A2ATimelineSpec,
		"a2a.sse_reconnect":         snapshot.A2ATimelineSpec,
		"a2a.delivery_fallback":     snapshot.A2ATimelineSpec,
		"a2a.version_mismatch":      snapshot.A2ATimelineSpec,
		"a2a.async_submit":          snapshot.A2ATimelineSpec,
		"a2a.async_report_deliver":  snapshot.A2ATimelineSpec,
		"a2a.async_report_retry":    snapshot.A2ATimelineSpec,
		"a2a.async_report_dedup":    snapshot.A2ATimelineSpec,
		"a2a.async_report_drop":     snapshot.A2ATimelineSpec,
		"scheduler.enqueue":         snapshot.SchedulerTimelineSpec,
		"scheduler.delayed_enqueue": snapshot.SchedulerTimelineSpec,
		"scheduler.delayed_wait":    snapshot.SchedulerTimelineSpec,
		"scheduler.delayed_ready":   snapshot.SchedulerTimelineSpec,
		"scheduler.claim":           snapshot.SchedulerTimelineSpec,
		"scheduler.heartbeat":       snapshot.SchedulerTimelineSpec,
		"scheduler.lease_expired":   snapshot.SchedulerTimelineSpec,
		"scheduler.requeue":         snapshot.SchedulerTimelineSpec,
		"scheduler.qos_claim":       snapshot.SchedulerTimelineSpec,
		"scheduler.fairness_yield":  snapshot.SchedulerTimelineSpec,
		"scheduler.retry_backoff":   snapshot.SchedulerTimelineSpec,
		"scheduler.dead_letter":     snapshot.SchedulerTimelineSpec,
		"subagent.spawn":            snapshot.SchedulerTimelineSpec,
		"subagent.join":             snapshot.SchedulerTimelineSpec,
		"subagent.budget_reject":    snapshot.SchedulerTimelineSpec,
		"recovery.restore":          snapshot.SchedulerTimelineSpec,
		"recovery.replay":           snapshot.SchedulerTimelineSpec,
		"recovery.conflict":         snapshot.SchedulerTimelineSpec,
	}
	for reason, source := range requiredReasons {
		if !strings.Contains(source, reason) {
			violations = append(violations, Violation{
				Code:    "missing_reason_" + strings.ReplaceAll(reason, ".", "_"),
				Message: "missing required namespaced reason: " + reason,
			})
		}
	}
	requiredSchedulerTimelineFields := []string{
		"`run_id`",
		"`task_id`",
		"`attempt_id`",
	}
	for _, field := range requiredSchedulerTimelineFields {
		if !strings.Contains(snapshot.SchedulerTimelineSpec, field) {
			violations = append(violations, Violation{
				Code:    "missing_scheduler_timeline_field_" + strings.Trim(field, "`"),
				Message: "missing required scheduler timeline correlation field: " + field,
			})
		}
	}

	if !strings.Contains(snapshot.IdentifierDoc, "`peer_id`") ||
		!strings.Contains(snapshot.A2ATimelineSpec, "`peer_id`") ||
		!strings.Contains(snapshot.A2ARuntimeConfigSpec, "`peer_id`") {
		violations = append(violations, Violation{
			Code:    "missing_peer_id_canonical_naming",
			Message: "peer_id must be used as canonical A2A peer identifier field",
		})
	}
	requiredCorrelationFields := []string{
		"`workflow_id`",
		"`team_id`",
		"`step_id`",
		"`task_id`",
		"`attempt_id`",
		"`agent_id`",
		"`peer_id`",
	}
	for _, field := range requiredCorrelationFields {
		if !strings.Contains(snapshot.IdentifierDoc, field) {
			violations = append(violations, Violation{
				Code:    "missing_identifier_field_" + strings.Trim(field, "`"),
				Message: "identifier model missing canonical field: " + field,
			})
		}
	}

	requiredA2ATimelineFields := []string{
		"`task_id`",
		"`agent_id`",
		"`peer_id`",
		"`delivery_mode`",
		"`version_local`",
		"`version_peer`",
	}
	for _, field := range requiredA2ATimelineFields {
		if !strings.Contains(snapshot.A2ATimelineSpec, field) {
			violations = append(violations, Violation{
				Code:    "missing_a2a_timeline_field_" + strings.Trim(field, "`"),
				Message: "missing required A2A timeline correlation field: " + field,
			})
		}
	}

	requiredA2ASummaryFields := []string{
		"`a2a_delivery_mode`",
		"`a2a_delivery_fallback_used`",
		"`a2a_delivery_fallback_reason`",
		"`a2a_version_local`",
		"`a2a_version_peer`",
		"`a2a_version_negotiation_result`",
		"`a2a_async_report_total`",
		"`a2a_async_report_failed`",
		"`a2a_async_report_retry_total`",
		"`a2a_async_report_dedup_total`",
	}
	for _, field := range requiredA2ASummaryFields {
		if !strings.Contains(snapshot.A2ARuntimeConfigSpec, field) {
			violations = append(violations, Violation{
				Code:    "missing_a2a_summary_field_" + strings.Trim(field, "`"),
				Message: "missing required A2A additive summary field: " + field,
			})
		}
	}

	requiredComposedSummaryFields := []string{
		"`team_remote_task_total`",
		"`team_remote_task_failed`",
		"`workflow_remote_step_total`",
		"`workflow_remote_step_failed`",
		"`collab_handoff_total`",
		"`collab_delegation_total`",
		"`collab_aggregation_total`",
		"`collab_aggregation_strategy`",
		"`collab_fail_fast_total`",
		"`scheduler_queue_total`",
		"`scheduler_claim_total`",
		"`scheduler_reclaim_total`",
		"`scheduler_qos_mode`",
		"`scheduler_priority_claim_total`",
		"`scheduler_fairness_yield_total`",
		"`scheduler_retry_backoff_total`",
		"`scheduler_dead_letter_total`",
		"`scheduler_delayed_task_total`",
		"`scheduler_delayed_claim_total`",
		"`scheduler_delayed_wait_ms_p95`",
		"`subagent_child_total`",
		"`subagent_child_failed`",
		"`subagent_budget_reject_total`",
		"`recovery_enabled`",
		"`recovery_resume_boundary`",
		"`recovery_inflight_policy`",
		"`recovery_recovered`",
		"`recovery_replay_total`",
		"`recovery_timeout_reentry_total`",
		"`recovery_timeout_reentry_exhausted_total`",
		"`recovery_conflict`",
		"`recovery_conflict_code`",
		"`recovery_fallback_used`",
		"`recovery_fallback_reason`",
	}
	for _, field := range requiredComposedSummaryFields {
		if !strings.Contains(snapshot.IdentifierDoc, field) {
			violations = append(violations, Violation{
				Code:    "missing_identifier_summary_field_" + strings.Trim(field, "`"),
				Message: "identifier model missing composed additive summary field: " + field,
			})
		}
	}

	requiredComposedReasonsInDoc := []string{
		"`team.dispatch_remote`",
		"`team.collect_remote`",
		"`team.handoff`",
		"`team.delegation`",
		"`team.aggregation`",
		"`workflow.dispatch_a2a`",
		"`workflow.handoff`",
		"`workflow.delegation`",
		"`workflow.aggregation`",
		"`a2a.async_submit`",
		"`a2a.async_report_deliver`",
		"`a2a.async_report_retry`",
		"`a2a.async_report_dedup`",
		"`a2a.async_report_drop`",
		"`scheduler.enqueue`",
		"`scheduler.delayed_enqueue`",
		"`scheduler.delayed_wait`",
		"`scheduler.delayed_ready`",
		"`scheduler.claim`",
		"`scheduler.heartbeat`",
		"`scheduler.lease_expired`",
		"`scheduler.requeue`",
		"`scheduler.qos_claim`",
		"`scheduler.fairness_yield`",
		"`scheduler.retry_backoff`",
		"`scheduler.dead_letter`",
		"`subagent.spawn`",
		"`subagent.join`",
		"`subagent.budget_reject`",
		"`recovery.restore`",
		"`recovery.replay`",
		"`recovery.conflict`",
	}
	for _, reason := range requiredComposedReasonsInDoc {
		if !strings.Contains(snapshot.RuntimeConfigDoc, reason) {
			violations = append(violations, Violation{
				Code:    "missing_runtime_doc_reason_" + strings.ReplaceAll(strings.Trim(reason, "`"), ".", "_"),
				Message: "runtime config/diagnostics doc missing composed reason contract: " + reason,
			})
		}
	}

	requiredComposedDocFields := []string{
		"`composer_managed`",
		"`scheduler_backend_fallback`",
		"`scheduler_backend_fallback_reason`",
		"`collab_handoff_total`",
		"`collab_delegation_total`",
		"`collab_aggregation_total`",
		"`collab_aggregation_strategy`",
		"`collab_fail_fast_total`",
		"`team_remote_task_total`",
		"`team_remote_task_failed`",
		"`workflow_remote_step_total`",
		"`workflow_remote_step_failed`",
		"`scheduler_backend`",
		"`scheduler_queue_total`",
		"`scheduler_claim_total`",
		"`scheduler_reclaim_total`",
		"`scheduler_qos_mode`",
		"`scheduler_priority_claim_total`",
		"`scheduler_fairness_yield_total`",
		"`scheduler_retry_backoff_total`",
		"`scheduler_dead_letter_total`",
		"`scheduler_delayed_task_total`",
		"`scheduler_delayed_claim_total`",
		"`scheduler_delayed_wait_ms_p95`",
		"`subagent_child_total`",
		"`subagent_child_failed`",
		"`subagent_budget_reject_total`",
		"`recovery_enabled`",
		"`recovery_resume_boundary`",
		"`recovery_inflight_policy`",
		"`recovery_recovered`",
		"`recovery_replay_total`",
		"`recovery_timeout_reentry_total`",
		"`recovery_timeout_reentry_exhausted_total`",
		"`recovery_conflict`",
		"`recovery_conflict_code`",
		"`recovery_fallback_used`",
		"`recovery_fallback_reason`",
	}
	requiredComposerRuntimeFields := []string{
		"`composer_managed`",
		"`scheduler_backend_fallback`",
		"`scheduler_backend_fallback_reason`",
		"`scheduler_qos_mode`",
		"`scheduler_priority_claim_total`",
		"`scheduler_fairness_yield_total`",
		"`scheduler_retry_backoff_total`",
		"`scheduler_dead_letter_total`",
		"`scheduler_delayed_task_total`",
		"`scheduler_delayed_claim_total`",
		"`scheduler_delayed_wait_ms_p95`",
		"`recovery_enabled`",
		"`recovery_resume_boundary`",
		"`recovery_inflight_policy`",
		"`recovery_recovered`",
		"`recovery_replay_total`",
		"`recovery_timeout_reentry_total`",
		"`recovery_timeout_reentry_exhausted_total`",
		"`recovery_conflict`",
		"`recovery_conflict_code`",
		"`recovery_fallback_used`",
		"`recovery_fallback_reason`",
	}
	for _, field := range requiredComposerRuntimeFields {
		if !strings.Contains(snapshot.SchedulerRuntimeConfigSpec, field) {
			violations = append(violations, Violation{
				Code:    "missing_scheduler_runtime_spec_field_" + strings.Trim(field, "`"),
				Message: "scheduler runtime config spec missing composer additive field: " + field,
			})
		}
	}
	for _, field := range requiredComposedDocFields {
		if !strings.Contains(snapshot.RuntimeConfigDoc, field) {
			violations = append(violations, Violation{
				Code:    "missing_runtime_doc_field_" + strings.Trim(field, "`"),
				Message: "runtime config/diagnostics doc missing composed summary field: " + field,
			})
		}
	}
	requiredCompatibilityWindowMarkers := []struct {
		code   string
		marker string
	}{
		{code: "missing_runtime_doc_compatibility_window_title", marker: "Compatibility Window (A12/A13)"},
		{code: "missing_runtime_doc_compatibility_window_rule", marker: "additive + nullable + default"},
		{code: "missing_runtime_doc_compatibility_window_default_rule", marker: "missing additive fields resolve to documented default values"},
		{code: "missing_runtime_doc_compatibility_window_ignore_unknown_rule", marker: "unknown future additive fields are safely ignored"},
		{code: "missing_runtime_doc_compatibility_window_stable_existing_rule", marker: "pre-existing field semantics remain unchanged"},
	}
	for _, item := range requiredCompatibilityWindowMarkers {
		if !strings.Contains(snapshot.RuntimeConfigDoc, item.marker) {
			violations = append(violations, Violation{
				Code:    item.code,
				Message: "runtime config/diagnostics doc missing compatibility-window marker: " + item.marker,
			})
		}
	}

	requiredComposedEnvMappings := []string{
		"`BAYMAX_TEAMS_REMOTE_ENABLED`",
		"`BAYMAX_TEAMS_REMOTE_REQUIRE_PEER_ID`",
		"`BAYMAX_WORKFLOW_REMOTE_ENABLED`",
		"`BAYMAX_WORKFLOW_REMOTE_DEFAULT_RETRY_MAX_ATTEMPTS`",
		"`BAYMAX_SCHEDULER_ENABLED`",
		"`BAYMAX_SCHEDULER_BACKEND`",
		"`BAYMAX_SCHEDULER_LEASE_TIMEOUT`",
		"`BAYMAX_SCHEDULER_HEARTBEAT_INTERVAL`",
		"`BAYMAX_SCHEDULER_QOS_MODE`",
		"`BAYMAX_SCHEDULER_QOS_FAIRNESS_MAX_CONSECUTIVE_CLAIMS_PER_PRIORITY`",
		"`BAYMAX_SCHEDULER_DLQ_ENABLED`",
		"`BAYMAX_SCHEDULER_RETRY_BACKOFF_ENABLED`",
		"`BAYMAX_SCHEDULER_RETRY_BACKOFF_INITIAL`",
		"`BAYMAX_SCHEDULER_RETRY_BACKOFF_MAX`",
		"`BAYMAX_SCHEDULER_RETRY_BACKOFF_MULTIPLIER`",
		"`BAYMAX_SCHEDULER_RETRY_BACKOFF_JITTER_RATIO`",
		"`BAYMAX_SUBAGENT_MAX_DEPTH`",
		"`BAYMAX_SUBAGENT_MAX_ACTIVE_CHILDREN`",
		"`BAYMAX_SUBAGENT_CHILD_TIMEOUT_BUDGET`",
	}
	for _, mapping := range requiredComposedEnvMappings {
		if !strings.Contains(snapshot.RuntimeConfigDoc, mapping) {
			violations = append(violations, Violation{
				Code:    "missing_runtime_doc_env_mapping_" + strings.ToLower(strings.Trim(mapping, "`")),
				Message: "runtime config/diagnostics doc missing composed env mapping: " + mapping,
			})
		}
	}

	requiredComposedAcceptanceMarkers := []string{
		"`teams.remote.*`",
		"`workflow.remote.*`",
		"`team_remote_task_total`",
		"`workflow_remote_step_total`",
		"`scheduler.*`",
		"`scheduler.qos.*`",
		"`scheduler.dlq.*`",
		"`subagent.*`",
		"`scheduler_queue_total`",
		"`scheduler_qos_mode`",
		"`subagent_child_total`",
		"`recovery.*`",
		"`recovery_enabled`",
	}
	for _, marker := range requiredComposedAcceptanceMarkers {
		if !strings.Contains(snapshot.V1AcceptanceDoc, marker) {
			violations = append(violations, Violation{
				Code:    "missing_v1_acceptance_marker_" + strings.ReplaceAll(strings.Trim(marker, "`"), ".", "_"),
				Message: "v1 acceptance doc missing composed orchestration marker: " + marker,
			})
		}
	}
	if !strings.Contains(snapshot.V1AcceptanceDoc, "compatibility window") {
		violations = append(violations, Violation{
			Code:    "missing_v1_acceptance_compatibility_window_marker",
			Message: "v1 acceptance doc must mention A12/A13 compatibility window semantics",
		})
	}
	if !strings.Contains(snapshot.V1AcceptanceDoc, "A12/A13 additive summary fields") {
		violations = append(violations, Violation{
			Code:    "missing_v1_acceptance_a12_a13_compatibility_marker",
			Message: "v1 acceptance doc must pin compatibility window to A12/A13 additive summary fields",
		})
	}
	requiredUnifiedQuerySpecMarkers := []struct {
		code   string
		marker string
	}{
		{code: "unified_query_semantic_drift_missing_and_semantics", marker: "`AND` semantics"},
		{code: "unified_query_semantic_drift_missing_default_page_size", marker: "page_size=50"},
		{code: "unified_query_semantic_drift_missing_page_size_limit", marker: "page_size <= 200"},
		{code: "unified_query_semantic_drift_missing_default_sort", marker: "time desc"},
		{code: "unified_query_semantic_drift_missing_opaque_cursor", marker: "opaque cursor"},
		{code: "unified_query_semantic_drift_missing_empty_set_semantics", marker: "empty result set"},
	}
	for _, marker := range requiredUnifiedQuerySpecMarkers {
		if !strings.Contains(snapshot.UnifiedQuerySpec, marker.marker) {
			violations = append(violations, Violation{
				Code:    marker.code,
				Message: "unified query spec drift detected: missing marker " + marker.marker,
			})
		}
	}
	requiredUnifiedQueryRuntimeDocMarkers := []struct {
		code   string
		marker string
	}{
		{code: "missing_runtime_doc_unified_query_run_id_filter", marker: "`run_id`"},
		{code: "missing_runtime_doc_unified_query_team_id_filter", marker: "`team_id`"},
		{code: "missing_runtime_doc_unified_query_workflow_id_filter", marker: "`workflow_id`"},
		{code: "missing_runtime_doc_unified_query_task_id_filter", marker: "`task_id`"},
		{code: "missing_runtime_doc_unified_query_status_filter", marker: "`status`"},
		{code: "missing_runtime_doc_unified_query_time_range_filter", marker: "`time_range`"},
		{code: "missing_runtime_doc_unified_query_page_default", marker: "`page_size=50`"},
		{code: "missing_runtime_doc_unified_query_page_limit", marker: "`page_size<=200`"},
		{code: "missing_runtime_doc_unified_query_sort_default", marker: "`time desc`"},
		{code: "missing_runtime_doc_unified_query_cursor", marker: "opaque cursor"},
		{code: "missing_runtime_doc_unified_query_task_no_match", marker: "empty result set"},
	}
	for _, marker := range requiredUnifiedQueryRuntimeDocMarkers {
		if !strings.Contains(snapshot.RuntimeConfigDoc, marker.marker) {
			violations = append(violations, Violation{
				Code:    marker.code,
				Message: "runtime config/diagnostics doc missing unified query marker: " + marker.marker,
			})
		}
	}
	if !strings.Contains(strings.ToLower(snapshot.ComposerGateSpec), "unified query contract suites") {
		violations = append(violations, Violation{
			Code:    "missing_unified_query_gate_contract",
			Message: "shared quality gate spec must include unified query contract suites",
		})
	}
	if !strings.Contains(snapshot.MainlineContractIndexDoc, "Unified Query A18") ||
		!strings.Contains(snapshot.MainlineContractIndexDoc, "TestUnifiedQueryContractUnmatchedTaskIDEmptySet") ||
		!strings.Contains(snapshot.MainlineContractIndexDoc, "TestUnifiedQueryContractReplayIdempotentSummaries") {
		violations = append(violations, Violation{
			Code:    "missing_mainline_index_unified_query_mapping",
			Message: "mainline contract index must map unified query rows to concrete tests",
		})
	}

	deprecatedA2AFieldAliases := []string{
		"`a2aDeliveryMode`",
		"`a2aVersionLocal`",
		"`a2aVersionPeer`",
		"`a2aVersionNegotiationResult`",
	}
	for _, field := range deprecatedA2AFieldAliases {
		if strings.Contains(snapshot.IdentifierDoc, field) || strings.Contains(snapshot.A2ARuntimeConfigSpec, field) {
			violations = append(violations, Violation{
				Code:    "non_snake_case_a2a_field_detected",
				Message: "non-snake-case A2A field detected; use snake_case additive fields",
			})
			break
		}
	}

	if strings.Contains(snapshot.IdentifierDoc, "`a2a_peer`") || strings.Contains(snapshot.A2ARuntimeConfigSpec, "`a2a_peer`") {
		violations = append(violations, Violation{
			Code:    "deprecated_a2a_peer_field_detected",
			Message: "deprecated field a2a_peer detected; use peer_id instead",
		})
	}

	schedulerRuntimeSpecLower := strings.ToLower(snapshot.SchedulerRuntimeConfigSpec)
	hasSchedulerScope := strings.Contains(snapshot.SchedulerRuntimeConfigSpec, "`scheduler.*`") ||
		strings.Contains(schedulerRuntimeSpecLower, "scheduler and subagent")
	hasSubagentScope := strings.Contains(snapshot.SchedulerRuntimeConfigSpec, "`subagent.*`") ||
		strings.Contains(schedulerRuntimeSpecLower, "scheduler and subagent")
	if !strings.Contains(snapshot.TeamsRuntimeConfigSpec, "`teams.*`") ||
		!strings.Contains(snapshot.WorkflowRuntimeConfigSpec, "`workflow.*`") ||
		!strings.Contains(snapshot.A2ARuntimeConfigSpec, "`a2a.*`") ||
		!hasSchedulerScope ||
		!hasSubagentScope {
		violations = append(violations, Violation{
			Code:    "missing_domain_scoped_config_namespaces",
			Message: "teams/workflow/a2a/scheduler/subagent runtime config specs must declare domain-scoped namespaces",
		})
	}
	if !strings.Contains(snapshot.TeamsRuntimeConfigSpec, "teams remote-worker enablement and defaults") {
		violations = append(violations, Violation{
			Code:    "missing_teams_remote_config_contract",
			Message: "teams runtime config spec must include remote-worker enablement/default contract",
		})
	}
	if !strings.Contains(snapshot.WorkflowRuntimeConfigSpec, "workflow remote-step enablement and defaults") {
		violations = append(violations, Violation{
			Code:    "missing_workflow_remote_config_contract",
			Message: "workflow runtime config spec must include remote-step enablement/default contract",
		})
	}
	if !strings.Contains(snapshot.TeamsRuntimeConfigSpec, "remote execution totals and failure markers") &&
		!strings.Contains(snapshot.WorkflowRuntimeConfigSpec, "remote execution totals and failure markers") &&
		!strings.Contains(snapshot.SchedulerRuntimeConfigSpec, "scheduler/subagent summary") {
		violations = append(violations, Violation{
			Code:    "missing_composed_summary_contract",
			Message: "runtime config spec must include composed diagnostics summary contract",
		})
	}

	teamsBoundary := strings.ToLower(snapshot.TeamsBoundarySpec)
	workflowBoundary := strings.ToLower(snapshot.WorkflowBoundarySpec)
	a2aBoundary := strings.ToLower(snapshot.A2ABoundarySpec)
	schedulerBoundary := strings.ToLower(snapshot.SchedulerBoundarySpec)
	if !strings.Contains(teamsBoundary, "shared multi-agent contract gate") ||
		!strings.Contains(workflowBoundary, "shared multi-agent contract gate") ||
		!strings.Contains(a2aBoundary, "shared multi-agent contract gate") ||
		!strings.Contains(schedulerBoundary, "shared multi-agent contract gate") {
		violations = append(violations, Violation{
			Code:    "missing_blocking_shared_contract_gate",
			Message: "teams/workflow/a2a/scheduler boundary specs must declare blocking shared-contract gate",
		})
	}
	if !strings.Contains(strings.ToLower(schedulerBoundary), "orchestration/composer") {
		violations = append(violations, Violation{
			Code:    "missing_composer_boundary_contract",
			Message: "runtime boundary specs must include orchestration/composer ownership and dependency boundary",
		})
	}
	if !strings.Contains(strings.ToLower(snapshot.ComposerGateSpec), "composer contract suite") ||
		!strings.Contains(strings.ToLower(snapshot.ComposerGateSpec), "shared multi-agent gate") {
		violations = append(violations, Violation{
			Code:    "missing_composer_gate_contract",
			Message: "quality gate spec must include composer contract suite in shared multi-agent gate path",
		})
	}
	if !strings.Contains(strings.ToLower(snapshot.ComposerGateSpec), "disconnected parallel gate") {
		violations = append(violations, Violation{
			Code:    "missing_single_blocking_gate_marker",
			Message: "quality gate spec must explicitly prohibit disconnected parallel gates in the same contract domain",
		})
	}

	return violations
}
