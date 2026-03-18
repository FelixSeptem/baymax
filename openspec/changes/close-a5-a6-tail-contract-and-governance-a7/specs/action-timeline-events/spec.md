## ADDED Requirements

### Requirement: Action timeline SHALL use canonical scheduler/subagent reason taxonomy
Action timeline MUST use canonical scheduler/subagent reason set for closure governance.

Minimum required reasons:
- `scheduler.enqueue`
- `scheduler.claim`
- `scheduler.lease_expired`
- `scheduler.requeue`
- `subagent.spawn`
- `subagent.join`
- `subagent.budget_reject`

#### Scenario: Lease expiration takeover occurs
- **WHEN** scheduler lease expires and task is reclaimed
- **THEN** timeline reason codes use canonical scheduler taxonomy

### Requirement: Action timeline SHALL include attempt-level correlation on scheduler-managed paths
Scheduler-managed timeline events MUST include attempt-level correlation metadata for deterministic retry/takeover tracing.

#### Scenario: Duplicate attempt replay is processed
- **WHEN** timeline events for the same `task_id+attempt_id` are replayed
- **THEN** correlation remains deterministic and aggregate behavior remains idempotent
