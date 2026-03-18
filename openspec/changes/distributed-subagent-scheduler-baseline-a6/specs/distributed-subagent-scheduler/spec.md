## ADDED Requirements

### Requirement: Scheduler SHALL provide durable queue and lease-based claiming semantics
The distributed subagent scheduler MUST provide durable enqueue and lease-based claim semantics so worker restarts or process crashes do not lose pending tasks.

#### Scenario: Worker claims enqueued task
- **WHEN** a task is enqueued and an eligible worker polls for work
- **THEN** scheduler returns the task with a lease token and records claim ownership atomically

### Requirement: Scheduler SHALL support heartbeat and lease-expiry takeover
The scheduler MUST support heartbeat renewal for active leases and MUST requeue tasks when lease renewal expires within configured timeout.

#### Scenario: Worker crashes during task execution
- **WHEN** worker holding a lease stops heartbeating beyond lease timeout
- **THEN** scheduler marks lease expired and makes the task claimable by another worker

### Requirement: Scheduler result commit SHALL be idempotent
Scheduler task completion and failure commits MUST be idempotent under duplicate delivery, retry, and replay.

#### Scenario: Duplicate result commit arrives
- **WHEN** scheduler receives repeated completion for the same task attempt idempotency key
- **THEN** scheduler preserves a single logical terminal outcome and does not inflate aggregate counters

### Requirement: Scheduler SHALL enforce parent-child subagent guardrails
Scheduler MUST enforce parent-child guardrails including maximum depth, maximum active child runs, and bounded child execution timeout.

#### Scenario: Parent run exceeds child-depth guardrail
- **WHEN** a run requests spawning child subagents beyond configured depth
- **THEN** scheduler rejects the spawn with normalized budget/guardrail reason

### Requirement: Scheduler-managed execution SHALL preserve semantic equivalence across equivalent run modes
For equivalent logical requests and effective configuration, scheduler-managed execution MUST preserve semantic equivalence of terminal state and aggregate counters across Run and Stream paths.

#### Scenario: Equivalent scheduler-managed request via Run and Stream
- **WHEN** equivalent requests execute through scheduler-managed Run and Stream paths
- **THEN** terminal status category and scheduler aggregate counters remain semantically equivalent
