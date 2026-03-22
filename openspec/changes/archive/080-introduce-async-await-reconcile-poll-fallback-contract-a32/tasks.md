## 1. Preconditions and Data Contract

- [x] 1.1 Confirm A31 lifecycle baseline (`awaiting_report`, timeout, late-report policy) is merged and usable as A32 dependency.
- [x] 1.2 Extend scheduler task/attempt persistence model with remote correlation key and terminal resolution-source markers using additive compatibility semantics.
- [x] 1.3 Define callback/poll terminal arbitration helper with deterministic `first_terminal_wins + record_conflict` behavior.

## 2. Reconcile Poll Fallback Implementation

- [x] 2.1 Implement reconcile dispatcher for `awaiting_report` tasks with configurable `interval`, `batch_size`, and `jitter_ratio`.
- [x] 2.2 Integrate A2A status/result polling path and normalize reconcile result classifications (`terminal`, `not_found`, retryable error, non-retryable error).
- [x] 2.3 Implement `not_found_policy=keep_until_timeout` semantics so `not_found` does not directly force terminalization before timeout boundary.
- [x] 2.4 Route reconcile terminal outcomes through existing scheduler terminal commit contract and preserve idempotent convergence.

## 3. Query, Config, and Diagnostics Alignment

- [x] 3.1 Add `scheduler.async_await.reconcile.*` config fields with defaults (`enabled=false`, `interval=5s`, `batch_size=64`, `jitter_ratio=0.2`, `not_found_policy=keep_until_timeout`) and fail-fast validation on startup/hot reload.
- [x] 3.2 Extend Task Board query response with additive async observability fields (`resolution_source`, remote correlation field, conflict marker) while preserving deterministic pagination/cursor behavior.
- [x] 3.3 Add reconcile additive diagnostics aggregates and ensure replay-idempotent counters.

## 4. Contract Tests and Gate Integration

- [x] 4.1 Add scheduler/unit contract tests for reconcile polling cadence controls, not-found behavior, and terminal arbitration conflicts.
- [x] 4.2 Add integration contract tests for callback-loss fallback convergence, Run/Stream equivalence, and memory/file backend parity.
- [x] 4.3 Add replay contract tests for mixed callback/poll duplicated events to verify additive-counter stability.
- [x] 4.4 Integrate reconcile suites into `scripts/check-multi-agent-shared-contract.sh` and `scripts/check-multi-agent-shared-contract.ps1` as blocking checks.

## 5. Documentation and Contract Mapping

- [x] 5.1 Update `README.md` and core component docs with async-await reconcile fallback behavior and non-goals.
- [x] 5.2 Update `docs/runtime-config-diagnostics.md` with reconcile config fields, defaults, and diagnostics field semantics.
- [x] 5.3 Update `docs/mainline-contract-test-index.md` with reconcile suites and shared-gate mapping.
- [x] 5.4 Update `docs/development-roadmap.md` to reflect A32 scope, status, and dependency relation with A31.

## 6. Validation

- [x] 6.1 Run `go test ./...`.
- [x] 6.2 Run `go test -race ./...`.
- [x] 6.3 Run `golangci-lint run --config .golangci.yml`.
- [x] 6.4 Run `pwsh -File scripts/check-docs-consistency.ps1`.
- [x] 6.5 Run `pwsh -File scripts/check-multi-agent-shared-contract.ps1`.
- [x] 6.6 Run `openspec validate introduce-async-await-reconcile-poll-fallback-contract-a32 --strict`.
