## ADDED Requirements

### Requirement: Task Board query SHALL expose async terminal resolution source metadata
Task Board query response MUST expose normalized terminal resolution source metadata for async-await tasks with values:
- `callback`
- `reconcile_poll`
- `timeout`

This extension MUST be additive and MUST NOT change existing pagination/cursor behavior.

#### Scenario: Consumer queries terminal async task
- **WHEN** caller queries task board for an async task that reached terminal state
- **THEN** response includes terminal resolution source classification with existing cursor semantics unchanged

### Requirement: Task Board query SHALL expose remote correlation metadata for async-await tasks
Task Board query response MUST include remote correlation metadata (`remote_task_id` or semantically equivalent field) for async-await tasks when present.

Missing remote correlation metadata for non-async tasks MUST be represented with nullable additive semantics.

#### Scenario: Consumer queries awaiting-report and terminal async tasks
- **WHEN** task records include persisted remote correlation key from async acceptance path
- **THEN** query response returns corresponding remote correlation metadata without mutating read-only behavior

### Requirement: Task Board query SHALL expose terminal conflict observability marker
When callback/poll terminal conflict was recorded, query response MUST expose additive conflict marker for affected task records.

#### Scenario: Consumer queries task with callback-poll conflict
- **WHEN** async task has recorded terminal conflict after first-terminal-wins arbitration
- **THEN** query response includes conflict marker while preserving deterministic sort/page behavior

