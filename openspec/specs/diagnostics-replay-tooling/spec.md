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

