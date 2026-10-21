## Context

See `proposal.md` for the motivation. Archive 172 gives workflow its own
continuation owner and bounded FileStore implementation. Composer currently
owns child dispatch, scheduler owns task/attempt/lease transitions, and its
recovery snapshot can reopen scheduler facts but cannot atomically claim a
logical child continuation. Teams and mailbox have their own execution and
lifecycle owners, while planbridge is only a compiler/delegator. None of those
owners may borrow workflow state or create a parallel recovery loop.

## Goals / Non-Goals

**Goals:**

- Deliver the first source-owner phase for local and A2A child/fanout
  continuation using composer plus scheduler facts.
- Preserve a bounded graph envelope and original identities across a new
  Composer/source-store reopen.
- Make duplicate, conflict, lease loss, timeout, retry, cancellation, and
  terminal commit behavior explicit and fail closed.
- Add manifest, planbridge, replay, integration, and documentation boundaries
  that make child support independently observable.
- Keep team parallel/vote, mailbox host-plan continuation, and planbridge
  resumed execution explicitly unsupported until later source-owner phases.

**Non-Goals:**

- No scheduler replacement, queue worker, mailbox owner, global terminal
  reducer, distributed store, multi-writer CAS, or cross-machine fencing.
- No use of `RecoverySnapshot`, mailbox lifecycle history, callback events,
  application snapshots, or a fresh `SpawnChild` as restore material.
- No team/member graph restore, mailbox pause/lease continuation, or resumed
  plan activation in this phase.

## Decisions

### Source ownership and phase boundary

The child continuation store is injected into Composer and records only child
source facts. Scheduler remains the owner of task/attempt/lease transitions and
terminal commits; Composer owns child graph admission, local/A2A execution
boundary, and aggregate child projection. Teams, mailbox, and planbridge
continue to expose unsupported cells. This keeps a supported capability tied to
one owner with evidence rather than inferring a form from shared DTOs.

### Continuation envelope

`ChildContinuation` contains a bounded `ChildContinuationRef`, normalized child
task and attempt state, parent/run/session/causation/authority/plan/write-scope/
effect identity, pause reason and pending action reference, consumed budget and
deadline, completed effect references, terminal arbitration facts, generation,
expiry, and predecessor/successor IDs. The public projection excludes payload
bodies, credentials, callbacks, and arbitrary scheduler internals. Canonical
normalization and digesting happen before persistence; stores detach all maps,
slices, and pointers from caller memory.

### Atomic claim and restore path

`ChildContinuationStore` exposes `Save`, `Load`, `ClaimResume`,
`MarkResumeRunning`, and `CommitResumeResult`. Claim identity includes form,
mode, ref digest, parent/run/session, causation, pause reason, predecessor,
request identity, and source revision. Memory and File stores share the same
state machine; FileStore reopens the same snapshot on a new Composer but only
advertises same-host single-writer sequential reopen. A duplicate claim returns
the retained claim/result projection. A running claim returns in-progress or
recovery-required and never auto-starts a second attempt.

Restore calls a dedicated child-owner entry that consumes the claimed native
task/attempt state and resumes pending work. It does not call `SpawnChild`,
publish a new message, insert a new scheduler task, or replay an event page.
Completed child effects are terminal facts and are skipped. Lease and timeout
decisions are validated against current scheduler/source state before the claim
and again before execution starts.

### Pause producers

The first implementation may produce a pause only at a source-owned child safe
point: before pending local/A2A action dispatch (`input_required` or
`approval_required`) and at an exhausted child resource boundary
(`budget_exhausted`). Every enabled producer must persist pending action,
consumed budget, original deadline and prior outcomes. If a producer cannot
provide a real source boundary, its manifest cell stays unsupported and the
bridge rejects before task activation.

### Capability and bridge behavior

The manifest adds independent child/fanout cells and evidence references. The
workflow cells delivered by archive 172 remain unchanged. Team, mailbox, and
resumed-plan cells remain unsupported with stable limitation values. planbridge
will accept a resumed child binding only after capability and source-owner
validation; otherwise it returns `unsupported_capability` before enqueue or
direct fallback. Existing inline child bindings remain terminal-only until the
new source restore entry is selected explicitly.

### Replay and diagnostics

Fixtures contain only bounded opaque references, normalized state, expected
typed outcomes, and capability declarations. Replay is a pure validator. It
does not construct Composer/Scheduler/Mailbox, claim or restore, invoke a
handler, or write RuntimeRecorder state. Source events and host business commit
remain separate.

## Risks / Trade-offs

- [Scheduler state differs from continuation] -> Compare source generation,
  task/attempt/lease identity and current snapshot before claim and again
  before start; reject conflicts instead of re-enqueueing.
- [Duplicate child effect after crash] -> Persist completed effect references,
  require one atomic claim, and gate dispatch on terminal child state.
- [False composite capability] -> Keep manifest cells independently evidenced;
  team/mailbox/resumed-plan remain unsupported in this phase.
- [FileStore writer collision] -> Fail closed with typed unavailable; advertise
  only same-host single-writer sequential reopen.
- [Large or sensitive continuation] -> Enforce field/count/envelope limits and
  keep bodies/credentials outside the public DTO and replay fixture.

## Migration Plan

1. Add child continuation types/store and tests without changing default child
   dispatch semantics.
2. Add explicit Composer restore admission and enable only child cells backed by
   passing source-process fixtures.
3. Add planbridge binding and read-only replay evidence; retain unsupported team,
   mailbox, and resumed-plan cells.
4. Update integration and package documentation, then run all focused and
   repository gates.
5. Roll back by withdrawing child capability advertisement and calling the
   existing terminal-only child path; persisted records remain for inspection
   and are never interpreted as a new task.

## Open Questions

None. The owner boundary, supported phase, store scope, identity claim, and
unsupported form matrix are fixed by this design.

## Example Impact Assessment

修改示例

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`docs/development-roadmap.md`、`docs/guides/integration.md`、`docs/runtime-module-boundaries.md`、`docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`orchestration/composer/README.md`、`orchestration/planbridge/README.md`、`examples/durable-continuation-restore/README.md`
- 维护 owner：composer/scheduler、planbridge、diagnostics/replay、documentation maintainers
- 验证证据：`openspec validate restore-composite-continuations-and-graph-state --strict`；`pwsh -File scripts/check-openspec-documentation-impact.ps1`；`pwsh -File scripts/check-docs-consistency.ps1`

## Documentation Impact Assessment

| 领域 | 结论 | 维护路径 | owner | 验证证据 |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | composer/scheduler | documentation impact gate |
| components | 修改文档 | `orchestration/composer/README.md` | composer/scheduler | package contract tests |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | composer/scheduler | documentation impact gate |
| contract/API | 修改文档 | `docs/guides/integration.md` | composer/planbridge | strict OpenSpec validation |
| diagnostics | 修改文档 | `docs/mainline-contract-test-index.md` | diagnostics/replay | replay gate |
| examples | 修改文档 | `examples/durable-continuation-restore/README.md` | documentation maintainers | example impact gate |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | planbridge | contract gate |
| best practices | 修改文档 | `docs/guides/integration.md` | documentation maintainers | docs consistency gate |
| roadmap | 修改文档 | `docs/development-roadmap.md` | documentation maintainers | roadmap status gate |
