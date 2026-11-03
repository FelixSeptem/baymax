## Context

See proposal.md. RunRecord carries recovery conflicts, gate counts and capability
decisions, but carries no business-effect ledger or source cursor truth. Existing
single-writer, cardinality and unified-query specs remain unchanged.

## Goals / Non-Goals

Provide a deterministic bounded read model with finite next actions and ownership.
Do not add execution interfaces, callbacks, diagnosis writes or polling.

## Decisions

Choose a pure function over an injected page, rather than a Store method that
would couple source/effect stores to diagnostics or a background service that
would create another lifetime. Supplement missing facts with reference-only
owner evidence; deriving effect status from terminal success would be incorrect.
Use exactly the mappings and bounds in docs/guides/attention-diagnostics.md.
Reject all duplicate Runs IDs, even identical entries: caller selects the snapshot
instead of projection guessing a latest record. Supplemental duplicate items
deduplicate. Validate all input before projecting. Output strings are new value
fields, not references to caller slices or maps. Stable tuple order precedes limit.

Replay uses the same pure function and compares the full expected result, including
owner/action/count/truncation. Strict JSON rejects unknown fields/trailing input
and duplicate, case-alias or non-ASCII object keys before typed decoding;
fixture size is bounded to 1MiB. Dedicated tests mutate owner, action and evidence
to prove drift. Real RuntimeRecorder -> QueryRuns -> projection integration proves
that projection does not modify the store. Real Composer admission tests pin
blocked capability outcomes (including unknown catalog models) and credential-only
negative cases in both Run and Stream. Unit tests cover both Run/Stream-shaped
records and all limits. Offline example uses only synthetic data.

## Risks / Trade-offs

Snapshot staleness -> callers refresh owners before acting; projection executes
nothing and claims no authority. Safe ASCII references still can contain secrets
-> callers must supply opaque identifiers only; raw reason/detail fields never
leave projection. Partial query page -> Total is explicitly snapshot-local.
Unknown classifications -> no inference, documented unsupported scope.

## Migration Plan

Add library API, tests, replay and example; wire both quality gates, validate all
public docs before implementation tasks are checked. Archive with the sequence
script, then commit/merge/push/verify remote/clean. Rollback removes caller use.

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
