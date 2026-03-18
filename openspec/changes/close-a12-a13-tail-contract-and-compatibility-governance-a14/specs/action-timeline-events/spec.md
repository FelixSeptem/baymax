## ADDED Requirements

### Requirement: Action timeline shared contract SHALL enforce async and delayed canonical reason completeness
Action timeline contract checks MUST enforce combined canonical reason completeness for A12 async reporting and A13 delayed dispatch in a single shared gate.

Minimum required canonical reason set MUST include:
- `a2a.async_submit`
- `a2a.async_report_deliver`
- `a2a.async_report_retry`
- `a2a.async_report_dedup`
- `a2a.async_report_drop`
- `scheduler.delayed_enqueue`
- `scheduler.delayed_wait`
- `scheduler.delayed_ready`

#### Scenario: Delayed reason is missing from timeline contract snapshot
- **WHEN** shared contract validation runs with delayed reason taxonomy drift
- **THEN** validation fails with explicit missing-reason classification and blocks merge

### Requirement: Timeline SHALL preserve scheduler correlation consistency for delayed and async interop paths
Timeline events on scheduler-managed delayed and async-reporting interop paths MUST include required correlation fields (`run_id`, `task_id`, `attempt_id`) where applicable.

#### Scenario: Delayed task later emits async reporting transitions
- **WHEN** one task traverses delayed dispatch and async reporting transitions
- **THEN** timeline records keep deterministic scheduler correlation keys across all required events

### Requirement: Timeline matrix semantics SHALL remain Run and Stream equivalent across sync async delayed modes
For equivalent requests executed through sync, async, and delayed communication modes, timeline reason and terminal status semantics MUST remain equivalent between Run and Stream paths for required matrix rows.

#### Scenario: Matrix row executes in Run and Stream
- **WHEN** one required cross-mode matrix case is executed in both Run and Stream
- **THEN** timeline reason taxonomy and terminal status category remain semantically equivalent
