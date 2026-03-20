## ADDED Requirements

### Requirement: Scheduler SHALL expose read-only Task Board query entrypoint
The scheduler MUST provide a read-only Task Board query entrypoint for listing task records by canonical filters, pagination, sorting, and cursor traversal semantics.

This entrypoint MUST preserve existing enqueue/claim/heartbeat/requeue/commit behavior and MUST NOT mutate queue state as a side effect of query.

#### Scenario: Host queries task board during active scheduling
- **WHEN** scheduler is processing tasks and host calls Task Board query API
- **THEN** query returns current snapshot-derived task records without changing scheduler execution state

#### Scenario: Host queries delayed and dead-letter tasks
- **WHEN** scheduler contains delayed and dead-letter tasks
- **THEN** query can filter and return these task states deterministically

### Requirement: Scheduler Task Board query SHALL remain recovery-compatible
Task Board query behavior MUST remain deterministic before and after scheduler snapshot restore.

#### Scenario: Query before and after snapshot restore
- **WHEN** scheduler state is snapshotted, restored, and queried with the same request
- **THEN** returned logical item set and ordering remain semantically equivalent
