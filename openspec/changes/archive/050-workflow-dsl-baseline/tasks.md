## 1. DSL Contract And Validation

- [x] 1.1 Define workflow DSL schema (`step/depends_on/condition/retry/timeout`) and validation error taxonomy.
- [x] 1.2 Implement static DAG validation (cycle, missing dependency, duplicate step ID, unsupported values).
- [x] 1.3 Add config contract fields and fail-fast validation for workflow baseline settings.

## 2. Workflow Engine Skeleton

- [x] 2.1 Add workflow module scaffold with planner/executor/checkpoint interfaces.
- [x] 2.2 Implement deterministic scheduling baseline (stable ordering for ready steps).
- [x] 2.3 Implement step execution adapter to call existing runner/tool/mcp/skill paths.

## 3. Runtime Semantics

- [x] 3.1 Implement step-level retry and timeout handling with bounded attempts.
- [x] 3.2 Implement minimal checkpoint persistence and resume semantics.
- [x] 3.3 Ensure workflow execution preserves Run/Stream semantic equivalence.

## 4. Observability And Diagnostics

- [x] 4.1 Extend timeline mapping with `workflow_id/step_id` correlation metadata.
- [x] 4.2 Extend diagnostics with workflow summary aggregates and replay-safe counters.
- [x] 4.3 Verify single-writer ingestion and idempotency under duplicated workflow events.

## 5. Validation And Documentation

- [x] 5.1 Add unit/integration tests for DSL parse/validate/schedule/retry/resume paths.
- [x] 5.2 Add contract tests for Run/Stream equivalence and replay stability.
- [x] 5.3 Update `docs/runtime-config-diagnostics.md`, `docs/runtime-module-boundaries.md`, and workflow example guidance.
