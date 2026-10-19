## 1. Proposal baseline, source audit and public contract wiring

- [x] 1.1 Update `docs/development-roadmap.md` to mark this Spec 2 change active and verify it agrees with `openspec list --json` through `pwsh -File scripts/check-openspec-roadmap-status-consistency.ps1`.
- [x] 1.2 Complete the documented support matrix for runner, isolated child, team, workflow and mailbox, including source/event namespaces, sequence scope, retention/cursor scope, terminal recovery and unsupported reasons; verify every source is explicit and no source is inferred from package/DTO presence.
- [x] 1.3 Synchronize every document named in the Public Semantic Documentation Commitment and Example Impact Assessment before any implementation task is marked complete; verify `pwsh -File scripts/check-openspec-documentation-impact.ps1` and `pwsh -File scripts/check-openspec-example-impact-declaration.ps1` pass with all declared paths in the branch diff.

## 2. Source-neutral native history contract

- [x] 2.1 Add bounded detached public `SourceRef`, `HistoryCapabilities`, `OpaqueCursor`, `NativeEventPage` and `NativeEventHistoryReader` types in the source-neutral core API; verify unit tests cover immutability, exact limits and invalid/oversized source or cursor facts.
- [x] 2.2 Define and document typed `ErrHistoryUnsupported`, `ErrCursorMalformed`, `ErrCursorForeign`, `ErrCursorExpired`, `ErrHistoryGap` and `ErrSourceUnavailable` outcomes; verify each can be distinguished without message parsing and no error path activates a source.
- [x] 2.3 Implement source capability audit/adapters for runner, child, team and workflow that explicitly return unsupported history before a read; verify they neither subscribe to callbacks nor create host cursors, latest fallbacks, Run, Stream, retry or restore side effects.
- [x] 2.4 Define bounded page semantics for positive limits, next cursor, retention floor and complete; verify source-local ordering and that parallel sources do not gain fabricated global order.

## 3. Mailbox owner journal and cursor replay

- [x] 3.1 Extend the existing mailbox owner/backend state with bounded source-history append facts for admit, lease, ack, nack, retry, lease loss, timeout, DLQ and source terminal transitions; verify original message/delivery/lease/attempt/causation and terminal facts persist without payloads, credentials, callback errors or business-commit assertions.
- [x] 3.2 Implement source-local mailbox sequence, retention floor/window and backend availability declarations; verify in-memory and retained backends advertise only their actual process/restart guarantees.
- [x] 3.3 Implement mailbox `Capabilities` and bounded `ReadAfter` with opaque versioned cursors; verify a newly constructed reader/consumer process can continue after a confirmed retained cursor without creating a new logical admission or parent Run.
- [x] 3.4 Implement fail-closed mailbox cursor validation for malformed, foreign, expired, gapped and unavailable outcomes; verify no case silently substitutes a floor/latest page or mutates mailbox delivery state.
- [x] 3.5 Add concurrency and snapshot-isolation coverage for append, retention and read pages; verify source reads cannot publish, claim, renew, ack, nack, requeue, retry, DLQ, invoke handlers or mutate records.

## 4. Manifest and event-adapter ownership boundaries

- [x] 4.1 Update the versioned capability manifest so `mailbox_async` reports only evidenced retained-history/after-cursor support and all other forms retain explicit source-owned unsupported reasons; verify unsupported form/API/DTO presence cannot be read as support.
- [x] 4.2 Preserve `host/eventadapter` as real-time projection only and add history-fact projection boundaries where needed; verify callback EventID/TraceID is never accepted as a journal identity/cursor and adapter-only requests do not create history.
- [x] 4.3 Add planbridge/mailbox integration coverage proving history availability/unavailability does not change `mailbox_async` admission, Run/Stream terminal parity, authority, occurrence, effect namespace or host business-commit behavior.

## 5. Restart, terminal and source-order conformance

- [x] 5.1 Add mailbox restart integration cases for crash-before-consumer-projection, confirmed-cursor replay and retained terminal evidence; verify replay returns the same source-owned terminal/lifecycle facts and does not run a model, tool, handler, retry, resume or restore.
- [x] 5.2 Add source-audit integration cases for runner, child, team and workflow; verify each reports declared unsupported history while retaining its existing real-time callback semantics and local parent/child correlation.
- [x] 5.3 Add negative tests for cross-source foreign cursors, malformed/version-invalid cursors, retention expiry, retention gaps, source unavailability, boundary page limits and caller mutation; verify all errors are typed and fail before side effects.

