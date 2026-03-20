## MODIFIED Requirements

### Requirement: Runtime SHALL provide independent report sink contract
The runtime MUST provide an independent mailbox result-delivery contract for terminal outcome delivery, decoupled from synchronous waiting APIs.

Async terminal outcomes MUST be published as mailbox `result` envelopes with stable correlation and idempotency metadata.

#### Scenario: Terminal outcome is delivered through mailbox result envelope
- **WHEN** async task reaches terminal status
- **THEN** runtime publishes correlated mailbox `result` envelope even if caller never invokes wait API

## ADDED Requirements

### Requirement: Legacy direct report-sink API SHALL be deprecated
Legacy direct report-sink contract from pre-mailbox async path MUST be marked deprecated and MUST NOT be the canonical contract surface.

#### Scenario: Maintainer validates async contract entrypoint
- **WHEN** maintainer reviews async reporting mainline contract
- **THEN** mailbox result delivery is canonical and legacy direct report-sink path is documented as deprecated
