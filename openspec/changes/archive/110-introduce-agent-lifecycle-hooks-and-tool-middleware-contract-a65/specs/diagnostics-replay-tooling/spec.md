## ADDED Requirements

### Requirement: A65 Replay Fixture Coverage
Diagnostics replay tooling MUST support A65 fixtures `hooks_middleware.v1`, `skill_discovery_sources.v1`, and `skill_preprocess_and_mapping.v1`.

#### Scenario: Fixture parsing compatibility
- **WHEN** replay runner loads A65 fixtures together with historical fixtures
- **THEN** parser MUST accept mixed versions and preserve deterministic normalized output

#### Scenario: Fixture schema validation
- **WHEN** required A65 fixture fields are missing or invalid
- **THEN** replay tooling MUST fail fast with deterministic schema mismatch classification

### Requirement: A65 Drift Classification
Replay tooling MUST classify hook/middleware/discovery/mapping drifts with canonical error taxonomy.

#### Scenario: Hook order drift classification
- **WHEN** hook execution order deviates from canonical sequence
- **THEN** replay MUST classify drift as `hooks_order_drift`

#### Scenario: Discovery source drift classification
- **WHEN** discovery source merge or dedup result deviates under identical input
- **THEN** replay MUST classify drift as `skill_discovery_source_drift`

#### Scenario: Bundle mapping drift classification
- **WHEN** prompt augmentation or whitelist mapping output deviates from configured policy
- **THEN** replay MUST classify drift as `skill_bundle_mapping_drift`
