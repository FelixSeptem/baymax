## ADDED Requirements

### Requirement: Diagnostics replay SHALL cover corpus, Badcase, experiment, and feedback drift
The diagnostics replay tool MUST parse versioned fixtures for corpus acceptance, Badcase reproduction, metric/rubric drift, experiment aggregation idempotency/conflict, approval-missing feedback, and local/distributed parity. Replay classifications MUST be deterministic and historical fixtures MUST remain compatible.

#### Scenario: New success fixture replays
- **WHEN** a fixture contains valid corpus, correlated run outcome, experiment aggregate, and approved feedback context
- **THEN** replay succeeds with the expected normalized digest and no drift

#### Scenario: Historical fixture omits extension fields
- **WHEN** an existing eval/tracing fixture has no corpus or experiment extension fields
- **THEN** replay succeeds using documented defaults and does not report extension drift

#### Scenario: Drift fixture is detected
- **WHEN** a fixture changes corpus version, rubric digest, reproduction outcome, aggregate digest, or approval context
- **THEN** replay returns the corresponding stable drift classification
