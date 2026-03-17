# security-alert-delivery-governance-s4 Specification

## Purpose
TBD - created by archiving change harden-security-s4-callback-delivery-reliability. Update Purpose after archive.
## Requirements
### Requirement: Runtime SHALL deliver deny alerts via asynchronous bounded callback pipeline by default
Runtime MUST use asynchronous callback delivery as default mode for deny security alerts.
The delivery queue MUST be bounded and MUST apply `drop_old` when overflow occurs.

#### Scenario: Queue overflow drops oldest pending alert
- **WHEN** alert delivery queue is full and a new deny alert arrives
- **THEN** runtime drops the oldest queued alert, enqueues the new alert, and records drop diagnostics

#### Scenario: Async mode does not block deny decision path
- **WHEN** runtime emits a deny security event under async delivery mode
- **THEN** runtime returns the original deny decision outcome without waiting for callback completion

### Requirement: Runtime SHALL apply timeout and bounded retries for callback delivery
For each deny alert dispatch attempt, runtime MUST apply delivery timeout and retry policy with at most 3 attempts.
Retry behavior MUST use bounded backoff and MUST stop after max attempts.

#### Scenario: Callback timeout triggers retry within retry budget
- **WHEN** callback execution exceeds configured timeout on first attempt
- **THEN** runtime retries delivery using configured backoff until success or retry budget is exhausted

#### Scenario: Retry budget exhausted marks delivery failed
- **WHEN** callback delivery fails for all configured attempts
- **THEN** runtime records failed dispatch status with retry count and failure reason, without changing deny decision outcome

### Requirement: Runtime SHALL enforce Hystrix-style circuit breaker for callback sink
Runtime MUST implement callback circuit breaker with `closed|open|half_open` states.
In `open` state, runtime MUST fail fast for callback dispatch and skip callback invocation until sleep window elapses.

#### Scenario: Consecutive callback failures open circuit
- **WHEN** callback failures exceed configured circuit threshold in closed state
- **THEN** runtime transitions circuit to open state and marks subsequent dispatches as fast-fail

#### Scenario: Sleep window transitions circuit to half-open probe
- **WHEN** open-state sleep window expires
- **THEN** runtime allows probe dispatch in half-open state and transitions to closed on success or back to open on failure

### Requirement: Run and Stream SHALL keep S4 delivery semantics equivalent
For equivalent inputs and effective configuration, Run and Stream MUST produce semantically equivalent deny alert delivery behavior, including queue policy, retry semantics, and circuit state transitions.

#### Scenario: Equivalent deny alert delivery semantics in Run and Stream
- **WHEN** equivalent deny events are emitted in Run and Stream with same delivery config
- **THEN** delivery-mode, retry-count, and circuit-state diagnostics remain semantically equivalent

