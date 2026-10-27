## Context

The existing `core/history.Journal` assigns source-local identities and can
persist a bounded event page. An optional `ProjectionLedger` stores matching
commit records and consumer acknowledgements, but journal append and ledger
persistence are separate same-host single-writer operations. When the journal
rename succeeds and the ledger rename fails, the source fact is durable while
the ledger is behind. The current API reports
`ErrProjectionRecoveryRequired`, but offers no explicit way to reconcile the
source fact into the ledger after a fresh-process reopen.

The source owner remains the authority for source identity, sequence, terminal
evidence, retention, and cursor interpretation. The projection ledger is a
separate source-owned evidence store; it is not Danny's business database and
does not authorize execution recovery.

## Goals / Non-Goals

**Goals:**

- Add a bounded, explicit reconciliation seam for a matching journal and
  projection ledger.
- Preserve source identity, sequence, digest, terminal evidence, retention
  floor, and cursor semantics across a crash window and process restart.
- Make duplicate reconciliation idempotent and conflicting source facts
  fail-closed before mutation.
- Keep capability promotion, owner injection, and same-host single-writer
  deployment evidence explicit for every source form.
- Prove the boundary with owner tests, read-only replay, crash-window fixtures,
  and Run/Stream/no-side-effect checks.

**Non-Goals:**

- No new journal, queue, scheduler, worker, recovery loop, or hosted control
  plane.
- No multi-writer CAS, cross-machine fencing, cross-database atomic commit, or
  exactly-once business projection.
- No execution retry, resume, checkpoint restore, parent rerun, mailbox lease
  transition, or terminal arbitration.
- No implicit creation of a journal or projection ledger and no promotion from
  package, DTO, callback, or mailbox lifecycle history.

## Decisions

### Reconcile through the existing source owner stores

Add a public recovery method on the existing history boundary that accepts a
detached source event/page and writes only to the explicitly bound
`ProjectionLedger`. Keep journal and ledger construction separate and preserve
the existing `WithProjectionLedger` opt-in. The method validates the source
event's existing `NativeHistoryIdentity` and derives the same canonical digest
used by normal ledger commits. This keeps ownership in `core/history` instead
of creating an application-level recovery service.

An alternative would be to add a background recovery worker or make
`ProjectionLedger` read the journal path implicitly. Both would create hidden
ownership and lifecycle side effects, so they are rejected.

### Use a source-fact recovery result with explicit duplicate state

Return the retained `SourceCommitRecord` plus a duplicate indicator from
reconciliation. A duplicate is valid only when source, sequence, event ID,
digest, terminal evidence, and cursor all match the existing record. A
different fact with the same event ID returns `ErrHistoryConflict`; a sequence
gap returns `ErrHistoryGap` or `ErrProjectionRecoveryRequired` according to the
existing source boundary. No mutation occurs before all validation succeeds.

An alternative would be to silently ignore duplicates. That loses evidence for
the caller and makes recovery auditability ambiguous, so duplicate state is
explicit.

### Keep replay and reconciliation as separate operations

`ReadAfter` and `ReadCommittedAfter` remain read-only. Reconciliation is an
explicit persistence operation whose only write target is the supplied source
projection ledger. Replay fixtures call neither source owner nor reconciliation;
integration tests exercise the real crash window separately. This distinction
prevents callback replay or history reads from being misrepresented as
execution restore.

### Promote capability only with matching durable evidence

Capability projection checks the form, owner namespace, journal/ledger source
identity, API/contract versions, evidence reference, and deployment scope.
Memory journals and memory ledgers remain history-only or recovery-unsupported;
file-backed matching stores may report recovery only within the existing
same-host, single-writer, sequential-reopen boundary. No capability is inferred
from a source reader or owner package.

### Owner integration is additive and form-local

Runner, Composer child, teams, and workflow expose or test the same explicit
recovery seam while retaining their existing `WithNativeEventHistory` APIs.
Mailbox continues to use its own lifecycle history and source owner; it does
not receive a generic runner projection ledger or execution-recovery path.
Planbridge only consumes manifest facts and never opens or reconciles stores.

## Risks / Trade-offs

- [Risk] A recovery caller may mistake ledger repair for business projection.
  → Keep the result and docs source-evidence-only, prohibit business writes in
  tests, and preserve the separate acknowledgement reference contract.
- [Risk] Source and ledger retention windows may differ. → Validate floor,
  sequence, and cursor before mutation; return typed expiry/gap errors instead
  of latest fallback.
- [Risk] A source fact may be malformed or have changed payload evidence. →
  snapshot and digest before write; reject conflict and leave the ledger
  untouched.
- [Risk] Existing owner capability cells could be widened accidentally. →
  require matching per-form evidence and add negative manifest tests for
  mailbox/direct/composite evidence leakage.
- [Risk] Recovery path could become an implicit retry path. → use counting fake
  owners and gate scans to prove no Run, Stream, retry, resume, lease, ack,
  nack, DLQ, model, tool, or diagnostics invocation.

## Migration Plan

1. Add the additive recovery DTO/result and validation rules in `core/types`.
2. Implement journal-to-ledger reconciliation in `core/history` using the
   existing source identity and digest rules.
3. Add owner integration tests and capability evidence for runner, child, team,
   workflow, and mailbox boundaries without changing default unsupported cells.
4. Add crash-window/restart fixtures, offline replay classifications, example
   markers, and shell/PowerShell gates.
5. Run focused tests, full Go/race/lint/quality/docs gates, then archive the
   change. Rollback removes reconciliation evidence and returns the source to
   `recovery-required`; it never deletes source facts or reruns uncertain
   effects.

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`, `docs/development-roadmap.md`,
  `docs/guides/integration.md`, `docs/runtime-module-boundaries.md`,
  `docs/runtime-config-diagnostics.md`, `docs/mainline-contract-test-index.md`,
  `core/history/README.md`, `core/types/README.md`,
  `examples/native-execution-events/README.md`
- 维护 owner：runtime-core/source-history owners, planbridge capability owner,
  diagnostics-replay and examples maintainers
- 验证证据：strict OpenSpec validation, documentation/example-impact and
  roadmap gates, source-history contract/replay gates, focused owner tests,
  `go test ./...`, `go test -race ./...`, golangci-lint, quality gate, and
  docs consistency verification recorded in `verification.md`.

## Example Impact Assessment

修改示例

理由：the offline native execution events example will demonstrate explicit
source-fact reconciliation and its fail-closed duplicate/conflict boundaries.

## Documentation Impact Assessment

| Area | Outcome | Affected paths | Owner | Verification |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | runtime-core/source owners | no-second-store and ownership gate |
| components | 修改文档 | `core/history/README.md`, `core/types/README.md` | history/types owners | recovery API tests |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime config owner | no-new-config gate |
| contract/API | 修改文档 | `docs/guides/integration.md` | source-history owner | strict spec and integration tests |
| diagnostics | 修改文档 | `docs/mainline-contract-test-index.md` | replay maintainer | fixture/gate index tests |
| examples | 修改文档 | `examples/native-execution-events/README.md` | examples maintainer | offline smoke test |
| CLI/integration | 修改文档 | `docs/guides/integration.md` | source/planbridge owners | fail-closed recovery tests |
| best practices | 修改文档 | `README.md` | documentation owner | docs consistency gate |
| roadmap | 修改文档 | `docs/development-roadmap.md` | roadmap owner | roadmap status consistency gate |
