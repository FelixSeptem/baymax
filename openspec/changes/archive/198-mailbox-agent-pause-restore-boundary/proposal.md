## Why

Mailbox already has an opt-in handler continuation implementation, but its public capability declaration and regression evidence are not complete enough to distinguish a recoverable Agent pause from ordinary mailbox lifecycle replay. Without a completed contract, hosts must keep handler pause cells unsupported even though the owner-native path can preserve a delivery and resume it after a sequential process reopen.

This change makes the bounded capability explicit and verifiable. It does not turn the default mailbox worker, mailbox redelivery, or planbridge into an Agent continuation owner.

## What Changes

- Promote the explicit `NewContinuationWorker` + durable `FileStore` + registered resumer/revalidator path to a public, deployment-scoped handler continuation capability.
- Require independent evidence for `input_required`, `approval_required`, and `budget_exhausted`, including fresh-process restore, bounded checkpoint identity, lease/revalidation checks, terminal projection, and completed-effect non-replay.
- Preserve continuation lineage when a resumer reaches another safe pause; successor checkpoints remain source-owned and duplicate resumes converge.
- Keep `NewWorker` lifecycle-only, keep mailbox lifecycle replay separate from Agent restore, and keep planbridge as a thin delegate with resumed plan execution unsupported.
- Add a dedicated replay/contract gate and an offline continuation example marker set.

## Example Impact Assessment

修改示例

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`, `docs/development-roadmap.md`, `docs/mainline-contract-test-index.md`, `docs/guides/integration.md`, `docs/runtime-module-boundaries.md`, `docs/runtime-config-diagnostics.md`, `orchestration/mailbox/README.md`, `orchestration/planbridge/README.md`, `examples/host-execution-plan/README.md`
- 维护 owner：orchestration/runtime maintainers
- 验证证据：`pwsh -File scripts/check-mailbox-agent-pause-restore-contract.ps1`, `pwsh -File scripts/check-docs-consistency.ps1`, `pwsh -File scripts/check-openspec-documentation-impact.ps1`, `pwsh -File scripts/check-openspec-example-impact-declaration.ps1`, `go test ./orchestration/mailbox ./orchestration/planbridge ./tool/diagnosticsreplay -count=1`

## Documentation Impact Assessment

| 文档域 | 变更 | 路径 | Owner | 证据 |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | orchestration/runtime maintainers | owner and deployment boundary |
| components | 修改文档 | `orchestration/mailbox/README.md`, `orchestration/planbridge/README.md` | orchestration maintainers | package contract and delegation boundary |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime maintainers | states that no runtime keys/defaults changed |
| contract/API | 修改文档 | `docs/guides/integration.md`, `docs/mainline-contract-test-index.md` | orchestration/runtime maintainers | public matrix and test index |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime maintainers | states that no diagnostics or QueryRuns fields changed |
| examples | 修改文档 | `examples/host-execution-plan/README.md` | examples maintainers | offline marker and support matrix |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | integration maintainers | host integration boundary |
| best practices | 修改文档 | `README.md` | runtime maintainers | capability summary |
| roadmap | 修改文档 | `docs/development-roadmap.md` | roadmap owner | active status and scope |

## Capabilities

### New Capabilities

None. The capability already has a public owner package; this change completes its evidence and declaration rather than introducing a second API.

### Modified Capabilities

- `multi-agent-mailbox-contract`: formalize configured handler continuation, successor pause lineage, fresh-process restore, and lifecycle-owned terminal effects.
- `host-runtime-capability-manifest`: promote mailbox handler pause cells only from complete owner evidence and preserve the explicit deployment limitation.

## Impact

- Affected code: `orchestration/mailbox`, `orchestration/planbridge` capability admission, and related contract tests.
- Affected docs/examples: the public support matrix, roadmap item I, integration/module-boundary notes, and the offline host execution plan example.
- No runtime configuration keys, QueryRuns fields, provider dependencies, or cross-project transaction semantics are added.
- Cross-machine, multi-writer, multi-instance, and mailbox-owned planbridge resume remain unsupported.
