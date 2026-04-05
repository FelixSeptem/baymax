## ADDED Requirements

### Requirement: Replay Tooling SHALL Support A63 Naming Migration Compatibility
Diagnostics replay tooling MUST support mixed fixtures containing semantic-primary and legacy naming variants for A63-governed fields without changing canonical replay semantics.

#### Scenario: Mixed fixture includes semantic and legacy naming variants
- **WHEN** replay validation executes fixtures that contain both semantic and legacy variants for the same logical field
- **THEN** replay result MUST remain deterministic and semantically equivalent

#### Scenario: Naming migration changes replay classification
- **WHEN** replay output classification differs only due naming migration without semantic behavior change
- **THEN** validation MUST classify as migration compatibility issue and block merge until mapping is corrected

### Requirement: Replay Contract Suites SHALL Guard Against Naming-Only Semantic Drift
Replay suites for A63 scope MUST detect unintended semantic drift introduced by naming consolidation, including parser compatibility and aggregate idempotency regressions.

#### Scenario: Parser compatibility regression breaks historical fixture
- **WHEN** historical fixture cannot be parsed after naming consolidation
- **THEN** replay contract suite MUST fail fast and block merge

#### Scenario: Aggregation idempotency remains stable across naming migration
- **WHEN** equivalent fixture set is replayed before and after naming migration
- **THEN** canonical aggregates and drift taxonomy outcomes MUST remain equivalent

