## ADDED Requirements

### Requirement: Replay tooling SHALL support memory fixture contract version memory v1
Diagnostics replay tooling MUST support versioned memory fixture contract `memory.v1`.

`memory.v1` fixture validation MUST cover at minimum:
- effective memory mode,
- provider and profile,
- operation counters,
- fallback classification,
- canonical reason codes.

#### Scenario: Replay validates canonical memory v1 fixture
- **WHEN** tooling replays valid `memory.v1` fixture with expected canonical output
- **THEN** replay validation succeeds with deterministic pass result

#### Scenario: Replay receives malformed memory fixture version
- **WHEN** tooling receives malformed or unsupported memory fixture schema
- **THEN** replay fails fast with deterministic validation reason code

### Requirement: Replay drift classification SHALL include canonical memory drift classes
Replay tooling MUST classify memory semantic drift using canonical classes:
- `memory_mode_drift`
- `memory_profile_drift`
- `memory_contract_version_drift`
- `memory_fallback_drift`
- `memory_error_taxonomy_drift`
- `memory_operation_aggregate_drift`

#### Scenario: Replay detects fallback behavior drift
- **WHEN** replay output fallback behavior differs from fixture expectation
- **THEN** replay validation fails with deterministic `memory_fallback_drift` classification

#### Scenario: Replay detects operation aggregate drift
- **WHEN** equivalent replay input produces non-equivalent memory operation aggregates
- **THEN** replay validation fails with deterministic `memory_operation_aggregate_drift` classification

### Requirement: Memory replay fixture support SHALL preserve backward-compatible mixed-fixture validation
Adding `memory.v1` support MUST NOT break validation of previously archived fixture versions.

#### Scenario: Mixed fixture suite includes A52 and memory v1 fixtures
- **WHEN** replay gate runs fixture suite containing historical fixtures and `memory.v1`
- **THEN** all fixture generations are parsed and validated deterministically without regression

#### Scenario: Historical fixture parser regression is introduced
- **WHEN** memory fixture support change breaks legacy fixture parsing
- **THEN** replay validation fails and blocks merge
