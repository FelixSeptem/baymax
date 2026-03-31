## ADDED Requirements

### Requirement: Replay tooling SHALL validate observability export and bundle fixtures
Diagnostics replay tooling MUST support observability export and diagnostics bundle fixture validation using versioned fixture contract `observability.v1`.

Fixture validation MUST cover canonical fields at minimum:
- export profile and status,
- export degradation and failure reason taxonomy,
- bundle schema version and generation result,
- bundle redaction and gate-fingerprint metadata.

#### Scenario: Observability fixture matches canonical output
- **WHEN** replay tooling processes valid `observability.v1` fixture and actual output matches expected normalized semantics
- **THEN** replay validation succeeds with deterministic pass result

#### Scenario: Observability fixture schema is malformed
- **WHEN** replay tooling receives malformed or unsupported `observability.v1` fixture schema
- **THEN** replay validation fails fast with deterministic validation reason code

### Requirement: Replay drift classification SHALL include observability and bundle drift classes
Replay tooling MUST classify observability and bundle semantic drift with canonical classes:
- `observability_export_profile_drift`
- `observability_export_status_drift`
- `observability_export_reason_drift`
- `diagnostics_bundle_schema_drift`
- `diagnostics_bundle_redaction_drift`
- `diagnostics_bundle_fingerprint_drift`

#### Scenario: Replay detects export status drift
- **WHEN** actual export status semantics differ from fixture expectation
- **THEN** replay validation fails with deterministic `observability_export_status_drift` classification

#### Scenario: Replay detects bundle redaction drift
- **WHEN** bundle output includes non-redacted secret-like fields compared with fixture expectation
- **THEN** replay validation fails with deterministic `diagnostics_bundle_redaction_drift` classification

### Requirement: Replay tooling SHALL preserve backward compatibility for pre-A55 fixtures
Adding `observability.v1` support MUST NOT break validation of existing fixture suites.

#### Scenario: Mixed fixture suites execute in one replay gate flow
- **WHEN** replay gate runs archived fixtures and `observability.v1` fixtures together
- **THEN** parser and validation remain backward compatible and deterministic for all suites
