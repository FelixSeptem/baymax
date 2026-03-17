## ADDED Requirements

### Requirement: Runtime config SHALL expose S2 tool-security governance controls with deterministic precedence
Runtime configuration MUST expose S2 tool-security governance fields with precedence `env > file > default`, including:
- governance mode (default `enforce`),
- `namespace+tool` permission policy entries,
- process-scoped rate-limit policy entries,
- deny behavior controls for permission and rate-limit violations.

Invalid policy keys, malformed `namespace+tool` selectors, or unsupported mode values MUST fail fast during startup and hot reload.

#### Scenario: Startup with default S2 governance config
- **WHEN** runtime starts without explicit S2 governance overrides
- **THEN** effective config resolves governance mode as `enforce` with valid default deny semantics

#### Scenario: Startup with env and file governance overrides
- **WHEN** S2 governance fields are defined in both YAML and environment variables
- **THEN** effective values resolve by `env > file > default`

#### Scenario: Invalid namespace+tool policy key
- **WHEN** runtime receives malformed `namespace+tool` selector in permission or rate-limit policy
- **THEN** startup or hot reload is rejected with fail-fast validation error

### Requirement: Runtime config SHALL expose model I/O security filtering controls with deterministic precedence
Runtime configuration MUST expose model input/output security filtering controls with precedence `env > file > default`, including filter enablement, stage-specific execution controls, and extension registration settings.

Invalid stage configuration or malformed filter policy values MUST fail fast during startup and hot reload.

#### Scenario: Startup with default model I/O filtering config
- **WHEN** runtime starts without explicit I/O filtering overrides
- **THEN** effective config resolves to valid default input/output filtering controls

#### Scenario: Startup with I/O filtering overrides
- **WHEN** input/output filtering settings are defined in both YAML and environment variables
- **THEN** effective values resolve by `env > file > default`

#### Scenario: Invalid I/O filter stage configuration
- **WHEN** runtime receives unsupported stage option or malformed filter policy value
- **THEN** startup or hot reload fails fast with validation error

### Requirement: Runtime SHALL apply S2 security config hot reload with atomic switch and rollback safety
Security governance and I/O filtering config updates MUST be validated and atomically activated on success; invalid updates MUST be rejected and previous snapshot MUST remain active.

#### Scenario: Valid S2 security config update arrives
- **WHEN** watched config file changes to valid S2 governance or I/O filtering settings
- **THEN** runtime atomically switches to new snapshot and subsequent requests observe updated security policy immediately

#### Scenario: Invalid S2 security config update arrives
- **WHEN** watched config file changes to invalid S2 governance or I/O filtering settings
- **THEN** runtime rejects the update, preserves previous active snapshot, and emits reload error diagnostics

### Requirement: Runtime diagnostics SHALL expose additive S2 security governance and I/O filtering fields
Runtime diagnostics MUST expose additive fields for S2 security decisions, including at minimum:
- policy kind (`permission|rate_limit|io_filter`),
- selector context (`namespace+tool` when applicable),
- filter stage (`input|output` when applicable),
- deny/match outcome,
- normalized reason code.

These fields MUST be backward-compatible and MUST NOT change existing diagnostics field semantics.

#### Scenario: Permission denial diagnostics
- **WHEN** runtime denies tool execution by permission policy
- **THEN** diagnostics include additive permission-deny fields with selector and reason code

#### Scenario: Rate-limit denial diagnostics
- **WHEN** runtime denies tool execution by process-scoped rate limit
- **THEN** diagnostics include additive rate-limit fields with selector, window context, and reason code

#### Scenario: I/O filter diagnostics
- **WHEN** runtime evaluates input or output filtering and records match or deny outcomes
- **THEN** diagnostics include additive filter stage/result fields with normalized reason code