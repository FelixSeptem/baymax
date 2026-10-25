## Context

See `proposal.md` for motivation. The current team engine executes serial/parallel/vote plans directly from in-memory task records. It has no pause boundary, graph store, claim or new-process restore. The mailbox already owns message identity, idempotent publish, delivery/lease IDs, attempt causation, ack/nack/requeue/retry/DLQ and retained lifecycle history, but host-plan binding/outcome is an admission/lifecycle projection rather than executable continuation state.

Existing direct, child and workflow implementations establish the required vocabulary: source-owned envelope, persisted pause boundary, atomic idempotent claim, successor lineage, typed unavailable/conflict/recovery-required, source deployment scope, and capability evidence. The new implementation must reuse those semantics without treating another form's store, history or DTO as proof of support.

## Goals / Non-Goals

**Goals:**

- Implement source-owned continuation for team parallel/vote and mailbox host-plan forms under explicitly configured owner stores.
- Preserve original logical run, parent run, session, causation, authority, plan, write scope, effect namespace and occurrence identities.
- Preserve completed member/effect results, pending work, aggregate/vote state, budget consumption, original deadline, mailbox delivery/lease/attempt state and terminal arbitration material.
- Restore only pending source work after an atomic claim; equivalent resumes converge and conflicting resumes fail closed.
- Publish form/reason-specific evidence through the existing capability manifest and reject unsupported paths before activation.

**Non-Goals:**

- No second mailbox owner, queue, worker, scheduler, event journal or host-side resume loop.
- No conversion of mailbox lifecycle history, callback replay, scheduler snapshot, host projection or identical run ID into execution state.
- No resumed planbridge execution; it remains an independent capability cell until it owns its own source graph material.
- No multi-writer, cross-machine, distributed claim or business database transaction guarantee.
- No automatic retry, re-enqueue, re-consume or parent re-Run after an ambiguous source state.

## Decisions

### 1. Share continuation admission semantics, keep source state owner-specific

Team and mailbox will use a common bounded identity/claim contract where it avoids drift: contract/schema version, ref/digest, original identity binding, pause reason, expiry, deployment scope, claim identity, duplicate projection, successor lineage and typed failures. Their persisted material remains owner-specific.

- Team envelope carries normalized plan digest/version, strategy, member records, runnable/pending sets, completed effects, vote/fanout aggregate state, pending action, budget/deadline and terminal arbitration.
- Mailbox envelope carries normalized host-plan binding, original record/message ID, delivery/lease/attempt identity, lifecycle state, pending action, effect references, budget/deadline and lifecycle/terminal arbitration references.

This avoids a generic graph abstraction that would hide mailbox lease semantics or team aggregate semantics. Fully separate stores were considered but rejected because conflict, duplicate and recovery-required behavior would drift.

### 2. Pause only at source-owned safe boundaries

Team pauses occur before dispatching an unstarted member/effect. A parallel/vote plan may persist only after owner state identifies finished members and a deterministic pending set. If an in-flight member cannot provide safe source material, the owner returns typed `recovery-required` or remains unsupported rather than claiming a pause.

Mailbox pauses occur while the original record is held by its existing delivery/lease owner and before the pending effect/action is executed. Resume validates the persisted record against current mailbox state, consumer/lease ownership, attempt causation and expiry before starting work.

### 3. Atomic owner claims and terminal commits decide recovery

Every resume validates owner namespace, form/graph/message binding, schema, digest, expiry, deployment scope, authority/plan/write-scope/effect namespace, occurrence/causation, predecessor/successor and source budget facts. The owner atomically persists claim state before invoking continuation work. Exact duplicates receive the retained claim/result; changed input fails closed. A process restart with a claimed/running record but no durable result returns `recovery-required` and does not restart automatically.

Team terminal results are committed through the team owner with aggregate evidence. Mailbox terminal results are projected from the original mailbox lifecycle owner: ack/nack/requeue/retry/DLQ/lease loss are source facts, not host business commits.

### 4. Capability promotion is evidence-gated and form-specific

