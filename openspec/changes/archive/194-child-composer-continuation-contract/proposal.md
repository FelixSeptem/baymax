## Why

Composer child execution wiring exists, but `subagent_fanout` remains unsupported because the public contract does not yet prove durable child source position and fresh-process recovery for each pause reason. Hosts need a form-local declaration that distinguishes executable children from children that can safely continue after a process boundary.

## What Changes

- Define child/composer continuation as a source-owned, bounded checkpoint contract carrying parent/child identity, assignment, consumed budget, effect boundary, pause reason, and stable native source position.
- Require a fresh Composer/store reopen path and evidence for `input_required`, `approval_required`, and `budget_exhausted` independently.
- Define bounded child source cursor replay, retention floor, and typed expired/gap outcomes; missing or ambiguous source facts fail closed.
- Keep capability promotion evidence-based. Incomplete cells remain `unsupported_capability`; planner fallback to direct execution is forbidden.
- Verify restored work does not re-run completed child/tool effects and preserve native task/attempt lineage.

## Capabilities

### New Capabilities
- `child-composer-continuation`: Public source-owned child checkpoint, cursor, fresh-process restore, and pause-cell capability contract.

### Modified Capabilities
- `composite-continuation-restore`: Clarify child native source-position retention and fresh-process evidence without changing team, mailbox, or workflow support.
- `host-runtime-capability-manifest`: Require child pause-cell promotion to reference independently verified fresh-process evidence and preserve unsupported cells otherwise.

## Impact

- Affected code: `orchestration/composer`, `orchestration/planbridge`, and shared native history/capability types only where child ownership requires an additive contract.
- Affected verification: Composer reopen/restore tests, capability manifest tests, example contract, and dedicated offline contract gate.
- Deployment remains same-host/single-writer where the configured file owner only proves that profile; this change does not claim multi-writer or cross-machine recovery.

## Example Impact Assessment

修改示例

## Documentation Impact Assessment

| Area | Outcome | Document | Owner | Evidence |
| --- | --- | --- | --- | --- |
| architecture | 无需文档变更（附理由） | — | — | 不引入新的跨模块架构边界 |
| components | 修改文档 | `orchestration/composer/README.md` | Composer continuation owner | source cursor binding 与恢复前检查 |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime config documentation owner | 明确本提案不新增配置键或默认值 |
| contract/API | 修改文档 | `orchestration/planbridge/README.md` | capability manifest owner | pause cell 能力提升条件 |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime config documentation owner | 明确 continuation 不改变诊断字段 |
| examples | 修改文档 | `examples/durable-continuation-restore/README.md` | continuation example owner | child reopen 与 cursor fail-closed markers |
| CLI/integration | 无需文档变更（附理由） | — | — | 不新增 CLI 或集成入口 |
| best practices | 无需文档变更（附理由） | — | — | 无独立实践语义变化 |
| roadmap | 修改文档 | `docs/development-roadmap.md` | roadmap owner | F active 与部署边界 |
| public overview | 修改文档 | `README.md` | repository overview owner | 展示当前 active change 与保守能力状态 |

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`orchestration/composer/README.md`, `orchestration/planbridge/README.md`, `examples/durable-continuation-restore/README.md`, `docs/development-roadmap.md`, `docs/runtime-config-diagnostics.md`, `README.md`
- 维护 owner：Composer continuation owner; capability manifest owner; continuation example owner; roadmap owner; runtime config documentation owner; repository overview owner
- 验证证据：`go test ./orchestration/composer`; `go test ./orchestration/planbridge`; `go test ./examples/durable-continuation-restore`; `pwsh -File scripts/check-openspec-roadmap-status-consistency.ps1`; `pwsh -File scripts/check-docs-consistency.ps1`
