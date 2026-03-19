# multi-agent-delayed-dispatch Specification

## Purpose
TBD - created by archiving change introduce-delayed-dispatch-not-before-contract-a13. Update Purpose after archive.
## Requirements
### Requirement: Runtime SHALL support task-level delayed dispatch via not_before
The runtime MUST support task-level delayed dispatch by an optional `not_before` field that controls earliest claim eligibility.

#### Scenario: Delayed task is submitted with future not_before
- **WHEN** a task is enqueued with `not_before` greater than current scheduler time
- **THEN** the task remains non-claimable until scheduler time reaches `not_before`

### Requirement: Delayed dispatch SHALL remain backward compatible by default
Delayed dispatch support MUST be backward compatible, and tasks without `not_before` MUST keep existing immediate-claim behavior.

#### Scenario: Task has no not_before
- **WHEN** task is enqueued without delayed dispatch field
- **THEN** claim eligibility follows existing queue semantics without added delay

### Requirement: Delayed dispatch SHALL preserve deterministic recovery semantics
Delayed dispatch state MUST be recoverable and deterministic across restart/replay so tasks are not claimed earlier than intended after restore.

#### Scenario: Scheduler restores delayed task from snapshot
- **WHEN** scheduler restarts and restores a queued task with future `not_before`
- **THEN** restored task remains non-claimable until restored scheduler time reaches `not_before`

