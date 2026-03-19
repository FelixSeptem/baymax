## 1. Collaboration Primitive Core

- [ ] 1.1 Create `orchestration/collab` package with unified primitive contracts for `handoff`, `delegation`, and `aggregation`.
- [ ] 1.2 Define primitive request/response models and normalized terminal outcome mapping.
- [ ] 1.3 Implement aggregation strategy support for `all_settled` and `first_success` only.
- [ ] 1.4 Set default aggregation strategy to `all_settled` in primitive execution path.
- [ ] 1.5 Set default failure policy to `fail_fast` in primitive execution path.
- [ ] 1.6 Keep primitive-layer retry disabled by default and rely on existing scheduler/retry governance.

## 2. Teams Workflow Composer Integration

- [ ] 2.1 Integrate teams remote execution path with shared collaboration delegation primitive.
- [ ] 2.2 Integrate workflow delegation/handoff/aggregation markers with shared collaboration primitive path.
- [ ] 2.3 Integrate composer child execution path with collaboration primitive entrypoints.
- [ ] 2.4 Ensure integration preserves existing terminal semantics and idempotent convergence.
- [ ] 2.5 Add compatibility adapters to avoid breaking existing module-local API usage.

## 3. Config and Diagnostics Contract

- [ ] 3.1 Add runtime config fields for collaboration primitive controls under `composer.collab.*`.
- [ ] 3.2 Enforce default config values: enabled=false, strategy=all_settled, policy=fail_fast, retry.enabled=false.
- [ ] 3.3 Add fail-fast validation for invalid strategy/policy/retry combinations on startup and hot reload.
- [ ] 3.4 Add additive diagnostics fields: `collab_handoff_total`, `collab_delegation_total`, `collab_aggregation_total`, `collab_aggregation_strategy`, `collab_fail_fast_total`.
- [ ] 3.5 Add parser-compatibility tests for `additive + nullable + default` behavior on new fields.

## 4. Timeline and Reason Taxonomy

- [ ] 4.1 Add collaboration primitive reason mapping using existing namespaces only.
- [ ] 4.2 Enforce no new top-level `collab.*` namespace in timeline reason taxonomy.
- [ ] 4.3 Add required collaboration reason coverage checks (`team.*` and `workflow.*` collaboration reasons) in shared contract assertions.
- [ ] 4.4 Ensure collaboration timeline events include required correlation fields where applicable.

## 5. Contract Tests and Shared Gate

- [ ] 5.1 Add integration contract tests for collaboration primitives in sync mode.
- [ ] 5.2 Add integration contract tests for collaboration primitives in async-reporting mode.
- [ ] 5.3 Add integration contract tests for collaboration primitives in delayed-dispatch mode.
- [ ] 5.4 Add Run/Stream equivalence tests for collaboration primitive matrix.
- [ ] 5.5 Add replay-idempotency and recovery-consistency tests for collaboration primitive aggregates.
- [ ] 5.6 Integrate collaboration suites into `check-multi-agent-shared-contract.sh` and `.ps1`.
- [ ] 5.7 Extend `tool/contributioncheck` snapshot assertions and failure-code coverage for collaboration contracts.

## 6. Documentation and Delivery

- [ ] 6.1 Update `README.md` with collaboration primitive positioning and minimal usage example.
- [ ] 6.2 Update `docs/runtime-config-diagnostics.md` with `composer.collab.*` config and additive diagnostics fields.
- [ ] 6.3 Update `docs/mainline-contract-test-index.md` with A16 coverage mapping rows.
- [ ] 6.4 Update `docs/development-roadmap.md` with A16 scope and sequencing relative to A15.
- [ ] 6.5 Run `go test ./...`.
- [ ] 6.6 Run `go test -race ./...`.
- [ ] 6.7 Run `golangci-lint run --config .golangci.yml`.
- [ ] 6.8 Run `pwsh -File scripts/check-docs-consistency.ps1`.
- [ ] 6.9 Run `pwsh -File scripts/check-multi-agent-shared-contract.ps1`.
