## ADDED Requirements

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
