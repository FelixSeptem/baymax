## ADDED Requirements

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
