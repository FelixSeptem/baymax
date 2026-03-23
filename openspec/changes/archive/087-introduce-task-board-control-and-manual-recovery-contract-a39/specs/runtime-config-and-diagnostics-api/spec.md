## ADDED Requirements

### Requirement: Runtime config SHALL expose task-board control governance with deterministic precedence
Runtime configuration MUST expose scheduler task-board control governance under `scheduler.task_board.control.*` with precedence `env > file > default`.

Minimum required fields:
- `scheduler.task_board.control.enabled`
- `scheduler.task_board.control.max_manual_retry_per_task`

Default values MUST be:
- `scheduler.task_board.control.enabled=false`
- `scheduler.task_board.control.max_manual_retry_per_task=3`

Invalid startup or hot-reload values MUST fail fast and MUST preserve previous valid snapshot.

#### Scenario: Runtime starts with default task-board control governance
- **WHEN** no task-board control overrides are configured
- **THEN** effective configuration resolves to default disabled state and default manual retry budget

#### Scenario: Hot reload provides invalid manual retry budget
- **WHEN** hot reload sets `scheduler.task_board.control.max_manual_retry_per_task<=0`
- **THEN** runtime rejects update and keeps previous active snapshot unchanged

### Requirement: Runtime diagnostics SHALL expose additive manual-control aggregates
Runtime diagnostics MUST expose additive manual-control summary fields while preserving compatibility-window semantics (`additive + nullable + default`).

Minimum required fields:
- `task_board_manual_control_total`
- `task_board_manual_control_success_total`
- `task_board_manual_control_rejected_total`
- `task_board_manual_control_idempotent_dedup_total`

Runtime diagnostics MUST additionally preserve action-level breakdown by canonical action and reason namespace (`scheduler.manual_cancel`, `scheduler.manual_retry`).

#### Scenario: Consumer queries diagnostics after manual control operations
- **WHEN** run/task lifecycle contains manual cancel and manual retry operations
- **THEN** diagnostics response includes additive manual-control aggregates and canonical reason-aligned breakdown

#### Scenario: Manual control events are replayed
- **WHEN** equivalent manual-control events are replayed for one run
- **THEN** logical aggregates remain stable after first ingestion
