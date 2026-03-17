## 1. DSL Contract And Validation

- [ ] 1.1 Define workflow DSL schema (`step/depends_on/condition/retry/timeout`) and validation error taxonomy.
- [ ] 1.2 Implement static DAG validation (cycle, missing dependency, duplicate step ID, unsupported values).
- [ ] 1.3 Add config contract fields and fail-fast validation for workflow baseline settings.

## 2. Workflow Engine Skeleton

- [ ] 2.1 Add workflow module scaffold with planner/executor/checkpoint interfaces.
- [ ] 2.2 Implement deterministic scheduling baseline (stable ordering for ready steps).
- [ ] 2.3 Implement step execution adapter to call existing runner/tool/mcp/skill paths.

## 3. Runtime Semantics

- [ ] 3.1 Implement step-level retry and timeout handling with bounded attempts.
- [ ] 3.2 Implement minimal checkpoint persistence and resume semantics.
- [ ] 3.3 Ensure workflow execution preserves Run/Stream semantic equivalence.

## 4. Observability And Diagnostics

- [ ] 4.1 Extend timeline mapping with `workflow_id/step_id` correlation metadata.
- [ ] 4.2 Extend diagnostics with workflow summary aggregates and replay-safe counters.
- [ ] 4.3 Verify single-writer ingestion and idempotency under duplicated workflow events.

## 5. Validation And Documentation

- [ ] 5.1 Add unit/integration tests for DSL parse/validate/schedule/retry/resume paths.
- [ ] 5.2 Add contract tests for Run/Stream equivalence and replay stability.
- [ ] 5.3 Update `docs/runtime-config-diagnostics.md`, `docs/runtime-module-boundaries.md`, and workflow example guidance.