## 6. Replay fixtures, gates, examples and documentation

- [x] 6.1 Add versioned reference-only history/cursor fixtures and diagnostics replay classifications for capability matrix, cursor restart, terminal recovery, malformed/foreign/expired/gapped/unsupported/unavailable paths and unsafe latest fallback; verify fixture replay constructs no source owner, runner, handler, model, tool, host callback, retry or restore path.
- [x] 6.2 Add equivalent shell and PowerShell native-history contract gates and wire them into quality gates; verify both detect identity/order/cursor/retention/no-side-effect drift.
- [x] 6.3 Update `examples/native-execution-events/README.md` before modifying the example code with semantic anchors, source ownership, mailbox history path, expected success/error markers and rollback notes; then provide an offline example showing restart replay and explicit unsupported paths without a provider or business database.
- [x] 6.4 Synchronize README, integration, best-practices, package, module-boundary, runtime-config/diagnostics and contract-test-index documentation; verify they distinguish history replay from callback projection, execution recovery, checkpoint restore and business commit.

## 7. Final verification and archival evidence

- [x] 7.1 Run targeted core/mailbox/planbridge/eventadapter/integration/diagnostics-replay tests, then `go test ./...`, `go test -race ./...` and `golangci-lint run --config .golangci.yml`; record commands, exit codes, risks, rollback and any environmental limitation in `verification.md`.
- [x] 7.2 Run `pwsh -File scripts/check-quality-gate.ps1`, `pwsh -File scripts/check-docs-consistency.ps1`, native-history shell/PowerShell gates, `openspec validate retain-source-native-event-history-and-cursor-replay --strict`, documentation/example-impact/roadmap consistency gates and `git diff --check`; verify all contract and documentation commitments before archive.

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`docs/development-roadmap.md`、`docs/guides/integration.md`、`docs/guides/best-practices.md`、`docs/runtime-module-boundaries.md`、`docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`core/types/README.md`、`orchestration/mailbox/README.md`、`host/eventadapter/README.md`、`orchestration/planbridge/README.md`、`examples/native-execution-events/README.md`
- 维护 owner：runtime-core、mailbox、host-integration、documentation 与 examples
  maintainers
- 验证证据：OpenSpec strict validation、documentation impact/example impact/
  roadmap consistency gates、history/cursor contract gates、`go test ./...`、
  `go test -race ./...` 与 `golangci-lint run --config .golangci.yml`

## Example Impact Assessment

修改示例

- 影响路径：`examples/native-execution-events/README.md` 与其离线示例代码。
- 理由：示例必须区分实时 callback projection 与 mailbox source journal，展示
  cursor restart replay、typed unsupported/gap/expiry 结果，以及 replay 不触发
  执行或业务写入。

## Documentation Impact Assessment

| Area | Outcome | Affected paths | Owner | Verification |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | runtime-core maintainers | module ownership and read-only boundary reviewed; docs consistency gate |
| components | 修改文档 | `core/types/README.md`, `orchestration/mailbox/README.md`, `host/eventadapter/README.md`, `orchestration/planbridge/README.md` | package owners | package history/cursor contracts and unsupported boundaries; package and history gates |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime configuration maintainers | explicit no-new-key/diagnostic limits documented; docs consistency gate |
| contract/API | 修改文档 | `README.md`, `docs/guides/integration.md` | host-integration maintainers | support matrix, API bounds and typed errors; integration and history gates |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md`, `docs/mainline-contract-test-index.md` | observability maintainers | reference-only replay and RuntimeRecorder boundary; replay contract gate |
| examples | 修改文档 | `examples/native-execution-events/README.md` | examples maintainers | offline restart replay markers and rollback; example smoke/history gate |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | host-integration maintainers | actual reader call sequence and source audit; integration/history tests |
| best practices | 修改文档 | `docs/guides/best-practices.md` | documentation maintainers | confirmed cursor, expiry/gap and no-latest guidance; docs consistency gate |
| roadmap | 修改文档 | `docs/development-roadmap.md` | roadmap maintainers | active change slug matches `openspec list --json`; roadmap consistency gate |
