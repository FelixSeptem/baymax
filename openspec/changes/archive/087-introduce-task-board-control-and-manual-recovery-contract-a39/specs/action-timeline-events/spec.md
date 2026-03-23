## ADDED Requirements

### Requirement: Timeline SHALL include canonical task-board manual-control reasons
Action timeline MUST include canonical scheduler namespaced reasons for task-board manual-control actions:
- `scheduler.manual_cancel`
- `scheduler.manual_retry`

These reasons MUST remain within existing `scheduler.*` namespace and MUST preserve compatibility with shared reason-taxonomy contract checks.

#### Scenario: Timeline records manual cancel transition
- **WHEN** scheduler applies manual cancel action through task-board control path
- **THEN** timeline event includes reason `scheduler.manual_cancel` with canonical correlation fields

#### Scenario: Timeline records manual retry transition
- **WHEN** scheduler applies manual retry action through task-board control path
- **THEN** timeline event includes reason `scheduler.manual_retry` with canonical correlation fields
