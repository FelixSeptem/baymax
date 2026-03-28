## ADDED Requirements

### Requirement: Runtime config SHALL expose arbitration version-governance controls with deterministic precedence
Runtime configuration MUST expose arbitration version-governance controls under `runtime.arbitration.version.*` with precedence `env > file > default`.

Minimum required controls:
- `runtime.arbitration.version.enabled`
- `runtime.arbitration.version.default`
- `runtime.arbitration.version.compat_window`
- `runtime.arbitration.version.on_unsupported`
- `runtime.arbitration.version.on_mismatch`

Recommended defaults:
- `enabled=true`
- `default=a49.v1`
- `compat_window=1`
- `on_unsupported=fail_fast`
- `on_mismatch=fail_fast`

Invalid startup or hot-reload values MUST fail fast and MUST preserve previous valid active snapshot.

#### Scenario: Runtime starts without explicit arbitration-version overrides
- **WHEN** arbitration version-governance fields are not configured
- **THEN** runtime resolves documented default values deterministically

#### Scenario: Hot reload provides invalid arbitration-version policy
- **WHEN** hot reload sets unsupported `on_unsupported` or `on_mismatch` value
- **THEN** runtime rejects update and keeps previous active configuration snapshot unchanged

### Requirement: Runtime diagnostics SHALL expose additive arbitration version-governance fields
Runtime diagnostics MUST expose additive arbitration version-governance fields while preserving compatibility-window semantics (`additive + nullable + default`).

Minimum required fields:
- `runtime_arbitration_rule_requested_version`
- `runtime_arbitration_rule_effective_version`
- `runtime_arbitration_rule_version_source`
- `runtime_arbitration_rule_policy_action`
- `runtime_arbitration_rule_unsupported_total`
- `runtime_arbitration_rule_mismatch_total`

Version-governance fields MUST remain replay-idempotent and bounded-cardinality.

#### Scenario: Consumer queries diagnostics after version-governed arbitration
- **WHEN** runtime evaluates cross-domain arbitration with version-governance enabled
- **THEN** diagnostics include additive version-governance fields with deterministic values

#### Scenario: Equivalent version-governance events are replayed
- **WHEN** recorder ingests duplicate arbitration version-governance events for one run
- **THEN** logical version-governance aggregates remain stable after first ingestion
