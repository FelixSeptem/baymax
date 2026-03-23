## ADDED Requirements

### Requirement: Scheduler SHALL expose manual task-board control entrypoint
Scheduler MUST expose a manual control entrypoint for task-board actions without bypassing existing state machine invariants.

At minimum the entrypoint MUST support:
- `cancel`
- `retry_terminal`

The entrypoint MUST preserve existing enqueue/claim/heartbeat/requeue/commit semantics for unaffected tasks.

#### Scenario: Host executes scheduler manual control
- **WHEN** caller invokes scheduler task-board control entrypoint with valid action and target task
- **THEN** scheduler applies deterministic state transition and returns normalized control result payload

#### Scenario: Host executes control on unsupported state
- **WHEN** caller invokes valid action for disallowed task state
- **THEN** scheduler fails fast and does not mutate task/attempt runtime state

### Requirement: Scheduler manual control SHALL emit canonical timeline reasons
Scheduler manual control paths MUST emit canonical scheduler reason codes:
- `scheduler.manual_cancel`
- `scheduler.manual_retry`

Reason semantics MUST remain namespaced under `scheduler.*` and MUST be consumable by existing timeline/diagnostics aggregation.

#### Scenario: Manual cancel emits canonical reason
- **WHEN** scheduler successfully applies manual cancel action
- **THEN** emitted timeline includes reason `scheduler.manual_cancel` with task/attempt correlation metadata

#### Scenario: Manual retry emits canonical reason
- **WHEN** scheduler successfully applies manual retry action
- **THEN** emitted timeline includes reason `scheduler.manual_retry` with task/attempt correlation metadata
