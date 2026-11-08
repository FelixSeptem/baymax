# orchestration/teams

`orchestration/teams` owns serial, parallel and vote aggregation. A default
engine is terminal-only: it does not create a journal, checkpoint store or
cross-process continuation implicitly.

## Native continuation

Parallel and vote continuation is an explicit owner opt-in through
`WithTeamContinuationStore`. The sealed graph identity binds normalized
strategy, member/participant identity, failure/backpressure policy, worker
bound and vote tie-break so a fresh process cannot silently change dispatch or
winner semantics. `FileTeamContinuationStore` supports only
same-host, single-writer, sequential reopen. `MemoryTeamContinuationStore` is
process-local and is never advertised as cross-process restore.

At a dispatch-safe boundary, the owner persists the original graph/team/run,
parent/session/causation, authority/plan/write-scope/effect/occurrence facts,
pending and settled member references, aggregate state, consumed budget and
original deadline. `input_required`, `approval_required` and
`budget_exhausted` are distinct pause reasons. A resume atomically claims the
same source checkpoint, executes only pending members and converges exact
duplicates. Corrupt, foreign, stale, expired, conflicting or interrupted claims
fail closed; completed members/effects are not replayed and a replacement parent
Run is not created.

When a resumed team reaches another pause boundary, the owner persists a
successor checkpoint with the original logical run/session/authority identity,
the next generation, and the immediate predecessor checkpoint reference. The
predecessor resume projection and sealed successor reference are committed as
one owner-store transition. A fresh FileStore reopen can therefore validate the
chain and resume only the successor pending members; duplicate requests return
the retained projection, while foreign, stale, malformed, or conflicting
successor lineage fails closed. This does not expand FileStore beyond same-host
single-writer sequential reopen or provide distributed recovery.
This lineage change adds no configuration keys or defaults, diagnostic schema
fields, or example runtime paths and markers.

The public capability cell is evidence-gated. A configured owner may promote
team checkpoint/restore and pause cells only with matching policy-binding,
duplicate-claim, ambiguous-in-flight and Run/Stream evidence; an unconfigured,
memory-only or incompatible deployment remains
`unsupported_capability`/`recovery-required`.
Team continuation does not make child, workflow, mailbox or resumed planbridge
continuation supported.

## Host context binding

An explicitly paused team may persist a reference-only
`hostprojection.SourceBinding` for a pending **local** member. On resume, the
team owner validates the exact member/graph/generation binding before invoking
the current host resolver, rechecks the original deadline after resolution,
then routes transient material only through that member's
`HostContextTaskRunner`. Settled members never reactivate and unbound parallel
siblings continue through their normal runner without inherited material. Remote
member context injection is unsupported until a source-owned remote execution
seam provides equivalent evidence.

This owner API evidence does not promote the default planbridge capability
manifest. A foreign/missing binding or an unsupported member runner rejects
before resolver, resume claim, or member activation; it never falls back to a
replacement parent Run.

## Verification

`continuation_test.go` covers fresh-process reopen for parallel and vote teams,
duplicate claim, all three pause reasons, repeated-pause provenance, Run/Stream
parity, pending-member binding, sibling isolation, vote aggregate continuity,
foreign-binding rejection before claim, ambiguous in-flight rejection, legacy
checkpoint fail-closed behavior, and the post-resolver deadline fence.
`scripts/check-team-continuation-contract.sh/.ps1` runs the team, planbridge
and core capability suites and checks that policy/provenance and owner evidence
remain present. Mailbox lifecycle evidence remains covered by its separate
mailbox continuation gate; it does not make mailbox an Agent checkpoint owner.
