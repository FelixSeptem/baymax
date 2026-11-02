## Why

An embedding host that polls or receives callbacks from an external source has
no narrow contract for binding source metadata, an external identity, durable
cursor progress, delivery disposition, health, and shutdown. Reusing a callback
EventID, observer status, or a source-history cursor leaves duplicate effects
and restart behavior ambiguous.

## What Changes

- Introduce an explicit, transport-neutral source-adapter lifecycle for a
  host-supplied source, consumer, and source-bound cursor state store.
- Bind every poll page and cursor to validated source metadata and immutable
  external identities; persist progress only after an explicit `sent` or
  `gone` consumer disposition.
- Preserve retryable or interrupted facts at the prior cursor, expose a bounded
  health claim, and make abort/dispose cooperative and idempotent.
- Add focused unit, fresh-process/store, offline replay, example, and
  shell/PowerShell gate evidence without changing callback observer behavior.

## Capabilities

### New Capabilities

- `source-adapter-lifecycle`: host-managed, source-bound polling and callback
  delivery lifecycle with cursor-after-commit and finite outcome semantics.

### Modified Capabilities

- `native-execution-event-adapter`: clarify that the synchronous observer is
  not the durable source-adapter lifecycle and remains without cursor or retry
  ownership.

## Impact

Owner: `host/sourceadapter`, including local external-source identity primitives.
The initial scope supplies an embeddable library API and injected in-memory/file
state implementations only; it creates no global source store, control plane,
queue, worker, terminal owner, runtime configuration key, diagnostics writer,
or business-effect ledger. The host retains consumer effect CAS. Rollback stops
the adapter and leaves the last confirmed source-bound cursor for operator
inspection; it never retries or fabricates a terminal state.

## Example Impact Assessment

新增示例

新增离线 `examples/source-adapter-lifecycle`，展示 sent/gone/retryable、
cursor-after-commit、abort 和 restart；不连接真实外部 source。

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`docs/guides/source-adapter-lifecycle.md`、`docs/runtime-module-boundaries.md`、`docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`docs/development-roadmap.md`、`examples/source-adapter-lifecycle/README.md`
- 维护 owner：`host/sourceadapter` maintainers
- 验证证据：`pwsh -File scripts/check-source-adapter-lifecycle-contract.ps1`、`pwsh -File scripts/check-docs-consistency.ps1`、`openspec validate --all`

## Documentation Impact Assessment

| area | outcome | affected paths | owner | evidence |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | architecture maintainers | boundary tests and dedicated gate |
| components | 修改文档 | `docs/guides/source-adapter-lifecycle.md` | host integration maintainers | source-adapter lifecycle suite |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | host integration maintainers | constructor bounds and timeout; no env/file/hot-update keys |
| contract/API | 修改文档 | `docs/guides/source-adapter-lifecycle.md` | host integration maintainers | bounds and restart tests |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md` | host integration maintainers | finite API health; no RuntimeRecorder write or QueryRuns field |
| examples | 新增文档 | `examples/source-adapter-lifecycle/README.md` | examples maintainers | offline example marker suite |
| CLI/integration | 修改文档 | `README.md` | documentation maintainers | library-first integration entry |
| best practices | 修改文档 | `docs/guides/source-adapter-lifecycle.md` | host integration maintainers | cursor-after-commit and CAS notes |
| roadmap | 修改文档 | `docs/development-roadmap.md` | documentation maintainers | roadmap status consistency |
