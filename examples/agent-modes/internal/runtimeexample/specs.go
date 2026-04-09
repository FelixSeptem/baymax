package runtimeexample

import "sort"

// ModeSpec defines one agent-mode semantic contract profile used by both examples and gates.
type ModeSpec struct {
	Pattern           string
	Phase             string
	SemanticAnchor    string
	Classification    string
	RuntimeDomains    []string
	MinimalMarkers    []string
	ProductionMarkers []string
	Contracts         []string
	Gates             []string
	Replay            []string
}

// ExpectedMarkers returns the variant-scoped semantic evidence markers.
func (s ModeSpec) ExpectedMarkers(variant string) []string {
	markers := make([]string, 0, len(s.MinimalMarkers)+len(s.ProductionMarkers))
	markers = append(markers, s.MinimalMarkers...)
	if variant == "production-ish" {
		markers = append(markers, s.ProductionMarkers...)
	}
	return markers
}

func cloneStrings(src []string) []string {
	if len(src) == 0 {
		return nil
	}
	out := make([]string, len(src))
	copy(out, src)
	return out
}

func cloneSpec(in ModeSpec) ModeSpec {
	in.RuntimeDomains = cloneStrings(in.RuntimeDomains)
	in.MinimalMarkers = cloneStrings(in.MinimalMarkers)
	in.ProductionMarkers = cloneStrings(in.ProductionMarkers)
	in.Contracts = cloneStrings(in.Contracts)
	in.Gates = cloneStrings(in.Gates)
	in.Replay = cloneStrings(in.Replay)
	return in
}

// Lookup returns a copy of the mode spec.
func Lookup(pattern string) (ModeSpec, bool) {
	spec, ok := modeSpecs[pattern]
	if !ok {
		return ModeSpec{}, false
	}
	return cloneSpec(spec), true
}

// RequiredPatterns returns all canonical mode names sorted.
func RequiredPatterns() []string {
	keys := make([]string, 0, len(modeSpecs))
	for pattern := range modeSpecs {
		keys = append(keys, pattern)
	}
	sort.Strings(keys)
	return keys
}

// AllSpecs returns all mode specs in canonical pattern order.
func AllSpecs() []ModeSpec {
	patterns := RequiredPatterns()
	out := make([]ModeSpec, 0, len(patterns))
	for _, pattern := range patterns {
		spec, ok := Lookup(pattern)
		if !ok {
			continue
		}
		out = append(out, spec)
	}
	return out
}

