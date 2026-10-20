## Purpose

This capability provides source-owned, durable direct-run continuation across Engine and process boundaries while making every unsupported composite pause/restore combination explicit before source activation.

## ADDED Requirements

### Requirement: Checkpoints SHALL be durable, bounded, and source-bound

The runtime SHALL expose an injectable checkpoint store that persists a versioned checkpoint reference containing checkpoint ID, schema/version, digest, source/run/session identity, pause reason, source position or replay reference, creation and expiry times, causal predecessor, and resume-attempt identity. The persisted form MUST be bounded, detached from caller memory, and free of credentials or business payloads. A process-local store MUST NOT advertise cross-process restore capability.

The source binding MUST preserve original parent run, host correlation, authority/plan/write-scope digests, occurrence and effect namespace when present. A reference or digest MUST NOT substitute for source continuation material or authenticate the caller. Loading MUST validate both the reference and actual retained continuation material before admission.

#### Scenario: Reopened store loads the same checkpoint
- **WHEN** a checkpoint is saved, the Engine process exits, and a new Engine opens the same durable store
- **THEN** the new Engine can load the checkpoint with the original source/run/session identity and digest

#### Scenario: Unrestorable checkpoint is rejected
- **WHEN** a checkpoint contains an unsupported schema, oversized field, foreign source binding, or non-reconstructable continuation reference
- **THEN** load fails with a typed unavailable/corrupt/foreign result and no Run, provider, tool, or executor is started

#### Scenario: Exact envelope boundary and one beyond
- **WHEN** a checkpoint reaches a declared field/count/envelope bound and another checkpoint exceeds it by one
- **THEN** the valid boundary is accepted and the excess is rejected before unbounded copying or persistence

### Requirement: Resume SHALL use an atomic source claim and preserve one logical Run

The source SHALL provide an atomic resume claim operation that validates digest, schema, ownership, expiry, pause state, causal predecessor, and full idempotency identity. Equivalent duplicate requests SHALL return the retained claim/result projection; conflicting or already-consumed requests SHALL fail closed. Successful restore MUST preserve the original logical parent/run/session and MUST NOT create a replacement Run or rerun completed tool work.

#### Scenario: Exact duplicate resume is idempotent
- **WHEN** the same checkpoint and normalized resume identity are claimed twice
- **THEN** both calls return the same admitted/result projection and only one continuation can execute

#### Scenario: Conflicting claim is rejected
- **WHEN** a second request changes the checkpoint digest, mode, pause reason, causal predecessor, or resume identity
- **THEN** the request receives a typed conflict/in-progress result without consuming or replacing the source checkpoint

#### Scenario: Old checkpoint cannot consume a successor
- **WHEN** a resumed Run pauses at a new checkpoint and an old resume request is repeated
- **THEN** it returns only its retained prior result/projection or a typed stale/expired result and never consumes the successor checkpoint

### Requirement: Restore SHALL be fail-closed across lifecycle races

Checkpoint save, load, claim, continuation start, and terminal recording failures MUST remain visible as typed unavailable/corrupt/expired/foreign/conflict outcomes. Cancellation, timeout, retry, and resume ordering MUST be decided from source state; an accepted cancellation request is not terminal cancellation until source evidence records it. Store failure MUST NOT fall back to a new Run, retry, latest state, or side-effecting replay.

#### Scenario: Store unavailable before admission
- **WHEN** the durable store cannot load or claim a checkpoint
- **THEN** admission fails before provider/tool/executor activation and no new parent Run is allocated

#### Scenario: Cancellation races with resume
- **WHEN** source cancellation commits before the resume claim boundary
- **THEN** resume is rejected from current source evidence and terminal cancellation is reported only if the source recorded that terminal

#### Scenario: Crash after claim without a durable result
- **WHEN** the source restarts with a retained claimed/running attempt and no successor or terminal commit
- **THEN** exact duplicates return an explicit in-progress or recovery-required projection, conflicting resumes reject, and no continuation is automatically started again

#### Scenario: Terminal persistence fails
- **WHEN** execution finishes but its source result/terminal commit fails
- **THEN** the source reports unavailable with the retained attempt identity, never reports durable completion, and never reruns the attempt to reconstruct a result

### Requirement: Direct pause reasons SHALL be capability-evidenced

The direct durable profile SHALL produce and durably restore all three pause reasons: `input_required`, `approval_required`, and `budget_exhausted`. Each reason MUST have source-owned pause evidence and a restorable continuation boundary. Until verified and enabled, a cell MUST return `unsupported_capability` before activation; all three cells are mandatory acceptance criteria for completion of this profile. Approval MUST pause before the pending high-risk tool executes. Budget pause MUST preserve consumed resources and pending work, revalidate source-owned current budget facts on resume, and MUST NOT silently replenish budget or reset elapsed deadlines.

#### Scenario: Each supported direct pause restores after restart
- **WHEN** a direct Run is durably paused for any declared pause reason and a new Engine submits a valid resume
- **THEN** the original Run resumes from the saved boundary without rerunning completed work

#### Scenario: Unsupported pause reason is rejected
- **WHEN** a direct or composite form requests a pause reason not declared by its source capability manifest
- **THEN** admission returns `unsupported_capability` before source activation

#### Scenario: Approval is required before tool dispatch
- **WHEN** the explicitly enabled durable approval path pauses a high-risk tool call
- **THEN** the pending call and prior completed outcomes are preserved, the tool count remains zero until an authorized approval, and deny/timeout record source cancellation without executing that call

#### Scenario: Budget has not been increased by its owner
- **WHEN** a budget-exhausted checkpoint receives resume without valid current source budget capacity
- **THEN** the source rejects resume without consuming the checkpoint or repeating completed work

### Requirement: Run and Stream restore SHALL remain semantically equivalent

Equivalent direct Run and Stream checkpoints SHALL preserve source identity, iteration boundary, saved outcomes, pause/resume events, terminal classification, and error precedence. Checkpoint load, inspection and fixture replay MUST be read-only. A restore entry point can continue the source only after all binding checks and an atomic resume claim; replay MUST never claim or resume.

#### Scenario: Run/Stream parity after process restart
- **WHEN** equivalent Run and Stream checkpoints are restored through separate new Engine instances
- **THEN** their source facts and terminal outcomes are semantically equivalent apart from transport projection

### Requirement: Composite forms SHALL declare unsupported continuation explicitly

Child/fanout, team parallel/vote, workflow, mailbox, and host plan resumed execution SHALL remain `unsupported_capability` for pause reasons and cross-process restore unless a later source-owned capability entry supplies graph state, member/step state, dependencies, budgets, native IDs, and terminal evidence. Existing workflow checkpoints, mailbox history, or event callbacks MUST NOT be treated as runner continuation.

#### Scenario: Composite request is rejected before activation
- **WHEN** a plan requests composite pause or restore for a form whose manifest cell is unsupported
- **THEN** admission rejects before enqueue, publish, Run, Stream, retry, or member/step execution
