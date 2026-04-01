## ADDED Requirements

### Requirement: Replay tooling SHALL validate policy precedence fixtures
Diagnostics replay tooling MUST support policy precedence fixture validation using versioned fixture contract `policy_stack.v1`.

Fixture validation MUST cover at minimum:
- winner stage
- deny source
- decision path
- tie-break reason

#### Scenario: Policy precedence fixture matches canonical output
- **WHEN** replay tooling processes valid `policy_stack.v1` fixture and normalized output matches expected semantics
- **THEN** replay validation succeeds with deterministic pass result

#### Scenario: Policy precedence fixture schema is malformed
- **WHEN** replay tooling receives malformed or unsupported `policy_stack.v1` fixture schema
- **THEN** replay validation fails fast with deterministic validation reason code

### Requirement: Replay drift classification SHALL include canonical policy-stack drift classes
Replay tooling MUST classify policy-stack semantic drift using canonical classes:
- `precedence_conflict`
- `tie_break_drift`
- `deny_source_mismatch`

#### Scenario: Replay detects precedence conflict drift
- **WHEN** actual winner stage violates expected precedence matrix
- **THEN** replay validation fails with deterministic `precedence_conflict` classification

#### Scenario: Replay detects deny source mismatch
- **WHEN** actual deny source differs from expected canonical source
- **THEN** replay validation fails with deterministic `deny_source_mismatch` classification

### Requirement: Policy fixture support SHALL preserve mixed-fixture backward compatibility
Adding `policy_stack.v1` support MUST NOT break existing fixture validations.

#### Scenario: Mixed fixture suites run in one gate flow
- **WHEN** replay gate executes `a50.v1`、`react.v1`、`sandbox_egress.v1` 与 `policy_stack.v1`
- **THEN** all fixture generations are validated deterministically without parser regression

#### Scenario: Historical fixture parser regression is introduced
- **WHEN** policy fixture support change breaks legacy fixture parsing
- **THEN** replay validation fails and blocks merge
