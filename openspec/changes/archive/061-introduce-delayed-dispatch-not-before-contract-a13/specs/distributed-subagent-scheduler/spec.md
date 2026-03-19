## ADDED Requirements

### Requirement: Scheduler claim eligibility SHALL enforce not_before gate
Scheduler claim logic MUST enforce `not_before` gate before a queued task can be claimed.

#### Scenario: Queue contains ready and not-ready delayed tasks
- **WHEN** scheduler scans queue and some tasks have `not_before` in the future
- **THEN** scheduler skips non-ready delayed tasks and may claim other eligible tasks

### Requirement: Scheduler delayed gate SHALL compose with retry backoff gate
Scheduler claim eligibility MUST satisfy both delayed dispatch gate and retry backoff gate when both are present.

#### Scenario: Task has both future not_before and retry next_eligible_at
- **WHEN** scheduler evaluates claim eligibility for task with both gates
- **THEN** task becomes claimable only after both gates are satisfied

### Requirement: Scheduler delayed dispatch SHALL compose with QoS and fairness
When delayed tasks become eligible, scheduler MUST apply existing QoS/fairness selection semantics without bypass.

#### Scenario: Multiple delayed tasks reach eligibility under priority mode
- **WHEN** delayed tasks become eligible in a queue using priority mode
- **THEN** claim ordering follows configured QoS/fairness contract among eligible tasks
