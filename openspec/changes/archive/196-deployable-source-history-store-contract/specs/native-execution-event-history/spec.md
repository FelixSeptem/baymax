## ADDED Requirements

### Requirement: Native history construction SHALL use an owner-store binding for durable claims

Runner, child, team, workflow, and mailbox history declarations that use the deployable owner-store contract MUST identify the explicit owner-store binding and its deployment evidence. Existing explicitly injected journal constructors retain their established compatibility semantics, but callers MUST NOT infer a stronger profile than their declared backing store proves. The declaration MUST preserve source-local ordering, cursor version, retention floor, writer scope, and coordinator transaction scope.

#### Scenario: File-backed binding declares its local boundary

- **WHEN** a source owner opens a file-backed journal through an explicit binding
- **THEN** capabilities identify the source namespace, source-local sequence, retention floor, cursor scope, and same-host writer/transaction scope
- **AND** they do not claim cross-machine coordination or external projection atomicity

#### Scenario: Unbound journal cannot infer a broader profile

- **WHEN** a caller uses an existing directly injected journal and requests a deployment profile beyond its declared backing-store scope
- **THEN** capability admission returns an unsupported result
- **AND** the reader does not infer cross-machine or multi-writer support from a file path, Run ID, callback, or successful reopen

### Requirement: Coordinated append and retention advancement SHALL be one committed source transition

For a binding with a live coordinator, event identity allocation, source-local sequence, retention-floor evolution, and detached journal snapshot publication MUST be committed by one fenced CAS revision or none become visible. A stale writer, CAS conflict, malformed state, or coordinator outage MUST return a typed error without exposing a partial event or silently falling back to latest.

#### Scenario: Retention floor advances atomically

- **WHEN** a coordinated append exceeds the configured retention bound
- **THEN** the new event, next sequence, and new retention floor are visible together in one committed revision
- **AND** a failed CAS leaves the prior journal state intact

#### Scenario: Concurrent writers preserve source order

- **WHEN** two valid writers append through separate owner processes sharing one coordinator
- **THEN** committed events receive one contiguous source-local order with unique stable identities
- **AND** the losing writer receives a typed conflict or stale result without rerunning an effect

### Requirement: History capability evidence SHALL distinguish source replay from projection recovery

An owner-store history declaration MUST separately report retained source replay, committed source facts, projection ledger binding, and projection recovery scope. A reader with no explicitly bound ledger MUST NOT advertise projection recovery or atomic external projection, even when source replay succeeds after restart.

#### Scenario: Source replay without ledger remains limited

- **WHEN** a fresh process reopens a durable source journal without a matching projection ledger
- **THEN** it can report retained source facts and bounded cursors
- **AND** projection recovery and external transaction support remain unavailable
