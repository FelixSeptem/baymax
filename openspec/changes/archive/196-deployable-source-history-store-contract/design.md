## Context

The existing `core/history` journal, projection ledger, and `core/sourcecoordination` coordinator already provide bounded source facts, typed cursor errors, local file persistence, and fenced revision CAS. They are currently composed independently by each owner, so deployment evidence can be omitted or become inconsistent. See `proposal.md` for the motivation and the existing capability specs for the public behavior being extended.

## Goals / Non-Goals

**Goals:**

- Introduce one explicit, validated owner-store binding that keeps source identity, journal, optional projection ledger, coordinator, writer fence, and deployment request together.
- Make capability publication and append/reconcile admission fail closed before execution activation when the binding is missing, stale, unavailable, or over-requested.
- Preserve stable source-local event identity, bounded cursor semantics, idempotent overlap handling, and source/projection reconciliation without a cross-project transaction.
- Prove the selected profile with fresh-process and concurrent-writer tests, deterministic fixtures, and dedicated shell/PowerShell gates.

**Non-Goals:**

- No new database, hosted coordinator, global service locator, or implicit default store.
- No claim of cross-machine coordination from the existing file coordinator; its supported profile remains the evidence-backed same-host multi-instance CAS/fencing profile.
- No atomic transaction with Danny or any external projection database.
- No change to runner, child, team, workflow, or mailbox execution ownership; history replay remains read-only and cannot resume or rerun effects.

## Decisions

1. **Use an explicit owner-store factory.** A factory returns a detached binding object containing the validated source reference, journal, optional projection ledger, coordinator lease, and deployment scope. This removes repeated injection mistakes while keeping ownership visible. A service locator or package-global default was rejected because it would hide source ownership and make tests non-deterministic.
2. **Validate deployment scope at construction and capability read.** The binding checks the requested profile against coordinator evidence and revalidates the lease before publishing capabilities or mutating source facts. This prevents stale capability claims after a lease expires. Lazy validation alone was rejected because it could advertise support before the first append.
3. **Keep source and projection stores separate.** Journal append commits source facts first; ledger reconciliation is explicit and idempotent. A failed ledger write yields recovery-required and never reruns the effect. A distributed transaction was rejected because it would couple Baymax to Danny storage and exceed the source-owner boundary.
4. **Use stable source identity for duplicate append.** An append carrying an already committed source event identity and identical detached fact converges to the existing identity; conflicting payload/digest or source metadata returns `ErrHistoryConflict`. Sequence allocation remains source-local and is advanced only by the committed CAS transition.
5. **Expose unsupported profiles explicitly.** Unconfigured, memory-only, cross-machine, and unavailable stores return typed unsupported/unavailable evidence and never downgrade to a local store. This preserves the planner's fail-closed behavior.

## Risks / Trade-offs

- [Same-host limitation] File locking cannot prove cross-machine fencing. -> Advertise only the validated local profile and add negative tests for cross-machine requests.
- [Factory adoption gap] Existing owners may still construct journals directly. -> Keep direct constructors valid for tests but mark their deployment capability unsupported unless an explicit binding is supplied; update owner docs and gates to require the factory for durable claims.
- [Projection lag] Source append and projection ledger can commit separately. -> Persist source identity/cursor, return recovery-required on ledger failure, and provide bounded reconciliation with duplicate/conflict checks.
- [Lease expiry during append] A lease can expire between validation and CAS. -> Coordinator CAS remains the final authority; surface stale/unavailable errors and do not retry effects.

## Migration Plan

1. Add the owner-store binding and contract fixtures without changing unconfigured behavior.
2. Migrate capability evidence and the host execution-plan example to construct an explicit binding.
3. Run focused and full gates; any profile without evidence remains unsupported.
4. Roll back by removing the binding at the owner boundary; direct history readers remain read-only and no execution state is mutated.

## Example Impact Assessment

修改示例

理由：host execution plan 示例的 source-history setup 与 expected markers 必须展示显式 owner-store factory、deployment scope 和 unsupported profile 行为。

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`docs/development-roadmap.md`、`docs/guides/integration.md`、`docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`core/types/README.md`、`examples/host-execution-plan/README.md`
- 维护 owner：runtime contracts owner、source history owner、examples/documentation maintainer
- 验证证据：`go test ./core/history ./core/sourcecoordination ./core/types`、`pwsh -File scripts/check-source-history-store-contract.ps1`、`pwsh -File scripts/check-quality-gate.ps1`、`pwsh -File scripts/check-docs-consistency.ps1`、`openspec validate --all`

## Documentation Impact Assessment

| area | outcome | affected paths | owner | evidence |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/guides/integration.md` | runtime contracts owner | binding and profile tests |
| components | 修改文档 | `core/types/README.md` | source history owner | history capability tests |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime contracts owner | fail-closed admission tests |
| contract/API | 修改文档 | `docs/guides/integration.md` | runtime contracts owner | factory/reconcile tests |
| diagnostics | 修改文档 | `docs/mainline-contract-test-index.md` | diagnostics/replay maintainer | dedicated gate |
| examples | 修改文档 | `examples/host-execution-plan/README.md` | examples/documentation maintainer | example markers |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | runtime contracts owner | bounded owner-store admission and reconciliation |
| best practices | 修改文档 | `README.md` | documentation maintainer | docs consistency |
| roadmap | 修改文档 | `docs/development-roadmap.md` | documentation maintainer | roadmap consistency |
