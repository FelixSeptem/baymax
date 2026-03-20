## 1. Mailbox Core Contract

- [x] 1.1 Add `orchestration/mailbox` package with envelope models (`command/event/result`) and canonical metadata fields.
- [x] 1.2 Add mailbox lifecycle API: `Publish`, `Consume`, `Ack`, `Nack`, `Requeue`, `Stats`, `Snapshot`, `Restore`.
- [x] 1.3 Implement envelope validation with fail-fast errors for missing IDs, invalid kind, and invalid timing fields.
- [x] 1.4 Implement idempotency-key convergence semantics for duplicate publish/delivery paths.

## 2. Backend and Query

- [x] 2.1 Implement mailbox `memory` backend with deterministic ordering semantics.
- [x] 2.2 Implement mailbox `file` backend with snapshot/restore semantics and startup consistency checks.
- [x] 2.3 Implement backend fallback rule: `file` initialization failure falls back to `memory` with explicit diagnostics marker.
- [x] 2.4 Implement mailbox query API with canonical filters, `AND` semantics, default sort `updated_at desc`, default page size `50`, max `200`, and opaque cursor.
- [x] 2.5 Add query validation for invalid `state/time_range/page_size/sort/cursor` with fail-fast behavior.

## 3. Multi-Agent Flow Convergence

- [x] 3.1 Add mailbox bridge for synchronous command->result flow used by orchestration integration paths.
- [x] 3.2 Add mailbox bridge for asynchronous result reporting and retry/dedup convergence.
- [x] 3.3 Add mailbox delayed-dispatch handling via envelope `not_before` and `expire_at`.
- [x] 3.4 Add mailbox correlation mapping with `run_id/task_id/workflow_id/team_id` for diagnostics/query composition.

## 4. Runtime Config and Diagnostics

- [x] 4.1 Add `mailbox.*` configuration domain in `runtime/config` with deterministic precedence `env > file > default`.
- [x] 4.2 Add startup/hot-reload fail-fast validation for mailbox backend/retry/ttl/query limits.
- [x] 4.3 Add mailbox diagnostics aggregate fields and query entrypoint in `runtime/diagnostics` and manager surface.
- [x] 4.4 Ensure mailbox diagnostics still follow RuntimeRecorder single-writer path requirements.

## 5. Deprecation and Gate Convergence

- [x] 5.1 Mark A11/A12/A13 legacy API entrypoints as deprecated and switch mainline examples/docs to mailbox path.
- [x] 5.2 Update shared multi-agent gate scripts to run mailbox contract suites as blocking checks.
- [x] 5.3 Update contract index mappings for mailbox flow rows (sync/async/delayed/query + backend parity).
- [x] 5.4 Remove old compatibility assumptions from docs/spec references where mailbox now defines canonical behavior.

## 6. Contract Tests and Validation

- [x] 6.1 Add unit contract tests for envelope validation, idempotency, ack/nack/retry, ttl/expiry, and dlq semantics.
- [x] 6.2 Add unit contract tests for mailbox query filtering, pagination defaults, cursor determinism, and fail-fast invalid input.
- [x] 6.3 Add integration contract tests for sync/async/delayed convergence through mailbox and Run/Stream semantic equivalence.
- [x] 6.4 Add integration contract tests for memory/file backend parity and restore/replay determinism.
- [x] 6.5 Run `go test ./...`.
- [x] 6.6 Run `go test -race ./...`.
- [x] 6.7 Run `golangci-lint run --config .golangci.yml`.
- [x] 6.8 Run `pwsh -File scripts/check-docs-consistency.ps1`.
- [x] 6.9 Run `pwsh -File scripts/check-multi-agent-shared-contract.ps1`.
- [x] 6.10 Run `openspec validate introduce-unified-mailbox-coordination-contract-a30 --strict`.