var modeSpecs = map[string]ModeSpec{
	"rag-hybrid-retrieval": {
		Pattern:           "rag-hybrid-retrieval",
		Phase:             "P0",
		SemanticAnchor:    "retrieval.candidate_rerank_fallback",
		Classification:    "rag.hybrid_retrieval",
		RuntimeDomains:    []string{"memory", "context/assembler"},
		MinimalMarkers:    []string{"retrieval_candidates_built", "retrieval_rerank_applied", "retrieval_fallback_classified"},
		ProductionMarkers: []string{"governance_retrieval_budget_gate", "governance_retrieval_replay_bound"},
		Contracts:         []string{"memory-scope-and-builtin-filesystem-v2-governance-contract"},
		Gates:             []string{"check-memory-scope-and-search-contract.*"},
		Replay:            []string{"memory_scope.v1"},
	},
	"structured-output-schema-contract": {
		Pattern:           "structured-output-schema-contract",
		Phase:             "P0",
		SemanticAnchor:    "schema.validate_compat_drift",
		Classification:    "structured_output.schema_contract",
		RuntimeDomains:    []string{"core/types", "runtime/diagnostics"},
		MinimalMarkers:    []string{"schema_contract_loaded", "schema_compat_window_checked", "schema_drift_signal_emitted"},
		ProductionMarkers: []string{"governance_schema_gate_enforced", "governance_schema_replay_bound"},
		Contracts:         []string{"diagnostics-replay-tooling"},
		Gates:             []string{"check-diagnostics-replay-contract.*"},
		Replay:            []string{"cross-domain-primary-reason-arbitration-contract.v1"},
	},
	"skill-driven-discovery-hybrid": {
		Pattern:           "skill-driven-discovery-hybrid",
		Phase:             "P0",
		SemanticAnchor:    "discovery.source_priority_score_mapping",
		Classification:    "skill.hybrid_discovery",
		RuntimeDomains:    []string{"skill/loader", "context/assembler"},
		MinimalMarkers:    []string{"discovery_sources_prioritized", "discovery_score_reconciled", "discovery_mapping_emitted"},
		ProductionMarkers: []string{"governance_skill_gate_enforced", "governance_skill_replay_bound"},
		Contracts:         []string{"skill-trigger-scoring"},
		Gates:             []string{"check-react-contract.*"},
		Replay:            []string{"react.v1"},
	},
	"mcp-governed-stdio-http": {
		Pattern:           "mcp-governed-stdio-http",
		Phase:             "P0",
		SemanticAnchor:    "transport.profile_failover_governance",
		Classification:    "mcp.transport_governance",
		RuntimeDomains:    []string{"mcp/stdio", "mcp/http", "mcp/profile"},
		MinimalMarkers:    []string{"transport_profile_selected", "transport_failover_decided", "transport_reason_trace_emitted"},
		ProductionMarkers: []string{"governance_transport_gate_enforced", "governance_transport_replay_bound"},
		Contracts:         []string{"mcp-runtime-reliability-profiles"},
		Gates:             []string{"check-quality-gate.*"},
		Replay:            []string{"cross-domain-primary-reason-arbitration-contract.v1"},
	},
	"hitl-governed-checkpoint": {
		Pattern:           "hitl-governed-checkpoint",
		Phase:             "P0",
		SemanticAnchor:    "hitl.await_resume_reject_timeout_recover",
		Classification:    "hitl.checkpoint_governance",
		RuntimeDomains:    []string{"orchestration/composer", "runtime/diagnostics"},
		MinimalMarkers:    []string{"hitl_checkpoint_awaited", "hitl_resume_reject_classified", "hitl_timeout_recoverable"},
		ProductionMarkers: []string{"governance_hitl_gate_enforced", "governance_hitl_replay_bound"},
		Contracts:         []string{"react-loop-and-tool-calling-parity-contract"},
		Gates:             []string{"check-react-contract.*"},
		Replay:            []string{"react.v1"},
	},
	"context-governed-reference-first": {
		Pattern:           "context-governed-reference-first",
		Phase:             "P0",
		SemanticAnchor:    "context.reference_first_isolate_edit_tiering",
		Classification:    "context.reference_first_governance",
		RuntimeDomains:    []string{"context/assembler", "context/guard", "context/journal"},
		MinimalMarkers:    []string{"context_reference_first_selected", "context_isolate_handoff_applied", "context_edit_gate_evaluated"},
		ProductionMarkers: []string{"governance_context_tiering_enforced", "governance_context_replay_bound"},
		Contracts:         []string{"jit-context-organization-and-reference-first-assembly-contract", "context-compression-production-hardening-contract"},
		Gates:             []string{"check-context-jit-organization-contract.*", "check-context-compression-production-contract.*"},
		Replay:            []string{"context_reference_first.v1", "context_compression_production.v1"},
	},
	"sandbox-governed-toolchain": {
		Pattern:           "sandbox-governed-toolchain",
		Phase:             "P0",
		SemanticAnchor:    "sandbox.allow_deny_egress_fallback",
		Classification:    "sandbox.toolchain_governance",
		RuntimeDomains:    []string{"runtime/security", "tool/local"},
		MinimalMarkers:    []string{"sandbox_allow_deny_classified", "sandbox_egress_allowlist_checked", "sandbox_fallback_path_emitted"},
		ProductionMarkers: []string{"governance_sandbox_gate_enforced", "governance_sandbox_replay_bound"},
		Contracts:         []string{"security-sandbox-contract"},
		Gates:             []string{"check-security-sandbox-contract.*", "check-sandbox-egress-allowlist-contract.*"},
		Replay:            []string{"sandbox_egress.v1"},
	},
	"realtime-interrupt-resume": {
		Pattern:           "realtime-interrupt-resume",
		Phase:             "P0",
		SemanticAnchor:    "realtime.cursor_idempotent_interrupt_resume",
		Classification:    "realtime.resume_recovery",
		RuntimeDomains:    []string{"core/runner", "runtime/diagnostics"},
		MinimalMarkers:    []string{"realtime_cursor_idempotent", "realtime_interrupt_captured", "realtime_resume_recovered"},
		ProductionMarkers: []string{"governance_realtime_gate_enforced", "governance_realtime_replay_bound"},
		Contracts:         []string{"realtime-event-protocol-and-interrupt-resume-contract"},
		Gates:             []string{"check-realtime-protocol-contract.*"},
		Replay:            []string{"realtime_event_protocol.v1"},
	},
	"multi-agents-collab-recovery": {
		Pattern:           "multi-agents-collab-recovery",
		Phase:             "P0",
		SemanticAnchor:    "collab.mailbox_taskboard_recovery",
		Classification:    "multi_agents.collaboration_recovery",
		RuntimeDomains:    []string{"orchestration/collab", "orchestration/mailbox", "orchestration/scheduler"},
		MinimalMarkers:    []string{"collab_mailbox_orchestrated", "collab_task_board_reconciled", "collab_recovery_continued"},
		ProductionMarkers: []string{"governance_collab_gate_enforced", "governance_collab_replay_bound"},
		Contracts:         []string{"multi-agent-collaboration-primitives", "long-running-recovery-boundary"},
		Gates:             []string{"check-multi-agent-shared-contract.*"},
		Replay:            []string{"cross-domain-primary-reason-arbitration-contract.v1"},
	},
	"workflow-branch-retry-failfast": {
		Pattern:           "workflow-branch-retry-failfast",
		Phase:             "P1",
		SemanticAnchor:    "workflow.branch_retry_failfast",
		Classification:    "workflow.retry_failfast",
		RuntimeDomains:    []string{"orchestration/workflow", "runtime/config"},
		MinimalMarkers:    []string{"workflow_branch_routed", "workflow_retry_budgeted", "workflow_failfast_classified"},
		ProductionMarkers: []string{"governance_workflow_gate_enforced", "governance_workflow_replay_bound"},
		Contracts:         []string{"workflow-graph-composability-contract"},
		Gates:             []string{"check-multi-agent-shared-contract.*"},
		Replay:            []string{"cross-domain-primary-reason-arbitration-contract.v1"},
	},
	"mapreduce-large-batch": {
		Pattern:           "mapreduce-large-batch",
		Phase:             "P1",
		SemanticAnchor:    "mapreduce.shard_reduce_retry",
		Classification:    "mapreduce.large_batch",
		RuntimeDomains:    []string{"orchestration/teams", "runtime/diagnostics"},
		MinimalMarkers:    []string{"mapreduce_shards_fanned_out", "mapreduce_reduce_aggregated", "mapreduce_retry_classified"},
		ProductionMarkers: []string{"governance_mapreduce_gate_enforced", "governance_mapreduce_replay_bound"},
		Contracts:         []string{"composed-orchestration-contract"},
		Gates:             []string{"check-multi-agent-shared-contract.*"},
		Replay:            []string{"cross-domain-primary-reason-arbitration-contract.v1"},
	},
	"state-session-snapshot-recovery": {
		Pattern:           "state-session-snapshot-recovery",
		Phase:             "P1",
		SemanticAnchor:    "snapshot.export_restore_replay",
		Classification:    "state.session_snapshot_recovery",
		RuntimeDomains:    []string{"orchestration/snapshot", "runtime/diagnostics"},
		MinimalMarkers:    []string{"snapshot_export_emitted", "snapshot_restore_verified", "snapshot_replay_idempotent"},
		ProductionMarkers: []string{"governance_snapshot_gate_enforced", "governance_snapshot_replay_bound"},
		Contracts:         []string{"unified-state-and-session-snapshot-contract"},
		Gates:             []string{"check-state-snapshot-contract.*"},
		Replay:            []string{"state_session_snapshot.v1"},
	},
	"policy-budget-admission": {
		Pattern:           "policy-budget-admission",
		Phase:             "P1",
		SemanticAnchor:    "policy.precedence_budget_admission_trace",
		Classification:    "policy.budget_admission",
		RuntimeDomains:    []string{"runtime/config", "runtime/diagnostics"},
		MinimalMarkers:    []string{"policy_precedence_applied", "budget_admission_decided", "decision_trace_recorded"},
		ProductionMarkers: []string{"governance_policy_gate_enforced", "governance_policy_replay_bound"},
		Contracts:         []string{"policy-precedence-and-decision-trace-contract", "runtime-cost-latency-budget-and-admission-contract"},
		Gates:             []string{"check-policy-precedence-contract.*", "check-runtime-budget-admission-contract.*"},
		Replay:            []string{"policy_stack.v1", "budget_admission.v1"},
	},
	"tracing-eval-smoke": {
		Pattern:           "tracing-eval-smoke",
		Phase:             "P1",
		SemanticAnchor:    "trace.eval_feedback_loop",
		Classification:    "tracing.eval_interop",
		RuntimeDomains:    []string{"observability/trace", "runtime/diagnostics"},
		MinimalMarkers:    []string{"tracing_span_emitted", "eval_signal_recorded", "trace_eval_loop_closed"},
		ProductionMarkers: []string{"governance_tracing_gate_enforced", "governance_tracing_replay_bound"},
		Contracts:         []string{"otel-tracing-and-agent-eval-interoperability-contract"},
		Gates:             []string{"check-agent-eval-and-tracing-interop-contract.*"},
		Replay:            []string{"otel_semconv.v1", "agent_eval.v1"},
	},
	"react-plan-notebook-loop": {
		Pattern:           "react-plan-notebook-loop",
		Phase:             "P1",
		SemanticAnchor:    "react.plan_notebook_change_hooks",
		Classification:    "react.plan_notebook_loop",
		RuntimeDomains:    []string{"core/runner", "runtime/diagnostics"},
		MinimalMarkers:    []string{"react_plan_notebook_synced", "react_change_hook_emitted", "react_tool_loop_closed"},
		ProductionMarkers: []string{"governance_react_gate_enforced", "governance_react_replay_bound"},
		Contracts:         []string{"react-plan-notebook-and-plan-change-hook-contract"},
		Gates:             []string{"check-react-plan-notebook-contract.*"},
		Replay:            []string{"react_plan_notebook.v1"},
	},
	"hooks-middleware-extension-pipeline": {
		Pattern:           "hooks-middleware-extension-pipeline",
		Phase:             "P1",
		SemanticAnchor:    "middleware.onion_bubble_passthrough",
		Classification:    "hooks.middleware_pipeline",
		RuntimeDomains:    []string{"core/runner", "tool/local"},
		MinimalMarkers:    []string{"middleware_onion_order_verified", "middleware_error_bubbled", "middleware_extension_passthrough"},
		ProductionMarkers: []string{"governance_hooks_gate_enforced", "governance_hooks_replay_bound"},
		Contracts:         []string{"agent-lifecycle-hooks-and-tool-middleware-contract"},
		Gates:             []string{"check-hooks-middleware-contract.*"},
		Replay:            []string{"hooks_middleware.v1"},
	},
	"observability-export-bundle": {
		Pattern:           "observability-export-bundle",
		Phase:             "P1",
		SemanticAnchor:    "observability.export_bundle_replay",
		Classification:    "observability.export_bundle",
		RuntimeDomains:    []string{"observability/event", "runtime/diagnostics"},
		MinimalMarkers:    []string{"observability_export_collected", "observability_bundle_emitted", "observability_replay_linked"},
		ProductionMarkers: []string{"governance_observability_gate_enforced", "governance_observability_replay_bound"},
		Contracts:         []string{"observability-export-and-diagnostics-bundle-contract"},
		Gates:             []string{"check-observability-export-and-bundle-contract.*"},
		Replay:            []string{"observability.v1"},
	},
	"adapter-onboarding-manifest-capability": {
		Pattern:           "adapter-onboarding-manifest-capability",
		Phase:             "P2",
		SemanticAnchor:    "adapter.manifest_capability_fallback",
		Classification:    "adapter.onboarding",
		RuntimeDomains:    []string{"adapter/manifest", "adapter/capability"},
		MinimalMarkers:    []string{"adapter_manifest_loaded", "adapter_capability_negotiated", "adapter_fallback_mapped"},
		ProductionMarkers: []string{"governance_adapter_gate_enforced", "governance_adapter_replay_bound"},
		Contracts:         []string{"adapter-manifest-and-runtime-compatibility", "adapter-capability-negotiation-and-fallback", "adapter-contract-profile-versioning-and-replay"},
		Gates:             []string{"check-adapter-manifest-contract.*", "check-adapter-capability-contract.*", "check-adapter-contract-replay.*"},
		Replay:            []string{"adapter_contract_profile.v1"},
	},
	"security-policy-event-delivery": {
		Pattern:           "security-policy-event-delivery",
		Phase:             "P2",
		SemanticAnchor:    "security.policy_event_delivery",
		Classification:    "security.policy_delivery",
		RuntimeDomains:    []string{"runtime/security", "observability/event"},
		MinimalMarkers:    []string{"security_policy_decision_emitted", "security_event_delivery_attempted", "security_deny_semantic_preserved"},
		ProductionMarkers: []string{"governance_security_gate_enforced", "governance_security_replay_bound"},
		Contracts:         []string{"security-baseline-s1", "security-event-delivery"},
		Gates:             []string{"check-security-policy-contract.*", "check-security-event-contract.*", "check-security-delivery-contract.*"},
		Replay:            []string{"cross-domain-primary-reason-arbitration-contract.v1"},
	},
	"config-hot-reload-rollback": {
		Pattern:           "config-hot-reload-rollback",
		Phase:             "P2",
		SemanticAnchor:    "config.reload_failfast_rollback",
		Classification:    "runtime.config_rollback",
		RuntimeDomains:    []string{"runtime/config", "runtime/diagnostics"},
		MinimalMarkers:    []string{"config_reload_attempted", "config_invalid_failfast", "config_atomic_rollback_verified"},
		ProductionMarkers: []string{"governance_config_gate_enforced", "governance_config_replay_bound"},
		Contracts:         []string{"runtime-config-and-diagnostics-api"},
		Gates:             []string{"check-quality-gate.*"},
		Replay:            []string{"cross-domain-primary-reason-arbitration-contract.v1"},
	},
	"workflow-routing-strategy-switch": {
		Pattern:           "workflow-routing-strategy-switch",
		Phase:             "P2",
		SemanticAnchor:    "routing.strategy_switch_confidence",
		Classification:    "workflow.strategy_switch",
		RuntimeDomains:    []string{"orchestration/workflow", "runtime/config"},
		MinimalMarkers:    []string{"routing_strategy_selected", "routing_confidence_evaluated", "routing_switch_committed"},
		ProductionMarkers: []string{"governance_routing_gate_enforced", "governance_routing_replay_bound"},
		Contracts:         []string{"workflow-graph-composability-contract"},
		Gates:             []string{"check-multi-agent-shared-contract.*"},
		Replay:            []string{"cross-domain-primary-reason-arbitration-contract.v1"},
	},
	"multi-agents-hierarchical-planner-validator": {
		Pattern:           "multi-agents-hierarchical-planner-validator",
		Phase:             "P2",
		SemanticAnchor:    "hierarchy.planner_validator_correction",
		Classification:    "multi_agents.hierarchy",
		RuntimeDomains:    []string{"orchestration/teams", "orchestration/workflow"},
		MinimalMarkers:    []string{"hierarchy_plan_decomposed", "hierarchy_validator_feedback_applied", "hierarchy_correction_loop_closed"},
		ProductionMarkers: []string{"governance_hierarchy_gate_enforced", "governance_hierarchy_replay_bound"},
		Contracts:         []string{"multi-agent-collaboration-primitives"},
		Gates:             []string{"check-multi-agent-shared-contract.*"},
		Replay:            []string{"cross-domain-primary-reason-arbitration-contract.v1"},
	},
	"mainline-mailbox-async-delayed-reconcile": {
		Pattern:           "mainline-mailbox-async-delayed-reconcile",
		Phase:             "P2",
		SemanticAnchor:    "mailbox.async_delayed_reconcile",
		Classification:    "mainline.mailbox_reconcile",
		RuntimeDomains:    []string{"orchestration/mailbox", "orchestration/invoke", "runtime/diagnostics"},
		MinimalMarkers:    []string{"mailbox_async_delayed_dispatched", "mailbox_reconcile_triggered", "mailbox_timeline_reason_emitted"},
		ProductionMarkers: []string{"governance_mailbox_gate_enforced", "governance_mailbox_replay_bound"},
		Contracts:         []string{"multi-agent-mailbox-contract", "multi-agent-async-await-reconcile-contract"},
		Gates:             []string{"check-multi-agent-shared-contract.*"},
		Replay:            []string{"cross-domain-primary-reason-arbitration-contract.v1"},
	},
	"mainline-task-board-query-control": {
		Pattern:           "mainline-task-board-query-control",
		Phase:             "P2",
		SemanticAnchor:    "taskboard.query_control_idempotency",
		Classification:    "mainline.taskboard_control",
		RuntimeDomains:    []string{"orchestration/scheduler", "runtime/diagnostics"},
		MinimalMarkers:    []string{"taskboard_query_filtered", "taskboard_control_validated", "taskboard_operation_idempotent"},
		ProductionMarkers: []string{"governance_taskboard_gate_enforced", "governance_taskboard_replay_bound"},
		Contracts:         []string{"multi-agent-task-board-control-contract"},
		Gates:             []string{"check-multi-agent-shared-contract.*"},
		Replay:            []string{"cross-domain-primary-reason-arbitration-contract.v1"},
	},
	"mainline-scheduler-qos-backoff-dlq": {
		Pattern:           "mainline-scheduler-qos-backoff-dlq",
		Phase:             "P2",
		SemanticAnchor:    "scheduler.qos_backoff_dlq",
		Classification:    "mainline.scheduler_qos",
		RuntimeDomains:    []string{"orchestration/scheduler", "runtime/diagnostics"},
		MinimalMarkers:    []string{"scheduler_qos_fairness_applied", "scheduler_backoff_budgeted", "scheduler_dlq_classified"},
		ProductionMarkers: []string{"governance_scheduler_gate_enforced", "governance_scheduler_replay_bound"},
		Contracts:         []string{"distributed-subagent-scheduler-qos"},
		Gates:             []string{"check-multi-agent-shared-contract.*"},
		Replay:            []string{"cross-domain-primary-reason-arbitration-contract.v1"},
	},
	"mainline-readiness-admission-degradation": {
		Pattern:           "mainline-readiness-admission-degradation",
		Phase:             "P2",
		SemanticAnchor:    "readiness.admission_degradation",
		Classification:    "mainline.readiness_admission",
		RuntimeDomains:    []string{"runtime/config", "runtime/diagnostics", "orchestration/composer"},
		MinimalMarkers:    []string{"readiness_preflight_evaluated", "admission_degradation_classified", "readiness_rollback_guarded"},
		ProductionMarkers: []string{"governance_readiness_gate_enforced", "governance_readiness_replay_bound"},
		Contracts:         []string{"runtime-readiness-preflight-contract", "runtime-readiness-admission-guard-contract"},
		Gates:             []string{"check-quality-gate.*"},
		Replay:            []string{"readiness-timeout-health-replay-fixture-gate.v1"},
	},
	"custom-adapter-mcp-model-tool-memory-pack": {
		Pattern:           "custom-adapter-mcp-model-tool-memory-pack",
		Phase:             "P2",
		SemanticAnchor:    "adapterpack.manifest_capability_memory",
		Classification:    "adapter.custom_pack",
		RuntimeDomains:    []string{"adapter/scaffold", "mcp/profile", "memory"},
		MinimalMarkers:    []string{"adapter_pack_manifest_resolved", "adapter_pack_capability_fallback", "adapter_pack_memory_scope_bound"},
		ProductionMarkers: []string{"governance_adapter_pack_gate_enforced", "governance_adapter_pack_replay_bound"},
		Contracts:         []string{"external-adapter-template-and-migration-mapping", "external-adapter-conformance-harness", "adapter-scaffold-generator"},
		Gates:             []string{"check-adapter-conformance.*", "check-adapter-scaffold-drift.*"},
		Replay:            []string{"adapter_contract_profile.v1"},
	},
	"custom-adapter-health-readiness-circuit": {
		Pattern:           "custom-adapter-health-readiness-circuit",
		Phase:             "P2",
		SemanticAnchor:    "adapterhealth.readiness_backoff_circuit",
		Classification:    "adapter.health_readiness",
		RuntimeDomains:    []string{"adapter/health", "runtime/config"},
		MinimalMarkers:    []string{"adapter_health_probe_sampled", "adapter_readiness_circuit_transitioned", "adapter_backoff_recovery_classified"},
		ProductionMarkers: []string{"governance_adapter_health_gate_enforced", "governance_adapter_health_replay_bound"},
		Contracts:         []string{"adapter-runtime-health-probe-contract", "adapter-health-backoff-and-circuit-governance-contract"},
		Gates:             []string{"check-adapter-conformance.*"},
		Replay:            []string{"readiness-timeout-health-replay-fixture-gate.v1"},
	},
}
