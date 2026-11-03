## Why

Hosts currently inspect hundreds of RunRecord fields and still cannot distinguish
actionable recovery/gate failures from healthy or automatically retrying work.
The smallest reproduction is a mixed diagnostics page plus an explicitly pending
host effect: terminal success alone cannot answer whether that effect committed.

## What Changes

- Add a pure bounded attention projection over an injected diagnostics page.
- Use finite codes, next actions and owners; accept explicit reference-only
  source/effect/authorization evidence where RunRecord has no authoritative fact.
- Reject ambiguous/oversized input atomically, sort/deduplicate deterministically,
  and expose truncation without altering QueryRuns pagination.
- Add recorder integration, offline example, replay drift fixtures and both gates.

## Capabilities

### New Capabilities

- `attention-diagnostics-projection`: read-only actionable diagnostics projection.

### Modified Capabilities

None. Existing diagnostics writer, query and execution semantics remain owned by
their existing contracts.

## Impact

Owner: runtime/diagnostics maintainers. Initial deployment is a library function,
not a panel, notification service or control plane. No new writer, runtime config
key, background work, retry, resume, tool or effect execution is introduced.
Rollback removes the caller's projection call; original diagnostics and owner
stores are untouched. Input is a caller snapshot, not proof of current authority.

## Example Impact Assessment

新增示例

新增离线 `examples/attention-diagnostics`，用合成引用展示 mixed page、
healthy/retrying 无行动项、显式 effect/source evidence 和截断，不调用外部系统。

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`docs/guides/attention-diagnostics.md`、`runtime/diagnostics/README.md`、`docs/runtime-config-diagnostics.md`、`docs/runtime-module-boundaries.md`、`docs/mainline-contract-test-index.md`、`docs/development-roadmap.md`
- 维护 owner：runtime/diagnostics maintainers
- 验证证据：`pwsh -File scripts/check-attention-diagnostics-contract.ps1`、`pwsh -File scripts/check-docs-consistency.ps1`、`openspec validate --all`

## Documentation Impact Assessment

| area | outcome | affected paths | owner | evidence |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | diagnostics maintainers | pure read model dependency boundary |
| components | 修改文档 | `runtime/diagnostics/README.md` | diagnostics maintainers | projection unit tests |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | diagnostics maintainers | API defaults and limits; no new config |
| contract/API | 修改文档 | `docs/guides/attention-diagnostics.md` | diagnostics maintainers | finite mappings and negative tests |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md` | diagnostics maintainers | no writer/query schema change |
| examples | 修改文档 | `docs/guides/attention-diagnostics.md` | examples maintainers | synthetic offline example |
| CLI/integration | 修改文档 | `README.md` | diagnostics maintainers | library integration entry |
| best practices | 修改文档 | `docs/guides/attention-diagnostics.md` | diagnostics maintainers | no terminal-to-effect inference |
| roadmap | 修改文档 | `docs/development-roadmap.md` | documentation maintainers | roadmap consistency gate |
