# multi-agent-async-reporting Specification

## Purpose
TBD - created by archiving change introduce-async-agent-reporting-contract-a12. Update Purpose after archive.
## Requirements
### Requirement: Runtime SHALL provide non-blocking async submit contract for multi-agent paths
The runtime MUST provide an async submit contract that allows callers to submit remote tasks and return immediately without blocking on terminal wait.

#### Scenario: Caller submits async remote task
- **WHEN** caller invokes async submit on a supported multi-agent path
- **THEN** runtime returns an accepted task handle without waiting for terminal result

### Requirement: Runtime SHALL provide independent report sink contract
The runtime MUST provide an independent mailbox result-delivery contract for terminal outcome delivery, decoupled from synchronous waiting APIs.

Async terminal outcomes MUST be published as mailbox `result` envelopes with stable correlation and idempotency metadata.

#### Scenario: Terminal outcome is delivered through mailbox result envelope
- **WHEN** async task reaches terminal status
- **THEN** runtime publishes correlated mailbox `result` envelope even if caller never invokes wait API

### Requirement: Async reporting SHALL guarantee at-least-once delivery with idempotent convergence
Async report delivery MUST provide at-least-once semantics and MUST expose idempotent convergence behavior by stable report keys.

#### Scenario: Same terminal report is delivered multiple times
- **WHEN** report retry or replay causes duplicate terminal report deliveries
- **THEN** downstream aggregation converges to one logical terminal outcome

### Requirement: Async reporting SHALL classify delivery failure independently from business terminal status
Async report delivery failures MUST be classified independently and MUST NOT mutate already decided business terminal status.

#### Scenario: Report sink delivery fails after business success
- **WHEN** task execution reaches business terminal success and report sink delivery fails
- **THEN** task business terminal status remains success and delivery failure is recorded separately

### Requirement: Async reporting SHALL support bounded retry with exponential backoff and jitter
Async report delivery MUST support bounded retries and use exponential backoff with bounded jitter before retry attempts.

#### Scenario: Report sink transient error triggers retry
- **WHEN** report sink returns retryable delivery error
- **THEN** runtime retries delivery using configured bounded exponential backoff and jitter

### Requirement: Legacy direct report-sink API SHALL be deprecated
Legacy direct report-sink contract from pre-mailbox async path MUST be marked deprecated and MUST NOT be the canonical contract surface.

#### Scenario: Maintainer validates async contract entrypoint
- **WHEN** maintainer reviews async reporting mainline contract
- **THEN** mailbox result delivery is canonical and legacy direct report-sink path is documented as deprecated

### Requirement: Async reporting SHALL converge through awaiting-report lifecycle boundary
Async terminal reports MUST be committed only against scheduler attempts in `awaiting_report` lifecycle boundary and MUST preserve existing terminal commit idempotency contract.

#### Scenario: Report arrives for awaiting-report attempt
- **WHEN** async report is delivered for a task currently in `awaiting_report`
- **THEN** runtime commits terminal outcome through shared commit contract and keeps idempotent convergence behavior

### Requirement: Async reporting SHALL enforce late-report drop-and-record policy
When async report arrives after task has already converged to terminal state, runtime MUST not mutate terminal business outcome and MUST record late-report diagnostics.

#### Scenario: Report arrives after task already terminal
- **WHEN** async report is delivered for task no longer in `awaiting_report`
- **THEN** runtime treats the report as late, keeps terminal business status unchanged, and records drop-and-record diagnostics marker

### Requirement: Async reporting replay SHALL keep additive counters stable
Replayed equivalent async reports for the same task and attempt MUST keep async summary aggregates stable after first logical ingestion.

#### Scenario: Duplicate async report replay under recovery
- **WHEN** recovery path replays equivalent async reports
- **THEN** runtime keeps additive async-report counters and terminal summaries stable without inflation

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

