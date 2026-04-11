# Agent Mode Migration Playbook

This playbook defines how to promote each `agent-modes` example from `minimal` to `production-ish` with auditable semantic evidence.

## Global Promotion Checkpoints
- config: verify `env > file > default` precedence and atomic rollback on invalid reloads.
- permissions: verify sandbox and egress policies on required runtime path domains.
- observability: verify runtime recorder compatible diagnostics markers are emitted.
- replay: verify mapped replay fixtures remain additive-compatible.
- gates: run pattern coverage, smoke, semantic, readme sync, and quality gate scripts before merge.

## Variant Distinction Rules
- `minimal` MUST emit semantic anchor, runtime path evidence, and baseline semantic markers.
- `production-ish` MUST include all minimal markers plus governance semantic evidence (`verification.semantic.governance=enforced`).
- `production-ish` MUST NOT be a no-op copy of `minimal`; marker set and final answer signature must differ.

## Doc-First Delivery Flow
1. **Doc baseline first**: update `MATRIX.md` and the mode `README.md` files first, including semantic anchor, runtime path evidence, expected markers, and rollback notes.
2. **Implementation second**: only after doc baseline is complete, update mode code paths.
3. **Gate validation third**: run agent-mode semantic/readme/smoke/quality gates before task completion.
4. **Task completion rule**: mark a task done only when code/test/documentation/gate evidence are all present.

## Rollback Steps (Doc-First Scope)
1. Revert affected mode directory (`main.go`, `README.md`, semantic implementation file) as one unit.
2. Revert corresponding row updates in `MATRIX.md` and this playbook mapping section.
3. Re-run semantic/readme/smoke gates to confirm rollback returns to a stable baseline.
4. Record rollback reason and impacted pattern list in change-level acceptance notes.

## Semantic Evidence Fields
- `verification.mainline_runtime_path`: `ok|failed`, runtime entry health.
- `verification.semantic.phase`: rollout phase (`P0|P1|P2`).
- `verification.semantic.anchor`: pattern semantic anchor identifier.
- `verification.semantic.classification`: semantic classification label.
- `verification.semantic.runtime_path`: comma-separated runtime domain path evidence.
- `verification.semantic.expected_markers`: comma-separated expected marker list for variant.
- `verification.semantic.governance`: `baseline|enforced`.
- `verification.semantic.marker.<token>=ok`: concrete semantic evidence markers.

## a71 boundary
- a71 tracks only real-runtime replacement and governance checks for agent-mode examples.
- a71 does not reuse or backfill any a62 task status.

