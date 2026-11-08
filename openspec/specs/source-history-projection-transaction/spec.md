# source-history-projection-transaction Specification

## Purpose
Defines a source-owned durable commit and projection recovery boundary so
consumers can recover committed native facts without treating callback replay
or a repeated execution as a substitute for source state.

## Requirements

### Requirement: Source owners SHALL persist bounded commit records

A source owner that declares projection recovery support MUST persist a bounded,
versioned commit record containing the source identity, source sequence and
event identity, source terminal evidence when present, schema/version, digest,
creation time, and the owner deployment scope. The record MUST distinguish a
committed source fact from an unavailable or recovery-required transition.

#### Scenario: Committed terminal fact is reopened
- **WHEN** a source owner persists a terminal event and a new process opens the
  same owner store
- **THEN** the new process can load the same source identity, sequence, event
  identity, digest, and terminal evidence
- **AND** it does not execute or redispatch the source effect

#### Scenario: Uncertain append is fail-closed
- **WHEN** a source transition may have happened but its commit record cannot
  be durably confirmed
- **THEN** the owner returns a typed unavailable or recovery-required outcome
- **AND** it does not publish the transition as committed or rerun the effect

### Requirement: Projection progress SHALL be source-bound and idempotent

Projection progress MUST be associated with the source identity and the last
confirmed source commit/cursor. Reapplying an identical committed record MUST
return the existing projection result without duplicating it. A source identity
whose digest, sequence, terminal evidence, or payload differs from the stored
record MUST fail closed with a typed conflict.

#### Scenario: Consumer restarts before projection
- **WHEN** a consumer restarts with a confirmed source cursor before a retained
  committed fact
- **THEN** it can read and apply the committed fact exactly once by source
  identity
- **AND** repeating the same projection request returns the original result

#### Scenario: Same identity has conflicting evidence
- **WHEN** a projection request reuses an existing source event identity with a
  different digest or terminal evidence
- **THEN** the request returns a typed history conflict
- **AND** it does not overwrite the committed record or advance the cursor

### Requirement: Recovery reads SHALL be bounded and side-effect free

The source owner MUST expose a bounded read/recovery operation that loads
committed records after a source cursor and returns the next cursor, retention
floor, completion state, and recovery status. Recovery reads MUST NOT invoke
Run, Stream, retry, resume, model, tool, acknowledgement, lease transition,
business commit, or diagnostics writes.

#### Scenario: Crash before consumer projection
- **WHEN** the source journal commit succeeds and consumer projection fails
  before acknowledging its cursor
- **THEN** a later recovery read returns the same committed source evidence
- **AND** the source owner performs no lifecycle transition

#### Scenario: Recovery store is unavailable
- **WHEN** the owner cannot open or read its persisted commit store
- **THEN** the read returns `ErrSourceUnavailable`
- **AND** it does not return an empty/latest page or mark the source unsupported

### Requirement: Capability declarations SHALL expose commit recovery scope

Any source capability that advertises durable history or projection recovery
MUST declare whether source commit persistence, projection cursor advancement,
and recovery reads are single-writer sequential-reopen or source-owned
multi-writer transactions. A file rename or successful reopen MUST NOT imply
cross-machine coordination or atomicity with an external consumer database.

#### Scenario: Local file owner declares its boundary
- **WHEN** a planner reads a file-backed source capability
- **THEN** it sees the local writer/deployment scope and projection recovery
  limitation
- **AND** the manifest does not advertise multi-writer or cross-machine
  transaction support

#### Scenario: Missing transaction evidence remains unavailable
- **WHEN** a source has retained history but no declared commit/projection
  recovery contract
- **THEN** the durable recovery dimension is marked unavailable or unsupported
- **AND** admission does not infer support from the history reader alone

### Requirement: Source owners SHALL expose explicit journal-to-ledger reconciliation

An owner that has explicitly bound a source journal and projection ledger MUST
expose a bounded reconciliation operation that accepts only source facts from
that journal and persists the matching commit record in that ledger. The
operation MUST validate source identity, source-local sequence, event identity,
terminal evidence, cursor scope, and detached event digest before writing. It
MUST NOT create either store implicitly.

#### Scenario: Journal fact survives a ledger write failure

- **WHEN** a source journal has a durable event whose projection ledger did not
  persist the corresponding commit record
- **THEN** a fresh owner process can reconcile that exact source fact into the
  explicitly supplied ledger
- **AND** the result retains the original source identity, sequence, event ID,
  digest, terminal evidence, and cursor

#### Scenario: Missing owner binding remains unavailable

