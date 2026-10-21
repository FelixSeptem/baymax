## Why

Baymax currently exposes source-native retained history only for the mailbox
owner. Runner, isolated child, team, and workflow sources still expose live
callbacks or an explicit unsupported result, so a new consumer process cannot
replay source facts after a projection crash. Danny must not mistake a callback
EventID, an application observation sequence, or a mailbox-only capability for
a durable native journal.

This change turns the roadmap's source-history candidate into a truthful,
source-owned contract: every source declares its actual history boundary, and
each source that claims support provides bounded, read-only replay that survives
the supported process/storage boundary. It does not claim execution recovery.

## What Changes

- Define the public source-history contract for runner, isolated child, team,
  workflow, and mailbox sources: stable identity namespaces, source-local
  sequence scope, retention floor/window, opaque cursor scope/version, terminal
  recovery semantics, availability, and typed malformed/foreign/expired/gap/
  unavailable errors.
- Provide source-owned retained journals and bounded `ReadAfter` readers for
  the supported runner, child, team, and workflow stores, while preserving the
  existing mailbox MemoryStore/FileStore distinction. A source that cannot
  provide the declared guarantee remains explicitly unsupported or unavailable.
- Make source history and Danny's application projection independently
  recoverable after a consumer crash. Overlapping pages deduplicate by source
  event identity; conflicting payloads fail closed; expired or gapped cursors
  never fall back to latest or retention-floor replay.
- Keep replay structurally read-only. A history read cannot invoke Run,
  Stream, model, tool, handler, retry, resume, restore, ack, nack, lease,
  requeue, DLQ, scheduler, business commit, or diagnostics writes.
- Update the versioned capability manifest with a separate history/replay cell
  for every source/form. Package existence, callback EventID, DTO compilation,
  mailbox capability, and host observation order remain invalid evidence.
- Preserve the existing real-time event adapter as a projection only and add
  source-history fixtures, restart/gap/expiry/foreign/unavailable tests, and
  shell/PowerShell contract gates.

## Capabilities

### New Capabilities

None. This is an incremental source-owner expansion of the existing
`native-execution-event-history` contract; no second history capability or
parallel event store is introduced.

### Modified Capabilities

- `native-execution-event-history`: require an independently declared,
  source-owned history implementation or typed unsupported/unavailable result
  for runner, isolated child, team, workflow, and mailbox; define replay,
  retention, cursor, identity, and no-side-effect requirements.
- `host-runtime-capability-manifest`: expose history and after-cursor replay
  capability separately for each source/form with evidence and unsupported
  reasons.
- `native-execution-event-adapter`: retain the real-time projection-only
  boundary and reject callback identities as journal identities.
- `diagnostics-replay-tooling`: add reference-only source-history fixtures and
  replay classifications without constructing a source owner or executing a
  runtime action.

## Impact

- **Public API:** source-neutral history reader/DTOs and typed errors gain
  source-specific, durable capability facts; existing callers remain additive
  and must handle unsupported/unavailable outcomes.
- **Owners:** `core/types` owns bounded neutral types and validation;
  runner/child/team/workflow/mailbox owners own journal append and source
  ordering; `orchestration/planbridge` consumes manifest facts only;
  `host/eventadapter` remains projection-only; diagnostics replay remains
  reference-only.
- **Storage scope:** the default contract is same-host single-writer with
  persisted state reopenable by a later process where the owner/store supports
  it. Multi-writer CAS, cross-machine coordination, distributed journals, and
  multi-instance claim/transaction semantics are out of scope and must be
  marked unsupported until separately proven.
- **Out of scope:** checkpoint/restore, pause continuation, graph restore,
  planbridge resumed execution, retry/recovery automation, business commit,
  cross-source global ordering, new scheduler/mailbox owner, or remote event
  service.
- **Compatibility:** Run/Stream, callback delivery, mailbox lifecycle and
  business effects remain unchanged. A replay call is observational only.

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`docs/development-roadmap.md`
- 维护 owner：roadmap maintainers；实施阶段由 runtime-core、source-owner、mailbox、host-integration、observability、documentation 与 examples maintainers 同步其余公开文档
- 验证证据：OpenSpec strict validation、documentation/example-impact 与 roadmap consistency gates、source-history contract/replay gates、`go test ./...`、`go test -race ./...`、`golangci-lint run --config .golangci.yml`

## Example Impact Assessment

修改示例

- 影响路径：`examples/native-execution-events/README.md` 及其离线示例。
- 理由：示例需要展示多 source capability matrix、确认 cursor 后重启读取、
  typed unsupported/foreign/expired/gap/unavailable 结果，以及 replay 不触发
  执行和业务写入；不能继续只展示 mailbox callback 投影。

## Documentation Impact Assessment

| Area | Outcome | Affected paths | Owner | Verification |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/development-roadmap.md` | roadmap maintainers | roadmap consistency gate |
| components | 修改文档 | `docs/development-roadmap.md` | roadmap maintainers | roadmap consistency gate |
| configuration | 修改文档 | `docs/development-roadmap.md` | roadmap maintainers | roadmap consistency gate |
| contract/API | 修改文档 | `docs/development-roadmap.md` | roadmap maintainers | roadmap consistency gate |
| diagnostics | 修改文档 | `docs/development-roadmap.md` | roadmap maintainers | roadmap consistency gate |
| examples | 修改文档 | `docs/development-roadmap.md` | roadmap maintainers | roadmap consistency gate |
| CLI/integration | 修改文档 | `docs/development-roadmap.md` | roadmap maintainers | roadmap consistency gate |
| best practices | 修改文档 | `docs/development-roadmap.md` | roadmap maintainers | roadmap consistency gate |
| roadmap | 修改文档 | `docs/development-roadmap.md` | roadmap maintainers | roadmap consistency gate |
