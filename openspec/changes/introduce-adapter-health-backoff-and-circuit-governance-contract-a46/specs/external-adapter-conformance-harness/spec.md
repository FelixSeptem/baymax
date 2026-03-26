## ADDED Requirements

### Requirement: Adapter conformance harness SHALL include health-governance matrix suites
External adapter conformance harness MUST include adapter-health governance matrix suites as blocking validations.

The matrix MUST cover:
- backoff throttling behavior under repeated failure,
- circuit transition determinism (`closed|open|half_open`),
- half-open recovery and reopen behavior,
- strict/non-strict readiness mapping parity for required/optional adapters,
- replay-idempotent governance diagnostics aggregates.

#### Scenario: Harness executes health-governance matrix for one adapter fixture
- **WHEN** conformance harness runs adapter health suites
- **THEN** backoff/circuit/readiness/governance-observability assertions execute as required checks

#### Scenario: Governance semantics drift from canonical matrix
- **WHEN** harness detects state-transition or readiness-classification drift
- **THEN** conformance validation fails and returns non-zero status
