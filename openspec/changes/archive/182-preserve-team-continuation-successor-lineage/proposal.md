## Why

Team continuation envelopes declare predecessor/successor lineage, but the current owner always creates generation 1 on pause and commits only a resume result on the predecessor claim. A second pause therefore produces no source-owned link from the original checkpoint to the successor, so a fresh process cannot validate the complete team continuation chain. Close this gap before treating team graph restore material as complete.

## What Changes

- Preserve the original team checkpoint generation and predecessor identity when a resumed team pauses again.
- Commit the successor checkpoint reference and resume projection atomically in the source-owned continuation store.
- Validate successor identity, digest, graph/team/run/session/authority binding and generation before successor claim; exact duplicate resumes return the retained projection and conflicting lineage fails closed.
- Add fresh-store-reopen tests for multi-pause team continuation, Run/Stream parity, duplicate projection and malformed/foreign/conflicting successor cases.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `composite-continuation-restore`: require source-owned composite lineage to be persisted, validated, and consumed across successor checkpoints.
- `teams-collaboration-runtime`: require repeated team pause/restore to preserve checkpoint generation and predecessor/successor lineage without re-running settled members.

## Impact

- Code: `orchestration/teams/continuation.go`, team resume/pause flow in `orchestration/teams/engine.go`, and team continuation tests.
- Public source binding/outcome DTOs remain additive only if a successor reference is needed in the retained resume projection; no new scheduler, journal, checkpoint owner, or execution loop is introduced.
- Documentation: `docs/development-roadmap.md` and `orchestration/teams/README.md`; owner is the orchestration/teams maintainers. Verify with `pwsh -File scripts/check-team-mailbox-continuation-contract.ps1` and `pwsh -File scripts/check-docs-consistency.ps1`.
- Rollback: keep lineage validation and successor publication fail-closed; reverting the change restores the prior behavior, while checkpoints produced by the new version remain versioned and must not be silently treated as unrelated generation-1 roots.

## Documentation Impact Assessment

| Area | Outcome | Maintenance docs | Owner | Verification evidence |
| --- | --- | --- | --- | --- |
| architecture | 无需文档变更（附理由） | — | orchestration/teams maintainers | Existing owner boundary remains unchanged |
| components | 修改文档 | `orchestration/teams/README.md` | orchestration/teams maintainers | Team continuation contract gate |
| configuration | 修改文档 | `orchestration/teams/README.md` | orchestration/teams maintainers | README records that no configuration key or default changes |
| contract/API | 修改文档 | `orchestration/teams/README.md` | orchestration/teams maintainers | Successor lineage and duplicate projection tests |
| diagnostics | 修改文档 | `orchestration/teams/README.md` | diagnostics owner | README records that no diagnostic schema changes |
| examples | 修改文档 | `orchestration/teams/README.md` | examples owner | README records that example runtime paths and markers are unchanged |
| CLI/integration | 无需文档变更（附理由） | — | integration owner | No CLI or integration entry point changes |
| best practices | 无需文档变更（附理由） | — | orchestration/teams maintainers | README records recovery boundary |
| roadmap | 修改文档 | `docs/development-roadmap.md` | runtime architecture | Roadmap status consistency gate |

## Example Impact Assessment

- 无需示例变更（附理由）

This source-owned checkpoint-lineage correction does not change an example runtime path, command, or expected marker. Owner tests and the dedicated contract gate cover the new behavior.

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`docs/development-roadmap.md`, `orchestration/teams/README.md`
- 维护 owner：orchestration/teams maintainers
- 验证证据：`pwsh -File scripts/check-team-mailbox-continuation-contract.ps1`, `pwsh -File scripts/check-openspec-documentation-impact.ps1`, `pwsh -File scripts/check-docs-consistency.ps1`
