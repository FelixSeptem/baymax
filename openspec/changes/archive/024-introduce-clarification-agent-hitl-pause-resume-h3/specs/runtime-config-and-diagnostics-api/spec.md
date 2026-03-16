## ADDED Requirements

### Requirement: Runtime config SHALL define H3 clarification timeout policy
Runtime configuration MUST support H3 clarification fields with deterministic precedence `env > file > default`, including `enabled`, clarification timeout, and timeout policy. Default timeout policy MUST be `cancel_by_user`.

#### Scenario: Startup with default clarification config
- **WHEN** runtime starts without clarification overrides
- **THEN** clarification HITL is enabled with configured default timeout and `cancel_by_user` timeout policy

#### Scenario: Startup with clarification overrides
- **WHEN** clarification fields are configured in YAML and environment variables
- **THEN** effective values resolve by `env > file > default`

### Requirement: Runtime diagnostics SHALL expose minimal H3 clarification counters
Run diagnostics MUST expose minimal clarification counters including `await_count`, `resume_count`, and `cancel_by_user_count`.

#### Scenario: Consumer inspects run diagnostics with clarification flow
- **WHEN** a run triggers clarification wait and resume/cancel lifecycle
- **THEN** diagnostics include non-negative values for `await_count`, `resume_count`, and `cancel_by_user_count`

#### Scenario: Consumer inspects run diagnostics without clarification flow
- **WHEN** a run never triggers clarification
- **THEN** diagnostics expose zero-value clarification counters without breaking schema compatibility
