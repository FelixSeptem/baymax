## ADDED Requirements

### Requirement: Timeline SHALL include canonical async reporting reason taxonomy
Action timeline MUST include canonical reason taxonomy for async reporting lifecycle in the `a2a.*` namespace.

Minimum required reasons:
- `a2a.async_submit`
- `a2a.async_report_deliver`
- `a2a.async_report_retry`
- `a2a.async_report_dedup`
- `a2a.async_report_drop`

#### Scenario: Async report retries then succeeds
- **WHEN** async report delivery fails transiently and later succeeds
- **THEN** timeline emits canonical async reporting reasons for submit, retry, and delivery

### Requirement: Async reporting timeline SHALL preserve correlation fields
Timeline events for async reporting MUST preserve required correlation metadata including `task_id`, `agent_id`, `peer_id`, and attempt-level linkage when available.

#### Scenario: Scheduler-managed async report event is emitted
- **WHEN** async report event is emitted for scheduler-managed task
- **THEN** timeline payload includes `task_id` and `attempt_id` correlations where available

### Requirement: Async reporting timeline semantics SHALL remain Run and Stream equivalent
For equivalent async-reporting workloads, Run and Stream paths MUST produce semantically equivalent async reporting reason and status distributions.

#### Scenario: Equivalent async reporting flow via Run and Stream
- **WHEN** same logical async reporting flow is executed through Run and Stream
- **THEN** timeline reason/status semantics remain equivalent
