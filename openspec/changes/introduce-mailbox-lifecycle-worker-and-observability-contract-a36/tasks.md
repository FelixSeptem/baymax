## 1. Preconditions and Contract Baseline

- [ ] 1.1 Confirm A35 mailbox wiring contract suites are green before introducing worker lifecycle changes.
- [ ] 1.2 Freeze mailbox lifecycle canonical reason taxonomy set (`retry_exhausted`, `expired`, `consumer_mismatch`, `message_not_found`, `handler_error`).
- [ ] 1.3 Add/adjust contract fixtures for worker defaults (`enabled=false`, `poll_interval=100ms`, `handler_error_policy=requeue`).

## 2. Mailbox Worker Primitive

- [ ] 2.1 Implement library-level mailbox worker loop in `orchestration/mailbox` for `consume -> handler -> ack|nack|requeue`.
- [ ] 2.2 Implement configurable handler error policy with default `requeue`.
- [ ] 2.3 Ensure worker polling interval control with default `100ms` and strict `>0` validation.
- [ ] 2.4 Keep worker optional and default disabled to preserve existing behavior.

## 3. Runtime Config Wiring

- [ ] 3.1 Extend `runtime/config` with `mailbox.worker.enabled`, `mailbox.worker.poll_interval`, and `mailbox.worker.handler_error_policy`.
- [ ] 3.2 Add startup/hot-reload fail-fast validation for invalid worker settings.
- [ ] 3.3 Ensure invalid hot reload rolls back atomically to last valid snapshot.

## 4. Diagnostics and Reason Taxonomy

- [ ] 4.1 Record mailbox lifecycle diagnostics for consume/ack/nack/requeue/dead_letter/expired transitions.
- [ ] 4.2 Enforce canonical reason taxonomy mapping for lifecycle failure and retry paths.
- [ ] 4.3 Extend mailbox aggregates/query assertions to cover worker-driven lifecycle transitions.

## 5. Contract Tests and Gate Integration

- [ ] 5.1 Add unit tests for worker loop behavior, default policy, and handler-error transitions.
- [ ] 5.2 Add integration suites for worker-enabled lifecycle flows and worker-disabled no-op baseline.
- [ ] 5.3 Add Run/Stream equivalence and memory/file parity checks for mailbox lifecycle worker flows.
- [ ] 5.4 Add gate checks for non-canonical lifecycle reason drift as blocking failure.
- [ ] 5.5 Integrate lifecycle worker suites into `scripts/check-multi-agent-shared-contract.sh` and `.ps1`.

## 6. Documentation Synchronization

- [ ] 6.1 Update `README.md` and `orchestration/README.md` to describe mailbox worker defaults and lifecycle semantics.
- [ ] 6.2 Update `docs/runtime-config-diagnostics.md` with `mailbox.worker.*` fields and lifecycle diagnostics taxonomy.
- [ ] 6.3 Update `docs/mainline-contract-test-index.md` with mailbox lifecycle worker coverage and gate mappings.
- [ ] 6.4 Update `docs/development-roadmap.md` with A36 scope and status mapping.

## 7. Validation

- [ ] 7.1 Run `go test ./...`.
- [ ] 7.2 Run `go test -race ./...`.
- [ ] 7.3 Run `golangci-lint run --config .golangci.yml`.
- [ ] 7.4 Run `pwsh -File scripts/check-multi-agent-shared-contract.ps1`.
- [ ] 7.5 Run `pwsh -File scripts/check-quality-gate.ps1`.
- [ ] 7.6 Run `pwsh -File scripts/check-docs-consistency.ps1`.
- [ ] 7.7 Run `openspec validate introduce-mailbox-lifecycle-worker-and-observability-contract-a36 --strict`.
