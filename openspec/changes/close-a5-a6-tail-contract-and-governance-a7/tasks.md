## 1. Contract Closure Freeze

- [ ] 1.1 Freeze A7 closure scope and canonical scheduler/subagent reason taxonomy.
- [ ] 1.2 Freeze attempt-level correlation contract (`task_id+attempt_id`) for scheduler-managed timeline paths.
- [ ] 1.3 Freeze additive compatibility-window rules for A5/A6 new fields.

## 2. Diagnostics And Compatibility Closure

- [ ] 2.1 Implement bounded-cardinality aggregation constraints for scheduler/subagent additive fields.
- [ ] 2.2 Add replay-idempotent tests for repeated scheduler takeover and duplicate commit ingestion.
- [ ] 2.3 Publish compatibility-window semantics for A5/A6 additive fields in diagnostics contract docs.

## 3. Timeline And Boundary Gate Closure

- [ ] 3.1 Enforce canonical scheduler/subagent reason taxonomy in timeline mapping and tests.
- [ ] 3.2 Enforce attempt-level correlation presence on scheduler-managed timeline events.
- [ ] 3.3 Extend shared multi-agent contract gate for scheduler/subagent namespace + correlation + single-writer checks.

## 4. Quality Gate And Regression Closure

- [ ] 4.1 Add dedicated scheduler crash-recovery/takeover contract suite.
- [ ] 4.2 Add duplicate submit/commit idempotency gate scenarios.
- [ ] 4.3 Add Run/Stream semantic-equivalence gate scenarios for scheduler-managed flows.

## 5. Documentation And Index Closure

- [ ] 5.1 Update `docs/mainline-contract-test-index.md` with A5/A6 closure test mappings.
- [ ] 5.2 Update `docs/runtime-config-diagnostics.md` and `docs/runtime-module-boundaries.md` with A7 closure constraints.
- [ ] 5.3 Update `docs/v1-acceptance.md` and `docs/development-roadmap.md` to reflect A5/A6 closure status.
- [ ] 5.4 Execute validation gates: `go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`, and shared contract checks.
