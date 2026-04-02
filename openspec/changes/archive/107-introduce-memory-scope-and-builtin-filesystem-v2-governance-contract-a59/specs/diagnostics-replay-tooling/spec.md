## ADDED Requirements

### Requirement: Replay tooling SHALL validate memory governance fixtures
Diagnostics replay tooling MUST support memory governance fixture contracts:
- `memory_scope.v1`
- `memory_search.v1`
- `memory_lifecycle.v1`

Fixture validation MUST cover canonical fields for scope resolution, budget usage, search/rerank aggregates, and lifecycle action summaries.

#### Scenario: Memory governance fixtures match canonical output
- **WHEN** replay tooling processes valid memory governance fixtures and normalized output matches expected semantics
- **THEN** replay validation succeeds with deterministic pass result

#### Scenario: Memory governance fixture schema is malformed
- **WHEN** replay tooling receives malformed or unsupported memory governance fixture schema
- **THEN** replay validation fails fast with deterministic validation reason code

### Requirement: Replay drift classification SHALL include memory governance drift classes
Replay tooling MUST classify memory governance semantic drift using canonical classes:
- `scope_resolution_drift`
- `retrieval_quality_regression`
- `lifecycle_policy_drift`
- `recovery_consistency_drift`

#### Scenario: Replay detects retrieval quality regression
- **WHEN** replay output top-k/rerank metrics diverge from fixture expectation
- **THEN** replay validation fails with deterministic `retrieval_quality_regression` classification

#### Scenario: Replay detects lifecycle policy drift
- **WHEN** replay output lifecycle action differs from configured fixture policy
- **THEN** replay validation fails with deterministic `lifecycle_policy_drift` classification

### Requirement: Memory governance fixtures SHALL preserve mixed-fixture backward compatibility
Adding memory governance fixture support MUST NOT break validation for archived fixture suites.

#### Scenario: Mixed fixture suites execute in one gate flow
- **WHEN** replay gate runs historical fixtures and memory governance fixtures together
- **THEN** parser and validation remain backward compatible and deterministic for all suites

#### Scenario: Legacy fixture parser regression is introduced
- **WHEN** memory governance fixture support breaks legacy fixture parsing
- **THEN** replay validation fails and blocks merge