## Mode Mapping
| pattern | phase | semantic anchor | production governance markers | gates | replay |
| --- | --- | --- | --- | --- | --- |
| `context-governed-reference-first` | `P0` | `context.reference_first_isolate_edit_tiering` | `governance_context_tiering_enforced; governance_context_replay_bound` | `check-context-jit-organization-contract.*; check-context-compression-production-contract.*` | `context_reference_first.v1; context_compression_production.v1` |
| `hitl-governed-checkpoint` | `P0` | `hitl.await_resume_reject_timeout_recover` | `governance_hitl_gate_enforced; governance_hitl_replay_bound` | `check-react-contract.*` | `react.v1` |
| `mcp-governed-stdio-http` | `P0` | `transport.profile_failover_governance` | `governance_transport_gate_enforced; governance_transport_replay_bound` | `check-quality-gate.*` | `cross-domain-primary-reason-arbitration-contract.v1` |
| `multi-agents-collab-recovery` | `P0` | `collab.mailbox_taskboard_recovery` | `governance_collab_gate_enforced; governance_collab_replay_bound` | `check-multi-agent-shared-contract.*` | `cross-domain-primary-reason-arbitration-contract.v1` |
| `rag-hybrid-retrieval` | `P0` | `retrieval.candidate_rerank_fallback` | `governance_retrieval_budget_gate; governance_retrieval_replay_bound` | `check-memory-scope-and-search-contract.*` | `memory_scope.v1` |
| `realtime-interrupt-resume` | `P0` | `realtime.cursor_idempotent_interrupt_resume` | `governance_realtime_gate_enforced; governance_realtime_replay_bound` | `check-realtime-protocol-contract.*` | `realtime_event_protocol.v1` |
| `sandbox-governed-toolchain` | `P0` | `sandbox.allow_deny_egress_fallback` | `governance_sandbox_gate_enforced; governance_sandbox_replay_bound` | `check-security-sandbox-contract.*; check-sandbox-egress-allowlist-contract.*` | `sandbox_egress.v1` |
| `skill-driven-discovery-hybrid` | `P0` | `discovery.source_priority_score_mapping` | `governance_skill_gate_enforced; governance_skill_replay_bound` | `check-react-contract.*` | `react.v1` |
| `structured-output-schema-contract` | `P0` | `schema.validate_compat_drift` | `governance_schema_gate_enforced; governance_schema_replay_bound` | `check-diagnostics-replay-contract.*` | `cross-domain-primary-reason-arbitration-contract.v1` |
| `hooks-middleware-extension-pipeline` | `P1` | `middleware.onion_bubble_passthrough` | `governance_hooks_gate_enforced; governance_hooks_replay_bound` | `check-hooks-middleware-contract.*` | `hooks_middleware.v1` |
| `mapreduce-large-batch` | `P1` | `mapreduce.shard_reduce_retry` | `governance_mapreduce_gate_enforced; governance_mapreduce_replay_bound` | `check-multi-agent-shared-contract.*` | `cross-domain-primary-reason-arbitration-contract.v1` |
| `observability-export-bundle` | `P1` | `observability.export_bundle_replay` | `governance_observability_gate_enforced; governance_observability_replay_bound` | `check-observability-export-and-bundle-contract.*` | `observability.v1` |
| `policy-budget-admission` | `P1` | `policy.precedence_budget_admission_trace` | `governance_policy_gate_enforced; governance_policy_replay_bound` | `check-policy-precedence-contract.*; check-runtime-budget-admission-contract.*` | `policy_stack.v1; budget_admission.v1` |
| `react-plan-notebook-loop` | `P1` | `react.plan_notebook_change_hooks` | `governance_react_gate_enforced; governance_react_replay_bound` | `check-react-plan-notebook-contract.*` | `react_plan_notebook.v1` |
| `state-session-snapshot-recovery` | `P1` | `snapshot.export_restore_replay` | `governance_snapshot_gate_enforced; governance_snapshot_replay_bound` | `check-state-snapshot-contract.*` | `state_session_snapshot.v1` |
| `tracing-eval-smoke` | `P1` | `trace.eval_feedback_loop` | `governance_tracing_gate_enforced; governance_tracing_replay_bound` | `check-agent-eval-and-tracing-interop-contract.*` | `otel_semconv.v1; agent_eval.v1` |
| `workflow-branch-retry-failfast` | `P1` | `workflow.branch_retry_failfast` | `governance_workflow_gate_enforced; governance_workflow_replay_bound` | `check-multi-agent-shared-contract.*` | `cross-domain-primary-reason-arbitration-contract.v1` |
| `adapter-onboarding-manifest-capability` | `P2` | `adapter.manifest_capability_fallback` | `governance_adapter_gate_enforced; governance_adapter_replay_bound` | `check-adapter-manifest-contract.*; check-adapter-capability-contract.*; check-adapter-contract-replay.*` | `adapter_contract_profile.v1` |
| `config-hot-reload-rollback` | `P2` | `config.reload_failfast_rollback` | `governance_config_gate_enforced; governance_config_replay_bound` | `check-quality-gate.*` | `cross-domain-primary-reason-arbitration-contract.v1` |
| `custom-adapter-health-readiness-circuit` | `P2` | `adapterhealth.readiness_backoff_circuit` | `governance_adapter_health_gate_enforced; governance_adapter_health_replay_bound` | `check-adapter-conformance.*` | `readiness-timeout-health-replay-fixture-gate.v1` |
| `custom-adapter-mcp-model-tool-memory-pack` | `P2` | `adapterpack.manifest_capability_memory` | `governance_adapter_pack_gate_enforced; governance_adapter_pack_replay_bound` | `check-adapter-conformance.*; check-adapter-scaffold-drift.*` | `adapter_contract_profile.v1` |
| `mainline-mailbox-async-delayed-reconcile` | `P2` | `mailbox.async_delayed_reconcile` | `governance_mailbox_gate_enforced; governance_mailbox_replay_bound` | `check-multi-agent-shared-contract.*` | `cross-domain-primary-reason-arbitration-contract.v1` |
| `mainline-readiness-admission-degradation` | `P2` | `readiness.admission_degradation` | `governance_readiness_gate_enforced; governance_readiness_replay_bound` | `check-quality-gate.*` | `readiness-timeout-health-replay-fixture-gate.v1` |
| `mainline-scheduler-qos-backoff-dlq` | `P2` | `scheduler.qos_backoff_dlq` | `governance_scheduler_gate_enforced; governance_scheduler_replay_bound` | `check-multi-agent-shared-contract.*` | `cross-domain-primary-reason-arbitration-contract.v1` |
| `mainline-task-board-query-control` | `P2` | `taskboard.query_control_idempotency` | `governance_taskboard_gate_enforced; governance_taskboard_replay_bound` | `check-multi-agent-shared-contract.*` | `cross-domain-primary-reason-arbitration-contract.v1` |
| `multi-agents-hierarchical-planner-validator` | `P2` | `hierarchy.planner_validator_correction` | `governance_hierarchy_gate_enforced; governance_hierarchy_replay_bound` | `check-multi-agent-shared-contract.*` | `cross-domain-primary-reason-arbitration-contract.v1` |
| `security-policy-event-delivery` | `P2` | `security.policy_event_delivery` | `governance_security_gate_enforced; governance_security_replay_bound` | `check-security-policy-contract.*; check-security-event-contract.*; check-security-delivery-contract.*` | `cross-domain-primary-reason-arbitration-contract.v1` |
| `workflow-routing-strategy-switch` | `P2` | `routing.strategy_switch_confidence` | `governance_routing_gate_enforced; governance_routing_replay_bound` | `check-multi-agent-shared-contract.*` | `cross-domain-primary-reason-arbitration-contract.v1` |

## Production Migration Checklist
1. Run `minimal` and `production-ish` for the target pattern, and keep both outputs as evidence.
2. Confirm `production-ish` contains governance markers and differs from `minimal` marker set.
3. Confirm README contains `Run / Prerequisites / Real Runtime Path / Expected Output/Verification / Failure/Rollback Notes`.
4. Run `check-agent-mode-examples-smoke.*`, `check-agent-mode-real-runtime-semantic-contract.*`, and `check-agent-mode-readme-runtime-sync-contract.*`.
5. Run `check-quality-gate.*` before merge.