- **WHEN** reconciliation is requested without a matching source journal or
  projection ledger
- **THEN** the operation returns a typed unavailable or unsupported outcome
- **AND** it does not allocate a default store or rerun the source effect

### Requirement: Reconciliation SHALL be idempotent and conflict-safe

Reapplying an identical source fact MUST return the existing ledger record as a
duplicate without advancing sequence or retention state. A reused event ID
with different source identity, sequence, digest, terminal evidence, cursor,
or detached fact MUST return `ErrHistoryConflict` and MUST NOT overwrite or
advance the ledger.

#### Scenario: Duplicate recovery converges

- **WHEN** the same journal fact is reconciled twice
- **THEN** the second result identifies the existing record as a duplicate
- **AND** the ledger contains one record and one source-bound acknowledgement
  identity

#### Scenario: Conflicting recovery fails closed

- **WHEN** a source fact reuses an existing event ID but changes its digest or
  terminal evidence
- **THEN** reconciliation returns `ErrHistoryConflict`
- **AND** the original record and cursor remain unchanged

### Requirement: Recovery failures SHALL preserve source and execution boundaries

Malformed, foreign, expired, gapped, corrupt, or unavailable source facts MUST
return their typed error before ledger mutation. Reconciliation MAY persist only
the explicitly supplied source projection ledger; it MUST NOT call Run, Stream,
retry, resume, restore, model, tool, handler, lease transition, ack, nack, DLQ,
business commit, or diagnostics writes.

#### Scenario: Gap or unavailable store is not latest fallback

- **WHEN** the source cursor is expired, gapped, foreign, or the ledger cannot
  be opened
- **THEN** reconciliation returns the corresponding typed error
- **AND** it does not start from latest, retention floor, or a new execution

#### Scenario: Reconciliation is not execution recovery

- **WHEN** a source fact is reconciled after a consumer crash
- **THEN** only source projection evidence is repaired
- **AND** no source effect, parent run, child attempt, retry, or terminal state
  is executed or changed

### Requirement: Journal-to-ledger reconciliation SHALL require one explicit owner binding

Projection recovery MUST accept only source facts from the explicitly bound journal and the matching explicitly bound ledger. The operation MUST validate source identity, sequence, event identity, digest, terminal evidence, cursor scope, and deployment profile before ledger mutation. It MUST NOT create a default journal or ledger.

#### Scenario: Matching binding reconciles a committed fact

- **WHEN** a fresh process supplies a retained source fact from the bound journal and the matching projection ledger
- **THEN** reconciliation persists the source commit evidence with its original identity, sequence, digest, terminal evidence, and cursor
- **AND** it does not change source lifecycle state or execute an effect

#### Scenario: Missing binding is unavailable

- **WHEN** reconciliation is requested without a matching journal, ledger, or owner-store profile
- **THEN** it returns a typed unsupported or unavailable result
- **AND** it does not allocate a store, advance a cursor, or rerun the source effect

### Requirement: Projection reconciliation SHALL preserve source/projection non-atomicity explicitly

The owner MUST expose whether source append and projection ledger persistence are separate commits. If source append committed but ledger persistence is uncertain or unavailable, the result MUST be recovery-required or unavailable and MUST retain enough source identity/cursor evidence for later bounded reconciliation. It MUST NOT claim cross-project atomicity.

#### Scenario: Ledger failure after source commit

- **WHEN** a source event is durably appended but the separately bound projection ledger cannot persist it
- **THEN** the owner returns `ErrProjectionRecoveryRequired` or an equivalent typed recovery result
- **AND** it does not rerun, retry, resume, or replace the uncertain effect

#### Scenario: Duplicate reconciliation converges

- **WHEN** the exact source fact is reconciled after a consumer crash and then reconciled again
- **THEN** the second call returns the existing ledger record as a duplicate
- **AND** the ledger contains one source-bound record and cursor acknowledgement

### Requirement: Recovery reads SHALL remain bounded and side-effect free under deployment failure

Bounded recovery reads MUST return typed foreign, malformed, expired, gap, conflict, unavailable, or recovery-required errors before ledger mutation when the owner-store binding or source evidence is invalid. They MUST NOT invoke Run, Stream, retry, resume, restore, model, tool, handler, lease transition, acknowledgement, or business writes.

#### Scenario: Unavailable owner store does not fall back

- **WHEN** a bound journal, ledger, or coordinator cannot be opened or validated during recovery
- **THEN** the operation returns `ErrSourceUnavailable` or a typed recovery-required result
- **AND** it does not return an empty/latest page or start a new execution
