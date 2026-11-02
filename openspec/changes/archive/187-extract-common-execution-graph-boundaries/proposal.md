## Why / Why now

Workflow, team, and child execution each describe topology differently: workflow owns a dependency DAG, team strategy implies serial or independent member ordering, and composer dispatches one child at a time. Callers and conformance tests therefore have no source-neutral way to compare equivalent topology without treating owner-specific plan/checkpoint digests as interchangeable. Add a bounded topology projection contract now so equivalent graphs can be compared while execution and recovery remain owned by their existing forms.

## What Changes

- Add a source-neutral execution graph definition, deterministic normalization, bounded structural validation, stable topological ordering, and topology digest.
- Add owner-local projections for workflow definitions, team plans, and individual composer child dispatches; preserve form-specific capabilities and reject unsupported topology rather than inferring it.
- Keep owner plan digests, checkpoint digests, execution state, restore claims, terminal arbitration, and Run/Stream ownership unchanged. A topology digest is comparison evidence, not authorization or restore binding.
- Add cross-form conformance coverage for equivalent single-node, independent-node, and serial topology, plus negative drift and unsupported-projection cases.
- Publish a focused example and contract documentation; do not add a generic executor, scheduler, checkpoint store, or shared runtime state machine.

## Capabilities

### New Capabilities
- `execution-graph-contract`: bounded canonical topology representation and owner-local projections across workflow, team, and child dispatch forms.

### Modified Capabilities

## Impact

- New package: `orchestration/graph`.
- Owner adapters: `orchestration/workflow`, `orchestration/teams`, and `orchestration/composer`.
- Verification: cross-form contract tests and dedicated shell/PowerShell gate wired into standard quality checks.
- Documentation/example: `docs/guides/execution-graph.md`, module-boundary and contract-test index updates, roadmap status update, and `examples/execution-graph`.
- No configuration, mailbox behavior, native event history, source checkpoint format, or persisted restore identity changes.

## Example Impact Assessment

新增示例

增加 `examples/execution-graph`，展示三个 owner 对等价 topology 的只读投影与 digest 对比，并明确 topology digest 不具备授权或 checkpoint restore 语义。

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`docs/README.md`、`docs/guides/execution-graph.md`、`docs/runtime-module-boundaries.md`、`docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`docs/development-roadmap.md`、`examples/execution-graph/README.md`
- 维护 owner：`orchestration/graph` maintainers、runtime contract maintainers、documentation maintainers
- 验证证据：`pwsh -File scripts/check-execution-graph-contract.ps1`、`pwsh -File scripts/check-docs-consistency.ps1`、`openspec validate --all`

## Documentation Impact Assessment

| area | outcome | affected paths | owner | evidence |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | runtime contract maintainers | source-owner boundary review and docs consistency gate |
| components | 修改文档 | `docs/README.md`、`docs/guides/execution-graph.md` | orchestration and documentation maintainers | owner projection contract tests and documentation link validation |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime contract maintainers | explicitly documents that this contract adds no configuration keys or reload semantics |
| contract/API | 修改文档 | `docs/guides/execution-graph.md` | orchestration maintainers | graph package and cross-form contract gate |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime contract maintainers | explicitly documents that this contract adds no diagnostics or RuntimeRecorder fields |
| examples | 新增文档 | `examples/execution-graph/README.md` | examples maintainers | offline example marker test |
| CLI/integration | 无需文档变更（附理由） | — | — | no CLI, wire command, or transport integration changes |
| best practices | 修改文档 | `README.md` | documentation maintainers | active milestone snapshot and public capability boundary |
| roadmap | 修改文档 | `docs/development-roadmap.md` | documentation maintainers | roadmap status consistency gate |
