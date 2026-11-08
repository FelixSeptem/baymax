## ADDED Requirements

### Requirement: Owner-store deployment binding SHALL be explicit and validated

An owner that advertises durable or coordinated source history MUST construct an explicit source-store binding containing the source identity, selected journal/ledger stores, coordinator evidence, writer identity/fence, and requested deployment scope. The binding MUST validate all components before capability publication or source mutation. Missing, mismatched, stale, unavailable, or over-broad bindings MUST return a typed unsupported, unavailable, stale, foreign, or conflict result and MUST NOT allocate an implicit replacement store.

#### Scenario: Unconfigured owner remains unsupported

- **WHEN** a runner, child, team, workflow, or mailbox owner requests durable history without an explicit owner-store binding
- **THEN** capability admission returns `ErrHistoryUnsupported` or the owner-specific unsupported result
- **AND** no journal, coordinator, ledger, execution, or replacement run is created

#### Scenario: Requested profile exceeds evidence

- **WHEN** a caller requests cross-machine or distributed transaction semantics from a same-host file-backed binding
- **THEN** deployment validation returns `ErrDeploymentUnsupported` before source activation
- **AND** the binding continues to advertise only its evidence-backed local profile

### Requirement: Capability publication SHALL revalidate the owner lease

An explicit coordinated binding MUST revalidate its source-owner lease and deployment evidence whenever capabilities are read or a source mutation is admitted. Lease expiry, coordinator outage, foreign writer, or revision drift MUST make the binding unavailable or unsupported until a new explicit binding is supplied.

#### Scenario: Expired lease removes coordinated capability

- **WHEN** a coordinated owner lease expires before a capability or append request
- **THEN** the request returns a typed unavailable or stale-writer error
- **AND** it does not publish a coordinated capability or activate execution

#### Scenario: Fresh binding restores declared scope

- **WHEN** a new writer explicitly acquires a valid source-owner lease and constructs a matching binding
- **THEN** capabilities expose the validated owner namespace, fence and transaction scope
- **AND** the prior writer cannot mutate or replace the committed source facts

### Requirement: Coordinated source publication SHALL be idempotent by stable event identity

An owner-store binding MUST treat an identical already-committed detached source event as a duplicate that returns the original source identity without advancing sequence or retention state. A reused identity with a different source, sequence, digest, terminal evidence, or detached fact MUST fail with `ErrHistoryConflict` and MUST NOT overwrite committed state.

#### Scenario: Duplicate append converges

- **WHEN** two writers submit the same committed source event identity and identical detached facts
- **THEN** the later append returns the original identity as a duplicate
- **AND** the source sequence, retention floor, and committed payload remain unchanged

#### Scenario: Conflicting identity fails closed

- **WHEN** a writer reuses an event identity with a different digest or terminal evidence
- **THEN** append returns `ErrHistoryConflict`
- **AND** no source revision, cursor, or retention floor is changed
