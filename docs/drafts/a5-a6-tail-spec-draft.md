# A5/A6 Tail Spec Draft

更新时间：2026-03-18

## 目的

本草稿用于 A5（composed orchestration）与 A6（distributed subagent scheduler）主功能完成后的收口阶段，聚焦：
- 契约一致性补洞；
- 可观测与门禁闭环；
- 兼容性与迁移语义固定。

本草稿是“扫尾需求”，不重复定义 A5/A6 主能力。

## 建议变更能力

- `runtime-config-and-diagnostics-api`（MODIFIED）
- `action-timeline-events`（MODIFIED）
- `runtime-module-boundaries`（MODIFIED）
- `go-quality-gate`（MODIFIED）

## Spec Draft

## MODIFIED Requirements

### Requirement: Runtime diagnostics SHALL enforce bounded-cardinality multi-agent fields
Runtime diagnostics for composed orchestration and scheduler/subagent paths MUST keep bounded-cardinality semantics for additive fields to avoid unbounded growth in replay and high-concurrency runs.

At minimum, bounded-cardinality MUST be enforced for:
- reason-code aggregates by namespace,
- per-run child-attempt counters,
- lease and takeover counters.

#### Scenario: High-fanout run emits repeated scheduler reasons
- **WHEN** one run emits repeated `scheduler.*` and `subagent.*` events under retry/takeover paths
- **THEN** diagnostics aggregates remain bounded and replay-idempotent without cardinality explosion

### Requirement: Runtime config SHALL define explicit compatibility window for A5/A6 additive fields
Runtime config and diagnostics contracts MUST define a compatibility window for newly added A5/A6 fields, including additive defaults, nullable behavior, and migration notes for downstream consumers.

#### Scenario: Legacy consumer reads run summary after A5/A6 rollout
- **WHEN** a consumer parses only pre-A5 fields
- **THEN** runtime keeps prior semantics stable and new A5/A6 fields remain additive/optional

### Requirement: Action timeline SHALL provide canonical scheduler/subagent reason taxonomy
Action timeline contracts MUST define canonical reason taxonomy and minimal required reason set for scheduler/subagent closures.

At minimum, canonical reasons MUST include:
- `scheduler.enqueue`
- `scheduler.claim`
- `scheduler.lease_expired`
- `scheduler.requeue`
- `subagent.spawn`
- `subagent.join`
- `subagent.budget_reject`

#### Scenario: Scheduler takeover path is observed
- **WHEN** a claimed task loses lease and is reclaimed by another worker
- **THEN** timeline reasons follow canonical scheduler taxonomy and remain namespace-consistent

### Requirement: Action timeline SHALL require attempt-level correlation for scheduler paths
Scheduler-managed timeline events MUST carry attempt-level correlation metadata sufficient to disambiguate retries and takeovers.

Minimum required metadata:
- `task_id`
- `attempt_id`
- run linkage fields (`run_id` plus available parent linkage keys)

#### Scenario: Duplicate attempt replay is ingested
- **WHEN** equivalent timeline events for the same `task_id+attempt_id` are replayed
- **THEN** aggregation remains idempotent and attempt-level tracing remains deterministic

### Requirement: Boundary governance SHALL include scheduler/subagent closure checks in shared-contract gate
Shared multi-agent contract gate MUST validate scheduler/subagent closure rules in addition to existing team/workflow/a2a rules.

Minimum additional checks:
- scheduler/subagent reason namespace compliance,
- attempt-level correlation field presence,
- single-writer diagnostics ingestion path compliance.

#### Scenario: Change introduces non-canonical scheduler reason
- **WHEN** a change emits scheduler/subagent reason outside canonical namespace taxonomy
- **THEN** shared-contract gate fails and blocks merge

### Requirement: CI quality gate SHALL include crash-recovery and takeover contract suite
Quality gate MUST include a dedicated scheduler crash-recovery/takeover contract suite for A6 closure.

The suite MUST cover:
- worker crash with lease expiry takeover,
- duplicate submit/commit idempotency,
- equivalent Run/Stream semantics for scheduler-managed flows.

#### Scenario: CI executes scheduler contract gate
- **WHEN** scheduler contract suite runs in CI
- **THEN** takeover/idempotency/equivalence regressions fail the gate before merge

## 建议补充文档（非规范文本）

- `docs/mainline-contract-test-index.md`：新增 A5/A6 主链路映射行（含正向与异常场景）
- `docs/runtime-config-diagnostics.md`：补充 scheduler/subagent 字段解释、兼容窗口说明
- `docs/runtime-module-boundaries.md`：补充 scheduler owner 与禁止依赖方向
- `docs/v1-acceptance.md`：在已落地后更新 limitations 与 acceptance 条目

