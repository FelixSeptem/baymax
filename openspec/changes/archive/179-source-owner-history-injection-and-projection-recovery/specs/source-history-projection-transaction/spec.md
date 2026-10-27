## ADDED Requirements

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
