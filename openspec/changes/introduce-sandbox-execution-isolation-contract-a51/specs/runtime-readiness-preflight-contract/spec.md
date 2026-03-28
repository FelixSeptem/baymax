## ADDED Requirements

### Requirement: Readiness preflight SHALL evaluate sandbox-required availability deterministically
When sandbox governance is enabled with `required=true`, readiness preflight MUST evaluate sandbox executor availability and profile validity as blocking preconditions.

Unavailable or invalid required sandbox dependency MUST produce blocking readiness finding with canonical machine-readable code.

#### Scenario: Required sandbox executor is unavailable
- **WHEN** sandbox is enabled with `required=true` and executor probe fails
- **THEN** readiness preflight returns `blocked` with canonical sandbox-unavailable finding

#### Scenario: Required sandbox profile is invalid
- **WHEN** sandbox is enabled with `required=true` and selected profile validation fails
- **THEN** readiness preflight returns `blocked` with canonical sandbox-profile-invalid finding

#### Scenario: Required sandbox capability is not supported by backend
- **WHEN** sandbox is enabled with `required=true` and executor probe does not satisfy required capabilities
- **THEN** readiness preflight returns `blocked` with canonical sandbox-capability-mismatch finding

#### Scenario: Required sandbox session mode is unsupported
- **WHEN** sandbox is enabled with `required=true` and configured session mode is unsupported by executor/backend
- **THEN** readiness preflight returns `blocked` with canonical sandbox-session-mode-unsupported finding

### Requirement: Non-required sandbox degradation SHALL remain observable without forced blocking
When sandbox governance is enabled with `required=false`, sandbox dependency issues MUST remain observable and MUST follow readiness strict/non-strict classification semantics.

#### Scenario: Non-required sandbox executor unavailable under non-strict policy
- **WHEN** sandbox is enabled with `required=false`, executor probe fails, and readiness strict mode is disabled
- **THEN** readiness preflight returns degraded-class finding and keeps runtime runnable

#### Scenario: Non-required sandbox issue under strict policy
- **WHEN** sandbox is enabled with `required=false`, sandbox finding is degraded-class, and readiness strict mode is enabled
- **THEN** readiness classification escalates to blocked according to strict policy contract
