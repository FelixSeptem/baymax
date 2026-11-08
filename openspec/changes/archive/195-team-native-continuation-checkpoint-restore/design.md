## Context

See `proposal.md` for the motivation. The team package already owns a bounded continuation envelope, an atomic claim path, file-store reopen, repeated-pause lineage, and a capability source. The remaining contract risk is that `teamPlanDigest` currently covers only a subset of normalized plan semantics, while `TeamContinuation` does not retain the normalized aggregation policy needed to explain a vote winner after a fresh-process restore.

## Goals / Non-Goals

**Goals:**

- Make the sealed team graph identity cover every normalized input that can change member selection, failure handling, or vote aggregation.
- Preserve vote tie-break, participant boundary, and terminal aggregate provenance across pause, duplicate resume, successor publication, Run, and Stream.
- Make capability evidence reference concrete team policy-binding, ambiguous-in-flight, duplicate-claim, and transport-parity tests.
- Keep the supported deployment profile explicit: same-host, single-writer, sequential FileStore reopen.

**Non-Goals:**

- No multi-writer or cross-machine coordination, new database, mailbox owner, scheduler, or resumed planbridge execution.
- No redesign of the existing team task runner or aggregate algorithm.
- No automatic recovery of an ambiguous in-flight member; such state remains `recovery-required` or unsupported.
- No promotion of team capability from child, workflow, mailbox, history, or package presence.

## Decisions

### 1. Digest normalized aggregation policy, not executable callbacks

Extend the canonical team plan identity with stable scalar policy facts: strategy, vote tie-break, failure policy, backpressure mode, worker bound, timeout terminal status, and ordered member identity/target/role/priority facts. Runner callbacks and payloads remain host-owned and are never serialized. This binds all deterministic dispatch/aggregate decisions without attempting to hash opaque executable state.

Alternative rejected: persisting the full `Plan` or callback identity. That would leak executable/payload state and would not be portable across processes.

### 2. Persist explicit aggregate provenance in the continuation envelope

Add bounded aggregate metadata to `TeamContinuation`: normalized tie-break, participant IDs, and a deterministic aggregate digest derived from settled member statuses/votes and policy. Validation and successor checks require the metadata to remain unchanged for the logical generation. The existing `TeamTaskContinuation` remains the source for per-member vote/effect facts.

Alternative rejected: recomputing provenance only from the restored `Plan`. That permits policy drift and loses evidence about which members participated before the pause.

### 3. Verify policy at the existing claim boundary

On resume, normalize the supplied plan and compare its canonical digest plus aggregate metadata with the loaded continuation before `ClaimResume`. A mismatch returns the existing typed conflict/unsupported family and cannot activate a member. The current owner claim/commit atomicity and successor lineage remain the source of truth.

### 4. Keep capability promotion evidence explicit

The team capability source remains valid only for an explicitly configured owner whose store reports a supported deployment. Its evidence references point to tests covering policy binding, vote provenance, fresh-process restore, duplicate claim, ambiguous in-flight rejection, and Run/Stream parity. Memory stores continue to report process-local and never promote cross-process support.

## Risks / Trade-offs

- [Existing persisted checkpoints lack the new policy metadata] → treat them as corrupt/unsupported rather than guessing defaults or rerunning a team.
- [A future policy field is omitted from the digest] → add a focused policy-drift regression and require capability evidence to name the canonical digest test before promotion.
- [Vote output differs because host callbacks are not portable] → only settled vote/effect digests and normalized policy are persisted; pending callbacks remain host-owned and are rerun only when their source state is unambiguous.
- [A pause races with a parallel member] → retain the existing safe boundary and fail closed when the owner cannot establish settled versus in-flight state.

## Migration Plan

1. Add the policy/provenance fields and canonical digest updates with backward-incompatible validation for unsealed or stale envelopes; no existing constructor changes are required.
2. Add focused team and manifest tests, then update the dedicated team continuation gate and public documentation.
3. Roll back by disabling team continuation owner configuration; terminal-only team execution remains unchanged. Unknown old checkpoints are rejected safely.

## Open Questions

None. The roadmap explicitly defers deployable multi-writer source history to H.

## Example Impact Assessment

修改示例

理由：示例支持矩阵需要记录 team policy digest/provenance 与显式 File owner 的恢复条件；运行入口和 expected markers otherwise remain unchanged.

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`docs/development-roadmap.md`、`docs/guides/integration.md`、`docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`orchestration/teams/README.md`、`examples/host-execution-plan/README.md`
- 维护 owner：Runtime contracts owner、team owner maintainer、examples/documentation maintainer
- 验证证据：`go test ./orchestration/teams ./orchestration/planbridge ./core/types`、team continuation contract gate、`openspec validate --all`、`pwsh -File scripts/check-docs-consistency.ps1`

## Documentation Impact Assessment

| area | outcome | affected paths | owner | evidence |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/guides/integration.md` | runtime contracts owner | digest and restore boundary tests |
| components | 修改文档 | `orchestration/teams/README.md` | team owner maintainer | focused continuation suite |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime contracts owner | deployment scope capability test |
| contract/API | 修改文档 | `docs/guides/integration.md` | runtime contracts owner | OpenSpec and planbridge validation |
| diagnostics | 修改文档 | `docs/mainline-contract-test-index.md` | diagnostics/replay maintainer | gate mapping |
| examples | 修改文档 | `examples/host-execution-plan/README.md` | examples/documentation maintainer | team policy-binding and owner-bound rejection matrix |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | runtime contracts owner | pre-activation rejection tests |
| best practices | 修改文档 | `README.md` | documentation maintainer | docs consistency gate |
| roadmap | 修改文档 | `docs/development-roadmap.md` | documentation maintainer | roadmap status gate |
