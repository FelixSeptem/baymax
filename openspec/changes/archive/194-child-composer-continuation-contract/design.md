## Context

See proposal.md for motivation. `orchestration/composer` already owns a bounded `ChildContinuation`, memory/file stores, an atomic resume claim seam, and source-local execution. Shared `core/history` already defines opaque cursors, bounded `ReadAfter`, retention floors, and typed cursor-expired/history-gap errors. The capability manifest can promote child dimensions from explicit owner evidence, but child process-reopen evidence is not yet tied to a stable native source position.

## Goals / Non-Goals

**Goals:**
- Bind child checkpoint restore to child-owned history identity/cursor and preserve deployment scope.
- Make fresh-process reopen and each pause reason observable in public evidence.
- Keep the default manifest conservative and preserve no-fallback planner behavior.

**Non-Goals:**
- Team, mailbox, or workflow checkpoint implementation; generic cross-form checkpoint abstraction; multi-writer/cross-machine guarantees; replay-driven execution; changing parent Run/Stream terminal ownership.

## Decisions

1. Reuse `core/types.NativeEventHistoryReader` and the Composer-owned checkpoint store rather than introducing another cursor format. Child checkpoint references bind the source ref and opaque cursor; source facts remain read-only evidence and the checkpoint remains the executable restore authority.
2. Make pause evidence form-local and reason-local. A child declaration is supported only when checkpoint creation, fresh-process reopen, source cursor compatibility, terminal evidence, and effect non-replay tests are present for that reason. Missing evidence stays unsupported.
3. Keep file-backed deployment claims at the verified same-host/single-writer profile. Capability promotion copies the owner's declared deployment scope and does not infer broader coordination.
4. On cursor expiry/gap, surface typed history errors and require recovery; never seek to latest or re-run the parent/child to reconstruct missing state.

Alternatives considered: a new generic checkpoint package would couple F to the future team work; documentation-only evidence would not deliver the requested public restore contract; upgrading default capability based on package presence would be unsafe.

## Risks / Trade-offs

- [Risk] Native history cursor and checkpoint store can advance independently → validate their source identity and checkpoint-bound sequence at load/claim; any mismatch fails closed.
- [Risk] Existing child executor may not own enough native state to resume every safe point → only promote cells for which the executor and persisted envelope provide direct evidence.
- [Risk] File store offers limited deployment scope → expose that scope and keep broader profiles unsupported.

## Migration Plan

Additive API/spec changes only. Existing callers and persisted checkpoint versions remain readable; unsupported or absent cursor evidence never upgrades capability. Rollback consists of removing child capability promotion while retaining the safe checkpoint implementation. No data migration or cross-store transaction is introduced.

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
