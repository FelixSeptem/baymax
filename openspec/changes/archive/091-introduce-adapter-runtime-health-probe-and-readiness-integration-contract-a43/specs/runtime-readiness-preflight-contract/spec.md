## ADDED Requirements

### Requirement: Readiness preflight SHALL incorporate adapter-health findings
Runtime readiness preflight MUST evaluate adapter health findings as part of readiness classification when adapter-health feature is enabled.

Readiness output MUST preserve canonical finding schema while adding adapter-domain findings.

#### Scenario: Required adapter is unavailable under strict policy
- **WHEN** readiness preflight detects required adapter health status `unavailable` and `runtime.readiness.strict=true`
- **THEN** readiness overall status is `blocked` and includes canonical adapter-health blocking finding

#### Scenario: Optional adapter is unavailable under non-strict policy
- **WHEN** readiness preflight detects optional adapter health status `unavailable` and `runtime.readiness.strict=false`
- **THEN** readiness overall status is `degraded` and includes canonical adapter-health degraded finding

### Requirement: Adapter-health readiness mapping SHALL remain deterministic
For equivalent adapter inventory state and effective configuration, readiness classification and primary finding code MUST remain deterministic.

#### Scenario: Repeated preflight with unchanged adapter state
- **WHEN** host runs readiness preflight repeatedly without adapter or config changes
- **THEN** readiness status and adapter-health finding codes remain semantically equivalent

#### Scenario: Regression introduces non-canonical adapter finding code
- **WHEN** implementation emits adapter-health finding code outside canonical taxonomy
- **THEN** contract validation fails and blocks merge
