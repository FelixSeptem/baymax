## ADDED Requirements

### Requirement: Scheduler SHALL persist remote correlation key for awaiting-report reconciliation
Scheduler-managed async acceptance path MUST persist remote correlation key (for example `remote_task_id`) in task/attempt record so reconcile polling can continue across restart and recovery.

#### Scenario: Runtime restarts after async accepted
- **WHEN** async task has entered `awaiting_report` and runtime restores scheduler snapshot
- **THEN** scheduler record retains remote correlation key and reconcile polling can continue deterministically

### Requirement: Scheduler reconcile dispatcher SHALL apply deterministic polling cadence controls
Scheduler reconcile dispatcher MUST honor configured polling controls for eligible `awaiting_report` tasks:
- `interval`
- `batch_size`
- `jitter_ratio`

Dispatcher MUST skip polling when reconcile feature is disabled.

#### Scenario: Reconcile is disabled by default
- **WHEN** runtime starts with default configuration and tasks enter `awaiting_report`
- **THEN** scheduler does not execute reconcile poll loop

#### Scenario: Reconcile polls bounded batch when enabled
- **WHEN** reconcile is enabled and eligible awaiting-report tasks exceed configured `batch_size`
- **THEN** scheduler polls only bounded batch per cycle and defers remaining tasks to later cycles

### Requirement: Scheduler reconcile terminal commit SHALL reuse existing idempotent commit contract
Terminal outcomes obtained via reconcile poll MUST converge through existing scheduler terminal commit contract and MUST preserve idempotent behavior under duplicate poll callbacks.

#### Scenario: Duplicate terminal is observed from reconcile polls
- **WHEN** reconcile loop reads same terminal outcome multiple times for one task/attempt
- **THEN** scheduler commits one logical terminal outcome and keeps additive counters stable

### Requirement: Scheduler SHALL expose deterministic resolution-source classification for terminalized tasks
Scheduler terminal records MUST retain normalized resolution source classification (`callback|reconcile_poll|timeout`) for query and diagnostics consumption.

#### Scenario: Task converges by reconcile poll
- **WHEN** reconcile poll path commits first terminal outcome
- **THEN** terminal record is classified with `resolution_source=reconcile_poll`

