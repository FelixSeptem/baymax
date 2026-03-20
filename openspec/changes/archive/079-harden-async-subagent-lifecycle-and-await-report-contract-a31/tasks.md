## 1. Preconditions and Lifecycle Model

- [x] 1.1 Confirm A30 (`introduce-unified-mailbox-coordination-contract-a30`) is archived and mailbox main path is active before enabling A31 implementation.
- [x] 1.2 Extend scheduler task-state model to include `awaiting_report` and keep snapshot/restore compatibility for existing states.
- [x] 1.3 Add deterministic lifecycle transition from async accepted to `awaiting_report` in composer/scheduler bridge.

## 2. Async-Await Timeout and Late-Report Governance

- [x] 2.1 Implement async-await timeout tracking and deterministic terminalization (`failed` by default, `dead_letter` when configured policy applies).
- [x] 2.2 Implement late-report `drop_and_record` behavior so late reports do not mutate finalized business terminal outcome.
- [x] 2.3 Ensure duplicate or replayed async reports converge idempotently without additive-counter inflation.

## 3. Query, Config, and Diagnostics Contract Alignment

- [x] 3.1 Extend Task Board query state filter to support `awaiting_report` while preserving existing pagination/sort/cursor contract.
- [x] 3.2 Add `scheduler.async_await.*` config fields with deterministic precedence `env > file > default` and fail-fast startup/hot-reload validation.
- [x] 3.3 Add additive async-await diagnostics fields (`async_await_total`, `async_timeout_total`, `async_late_report_total`, `async_report_dedup_total`) with replay-idempotent behavior.

## 4. Contract Tests and Gate Integration

- [x] 4.1 Add scheduler/unit contract tests for awaiting-report transitions, timeout terminalization, and late-report handling.
- [x] 4.2 Add integration contract tests for Run/Stream semantic equivalence and memory/file backend parity under async-await lifecycle.
- [x] 4.3 Update shared multi-agent gate scripts to include async-await lifecycle suites as blocking checks.
- [x] 4.4 Update `docs/mainline-contract-test-index.md` mapping rows for async-await lifecycle coverage and gate paths.

## 5. Documentation and Validation

- [x] 5.1 Update `README.md`, `docs/runtime-config-diagnostics.md`, and `docs/development-roadmap.md` with A31 scope, defaults, and non-goals.
- [x] 5.2 Run `go test ./...`.
- [x] 5.3 Run `go test -race ./...`.
- [x] 5.4 Run `golangci-lint run --config .golangci.yml`.
- [x] 5.5 Run `pwsh -File scripts/check-docs-consistency.ps1`.
- [x] 5.6 Run `pwsh -File scripts/check-multi-agent-shared-contract.ps1`.
- [x] 5.7 Run `openspec validate harden-async-subagent-lifecycle-and-await-report-contract-a31 --strict`.
