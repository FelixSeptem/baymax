## ADDED Requirements

### Requirement: Sandbox deny alerts SHALL use S4 delivery governance semantics
Sandbox deny alerts MUST use the same S4 managed delivery semantics as existing deny events, including async bounded queue, retry budget, and circuit-breaker behavior.

#### Scenario: Sandbox deny alert follows async queue and retry policy
- **WHEN** runtime emits sandbox deny event under async delivery mode
- **THEN** alert dispatch uses bounded queue, timeout, retry, and records deterministic delivery diagnostics

#### Scenario: Sandbox deny alert under circuit-open state
- **WHEN** callback circuit is open while sandbox deny alert is dispatched
- **THEN** delivery fast-fails with canonical circuit-open diagnostics and original deny decision remains unchanged

### Requirement: Run and Stream SHALL preserve sandbox-alert delivery semantic equivalence
For equivalent inputs and effective configuration, Run and Stream MUST produce semantically equivalent sandbox-alert delivery outcomes.

#### Scenario: Equivalent sandbox deny alert delivery in Run and Stream
- **WHEN** equivalent Run and Stream requests produce sandbox deny alerts
- **THEN** delivery mode, retry count, queue-drop semantics, and circuit-state diagnostics remain semantically equivalent

