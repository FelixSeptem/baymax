## Why

Team parallel/vote continuation already has an explicit owner store and fresh-process restore path, but the persisted graph digest does not bind every aggregation decision input. In particular, a resumed vote may present a changed tie-break policy without changing the current team digest, so the source cannot prove that restored aggregate semantics match the paused run. This roadmap item closes that public contract gap before team continuation is treated as a deployable capability.

## What Changes

- Bind normalized team execution and aggregation policy inputs that affect resumed member dispatch or terminal aggregation into the team graph digest.
- Keep vote provenance, participant identity, and tie-break behavior stable across fresh-process restore and repeated pause generations.
- Advertise team pause/restore capability only when the configured team owner has matching checkpoint, cross-process restore, pause-reason, and deployment evidence; default and process-local owners remain unsupported for cross-process restore.
- Add focused positive/negative tests and a dedicated contract gate for policy drift, vote restore, duplicate claim, ambiguous in-flight state, and Run/Stream parity.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `teams-collaboration-runtime`: require restored team aggregation inputs and vote tie-break provenance to remain bound to the original continuation.
- `composite-continuation-restore`: require team graph identity to cover all semantics that influence resumed dispatch and aggregate resolution.
- `host-runtime-capability-manifest`: require team cross-process and pause cells to be promoted only from an explicitly configured team owner with compatible deployment evidence.

## Impact

- Code/API: `orchestration/teams` team digest, continuation capability projection, and existing team restore path; planbridge team manifest integration remains owner-specific.
- Tests/gates: team continuation and planbridge capability tests; dedicated team continuation contract gate.
- Documentation: `README.md`, `docs/development-roadmap.md`, `docs/guides/integration.md`, `docs/runtime-config-diagnostics.md`, `docs/mainline-contract-test-index.md`, and `orchestration/teams/README.md`.
- No new dependency, external store, mailbox ownership, distributed coordination, or resumed planbridge capability.

## Example Impact Assessment

修改示例

理由：示例支持矩阵需要明确 team continuation 的策略摘要绑定、File owner 部署条件和 policy-drift 拒绝边界；不改变示例的网络、Provider 或业务副作用。

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`docs/development-roadmap.md`、`docs/guides/integration.md`、`docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`orchestration/teams/README.md`、`examples/host-execution-plan/README.md`
- 维护 owner：Runtime contracts owner、team owner maintainer、examples/documentation maintainer
- 验证证据：`go test ./orchestration/teams ./orchestration/planbridge ./core/types`、team continuation contract gate、`openspec validate --all`、`pwsh -File scripts/check-docs-consistency.ps1`

## Documentation Impact Assessment

| area | outcome | affected paths | owner | evidence |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/guides/integration.md` | runtime contracts owner | owner-binding and fail-closed restore tests |
| components | 修改文档 | `orchestration/teams/README.md` | team owner maintainer | team continuation tests |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime contracts owner | deployment capability tests |
| contract/API | 修改文档 | `docs/guides/integration.md` | runtime contracts owner | OpenSpec validation and planbridge tests |
| diagnostics | 修改文档 | `docs/mainline-contract-test-index.md` | diagnostics/replay maintainer | dedicated gate mapping |
| examples | 修改文档 | `examples/host-execution-plan/README.md` | examples/documentation maintainer | team policy-binding and owner-bound rejection matrix |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | runtime contracts owner | pre-activation rejection tests |
| best practices | 修改文档 | `README.md` | documentation maintainer | docs consistency gate |
| roadmap | 修改文档 | `docs/development-roadmap.md` | documentation maintainer | roadmap status consistency gate |
