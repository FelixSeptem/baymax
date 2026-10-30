## Context

The team owner already persists bounded task/member state and uses a source-owned
claim record for duplicate resume protection. `persistPausedResult` currently
creates every checkpoint with generation 1 and no predecessor, while the claim
commit stores only the result. Workflow has an existing successor pattern that
the team path must match. See `proposal.md` for the motivation and the delta
specs for the public contract.

## Goals / Non-Goals

**Goals:**

- Make repeated team pause/restore a source-owned checkpoint chain.
- Preserve graph, run/session, causation, authority and generation identity.
- Publish successor reference and predecessor resume projection atomically in
  the existing team continuation store.
- Keep settled members/effects out of the resumed dispatch set and preserve
  Run/Stream parity.

**Non-Goals:**

- No new scheduler, event journal, queue, graph executor or distributed store.
- No expansion of FileStore beyond same-host single-writer sequential reopen.
- No changes to mailbox, child or workflow continuation ownership.
- No application-level replay or parent re-run fallback.

## Decisions

1. **Carry lineage through the existing owner request/result path.** The resume
   request carries the predecessor reference already loaded by the owner; the
   pause result creates a successor with `Generation = predecessor.Generation +
   1` and `PredecessorCheckpointID` set to the claimed checkpoint. This keeps
   lineage source-owned and avoids a second registry.
2. **Commit successor and projection under one store lock/write.** Extend the
   existing team claim/result record with the successor reference (and its
   digest). The store validates that the successor is sealed, belongs to the
   same namespace/graph/run/session/authority identity, has the next generation,
   and names the claimed predecessor before persisting the record. FileStore
   writes the complete snapshot atomically; a failed write restores the prior
   record in memory.
3. **Load and claim validate both ends of the chain.** Loading a successor
   verifies its digest and predecessor identity. Claiming a predecessor with a
   committed successor returns the retained projection for an equivalent request;
   a conflicting successor or request returns typed conflict/recovery-required.
4. **Use the existing execution reconstruction.** Resume still rebuilds task
   records from the sealed team continuation and dispatches only pending tasks.
   No task runner or effect executor is changed beyond the lineage data needed
   to make the existing path safe across a second pause.

### Alternatives considered

- **New lineage table/journal:** rejected because it would split source ownership
  and create a second recovery fact source.
- **Encode lineage only in host projection or application snapshot:** rejected;
  those artifacts cannot authorize source restore or provide atomic claim.
- **Reset generation and treat each pause as a new root:** rejected because it
  loses causal identity and permits foreign/conflicting successor admission.

## Risks / Trade-offs

- [Risk] Existing in-flight claims may have no successor after a crash. -> Keep
  the current `in-progress`/`recovery-required` fail-closed result; never infer
  or synthesize a successor.
- [Risk] Old checkpoints have no predecessor fields. -> Treat generation-1
  checkpoints as valid roots; apply successor validation only when a successor
  is present, preserving backward compatibility.
- [Risk] FileStore remains single-writer. -> Preserve deployment scope in the
  capability manifest and reject concurrent/distributed claims as before.

## Migration Plan

1. Add red tests for repeated team pause, fresh-store successor restore,
   duplicate projection and conflicting successor identity.
2. Implement sealed successor construction and atomic store persistence.
3. Run team continuation, planbridge, replay and documentation gates.
4. Roll back by reverting the owner change; do not reinterpret new lineage
   records as unrelated roots.

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

The change is confined to source-owned continuation lineage and does not alter example runtime paths or markers.

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`docs/development-roadmap.md`, `orchestration/teams/README.md`
- 维护 owner：orchestration/teams maintainers
- 验证证据：`pwsh -File scripts/check-team-mailbox-continuation-contract.ps1`, `pwsh -File scripts/check-openspec-documentation-impact.ps1`, `pwsh -File scripts/check-docs-consistency.ps1`
