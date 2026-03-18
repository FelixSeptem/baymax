## ADDED Requirements

### Requirement: A2A interoperability SHALL support scheduler-managed dispatch lifecycle
A2A task dispatch MUST support scheduler-managed lifecycle transitions including queued, claimed, and terminal commit phases without breaking existing submit/status/result contract.

#### Scenario: A2A task is dispatched by scheduler worker
- **WHEN** scheduler worker claims a remote-collaboration task and dispatches through A2A
- **THEN** A2A lifecycle remains queryable and terminal status maps to normalized A2A semantics

### Requirement: A2A scheduler integration SHALL preserve idempotent terminal mapping
A2A terminal outcomes committed through scheduler retry/takeover paths MUST remain idempotent and deterministic.

#### Scenario: A2A terminal commit is replayed after takeover
- **WHEN** takeover worker replays terminal commit for already-completed task attempt
- **THEN** A2A summary fields remain stable and duplicate commit does not alter logical terminal state

### Requirement: A2A scheduler integration SHALL preserve normalized error-layer mapping
When A2A execution fails under scheduler-managed retries, transport/protocol/semantic mapping MUST remain normalized and stable.

#### Scenario: Scheduler retries after transport failure
- **WHEN** remote collaboration fails with retryable transport error and scheduler retries claim execution
- **THEN** resulting A2A error class and `a2a_error_layer` remain normalized and deterministic
