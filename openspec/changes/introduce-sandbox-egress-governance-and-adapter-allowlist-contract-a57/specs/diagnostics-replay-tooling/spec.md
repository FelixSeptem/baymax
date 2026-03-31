## ADDED Requirements

### Requirement: Replay tooling SHALL support sandbox egress fixture contract version sandbox_egress.v1
Diagnostics replay tooling MUST support versioned fixture contract `sandbox_egress.v1`.

Fixture validation MUST cover at minimum:
- egress action decision,
- egress policy source,
- violation classification,
- allowlist decision and primary code.

#### Scenario: Replay validates canonical sandbox_egress.v1 fixture
- **WHEN** tooling processes valid `sandbox_egress.v1` fixture and normalized output matches expectation
- **THEN** replay validation succeeds with deterministic pass result

#### Scenario: Replay receives malformed sandbox_egress.v1 payload
- **WHEN** tooling receives malformed or unsupported `sandbox_egress.v1` fixture
- **THEN** replay fails fast with deterministic validation reason code

### Requirement: Replay drift classification SHALL include egress and allowlist drift classes
Replay tooling MUST classify A57 semantic drift using canonical classes:
- `sandbox_egress_action_drift`
- `sandbox_egress_policy_source_drift`
- `sandbox_egress_violation_taxonomy_drift`
- `adapter_allowlist_decision_drift`
- `adapter_allowlist_taxonomy_drift`

#### Scenario: Replay detects egress action drift
- **WHEN** replay output egress action differs from fixture expectation
- **THEN** validation fails with deterministic `sandbox_egress_action_drift` classification

#### Scenario: Replay detects allowlist taxonomy drift
- **WHEN** replay output allowlist reason taxonomy differs from fixture expectation
- **THEN** validation fails with deterministic `adapter_allowlist_taxonomy_drift` classification

### Requirement: A57 replay support SHALL preserve mixed-fixture backward compatibility
Adding `sandbox_egress.v1` support MUST NOT break validation of historical fixture versions.

#### Scenario: Mixed fixture suite includes A52 sandbox.v1 memory.v1 react.v1 and sandbox_egress.v1
- **WHEN** replay gate runs mixed fixture suite across multiple versions
- **THEN** all fixtures are validated deterministically without parser regression

#### Scenario: A57 fixture support breaks historical parser behavior
- **WHEN** tooling update for A57 introduces parser regression for archived fixtures
- **THEN** replay validation fails and blocks merge
