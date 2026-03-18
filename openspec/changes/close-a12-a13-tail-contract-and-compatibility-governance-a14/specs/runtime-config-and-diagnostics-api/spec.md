## ADDED Requirements

### Requirement: Runtime diagnostics SHALL expose unified A12/A13 additive summary compatibility contract
Runtime diagnostics contract MUST define unified compatibility semantics for async-reporting and delayed-dispatch additive summary fields under one compatibility window.

Minimum required additive fields:
- `a2a_async_report_total`
- `a2a_async_report_failed`
- `a2a_async_report_retry_total`
- `a2a_async_report_dedup_total`
- `scheduler_delayed_task_total`
- `scheduler_delayed_claim_total`
- `scheduler_delayed_wait_ms_p95`

#### Scenario: Consumer reads mixed async and delayed run summary
- **WHEN** run summary includes both async-reporting and delayed-dispatch aggregates
- **THEN** additive fields are present with documented names and legacy fields remain semantically unchanged

### Requirement: Diagnostics parser SHALL apply additive nullable default semantics for A12/A13 fields
Diagnostics parser and consumer-facing API contracts MUST preserve `additive + nullable + default` behavior for A12/A13 additive fields.

Compatibility rules MUST include:
- missing additive fields resolve to documented default values,
- unknown future additive fields are safely ignored,
- pre-existing field semantics remain unchanged.

#### Scenario: Legacy parser reads run summary with new additive fields
- **WHEN** parser built against older schema reads a newer run summary payload
- **THEN** parser succeeds without semantic regression on pre-existing fields

### Requirement: Combined async and delayed diagnostics aggregates SHALL remain replay-idempotent
For one run, repeated ingestion/replay of equivalent async-reporting and delayed-dispatch events MUST NOT inflate logical aggregates.

#### Scenario: Replay submits duplicated async and delayed events
- **WHEN** recorder replays equivalent async and delayed events for the same run
- **THEN** diagnostics aggregate counters remain stable after first logical ingestion
