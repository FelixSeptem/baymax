## ADDED Requirements

### Requirement: Action timeline SHALL encode recovery-boundary transitions using existing namespaces
Action timeline events for recovery-boundary behavior MUST use existing `recovery.*` and `scheduler.*` namespaces and MUST NOT add a new top-level namespace.

Minimum required reason coverage:
- `recovery.restore`
- `recovery.replay`
- `recovery.conflict`
- `scheduler.requeue`
- `scheduler.retry_backoff`

#### Scenario: Recovery timeout reentry is exhausted
- **WHEN** runtime exhausts timeout reentry budget under recovery boundary policy
- **THEN** timeline emits canonical recovery/scheduler reasons without new top-level namespace introduction

### Requirement: Recovery-boundary timeline events SHALL preserve required correlations
Recovery-boundary timeline events MUST preserve required correlation fields including `run_id`, `task_id`, and `attempt_id` on scheduler-managed paths.

#### Scenario: Recovery boundary transition occurs on scheduler-managed task
- **WHEN** restored task transitions through boundary-controlled reentry or terminal failure
- **THEN** timeline event includes required correlation metadata for deterministic replay auditing

### Requirement: Recovery-boundary timeline semantics SHALL preserve Run Stream equivalence
For equivalent recovery-boundary scenarios, Run and Stream timeline reason/status semantics MUST remain equivalent.

#### Scenario: Equivalent recovery-boundary replay via Run and Stream
- **WHEN** equivalent recovery-boundary replay scenario runs through Run and Stream
- **THEN** timeline reason and terminal status semantics remain equivalent
