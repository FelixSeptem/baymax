## Purpose

This capability provides source-owned, bounded continuation material for
composite executions and makes child, team, mailbox, and resumed-plan support
independent, evidence-based admission decisions.

## ADDED Requirements

### Requirement: Composite continuation material SHALL be source-owned and bounded

Each supported composite source SHALL persist a bounded continuation envelope
containing form and graph identity/version/digest, original parent/run/session/
causation/authority/occurrence/effect identities, native child/member/step or
message/delivery/lease identifiers, dependency state, pending and completed
actions, consumed budget/deadline, completed effect references, pause boundary,
terminal arbitration evidence, and predecessor/successor lineage. Payloads,
credentials, callbacks, and unbounded application bodies MUST remain outside
the public envelope.

#### Scenario: Child continuation retains source identity
- **WHEN** a child pauses at a declared safe point
- **THEN** its persisted continuation retains the original parent/run/session,
  task/attempt/lease, causation, budget, and completed-effect identities

#### Scenario: Existing recovery material is not continuation material
- **WHEN** admission receives a scheduler snapshot, mailbox history, callback
  event, or application projection without a source continuation envelope
- **THEN** it returns `unsupported_capability` or typed unavailable before any
  new child, member, message, or plan activation

### Requirement: Supported composite forms SHALL use atomic idempotent resume claims

A source owner SHALL validate schema, digest, ownership, expiry, graph/form
binding, authority/plan/write-scope/effect namespace, occurrence/causation,
current revision/budget facts, and predecessor/successor relation before an
atomic resume claim. Equivalent requests SHALL return the same retained
projection; conflicting, foreign, stale, expired, corrupt, unavailable, or
already-running requests SHALL fail closed.

#### Scenario: Duplicate child resume converges
- **WHEN** two equivalent requests race for one child continuation
- **THEN** exactly one source continuation is admitted and both callers observe
  the same claim or terminal projection

#### Scenario: Conflicting composite resume is rejected
- **WHEN** a request changes mode, form, graph digest, pause reason, causation,
  or resume identity
- **THEN** the source returns a typed conflict/in-progress result without
  consuming the continuation or starting another logical execution

### Requirement: Child and fanout restore SHALL not re-enqueue or rerun completed effects

When the child/fanout source advertises restore, a new Composer SHALL reopen the
configured source store, restore the same native child graph and scheduler
facts, skip completed members/effects, and continue only pending work. Restore
MUST NOT call a new parent Run, create a replacement task, enqueue a duplicate,
or use history replay as execution state.

#### Scenario: New Composer restores pending local child
- **WHEN** a local child continuation is persisted and a new Composer opens the
  same compatible store
- **THEN** it resumes the original task/attempt lineage and does not spawn a
  second task or repeat completed effect outcomes

#### Scenario: Child lease loss is source evidence
- **WHEN** a child lease expires before restore
- **THEN** admission returns the native lease/recovery outcome and does not
  silently reclaim, retry, or re-run the parent

### Requirement: Pause reasons SHALL be independently evidenced by form

The manifest SHALL declare `input_required`, `approval_required`, and
`budget_exhausted` independently for each composite form. A supported cell MUST
have a source-owned producer, persisted safe point, cross-process restore,
cancel/timeout/retry race tests, terminal evidence, and Run/Stream parity. A
missing cell SHALL return `unsupported_capability` before activation.

#### Scenario: Unsupported team pause is rejected
- **WHEN** a team plan requests approval continuation before team source evidence
  exists
- **THEN** admission rejects before member activation and does not borrow the
  workflow or child capability

#### Scenario: Child budget pause preserves consumed resources
- **WHEN** a child pauses for budget exhaustion
- **THEN** restore revalidates current source budget facts, preserves consumed
  resources and original deadline, and does not replenish or reset them

### Requirement: Composite lifecycle races SHALL fail closed

Cancellation, timeout, retry, lease loss, resume, claim/start, and terminal
commit ordering SHALL be decided from source evidence. Accepted cancellation is
not terminal cancellation until the source records it. Store or terminal commit
failure MUST surface as typed unavailable/recovery-required and MUST NOT trigger
automatic re-execution.

#### Scenario: Crash after child claim
- **WHEN** a process crashes after a child claim but before successor or terminal
  commit
- **THEN** duplicate requests return in-progress or recovery-required, conflicting
  requests fail closed, and no continuation starts automatically

### Requirement: Team, mailbox, and resumed-plan cells SHALL remain explicit until evidenced

Team parallel/vote, mailbox host-plan continuation, and planbridge resumed
execution SHALL remain `unsupported_capability` unless their own source owners
provide equivalent graph/message/lease material, atomic claim, restore, terminal
evidence, and capability references. Workflow or child support MUST NOT imply
support for another form.

#### Scenario: Resumed plan cannot borrow child support
- **WHEN** a host requests a resumed plan while only child continuation evidence
  is available
- **THEN** plan compilation rejects with `unsupported_capability` before enqueue,
  publish, Run, Stream, or direct fallback
