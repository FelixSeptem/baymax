## ADDED Requirements

### Requirement: Diagnostics SHALL use a single writer path per semantic event
The runtime MUST ensure each diagnostic semantic event is persisted through exactly one write path. For the same run/skill semantic event, the system MUST NOT allow both business-direct persistence and event-recorder persistence to hit storage.

#### Scenario: Run completion is emitted and recorded
- **WHEN** a run completion semantic event is produced
- **THEN** diagnostics storage receives exactly one persisted run record for that semantic event

#### Scenario: Skill lifecycle event is emitted and recorded
- **WHEN** a skill lifecycle semantic event is produced
- **THEN** diagnostics storage receives exactly one persisted skill record for that semantic event

### Requirement: Diagnostics persistence SHALL enforce idempotency for run and skill records
The diagnostics write layer MUST compute and enforce stable idempotency keys for run and skill records so retries, replays, or concurrent duplicate submissions do not create multiple logical records.

#### Scenario: Duplicate run record submissions under retry
- **WHEN** the same run diagnostic payload is submitted multiple times due to retry
- **THEN** storage keeps one logical run record according to idempotency policy

#### Scenario: Concurrent duplicate skill record submissions
- **WHEN** multiple goroutines submit the same skill diagnostic record concurrently
- **THEN** storage preserves one logical skill record and returns deterministic write outcome

### Requirement: Diagnostics idempotency key generation SHALL be deterministic and testable
The runtime MUST define deterministic idempotency key generation rules for run and skill diagnostics and MUST cover them with unit tests for normal and edge cases.

#### Scenario: Equal semantic payloads generate identical keys
- **WHEN** two diagnostic payloads represent the same semantic run/skill event
- **THEN** generated idempotency keys are identical

#### Scenario: Distinct semantic payloads generate different keys
- **WHEN** two diagnostic payloads differ on key uniqueness fields
- **THEN** generated idempotency keys are different