## ADDED Requirements

### Requirement: Replay tooling SHALL validate arbitration-version governance fixtures
Diagnostics replay tooling MUST support arbitration-version governance fixtures and MUST classify version-related semantic drift deterministically.

Drift classes MUST include at minimum:
- `version_mismatch`
- `unsupported_version`
- `cross_version_semantic_drift`

#### Scenario: Replay fixture matches expected version-governance output
- **WHEN** fixture expected requested/effective/source/policy output matches normalized actual output
- **THEN** replay validation succeeds with deterministic pass result

#### Scenario: Replay fixture detects unsupported-version drift
- **WHEN** actual output lacks expected unsupported-version classification
- **THEN** replay validation fails with deterministic `unsupported_version` drift classification

### Requirement: Replay tooling SHALL preserve backward-compatible fixture validation
Replay tooling MUST continue validating previously archived fixture schemas while adding version-governance fixture support.

#### Scenario: A47/A48 fixture validation runs with A50 tooling
- **WHEN** replay executes archived fixture suites and A50 fixture suites in one gate flow
- **THEN** archived fixture assertions remain valid and no cross-version parser regression is introduced
