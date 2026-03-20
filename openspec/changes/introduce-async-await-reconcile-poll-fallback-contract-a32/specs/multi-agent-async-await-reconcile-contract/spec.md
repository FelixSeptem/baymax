## ADDED Requirements

### Requirement: Async-await lifecycle SHALL provide reconcile poll fallback for awaiting-report tasks
For tasks in `awaiting_report`, runtime MUST provide periodic reconcile polling (`status/result`) as a fallback convergence path when callback delivery is delayed or unavailable.

Reconcile polling MUST be gated by runtime config and MUST NOT affect non-`awaiting_report` task states.

#### Scenario: Callback is missing but poll reaches terminal
- **WHEN** async task remains in `awaiting_report` and callback is not delivered within one reconcile interval
- **THEN** reconcile poll path retrieves terminal outcome and converges task through terminal commit contract

### Requirement: Async-await reconcile SHALL enforce first-terminal-wins with conflict recording
When callback and poll both produce terminal outcomes, runtime MUST enforce `first_terminal_wins` and MUST record conflict diagnostics for later conflicting terminal events.

Conflicting later terminal events MUST NOT mutate already converged business terminal state.

#### Scenario: Callback and poll return different terminal outcomes
- **WHEN** callback commits terminal success first and a later poll returns terminal failure for the same task/attempt
- **THEN** business terminal state remains success and runtime records one terminal-conflict marker

### Requirement: Async-await reconcile SHALL apply keep-until-timeout behavior for not-found polling result
When reconcile poll returns `not_found`, runtime MUST keep task in `awaiting_report` until configured `report_timeout` is reached.

`not_found` result MUST NOT directly force terminal failure before timeout boundary.

#### Scenario: Reconcile poll repeatedly returns not-found
- **WHEN** task is in `awaiting_report` and successive reconcile polls return `not_found`
- **THEN** task remains `awaiting_report` until timeout path converges terminal outcome

### Requirement: Async-await reconcile SHALL preserve replay-idempotent convergence
Replayed equivalent callback/poll events and duplicated poll cycles for one task/attempt MUST converge idempotently to one logical terminal outcome.

#### Scenario: Recovery replays equivalent reconcile and callback events
- **WHEN** runtime replays duplicated reconcile and callback events for an already converged task
- **THEN** terminal state and additive aggregates remain stable without counter inflation

### Requirement: Async-await reconcile SHALL preserve Run/Stream and backend semantic parity
For equivalent logical requests and effective configuration, reconcile fallback behavior MUST remain semantically equivalent across Run/Stream paths and memory/file scheduler backends.

#### Scenario: Equivalent reconcile fallback in Run and Stream across backends
- **WHEN** equivalent async-await request executes with callback loss under Run and Stream on memory and file backends
- **THEN** terminal category, resolution source, and additive aggregates remain semantically equivalent

