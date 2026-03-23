## 1. Preconditions and Contract Baseline

- [x] 1.1 Confirm A37 gate fail-fast changes are green before introducing new mailbox worker recovery semantics.
- [x] 1.2 Freeze updated mailbox lifecycle canonical reason taxonomy set (including `lease_expired`).
- [x] 1.3 Add/adjust contract fixtures for worker lease defaults (`inflight_timeout=30s`, `heartbeat_interval=5s`, `reclaim_on_consume=true`, `panic_policy=follow_handler_error_policy`).

## 2. Mailbox State and Worker Recovery Implementation

- [x] 2.1 Extend mailbox record/state model with lease metadata required for in-flight timeout reclaim.
- [x] 2.2 Implement stale in-flight reclaim path triggered from consume flow when `reclaim_on_consume=true`.
- [x] 2.3 Implement worker heartbeat renewal for active in-flight message processing.
- [x] 2.4 Implement handler panic recovery in worker loop and map recovered panic to configured handler-error policy path.
- [x] 2.5 Ensure reclaim/recover transitions emit deterministic lifecycle events and canonical reason mapping.

## 3. Runtime Config and Hot Reload Safety

- [x] 3.1 Extend `runtime/config` with `mailbox.worker.inflight_timeout`, `mailbox.worker.heartbeat_interval`, `mailbox.worker.reclaim_on_consume`, `mailbox.worker.panic_policy`.
- [x] 3.2 Set defaults to recommended values and keep existing worker defaults unchanged.
- [x] 3.3 Add startup and hot-reload fail-fast validation (`inflight_timeout>0`, `heartbeat_interval>0`, `heartbeat_interval<inflight_timeout`, enum validation for `panic_policy`).
- [x] 3.4 Ensure invalid reload rolls back atomically to last valid snapshot.

## 4. Diagnostics and Aggregation Coverage

- [x] 4.1 Extend mailbox diagnostics records to include reclaim and panic-recover lifecycle observability.
- [x] 4.2 Ensure reclaim events record canonical reason code `lease_expired`.
- [x] 4.3 Ensure panic-recovered path remains queryable and deterministic without breaking additive compatibility.
- [x] 4.4 Extend mailbox aggregates/query assertions for reclaim/recover transitions.

## 5. Contract Tests and Gate Integration

- [x] 5.1 Add unit tests for worker panic recovery and policy mapping (`requeue`/`nack`).
- [x] 5.2 Add unit/integration tests for stale in-flight reclaim and heartbeat no-premature-reclaim semantics.
- [x] 5.3 Add integration suites for worker crash/restart reclaim convergence, Run/Stream equivalence, and memory/file parity.
- [x] 5.4 Add taxonomy drift guard tests for `lease_expired` and canonical reason set.
- [x] 5.5 Integrate recover/reclaim suites into `scripts/check-multi-agent-shared-contract.sh` and `.ps1`.

## 6. Documentation Synchronization

- [x] 6.1 Update `README.md` and `orchestration/README.md` with worker lease/reclaim/recover semantics and defaults.
- [x] 6.2 Update `docs/runtime-config-diagnostics.md` with new `mailbox.worker.*` fields and reclaim/recover diagnostics fields.
- [x] 6.3 Update `docs/mainline-contract-test-index.md` with A38 contract test and gate path mappings.
- [x] 6.4 Update `docs/development-roadmap.md` with A38 scope and status mapping.

## 7. Validation

- [x] 7.1 Run `go test ./orchestration/mailbox ./runtime/config ./runtime/diagnostics ./integration -count=1`.
- [x] 7.2 Run `go test -race ./...`.
- [x] 7.3 Run `golangci-lint run --config .golangci.yml`.
- [x] 7.4 Run `pwsh -File scripts/check-multi-agent-shared-contract.ps1`.
- [x] 7.5 Run `pwsh -File scripts/check-quality-gate.ps1`.
- [x] 7.6 Run `pwsh -File scripts/check-docs-consistency.ps1`.
- [x] 7.7 Run `openspec validate harden-mailbox-worker-lease-reclaim-and-panic-recovery-contract-a38 --strict`.
