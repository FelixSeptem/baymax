## Purpose

Defines a source-owned durable commit and projection recovery boundary so
consumers can recover committed native facts without treating callback replay
or a repeated execution as a substitute for source state.

## ADDED Requirements

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
