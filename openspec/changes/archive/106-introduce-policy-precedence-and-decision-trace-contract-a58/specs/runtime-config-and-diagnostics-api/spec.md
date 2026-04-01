## ADDED Requirements

### Requirement: Runtime config SHALL expose policy precedence and tie-break controls
Runtime configuration MUST expose policy precedence controls with precedence `env > file > default`.

Minimum required controls:
- `runtime.policy.precedence.version`
- `runtime.policy.precedence.matrix.*`
- `runtime.policy.tie_breaker.mode`
- `runtime.policy.tie_breaker.source_order`
- `runtime.policy.explainability.enabled`

Invalid stage names, malformed matrix entries, or unsupported tie-break modes MUST fail fast at startup and rollback atomically on hot reload.

#### Scenario: Startup with unsupported tie-break mode
- **WHEN** configuration sets unsupported `runtime.policy.tie_breaker.mode`
- **THEN** runtime startup fails fast with validation error

#### Scenario: Hot reload with malformed precedence matrix
- **WHEN** hot reload payload includes invalid stage reference in `runtime.policy.precedence.matrix.*`
- **THEN** runtime rejects update and preserves previous active snapshot

### Requirement: Runtime diagnostics SHALL expose additive policy decision-trace fields
Run diagnostics MUST expose A58 additive fields while preserving compatibility contract `additive + nullable + default`.

Minimum required fields:
- `policy_decision_path`
- `deny_source`
- `winner_stage`
- `tie_break_reason`

Fields MUST remain bounded-cardinality and replay-idempotent.

#### Scenario: Consumer queries diagnostics after policy deny
- **WHEN** run contains one or more policy deny decisions
- **THEN** diagnostics include canonical policy decision-trace fields

#### Scenario: Consumer queries diagnostics for allow path
- **WHEN** run is allowed without same-stage conflicts
- **THEN** diagnostics include winner stage and omit tie-break reason when not applicable
