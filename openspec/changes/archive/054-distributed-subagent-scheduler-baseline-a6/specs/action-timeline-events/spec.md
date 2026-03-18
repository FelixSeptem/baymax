## ADDED Requirements

### Requirement: Action timeline SHALL encode scheduler and subagent reason namespaces
Action timeline MUST use normalized reason namespaces for distributed scheduling and subagent coordination.

For this milestone, minimum reason set MUST include:
- `scheduler.enqueue`
- `scheduler.claim`
- `scheduler.heartbeat`
- `scheduler.lease_expired`
- `scheduler.requeue`
- `subagent.spawn`
- `subagent.join`
- `subagent.budget_reject`

#### Scenario: Scheduler lease expires and task is requeued
- **WHEN** claimed task lease expires without heartbeat
- **THEN** timeline emits `scheduler.lease_expired` followed by `scheduler.requeue`

#### Scenario: Subagent spawn is rejected by guardrail
- **WHEN** parent run attempts to spawn child beyond guardrail limits
- **THEN** timeline emits `subagent.budget_reject` with normalized terminal category

### Requirement: Scheduler timeline SHALL carry cross-process correlation metadata
Timeline events on scheduler-managed paths MUST carry correlation metadata sufficient for cross-process tracing.

#### Scenario: Scheduler-managed subagent lifecycle emits events
- **WHEN** task transitions across enqueue, claim, and completion in different workers
- **THEN** timeline carries stable correlation metadata including at minimum `task_id`, `attempt_id`, and run linkage keys

### Requirement: Scheduler timeline semantics SHALL preserve Run and Stream equivalence
For equivalent scheduler-managed requests, Run and Stream MUST expose semantically equivalent timeline phase/status/reason outcomes.

#### Scenario: Equivalent scheduler-managed request through Run and Stream
- **WHEN** equivalent requests execute with scheduler enabled in Run and Stream
- **THEN** timeline reason categories and terminal phase semantics remain equivalent
