## ADDED Requirements

### Requirement: Replay tooling SHALL validate budget-admission fixtures
Diagnostics replay tooling MUST support budget-admission fixture validation with versioned fixture contract `budget_admission.v1`.

Fixture validation MUST cover at minimum:
- budget snapshot thresholds
- budget decision
- degrade action

#### Scenario: Budget-admission fixture matches canonical output
- **WHEN** replay tooling processes valid `budget_admission.v1` fixture and normalized output matches expected semantics
- **THEN** replay validation succeeds with deterministic pass result

#### Scenario: Budget-admission fixture schema is malformed
- **WHEN** replay tooling receives malformed or unsupported `budget_admission.v1` fixture schema
- **THEN** replay validation fails fast with deterministic validation reason code

### Requirement: Replay drift classification SHALL include canonical budget-admission drift classes
Replay tooling MUST classify budget-admission semantic drift using canonical classes:
- `budget_threshold_drift`
- `admission_decision_drift`
- `degrade_policy_drift`

#### Scenario: Replay detects budget threshold drift
- **WHEN** actual threshold evaluation output differs from expected fixture threshold semantics
- **THEN** replay validation fails with deterministic `budget_threshold_drift` classification

#### Scenario: Replay detects degrade policy drift
- **WHEN** actual degrade action selection differs from fixture policy expectation
- **THEN** replay validation fails with deterministic `degrade_policy_drift` classification

### Requirement: Budget fixture support SHALL preserve mixed-fixture backward compatibility
Adding `budget_admission.v1` support MUST NOT break existing archived fixture validations.

#### Scenario: Mixed fixture suites run in one gate flow
- **WHEN** replay gate executes historical fixtures together with `budget_admission.v1`
- **THEN** all fixture generations are parsed and validated deterministically without regression

#### Scenario: Historical parser regression is introduced
- **WHEN** budget fixture support breaks legacy fixture parsing
- **THEN** replay validation fails and blocks merge
