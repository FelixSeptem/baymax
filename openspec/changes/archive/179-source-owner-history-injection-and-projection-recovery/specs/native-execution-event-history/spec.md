## ADDED Requirements

### Requirement: Source history recovery SHALL bind to an explicit projection owner

Read-after-cursor history MAY supply facts to projection recovery only when the
caller explicitly binds the declaring source journal to a matching projection
ledger. The recovery result MUST preserve source-local ordering, retention
floor, cursor scope, event identity, and terminal evidence; a history reader
alone MUST NOT imply projection recovery or acknowledgement support.

#### Scenario: Durable history without ledger remains limited

- **WHEN** a file-backed source journal is opened without a matching projection
  ledger
- **THEN** its manifest continues to report retained history without projection
  recovery
- **AND** a consumer cannot infer an acknowledgement or atomic projection
  transaction from `ReadAfter`

#### Scenario: Reopened source facts feed the matching ledger

- **WHEN** a fresh process reads a confirmed cursor from a source journal and
  supplies the matching ledger owner
- **THEN** bounded facts can be reconciled without changing source lifecycle
  state
- **AND** the returned cursor and terminal evidence remain source-owned

### Requirement: History recovery SHALL preserve source-local ordering and overlap semantics

Recovery across bounded pages MUST accept identical overlap by source event
identity and digest, reject conflicting overlap with `ErrHistoryConflict`, and
reject gaps, foreign cursors, and expired cursors without silently changing
source position. Parallel child and team sources MUST retain independent source
order and MUST NOT be merged into a synthetic global cursor.

#### Scenario: Overlapping pages are safe to retry

- **WHEN** a consumer retries a recovery page after an acknowledgement timeout
- **THEN** identical source facts deduplicate by source event identity
- **AND** the next source cursor is not advanced twice

#### Scenario: Parallel sources do not share a cursor

- **WHEN** recovery reads are interleaved for two child or team sources
- **THEN** each source validates and advances only its own cursor
- **AND** a cursor from one source is rejected as foreign by the other
