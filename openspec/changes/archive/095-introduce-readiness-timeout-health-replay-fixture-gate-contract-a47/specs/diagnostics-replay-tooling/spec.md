## ADDED Requirements

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
