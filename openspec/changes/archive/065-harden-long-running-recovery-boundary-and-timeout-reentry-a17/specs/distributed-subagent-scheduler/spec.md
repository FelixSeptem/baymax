## ADDED Requirements

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
