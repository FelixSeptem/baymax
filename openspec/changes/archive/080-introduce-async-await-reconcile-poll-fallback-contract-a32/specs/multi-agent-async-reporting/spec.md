## ADDED Requirements

### Requirement: Async reporting SHALL support callback-plus-reconcile dual-source terminal convergence
Async reporting contract MUST treat callback and reconcile poll as dual terminal sources for `awaiting_report` tasks.

Callback path remains valid canonical delivery path, and reconcile poll path MUST act as fallback convergence path.

#### Scenario: Callback is unavailable and reconcile poll converges terminal
- **WHEN** async report callback is not delivered but remote status/result is available by reconcile polling
- **THEN** runtime converges task to terminal state without requiring callback delivery success

### Requirement: Async reporting terminal arbitration SHALL enforce first-terminal-wins semantics
When callback and reconcile poll both provide terminal results, arbitration MUST enforce `first_terminal_wins` and MUST classify later conflicting source as conflict evidence only.

#### Scenario: Reconcile commits first and callback arrives later with different terminal
- **WHEN** reconcile poll commits terminal failed and callback later reports terminal success for same task
- **THEN** terminal status remains failed and callback event is recorded as conflict without terminal overwrite

### Requirement: Async reporting failure classification SHALL remain independent from business terminal convergence
Delivery-path failures in callback and poll fallback MUST be recorded as delivery/reconcile diagnostics and MUST NOT mutate already decided business terminal status.

#### Scenario: Poll fallback converges success while callback delivery keeps failing
- **WHEN** callback path reports retryable delivery errors after poll already committed terminal success
- **THEN** business terminal success remains unchanged and delivery failure is recorded independently

