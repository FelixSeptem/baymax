## ADDED Requirements

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
