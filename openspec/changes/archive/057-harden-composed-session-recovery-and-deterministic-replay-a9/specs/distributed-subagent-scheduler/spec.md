## ADDED Requirements

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
