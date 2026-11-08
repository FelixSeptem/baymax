## ADDED Requirements

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
