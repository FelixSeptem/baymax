# Agent Mode Migration Playbook

This playbook defines the migration path from each `examples/agent-modes/*/minimal` example to production-ready rollout checkpoints.

## Global Promotion Checkpoints
- config: verify `env > file > default` precedence and rollback behavior under invalid reload input.
- permissions: verify sandbox and allowlist policies for required tools and network egress.
- capacity: define concurrency limits, retry budgets, and timeout budgets for sustained execution.
- observability: ensure diagnostics and tracing outputs are emitted through runtime recorder-compatible fields.
- replay: keep replay fixtures additive-compatible and parser-safe for drift detection.
- stability: track smoke latency (`p50/p95`), failure rate, retry rate, and flaky rate against baseline thresholds.
- gates: run all mapped contract gates and `check-quality-gate.*` before promotion.

## Mode Mapping
| pattern | production-ish focus | required gates | replay |
| --- | --- | --- | --- |
| `rag-hybrid-retrieval` | hybrid retrieval with memory primary, mcp fallback, and fallback classification output. | `check-memory-scope-and-search-contract.*` | `memory_scope.v1` |
| `structured-output-schema-contract` | schema validation plus parser compatibility and drift-friendly payload emission. | `check-diagnostics-replay-contract.*` | `cross-domain primary reason arbitration contract.v1` |
| `skill-driven-discovery-hybrid` | hybrid discovery with AGENTS.md, folder metadata, and hybrid scoring reconciliation. | `check-react-contract.*` | `react.v1` |
| `mcp-governed-stdio-http` | dual transport governance with stdio/http failover and fail-fast fallback reasoning. | `check-quality-gate.*` | `cross-domain primary reason arbitration contract.v1` |
| `hitl-governed-checkpoint` | await/resume/reject/timeout/recover governance matrix and replay-safe checkpoint signals. | `check-react-contract.*` | `react.v1` |
| `context-governed-reference-first` | reference-first, isolate handoff, edit gate, and tiering path aligned to context compression convergence evidence. | `check-context-jit-organization-contract.*` + `check-context-compression-production-contract.*` | `context_reference_first.v1` + `context_compression_production.v1` |
| `sandbox-governed-toolchain` | allowlist plus egress policy governance with deny fallback and classified reason output. | `check-security-sandbox-contract.*` + `check-sandbox-egress-allowlist-contract.*` | `sandbox_egress.v1` |
| `realtime-interrupt-resume` | resume semantics with cursor idempotency and replay-safe interruption classification. | `check-realtime-protocol-contract.*` | `realtime_event_protocol.v1` |
| `multi-agents-collab-recovery` | collaboration with mailbox and task-board control plus recovery replay continuity. | `check-multi-agent-shared-contract.*` | `cross-domain primary reason arbitration contract.v1` |
| `workflow-branch-retry-failfast` | branch routing with retry policy layering and fail-fast governance signal mapping. | `check-multi-agent-shared-contract.*` | `cross-domain primary reason arbitration contract.v1` |
| `mapreduce-large-batch` | large-batch mapreduce with controlled shard fanout, reduce aggregation, and retry-aware classification. | `check-multi-agent-shared-contract.*` | `cross-domain primary reason arbitration contract.v1` |
| `state-session-snapshot-recovery` | snapshot export, restore, and replay idempotency checkpoints across recovery boundaries. | `check-state-snapshot-contract.*` | `state_session_snapshot.v1` |
| `policy-budget-admission` | precedence and budget co-governance with deterministic decision trace classification. | `check-policy-precedence-contract.*` + `check-runtime-budget-admission-contract.*` | `policy_stack.v1` + `budget_admission.v1` |
| `tracing-eval-smoke` | tracing and eval interop path with replay-compatible diagnostics classification. | `check-agent-eval-and-tracing-interop-contract.*` | `otel_semconv.v1` + `agent_eval.v1` |
| `react-plan-notebook-loop` | react plus plan-notebook governance and hook-compatible change tracking semantics. | `check-react-plan-notebook-contract.*` | `react_plan_notebook.v1` |
| `hooks-middleware-extension-pipeline` | onion-chain middleware ordering, error bubbling, and extension pass-through governance path. | `check-hooks-middleware-contract.*` | `hooks_middleware.v1` |
| `observability-export-bundle` | observability export bundle with replay-aware package boundary and drift-safe fields. | `check-observability-export-and-bundle-contract.*` | `observability.v1` |
| `adapter-onboarding-manifest-capability` | manifest, capability negotiation, profile replay mapping, and scaffold drift governance checks. | `check-adapter-manifest-contract.*` + `check-adapter-capability-contract.*` + `check-adapter-contract-replay.*` | `adapter_contract_profile.v1` |
| `security-policy-event-delivery` | policy plus event plus delivery governance with deny semantics preserved under callback failure. | `check-security-policy-contract.*` + `check-security-event-contract.*` + `check-security-delivery-contract.*` | `cross-domain primary reason arbitration contract.v1` |
| `config-hot-reload-rollback` | hot reload fail-fast with atomic rollback and deterministic diagnostics reason mapping. | `check-quality-gate.*` | `cross-domain primary reason arbitration contract.v1` |
| `workflow-routing-strategy-switch` | strategy switch with confidence, cost, and capability-aware routing governance. | `check-multi-agent-shared-contract.*` | `cross-domain primary reason arbitration contract.v1` |
| `multi-agents-hierarchical-planner-validator` | hierarchical planner-validator orchestration with correction loop and governance checkpoints. | `check-multi-agent-shared-contract.*` | `cross-domain primary reason arbitration contract.v1` |
| `mainline-mailbox-async-delayed-reconcile` | mailbox async plus delayed dispatch plus reconcile path with canonical diagnostics evidence. | `check-multi-agent-shared-contract.*` | `cross-domain primary reason arbitration contract.v1` |
| `mainline-task-board-query-control` | query and control governance path with idempotent operation trace and replay classification. | `check-multi-agent-shared-contract.*` | `cross-domain primary reason arbitration contract.v1` |
| `mainline-scheduler-qos-backoff-dlq` | qos fairness with retry backoff and dead-letter classification under governance constraints. | `check-multi-agent-shared-contract.*` | `cross-domain primary reason arbitration contract.v1` |
| `mainline-readiness-admission-degradation` | readiness plus admission degradation path with strict classification and rollback-safe behavior. | `check-quality-gate.*` | `readiness-timeout-health replay fixture gate.v1` |
| `custom-adapter-mcp-model-tool-memory-pack` | adapter pack with manifest conformance, capability fallback, and replay-aware governance evidence. | `check-adapter-conformance.*` + `check-adapter-scaffold-drift.*` | `adapter_contract_profile.v1` |
| `custom-adapter-health-readiness-circuit` | health-readiness circuit governance with backoff, circuit transitions, and deterministic recovery signals. | `check-adapter-conformance.*` | `readiness-timeout-health replay fixture gate.v1` |

## Promotion Procedure
1. Start from `minimal` and confirm deterministic local run output for the target mode.
2. Execute `production-ish` and verify diagnostics/tracing markers plus contract gate mappings.
3. Complete the `Prod Delta Checklist` inside the mode-specific `production-ish/README.md`.
4. Run `check-agent-mode-smoke-stability-governance.*`; classification `example-smoke-latency-regression` and `example-smoke-flaky-regression` must stay green.
5. Run `check-agent-mode-migration-playbook-consistency.*` and `check-agent-mode-legacy-todo-cleanup.*`.
6. Run `check-agent-mode-pattern-coverage.*`, `check-agent-mode-examples-smoke.*`, and `check-quality-gate.*` before merge.

## Context-Governed Dependency Rule
- For `context-governed-reference-first`, completion requires green context compression and context organization gate outputs in the same branch validation record.
