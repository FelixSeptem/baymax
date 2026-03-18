## ADDED Requirements

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
