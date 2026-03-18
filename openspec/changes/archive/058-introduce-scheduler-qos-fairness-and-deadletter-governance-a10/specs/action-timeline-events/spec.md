## ADDED Requirements

### Requirement: Timeline SHALL include canonical scheduler QoS and dead-letter reasons
Action timeline MUST include canonical namespaced reasons for QoS claim decisions, fairness yielding, retry backoff scheduling, and dead-letter transitions.

Canonical reasons:
- `scheduler.qos_claim`
- `scheduler.fairness_yield`
- `scheduler.retry_backoff`
- `scheduler.dead_letter`

#### Scenario: Priority mode with fairness and dead-letter is active
- **WHEN** scheduler emits timeline events under qos/fairness/dlq paths
- **THEN** reasons remain namespaced and contract-checkable under scheduler/subagent taxonomy

### Requirement: QoS and dead-letter events SHALL preserve required correlations
Scheduler QoS and dead-letter timeline events MUST preserve required `task_id` and `attempt_id` correlations on scheduler-managed transitions.

#### Scenario: Task transitions through retry backoff to dead-letter
- **WHEN** scheduler emits backoff and dead-letter transition events
- **THEN** each required event includes task and attempt correlation fields
