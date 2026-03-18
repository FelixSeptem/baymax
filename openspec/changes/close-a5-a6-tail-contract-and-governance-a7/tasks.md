## 1. Contract Closure Freeze

- [ ] 1.1 Freeze A7 closure scope and canonical scheduler/subagent reason taxonomy.
- [ ] 1.1.1 Add canonical reason constants and mapper in `orchestration/scheduler` (new): `scheduler.enqueue`, `scheduler.claim`, `scheduler.lease_expired`, `scheduler.requeue`, `subagent.spawn`, `subagent.join`, `subagent.budget_reject`.
- [ ] 1.1.2 Add taxonomy contract test fixture and assertion in `tool/contributioncheck/multi_agent_contract.go` and `tool/contributioncheck/multi_agent_contract_test.go`.
- [ ] 1.1.3 Sync taxonomy wording to docs: `docs/runtime-config-diagnostics.md`, `docs/runtime-module-boundaries.md`, `docs/multi-agent-identifier-model.md`.
- [ ] 1.2 Freeze attempt-level correlation contract (`task_id+attempt_id`) for scheduler-managed timeline paths.
- [ ] 1.2.1 Define attempt-level correlation field contract in `openspec/specs/action-timeline-events/spec.md` and A7 delta specs.
- [ ] 1.2.2 Add timeline payload enforcement tests in `orchestration/scheduler/*_test.go` (new) and `integration/*scheduler*_contract_test.go` (new).
- [ ] 1.2.3 Extend shared-contract gate to fail when scheduler/subagent timeline events miss `task_id` or `attempt_id`.
- [ ] 1.3 Freeze additive compatibility-window rules for A5/A6 new fields.
- [ ] 1.3.1 Publish compatibility-window matrix (`additive + nullable + default`) in `docs/runtime-config-diagnostics.md`.
- [ ] 1.3.2 Add contract assertions for compatibility-window markers in `tool/contributioncheck/multi_agent_contract.go`.
- [ ] 1.3.3 Add regression doc checks to prevent removal of compatibility-window clauses.

## 2. Diagnostics And Compatibility Closure

- [ ] 2.1 Implement bounded-cardinality aggregation constraints for scheduler/subagent additive fields.
- [ ] 2.1.1 Extend `runtime/diagnostics/store.go` with scheduler/subagent additive counters (bounded-cardinality only).
- [ ] 2.1.2 Extend `observability/event/runtime_recorder.go` mapping so scheduler/subagent summary remains single-writer and additive.
- [ ] 2.1.3 Add store-level bounded-cardinality tests in `runtime/diagnostics/store_test.go`.
- [ ] 2.2 Add replay-idempotent tests for repeated scheduler takeover and duplicate commit ingestion.
- [ ] 2.2.1 Add replay-idempotency tests in `runtime/diagnostics/store_test.go` for repeated `task_id+attempt_id` ingestion.
- [ ] 2.2.2 Add integration contract tests in `integration/*scheduler*_contract_test.go` for duplicate submit/result replay.
- [ ] 2.2.3 Add runtime recorder replay assertions in `observability/event/runtime_recorder_test.go`.
- [ ] 2.3 Publish compatibility-window semantics for A5/A6 additive fields in diagnostics contract docs.
- [ ] 2.3.1 Ensure docs include legacy-consumer behavior examples and nullable fallback paths.
- [ ] 2.3.2 Link compatibility semantics in `docs/v1-acceptance.md` and `docs/mainline-contract-test-index.md`.

## 3. Timeline And Boundary Gate Closure

- [ ] 3.1 Enforce canonical scheduler/subagent reason taxonomy in timeline mapping and tests.
- [ ] 3.1.1 Add reason namespace checks in `tool/contributioncheck/multi_agent_contract.go` and negative cases in `tool/contributioncheck/multi_agent_contract_test.go`.
- [ ] 3.1.2 Add runtime-level namespace tests in `observability/event/runtime_recorder_test.go` and scheduler integration tests.
- [ ] 3.2 Enforce attempt-level correlation presence on scheduler-managed timeline events.
- [ ] 3.2.1 Add gate checks for correlation keys (`task_id`, `attempt_id`) with explicit violation codes.
- [ ] 3.2.2 Add integration tests asserting correlation is present on enqueue/claim/requeue/complete transitions.
- [ ] 3.3 Extend shared multi-agent contract gate for scheduler/subagent namespace + correlation + single-writer checks.
- [ ] 3.3.1 Extend `scripts/check-multi-agent-shared-contract.ps1` and `.sh` to include scheduler/subagent closure checks.
- [ ] 3.3.2 Update `docs/runtime-module-boundaries.md` with scheduler single-writer constraints and forbidden direct diagnostics writes.
- [ ] 3.3.3 Add CI required-check candidate entry for scheduler/subagent contract gate in docs/roadmap.

## 4. Quality Gate And Regression Closure

- [ ] 4.1 Add dedicated scheduler crash-recovery/takeover contract suite.
- [ ] 4.1.1 Add test file `integration/scheduler_recovery_contract_test.go` (new) covering worker crash, lease expiry, takeover success path.
- [ ] 4.1.2 Add deterministic fixtures/testdata for crash-recovery scenarios under `integration/testdata/scheduler/*`.
- [ ] 4.2 Add duplicate submit/commit idempotency gate scenarios.
- [ ] 4.2.1 Add duplicate submit/commit contract tests in `integration/scheduler_recovery_contract_test.go` and `runtime/diagnostics/store_test.go`.
- [ ] 4.2.2 Add gate row in `docs/mainline-contract-test-index.md` and required row assertion in `tool/contributioncheck/contract_index_test.go`.
- [ ] 4.3 Add Run/Stream semantic-equivalence gate scenarios for scheduler-managed flows.
- [ ] 4.3.1 Add Run/Stream equivalence tests in `integration/scheduler_recovery_contract_test.go`.
- [ ] 4.3.2 Ensure scheduler-managed equivalence covers status, execution order, additive summary counters, and reason namespace.
- [ ] 4.3.3 Add validation command path to CI scripts/quality gate docs once suite is green.

## 5. Documentation And Index Closure

- [x] 5.1 Update `docs/mainline-contract-test-index.md` with A5/A6 closure test mappings.
- [x] 5.2 Update `docs/runtime-config-diagnostics.md` and `docs/runtime-module-boundaries.md` with A7 closure constraints.
- [x] 5.3 Update `docs/v1-acceptance.md` and `docs/development-roadmap.md` to reflect A5/A6 closure status.
- [x] 5.4 Execute validation gates: `go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`, and shared contract checks.

## Validation Log (2026-03-18)

- [x] `go test ./...`
- [x] `$env:CGO_ENABLED='1'; go test -race ./...`
- [x] `golangci-lint run --config .golangci.yml`
- [x] `pwsh -File scripts/check-multi-agent-shared-contract.ps1`
- [x] `pwsh -File scripts/check-docs-consistency.ps1`
