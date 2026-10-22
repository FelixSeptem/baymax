## ADDED Requirements

### Requirement: Checkpoint stores SHALL expose deployment and claim scope

Every persisted checkpoint reference and resume claim MUST bind to a source
owner, deployment profile, writer fence, and claim transaction scope. A store
that supports sequential reopen MUST NOT advertise atomic multi-instance claim,
cross-machine CAS, or distributed crash recovery unless it supplies independent
source evidence for those guarantees.

#### Scenario: Same-host resume is admitted
- **WHEN** a new Engine reopens a persisted checkpoint with the current writer
  ownership and matching source binding
- **THEN** it may claim and resume within the declared same-host sequential
  profile
- **AND** the original logical parent identity is preserved

#### Scenario: Duplicate multi-instance claim is unsupported
- **WHEN** two instances present the same checkpoint without a declared
  multi-writer claim transaction
- **THEN** at least the non-owner request returns typed unsupported, stale-owner,
  or unavailable evidence
- **AND** neither request starts a replacement parent run

### Requirement: Store failure SHALL fail closed at the deployment boundary

Writer loss, fence mismatch, persistence quarantine, corrupt ownership state,
or unsupported deployment scope MUST prevent restore admission. The runtime
MUST NOT convert that failure into a fresh Run, a new parent, a re-enqueue, or
an unbounded history replay.

#### Scenario: Persistence failure cannot become rerun
- **WHEN** a checkpoint store cannot persist or validate its ownership state
- **THEN** restore returns a typed unavailable or unsupported result before
  source execution
- **AND** no model, tool, scheduler, mailbox, or parent Run is invoked