`CapabilityManifestWithTeamContinuation` and `CapabilityManifestWithMailboxContinuation` promote only the relevant form after source capability validation. Each of checkpoint creation, cross-process restore, `input_required`, `approval_required`, `budget_exhausted`, causal retry and mailbox continuation has distinct owner evidence. Default manifests and unrelated forms retain explicit unsupported reasons.

Planbridge validates these cells before dispatch. A supported mailbox continuation still delegates to the existing mailbox owner; planbridge never publishes, consumes or resumes directly.

### 5. File stores remain same-host single-writer sequential reopen

Owner file stores persist by atomic replacement and acquire the existing writer-scope contract. A new process may reopen a compatible source sequentially, but concurrent/multi-machine requests fail with deployment unsupported. This change uses the existing deployment vocabulary and does not prematurely promise distributed recovery.

## Risks / Trade-offs

- [Parallel task finishes while pause material is being sealed] → pause boundary captures only settled task records; ambiguous inflight work produces `recovery-required`, not a resumeable checkpoint.
- [Mailbox lease expires after save but before resume] → validate record state, delivery/lease/attempt causation at claim/start; surface lease/recovery evidence and do not re-consume or re-enqueue.
- [Owner file commit fails after side effect/terminal transition] → preserve attempt identity and return typed unavailable/recovery-required; no synthetic terminal or retry.
- [Aggregation hides a member conflict] → preserve per-member identity and return claim/terminal conflicts before aggregate execution starts.
- [Manifest claims support from package presence] → promotion APIs require exact source namespace, capability version, deployment scope and fixture evidence; default remains unsupported.

## Migration Plan

1. Add DTOs, stores and capability sources behind explicit owner options; existing constructors retain current terminal-only behavior.
2. Add team safe points and fresh-process restore for serially reproducible parallel/vote states; preserve old Run/Stream behavior when no owner store is configured.
3. Add mailbox host-plan continuation binding to existing mailbox owner; preserve existing Publish/Consume/Ack/Nack/Requeue APIs and history semantics.
4. Gate planbridge admission on exact capabilities; update examples/docs to distinguish configured support from default rejection.
5. Roll back by removing owner continuation configuration or disabling individual manifest cells. Persisted unknown/invalid checkpoints fail closed rather than running a fallback execution.

## Open Questions

None. The supported deployment profile is explicitly same-host single-writer sequential reopen; distributed ownership is deferred by roadmap scope.

## Example Impact Assessment

修改示例

理由：existing examples will add configured continuation/recovery markers and explicit unsupported fallback notes without changing their business payload model.

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`, `docs/development-roadmap.md`, `docs/guides/integration.md`, `docs/runtime-module-boundaries.md`, `docs/runtime-config-diagnostics.md`, `docs/mainline-contract-test-index.md`, `orchestration/teams/README.md`, `orchestration/mailbox/README.md`, `orchestration/planbridge/README.md`, `examples/host-execution-plan/README.md`, `examples/native-execution-events/README.md`
- 维护 owner：Runtime contracts owner、team owner 与 mailbox owner maintainers
- 验证证据：owner unit/integration tests、fresh-process fixtures、Run/Stream parity、dedicated PowerShell/shell contract gates、`go test -race ./...`、`golangci-lint run --config .golangci.yml`、质量门禁与 OpenSpec validation

## Documentation Impact Assessment

| area | outcome | affected paths | owner | evidence |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | runtime contracts owner | owner mutation boundary review |
| components | 修改文档 | `orchestration/teams/README.md`、`orchestration/mailbox/README.md`、`orchestration/planbridge/README.md` | team/mailbox maintainers | focused owner tests |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime contracts owner | same-host deployment profile review |
| contract/API | 修改文档 | `docs/guides/integration.md` | runtime contracts owner | OpenSpec strict validation |
| diagnostics | 修改文档 | `docs/mainline-contract-test-index.md` | diagnostics/replay maintainer | dedicated gate mapping review |
| examples | 修改文档 | `examples/host-execution-plan/README.md`、`examples/native-execution-events/README.md` | examples maintainer | expected-marker review |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | runtime contracts owner | pre-activation rejection review |
| best practices | 修改文档 | `README.md` | documentation maintainer | fail-closed restore guidance review |
| roadmap | 修改文档 | `docs/development-roadmap.md` | documentation maintainer | roadmap status gate |
