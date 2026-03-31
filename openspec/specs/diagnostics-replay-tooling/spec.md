# diagnostics-replay-tooling Specification

## Purpose
TBD - created by archiving change improve-dx-d1-api-reference-and-diagnostics-replay-e7. Update Purpose after archive.
## Requirements
### Requirement: Replay tooling SHALL accept diagnostics JSON as primary input
Diagnostics replay tooling MUST accept diagnostics JSON artifacts as input and MUST parse required timeline fields without requiring live runtime connectivity.

On malformed input, tooling MUST fail fast with deterministic machine-readable reason codes.

#### Scenario: Replay runs with valid diagnostics JSON
- **WHEN** tooling receives JSON payload containing supported diagnostics timeline schema
- **THEN** tooling produces replay output without requiring runtime API access

#### Scenario: Replay runs with malformed JSON
- **WHEN** tooling receives malformed JSON or missing required fields
- **THEN** tooling exits with deterministic validation reason code and no partial success status

### Requirement: Replay output SHALL support minimal timeline summary mode
Replay tooling MUST provide a minimal output mode that includes `phase`, `status`, `reason`, and `timestamp` fields, plus minimal correlation identifiers required for traceability.

#### Scenario: Minimal replay mode requested
- **WHEN** caller invokes replay in default minimal mode
- **THEN** output contains only required summary fields and deterministic ordering by replay sequence

#### Scenario: Missing optional details in source payload
- **WHEN** diagnostics source lacks optional extended fields
- **THEN** minimal replay output remains valid and omits unavailable optional fields without failure

### Requirement: Replay contract SHALL be regression-testable
The repository MUST provide contract tests for replay tooling using fixed sample inputs covering success and failure paths, and expected outputs/error codes MUST remain stable unless intentionally versioned.

#### Scenario: CI executes replay contract test suite
- **WHEN** standard test flow runs replay contract tests
- **THEN** expected normalized output snapshots and deterministic reason codes match version-controlled expectations

### Requirement: Replay tooling SHALL support readiness-timeout-health composite fixture mode
Diagnostics replay tooling MUST support composite fixture mode for readiness-timeout-health cross-domain validation.

Composite mode MUST:
- accept versioned fixture payload,
- emit normalized comparison output for canonical semantic fields,
- return deterministic error classification on fixture/schema mismatch.

#### Scenario: Composite fixture is replayed successfully
- **WHEN** tooling receives valid A47 composite fixture input
- **THEN** tooling emits deterministic normalized output with canonical semantic fields

#### Scenario: Composite fixture schema is invalid
- **WHEN** tooling receives malformed or unsupported fixture version
- **THEN** tooling fails fast with deterministic validation reason code

### Requirement: Replay tooling SHALL validate cross-domain primary-reason arbitration fixtures
Diagnostics replay tooling MUST support cross-domain primary-reason arbitration fixtures and MUST return deterministic drift classification on mismatch.

Drift classes MUST include at minimum:
- precedence drift
- tie-break drift
- taxonomy drift

#### Scenario: Replay fixture matches canonical arbitration output
- **WHEN** fixture expected arbitration output matches normalized actual output
- **THEN** replay validation succeeds with deterministic pass result

#### Scenario: Replay fixture detects precedence drift
- **WHEN** actual primary reason violates canonical precedence order
- **THEN** replay validation fails with deterministic precedence-drift classification

### Requirement: Replay tooling SHALL validate arbitration explainability fixtures
Diagnostics replay tooling MUST validate arbitration explainability fixtures, including secondary reason ordering, bounded count, remediation hint taxonomy, and rule-version stability.

Replay drift classes MUST include at minimum:
- `secondary_order_drift`
- `secondary_count_drift`
- `hint_taxonomy_drift`
- `rule_version_drift`

#### Scenario: Explainability fixture matches canonical output
- **WHEN** expected explainability fixture matches normalized replay output
- **THEN** replay validation passes deterministically

#### Scenario: Explainability fixture detects secondary-order drift
- **WHEN** replay output secondary reason ordering differs from canonical expectation
- **THEN** replay validation fails with deterministic `secondary_order_drift` classification

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

### Requirement: Replay tooling SHALL validate sandbox governance fixtures
Diagnostics replay tooling MUST support sandbox governance fixture validation with deterministic normalized output and drift classification.

Sandbox drift classes MUST include at minimum:
- `sandbox_policy_drift`
- `sandbox_fallback_drift`
- `sandbox_timeout_drift`
- `sandbox_capability_drift`
- `sandbox_resource_policy_drift`
- `sandbox_session_lifecycle_drift`

#### Scenario: Sandbox fixture matches canonical output
- **WHEN** replay tooling evaluates valid sandbox fixture and normalized output matches expected semantics
- **THEN** validation passes deterministically

#### Scenario: Sandbox fixture detects fallback drift
- **WHEN** replay output fallback behavior differs from canonical fixture expectation
- **THEN** validation fails with deterministic `sandbox_fallback_drift` classification

#### Scenario: Sandbox fixture detects capability drift
- **WHEN** replay output shows required capability satisfaction semantics diverging from canonical fixture
- **THEN** validation fails with deterministic `sandbox_capability_drift` classification

#### Scenario: Sandbox fixture detects session lifecycle drift
- **WHEN** replay output for per-call/per-session lifecycle semantics diverges from canonical fixture
- **THEN** validation fails with deterministic `sandbox_session_lifecycle_drift` classification

### Requirement: Replay tooling SHALL validate sandbox rollout-governance fixtures
Diagnostics replay tooling MUST support sandbox rollout-governance fixture validation using versioned fixture contract `a52.v1`.

Fixture validation MUST cover canonical fields:
- rollout phase
- health budget status
- capacity action
- freeze state and reason

#### Scenario: A52 rollout fixture matches canonical output
- **WHEN** replay tooling processes valid `a52.v1` fixture and actual output matches canonical expectation
- **THEN** replay validation succeeds with deterministic pass result

#### Scenario: A52 rollout fixture schema is malformed
- **WHEN** replay tooling receives malformed or unsupported `a52.v1` fixture schema
- **THEN** replay validation fails fast with deterministic validation reason code

### Requirement: Replay drift classification SHALL include rollout-governance drift classes
Replay tooling MUST classify rollout-governance semantic drift using canonical classes:
- `sandbox_rollout_phase_drift`
- `sandbox_health_budget_drift`
- `sandbox_capacity_action_drift`
- `sandbox_freeze_state_drift`

#### Scenario: Replay detects rollout phase drift
- **WHEN** actual rollout phase differs from expected fixture phase
- **THEN** replay validation fails with deterministic `sandbox_rollout_phase_drift` classification

#### Scenario: Replay detects capacity action drift
- **WHEN** actual capacity action differs from expected fixture action
- **THEN** replay validation fails with deterministic `sandbox_capacity_action_drift` classification

### Requirement: Replay tooling SHALL preserve backward compatibility for A51 fixtures
Adding A52 fixture support MUST NOT break existing A51 and earlier replay fixture validations.

#### Scenario: A51 and A52 fixtures run in single gate flow
- **WHEN** replay gate executes mixed fixture suites containing A51 and A52 fixture versions
- **THEN** both fixture generations are validated deterministically without parser regression

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

