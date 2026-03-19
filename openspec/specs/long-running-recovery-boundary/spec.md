# long-running-recovery-boundary Specification

## Purpose
TBD - created by archiving change harden-long-running-recovery-boundary-and-timeout-reentry-a17. Update Purpose after archive.
## Requirements
### Requirement: Recovery boundary SHALL enforce next_attempt_only semantics for resumed long-running tasks
Long-running recovery boundary MUST apply policy updates and execution resumption on next-attempt boundaries only.

#### Scenario: Recovery restores in-flight task attempt
- **WHEN** runtime restores an in-flight long-running task after restart
- **THEN** resumed execution keeps current attempt semantics unchanged and applies updated controls on next attempt only

### Requirement: Recovery boundary SHALL enforce no_rewind semantics for terminal tasks
Recovery boundary MUST prevent rewind of already terminal tasks and MUST NOT re-execute terminal steps during resume.

#### Scenario: Resume loads snapshot with terminal and in-flight tasks
- **WHEN** recovery restore processes a snapshot containing terminal tasks
- **THEN** terminal tasks remain terminal and are never scheduled again

### Requirement: Timeout reentry SHALL follow single_reentry_then_fail policy
Recovery boundary MUST allow at most one timeout-driven reentry per task and MUST fail task deterministically after reentry budget is exhausted.

#### Scenario: Task times out repeatedly after resume
- **WHEN** task hits timeout once, reenters, and times out again
- **THEN** runtime marks terminal failure and does not attempt further reentry

### Requirement: Recovery boundary policy SHALL be active only when recovery is enabled
Recovery boundary governance MUST be active when and only when `recovery.enabled=true`.

#### Scenario: Recovery is disabled
- **WHEN** runtime runs with `recovery.enabled=false`
- **THEN** long-running recovery boundary logic is not activated and baseline non-recovery path remains unchanged

