## ADDED Requirements

### Requirement: Runtime config SHALL expose sandbox egress governance controls with deterministic precedence
Runtime configuration MUST expose `security.sandbox.egress.*` controls with precedence `env > file > default`.

Minimum required controls:
- `security.sandbox.egress.enabled`
- `security.sandbox.egress.default_action`
- `security.sandbox.egress.by_tool`
- `security.sandbox.egress.allowlist`
- `security.sandbox.egress.on_violation`

Invalid enum or malformed allowlist entries MUST fail fast at startup and rollback atomically on hot reload.

#### Scenario: Startup with invalid egress default action
- **WHEN** configuration sets unsupported `security.sandbox.egress.default_action`
- **THEN** runtime startup fails fast with validation error

#### Scenario: Hot reload with malformed egress allowlist entry
- **WHEN** hot reload payload includes malformed allowlist host pattern
- **THEN** runtime rejects update and preserves previous active snapshot

### Requirement: Runtime config SHALL expose adapter allowlist activation controls with deterministic precedence
Runtime configuration MUST expose `adapter.allowlist.*` controls with precedence `env > file > default`.

Minimum required controls:
- `adapter.allowlist.enabled`
- `adapter.allowlist.enforcement_mode`
- `adapter.allowlist.entries`
- `adapter.allowlist.on_unknown_signature`

#### Scenario: Startup with invalid allowlist enforcement mode
- **WHEN** configuration sets unsupported `adapter.allowlist.enforcement_mode`
- **THEN** runtime startup fails fast with validation error

#### Scenario: Hot reload removes mandatory allowlist entry shape fields
- **WHEN** hot reload updates allowlist entries with missing identity fields
- **THEN** runtime rejects update and keeps previous active snapshot unchanged

### Requirement: Runtime diagnostics SHALL expose additive egress and allowlist fields
Run diagnostics MUST expose A57 additive fields while preserving compatibility contract `additive + nullable + default`.

Minimum required fields:
- `sandbox_egress_action`
- `sandbox_egress_violation_total`
- `sandbox_egress_policy_source`
- `adapter_allowlist_decision`
- `adapter_allowlist_block_total`
- `adapter_allowlist_primary_code`

Fields MUST remain bounded-cardinality and replay-idempotent.

#### Scenario: Consumer queries diagnostics after egress deny
- **WHEN** run contains one or more egress deny decisions
- **THEN** diagnostics include canonical egress additive fields and violation counters

#### Scenario: Consumer queries diagnostics for allowlist-blocked activation
- **WHEN** runtime blocks adapter activation due to allowlist policy
- **THEN** diagnostics include canonical allowlist decision and primary code fields
