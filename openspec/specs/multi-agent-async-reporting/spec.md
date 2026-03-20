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

