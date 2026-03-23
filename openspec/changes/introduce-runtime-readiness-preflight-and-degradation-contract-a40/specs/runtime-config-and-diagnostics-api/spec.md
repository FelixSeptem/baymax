## ADDED Requirements

### Requirement: Runtime config SHALL expose readiness controls with deterministic precedence
Runtime configuration MUST expose readiness controls under `runtime.readiness.*` with precedence `env > file > default`.

Minimum required controls and defaults:
- `runtime.readiness.enabled=true`
- `runtime.readiness.strict=false`
- `runtime.readiness.remote_probe_enabled=false`

Invalid startup and hot-reload values MUST fail fast and MUST preserve previous valid snapshot.

#### Scenario: Runtime starts with default readiness controls
- **WHEN** no readiness overrides are provided
- **THEN** effective config resolves to documented defaults

#### Scenario: Hot reload provides invalid readiness control value
- **WHEN** hot reload sets unsupported readiness enum/bool representation
- **THEN** runtime rejects update and keeps prior active config snapshot unchanged

### Requirement: Runtime diagnostics SHALL expose additive readiness summary fields
Runtime diagnostics MUST expose additive readiness summary fields while preserving compatibility-window semantics (`additive + nullable + default`).

Minimum required additive fields:
- `runtime_readiness_status`
- `runtime_readiness_finding_total`
- `runtime_readiness_blocking_total`
- `runtime_readiness_degraded_total`
- `runtime_readiness_primary_code`

Readiness diagnostics MUST preserve deterministic replay semantics and MUST NOT inflate logical counts for equivalent preflight events.

#### Scenario: Consumer queries readiness diagnostics after preflight
- **WHEN** host executes readiness preflight and queries diagnostics
- **THEN** diagnostics include readiness additive fields with stable status and counts

#### Scenario: Equivalent readiness preflight events are replayed
- **WHEN** recorder ingests duplicated readiness preflight events for one run/session
- **THEN** logical readiness aggregate counters remain stable after first ingestion
