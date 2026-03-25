## ADDED Requirements

### Requirement: Runtime config SHALL expose readiness-admission controls with deterministic precedence
Runtime configuration MUST expose readiness-admission controls under `runtime.readiness.admission.*` with precedence `env > file > default`.

Minimum required controls and defaults:
- `runtime.readiness.admission.enabled=false`
- `runtime.readiness.admission.mode=fail_fast`
- `runtime.readiness.admission.block_on=blocked_only`
- `runtime.readiness.admission.degraded_policy=allow_and_record`

Unsupported enum values or malformed booleans MUST fail fast at startup and hot reload, and runtime MUST keep previous valid snapshot on reload failure.

#### Scenario: Runtime starts with default readiness-admission controls
- **WHEN** readiness-admission fields are not explicitly configured
- **THEN** runtime resolves documented default values with admission disabled

#### Scenario: Hot reload provides invalid degraded policy
- **WHEN** hot reload sets unsupported `runtime.readiness.admission.degraded_policy`
- **THEN** runtime rejects update and retains previous active configuration snapshot

### Requirement: Runtime diagnostics SHALL expose additive readiness-admission summary fields
Runtime diagnostics MUST expose additive readiness-admission fields while preserving compatibility-window semantics (`additive + nullable + default`).

Minimum required fields:
- `runtime_readiness_admission_total`
- `runtime_readiness_admission_blocked_total`
- `runtime_readiness_admission_degraded_allow_total`
- `runtime_readiness_admission_bypass_total`
- `runtime_readiness_admission_mode`
- `runtime_readiness_admission_primary_code`

Readiness-admission diagnostics MUST preserve replay-idempotent logical aggregates.

#### Scenario: Consumer queries diagnostics after admission decisions
- **WHEN** managed runtime handles admission allow/deny decisions
- **THEN** diagnostics include additive readiness-admission counters and canonical mode/code fields

#### Scenario: Equivalent admission events are replayed
- **WHEN** recorder ingests duplicate admission events for one run
- **THEN** readiness-admission logical aggregate counters remain stable after first ingestion
