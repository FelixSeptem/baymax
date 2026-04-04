## ADDED Requirements

### Requirement: Replay Tooling SHALL Support A68 Realtime Fixture
Diagnostics replay tooling MUST support versioned fixture contract `realtime_event_protocol.v1`.

Fixture validation MUST cover at minimum:
- canonical realtime event taxonomy mapping
- sequence monotonicity and gap detection semantics
- interrupt/resume outcome semantics
- idempotent dedup semantics
- Run/Stream parity markers

#### Scenario: Replay validates canonical A68 fixture
- **WHEN** replay tooling processes valid `realtime_event_protocol.v1` fixture and normalized output matches canonical expectation
- **THEN** replay validation succeeds with deterministic pass result

#### Scenario: Replay receives malformed A68 fixture schema
- **WHEN** replay tooling receives malformed or unsupported `realtime_event_protocol.v1` schema
- **THEN** replay validation fails fast with deterministic validation reason code

### Requirement: Replay Drift Classification SHALL Include A68 Canonical Classes
Replay tooling MUST classify A68 semantic drift using canonical classes:
- `realtime_event_order_drift`
- `realtime_interrupt_semantic_drift`
- `realtime_resume_semantic_drift`
- `realtime_idempotency_drift`
- `realtime_sequence_gap_drift`

#### Scenario: Replay detects event-order drift
- **WHEN** replay output event order semantics diverge from fixture expectation
- **THEN** replay validation fails with deterministic `realtime_event_order_drift` classification

#### Scenario: Replay detects resume semantic drift
- **WHEN** replay output resume outcome semantics diverge from fixture expectation
- **THEN** replay validation fails with deterministic `realtime_resume_semantic_drift` classification

### Requirement: A68 Fixture Support SHALL Preserve Mixed-Fixture Backward Compatibility
Adding `realtime_event_protocol.v1` support MUST NOT break validation for historical fixture suites.

#### Scenario: Mixed fixture suites run in one gate flow
- **WHEN** replay gate executes archived fixtures together with `realtime_event_protocol.v1`
- **THEN** parser and validation remain deterministic without regression

#### Scenario: Historical parser regression is introduced
- **WHEN** A68 fixture support breaks legacy fixture parsing
- **THEN** replay validation fails and blocks merge

