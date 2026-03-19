## ADDED Requirements

### Requirement: Runtime config SHALL expose long-running recovery boundary controls
Runtime configuration MUST expose recovery boundary controls with deterministic precedence `env > file > default`.

Minimum required controls:
- `recovery.resume_boundary` (`next_attempt_only`)
- `recovery.inflight_policy` (`no_rewind`)
- `recovery.timeout_reentry_policy` (`single_reentry_then_fail`)
- `recovery.timeout_reentry_max_per_task` (default `1`)

#### Scenario: Runtime starts with recovery enabled and no boundary overrides
- **WHEN** `recovery.enabled=true` and no explicit boundary controls are provided
- **THEN** runtime applies default boundary values and validates them successfully

#### Scenario: Invalid boundary policy value is configured
- **WHEN** config provides unsupported resume boundary or inflight policy value
- **THEN** startup/hot reload fails fast and active configuration is not replaced

### Requirement: Runtime diagnostics SHALL expose additive recovery-boundary summary fields
Run diagnostics MUST expose additive recovery-boundary fields while preserving compatibility-window semantics.

Minimum required fields:
- `recovery_resume_boundary`
- `recovery_inflight_policy`
- `recovery_timeout_reentry_total`
- `recovery_timeout_reentry_exhausted_total`

#### Scenario: Consumer queries diagnostics for boundary-controlled recovery run
- **WHEN** run executes with recovery boundary controls active
- **THEN** diagnostics include additive boundary fields without breaking existing consumers

### Requirement: Recovery-boundary diagnostics SHALL remain additive nullable default compatible
New recovery-boundary fields MUST follow `additive + nullable + default` behavior and MUST NOT alter pre-existing field semantics.

#### Scenario: Legacy consumer parses run summary after boundary rollout
- **WHEN** legacy parser reads run summary containing recovery-boundary additive fields
- **THEN** legacy parsing remains compatible and pre-existing field meanings remain unchanged
