## Why

Baymax currently projects real-time native callbacks, but it does not expose
source-owned retained event history or a cursor that remains meaningful after a
consumer or process restarts. A host observation sequence, an adapter EventID,
and diagnostics queries are not a source journal and cannot safely be presented
to Danny as native replay. Spec 1 established the versioned capability manifest
and mailbox ownership boundary; this change makes the durable-history claim
truthful, bounded, and independently verifiable.

## What Changes

- Introduce a public, read-only native event-history contract with source
  capabilities, opaque cursors, bounded read-after-cursor pages, retention
  floor, terminal-recovery facts, and stable typed errors:
  `ErrHistoryUnsupported`, `ErrCursorMalformed`, `ErrCursorForeign`,
  `ErrCursorExpired`, `ErrHistoryGap`, and `ErrSourceUnavailable`.
- Add a source audit for runner, isolated child, team, workflow, and mailbox.
  Every source declares its event-identity namespace, ordering scope,
  retention/cursor semantics, terminal recovery behavior, and supported or
  unsupported state. Undeclared or unsupported sources fail explicitly.
- Make the existing mailbox owner the first retained-history implementation. It
  records bounded lifecycle and terminal evidence in a source-owned journal and
  replays it across a new reader/consumer process without publishing, claiming,
  acknowledging, retrying, resuming, invoking a handler, or creating a Run.
- Preserve the event adapter as a real-time projection only. It may consume
  events supplied by a history reader, but it does not own a journal, manufacture
  replay identities, or convert a cursor read into execution recovery.
- Update the host runtime capability manifest so `mailbox_async` advertises the
  delivered history/replay semantics while direct, child, team, and workflow
  retain explicit unsupported reasons until their own source owners implement
  retained history.
- Add bounded source fixtures, restart/cursor evidence, offline replay tooling,
  parity gates, maintained documentation, and an offline example. No runtime
  configuration key, host observation store, scheduler, retry loop, or new
  execution owner is introduced.

## Capabilities

### New Capabilities

- `native-execution-event-history`: source-owned, durable, read-only native
  event history, capabilities, cursors, retention and replay error contract.

### Modified Capabilities

- `host-runtime-capability-manifest`: declare history/replay capability,
  retention and evidence per execution form without inference from API surface.
- `native-execution-event-adapter`: preserve the adapter's real-time-only
  boundary and prohibit adapter-owned retained history or replay identity.
- `multi-agent-mailbox-contract`: require the existing mailbox owner to expose
  bounded retained source journal and replay semantics for supported mailbox
  sources.
- `diagnostics-replay-tooling`: define reference-only history/cursor fixtures
  and gates that detect lifecycle, identity, cursor and no-side-effect drift.

## Impact

- **Public API/contract:** new `NativeEventHistoryReader` and DTO/error surface;
  capability-manifest values become concrete source facts rather than future
  placeholders.
- **Affected owners:** `core/types` supplies source-neutral public DTOs and
  validation; `orchestration/mailbox` owns the first actual journal and reader;
  planbridge consumes the manifest only; `host/eventadapter` remains projection
  only.
- **Unsupported paths:** runner, local child, team, and workflow return
  `ErrHistoryUnsupported` before a read and retain no implied replay or global
  order. Their existing real-time callback contracts remain unchanged.
- **Compatibility:** additive API. Existing Run/Stream, mailbox lifecycle,
  handler, retry, diagnostics and business-commit behavior remains unchanged.
  A caller must opt in to the history reader and handle typed unsupported,
  unavailable, foreign, expired and gap outcomes.
- **Verification:** source-owner unit/integration tests, restart read-after-
  cursor tests, offline reference-only fixtures, shell/PowerShell contract
  gates, the existing full Go/race/lint gates, and OpenSpec/doc consistency
  gates.

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
