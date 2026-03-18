# distributed-subagent-scheduler Specification

## Purpose
TBD - created by archiving change distributed-subagent-scheduler-baseline-a6. Update Purpose after archive.
## Requirements
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

### Requirement: Scheduler initialization SHALL support fallback-to-memory backend
When configured scheduler backend initialization fails, composer-managed runtime MUST fallback to `memory` backend and MUST emit deterministic fallback diagnostics markers.

#### Scenario: File backend initialization fails at startup
- **WHEN** scheduler backend is configured as `file` and backend initialization fails
- **THEN** runtime falls back to `memory` backend, continues execution, and records fallback usage with an explicit reason marker

### Requirement: Scheduler config reload SHALL use next-attempt-only semantics
Scheduler-related hot-reload updates MUST apply to newly created or newly claimed attempts only, and MUST NOT retroactively change lease semantics of in-flight attempts.

#### Scenario: Scheduler lease config changes during an active attempt
- **WHEN** hot reload updates scheduler lease-related settings while a task attempt is already running
- **THEN** the running attempt keeps its existing lease semantics, and the updated settings apply from the next attempt boundary

### Requirement: Scheduler bridge SHALL converge local and A2A child terminals uniformly
Scheduler-managed local child-run and A2A child-run execution paths MUST converge through the same terminal commit idempotency contract.

#### Scenario: Duplicate terminal commits from mixed child targets
- **WHEN** duplicate terminal commits arrive for local and A2A child attempts
- **THEN** scheduler preserves a single logical terminal outcome and does not inflate additive counters

