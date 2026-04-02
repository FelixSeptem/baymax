## ADDED Requirements

### Requirement: A65 Runtime Config Governance
The runtime configuration surface SHALL include `runtime.hooks.*`, `runtime.tool_middleware.*`, `runtime.skill.discovery.*`, `runtime.skill.preprocess.*`, and `runtime.skill.bundle_mapping.*` with `env > file > default` precedence and fail-fast validation.

#### Scenario: Precedence resolution is deterministic
- **WHEN** the same config key is provided in env and file with different values
- **THEN** effective value MUST come from env source

#### Scenario: Invalid startup config fails fast
- **WHEN** startup config contains invalid lifecycle phase, invalid discovery mode, or invalid mapping policy
- **THEN** runtime initialization MUST fail fast with canonical validation error

#### Scenario: Invalid hot reload rolls back atomically
- **WHEN** hot reload receives invalid A65 config
- **THEN** runtime MUST preserve previous valid snapshot and record reload failure

### Requirement: A65 Diagnostics Fields Are Additive
Diagnostics output for hooks, middleware, and skill preprocess/mapping MUST be additive and MUST NOT redefine existing canonical fields.

#### Scenario: QueryRuns additive compatibility
- **WHEN** A65 diagnostics fields are present
- **THEN** existing QueryRuns consumers MUST parse legacy fields unchanged while new fields remain optional

#### Scenario: Reason taxonomy stability
- **WHEN** hook or middleware failures are recorded
- **THEN** diagnostics MUST use canonical reason taxonomy and MUST NOT duplicate existing semantic fields
