## ADDED Requirements

### Requirement: Timeline SHALL include canonical delayed-dispatch reasons
Action timeline MUST include canonical delayed-dispatch reason taxonomy in `scheduler.*` namespace.

Minimum required reasons:
- `scheduler.delayed_enqueue`
- `scheduler.delayed_wait`
- `scheduler.delayed_ready`

#### Scenario: Delayed task transitions from waiting to ready
- **WHEN** delayed task reaches `not_before` boundary and becomes claimable
- **THEN** timeline emits canonical delayed wait/ready reason semantics

### Requirement: Delayed-dispatch timeline SHALL preserve required correlation metadata
Delayed-dispatch timeline events MUST preserve scheduler-required correlation fields including `task_id` and attempt-level correlation where applicable.

#### Scenario: Delayed task is later claimed
- **WHEN** delayed task transitions to claim path
- **THEN** timeline keeps required correlation metadata for delayed and claim events
