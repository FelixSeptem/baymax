## ADDED Requirements

### Requirement: Task Board query state filter SHALL include awaiting-report state
Task Board query state filter MUST support `awaiting_report` as first-class lifecycle state while preserving existing deterministic pagination and cursor semantics.

#### Scenario: Query filters awaiting-report tasks
- **WHEN** caller submits Task Board query with `state=awaiting_report`
- **THEN** runtime returns only tasks in awaiting-report lifecycle with existing sort/page/cursor semantics unchanged

### Requirement: Task Board validation SHALL keep fail-fast behavior for unsupported states
Task Board query validation MUST continue fail-fast behavior for unsupported state values, and `awaiting_report` MUST be treated as supported state.

#### Scenario: Query uses unknown state value
- **WHEN** caller submits `state` outside supported set including `awaiting_report`
- **THEN** runtime returns validation error immediately

