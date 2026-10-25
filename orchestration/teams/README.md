# orchestration/teams

`orchestration/teams` owns serial, parallel and vote aggregation. A default
engine is terminal-only: it does not create a journal, checkpoint store or
cross-process continuation implicitly.

## Native continuation

Parallel and vote continuation is an explicit owner opt-in through
`WithTeamContinuationStore`. `FileTeamContinuationStore` supports only
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

The public capability cell is evidence-gated. A configured owner may promote
team checkpoint/restore and pause cells; an unconfigured, memory-only or
incompatible deployment remains `unsupported_capability`/`recovery-required`.
Team continuation does not make child, workflow, mailbox or resumed planbridge
continuation supported.

## Verification

`continuation_test.go` covers fresh-process reopen, duplicate claim, all three
pause reasons and Run/Stream parity.
`scripts/check-team-mailbox-continuation-contract.sh/.ps1` runs team, mailbox
and planbridge suites and rejects a continuation implementation that constructs
a second mailbox execution owner.
