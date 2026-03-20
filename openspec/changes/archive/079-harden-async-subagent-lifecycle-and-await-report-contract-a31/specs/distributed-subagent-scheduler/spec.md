## ADDED Requirements

### Requirement: Scheduler SHALL model awaiting-report as explicit async lifecycle state
Scheduler state model MUST include `awaiting_report` for async-accepted task attempts and MUST expose this state through snapshot and task query surfaces.

#### Scenario: Scheduler marks async-accepted attempt as awaiting-report
- **WHEN** scheduler-managed async child dispatch is accepted by remote peer
- **THEN** scheduler record transitions to `awaiting_report` and remains visible in snapshot and query APIs

### Requirement: Scheduler async-await timeout governance SHALL be deterministic
Scheduler MUST enforce configured async-await timeout and MUST converge terminal classification deterministically across memory and file backends.

#### Scenario: Awaiting-report timeout reaches terminal boundary
- **WHEN** scheduler task stays in `awaiting_report` longer than configured timeout
- **THEN** scheduler converges terminal state deterministically (`failed` by default, `dead_letter` when policy applies) and emits canonical lifecycle reason markers

### Requirement: Scheduler restore and replay SHALL preserve awaiting-report lifecycle semantics
Scheduler recovery restore path MUST preserve `awaiting_report` records and keep timeout/replay handling deterministic after restart.

#### Scenario: Recovery restores awaiting-report task
- **WHEN** runtime restores snapshot containing awaiting-report tasks and receives replayed reports
- **THEN** scheduler preserves deterministic terminal convergence without duplicate logical outcomes

