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

### Requirement: Scheduler SHALL restore task and attempt state from recovery snapshot
Scheduler recovery integration MUST restore queued/running/task-attempt state using deterministic mapping to existing task and attempt identifiers.

#### Scenario: Scheduler state is restored after restart
- **WHEN** scheduler loads recovery snapshot containing queued and running attempts
- **THEN** task/attempt identifiers are preserved and claim/commit semantics remain consistent with pre-restart state

### Requirement: Scheduler terminal replay under recovery SHALL remain idempotent
Scheduler terminal commits replayed during recovery MUST remain idempotent for both success and failure outcomes.

#### Scenario: Duplicate terminal commit appears in recovery replay
- **WHEN** recovery replays duplicate terminal commit for same task and attempt
- **THEN** scheduler keeps one logical terminal result and additive counters remain stable

### Requirement: Scheduler recovery conflict SHALL fail fast
If recovered scheduler state cannot be reconciled with runtime state, scheduler recovery MUST fail fast and stop resume flow.

#### Scenario: Recovery attempt mismatch is detected
- **WHEN** recovered current attempt identity conflicts with runtime claimable state
- **THEN** scheduler emits conflict classification and recovery terminates without best-effort continuation

### Requirement: Scheduler SHALL preserve compatibility when QoS is disabled
When QoS mode is not enabled, scheduler MUST preserve existing FIFO-compatible claim behavior and retry semantics.

#### Scenario: Existing scheduler integration runs under default config
- **WHEN** host uses scheduler without qos-specific config
- **THEN** scheduler behavior remains compatible with pre-A10 FIFO baseline

### Requirement: Scheduler terminal path SHALL include dead-letter terminal classification
Scheduler terminal outcomes MUST include explicit dead-letter classification when tasks are moved out of normal retry lifecycle.

#### Scenario: Retry-exhausted task enters dead-letter
- **WHEN** dead-letter policy is enabled and retry budget is exhausted
- **THEN** task terminal classification includes dead-letter reason and no further standard queue claims occur

### Requirement: Scheduler A2A adapter SHALL use shared synchronous invocation contract
Scheduler-managed A2A dispatch adapter MUST use shared synchronous invocation contract for submit/wait/normalize behavior instead of path-local duplicated logic.

#### Scenario: Scheduler claim executes remote child through A2A
- **WHEN** scheduler worker executes claimed task through A2A bridge
- **THEN** adapter invokes shared synchronous invocation and receives normalized terminal mapping

### Requirement: Scheduler retryability mapping SHALL follow normalized transport classification
Scheduler retryability decision for A2A execution MUST be derived from normalized error-layer classification where transport-layer failures are retryable and non-transport failures are non-retryable by default.

#### Scenario: Scheduler receives protocol-layer failure
- **WHEN** shared synchronous invocation returns protocol-layer failure
- **THEN** scheduler marks commit as failed and non-retryable

### Requirement: Scheduler canceled remote terminal SHALL converge deterministically
When remote A2A terminal state is `canceled`, scheduler terminal commit path MUST converge deterministically under existing terminal commit contract.

#### Scenario: A2A terminal status is canceled during scheduler-managed execution
- **WHEN** scheduler adapter receives canceled terminal from A2A
- **THEN** scheduler produces deterministic terminal commit outcome compatible with existing commit API

### Requirement: Scheduler SHALL converge async reports through terminal commit contract
Scheduler-managed async remote execution MUST converge report sink terminal outcomes through existing terminal commit contract with idempotent behavior.

#### Scenario: Scheduler receives async terminal report for claimed task
- **WHEN** async report arrives for a scheduler-managed task attempt
- **THEN** scheduler converges terminal state through commit contract and preserves idempotent semantics

### Requirement: Scheduler async report handling SHALL preserve retryability classification
Scheduler async report handling MUST preserve normalized retryability classification based on error-layer semantics.

#### Scenario: Async report indicates transport-layer failure
- **WHEN** async report carries transport-class failure classification
- **THEN** scheduler applies retryable handling consistent with scheduler retry policy

### Requirement: Scheduler async report replay SHALL remain recovery-safe
Scheduler async report replay under recovery MUST preserve deterministic convergence and must not inflate aggregate counters.

#### Scenario: Recovery replays async terminal reports
- **WHEN** recovered run replays already processed async terminal reports
- **THEN** scheduler keeps stable logical terminal outcomes and additive counters

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

### Requirement: Scheduler recovery path SHALL enforce no_rewind for terminal task records
Scheduler restore logic MUST preserve terminal records and MUST NOT enqueue restored terminal tasks for claim.

#### Scenario: Scheduler restore includes terminal commits
- **WHEN** scheduler restores snapshot containing terminal commit records
- **THEN** restored terminal tasks are not re-queued and remain terminal

### Requirement: Scheduler timeout reentry SHALL be bounded by recovery boundary policy
Scheduler-managed long-running task continuation after timeout MUST enforce single reentry budget and deterministic failure after budget exhaustion.

#### Scenario: Restored task hits timeout during resumed attempt
- **WHEN** scheduler-managed task times out during resumed execution and reentry budget is exhausted
- **THEN** scheduler converges task to terminal failed status without additional reentry

### Requirement: Scheduler recovery boundary semantics SHALL preserve Run Stream equivalence
For equivalent scheduler-managed recovery scenarios with boundary enforcement, Run and Stream paths MUST preserve semantic equivalence for terminal category and additive counters.

#### Scenario: Equivalent recovery boundary scenario via Run and Stream
- **WHEN** same scheduler recovery boundary scenario runs in Run and Stream paths
- **THEN** terminal classification and summary counters remain semantically equivalent

### Requirement: Scheduler SHALL expose read-only Task Board query entrypoint
The scheduler MUST provide a read-only Task Board query entrypoint for listing task records by canonical filters, pagination, sorting, and cursor traversal semantics.

This entrypoint MUST preserve existing enqueue/claim/heartbeat/requeue/commit behavior and MUST NOT mutate queue state as a side effect of query.

#### Scenario: Host queries task board during active scheduling
- **WHEN** scheduler is processing tasks and host calls Task Board query API
- **THEN** query returns current snapshot-derived task records without changing scheduler execution state

#### Scenario: Host queries delayed and dead-letter tasks
- **WHEN** scheduler contains delayed and dead-letter tasks
- **THEN** query can filter and return these task states deterministically

### Requirement: Scheduler Task Board query SHALL remain recovery-compatible
Task Board query behavior MUST remain deterministic before and after scheduler snapshot restore.

#### Scenario: Query before and after snapshot restore
- **WHEN** scheduler state is snapshotted, restored, and queried with the same request
- **THEN** returned logical item set and ordering remain semantically equivalent

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

