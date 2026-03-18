## ADDED Requirements

### Requirement: Timeline SHALL include canonical recovery reason semantics
Action timeline contract MUST include canonical recovery reasons for restore/replay/conflict paths and keep namespace consistency with existing multi-agent reasons.

Canonical reason set (namespaced):
- `recovery.restore`
- `recovery.replay`
- `recovery.conflict`

#### Scenario: Recovery restore and replay events are emitted
- **WHEN** composed runtime performs restore and replay steps
- **THEN** timeline reasons use canonical namespaced recovery semantics and remain contract-checkable

### Requirement: Recovery timeline SHALL preserve required correlation fields
Recovery-related scheduler and A2A timeline events MUST keep required correlation identifiers (`run_id`, `task_id`, `attempt_id`, and related cross-domain IDs where applicable).

Required correlation keys:
- `run_id`
- `task_id`
- `attempt_id`

#### Scenario: Recovery emits scheduler and A2A transitions
- **WHEN** recovery path emits scheduler claim/commit and A2A in-flight continuation events
- **THEN** required correlation fields are present for deterministic replay auditing
