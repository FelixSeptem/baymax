## ADDED Requirements

### Requirement: Runtime config SHALL expose adapter-health controls with deterministic precedence
Runtime configuration MUST expose adapter-health controls under `adapter.health.*` with precedence `env > file > default`.

Minimum required controls and defaults:
- `adapter.health.enabled=false`
- `adapter.health.strict=false`
- `adapter.health.probe_timeout=500ms`
- `adapter.health.cache_ttl=30s`

Invalid startup and hot-reload values MUST fail fast and MUST preserve previous valid snapshot.

#### Scenario: Runtime starts with default adapter-health controls
- **WHEN** no adapter-health overrides are provided
- **THEN** effective config resolves documented defaults and adapter-health feature remains disabled

#### Scenario: Hot reload applies invalid adapter-health durations
- **WHEN** hot reload sets `adapter.health.probe_timeout<=0` or `adapter.health.cache_ttl<=0`
- **THEN** runtime rejects update and keeps prior active configuration unchanged

### Requirement: Runtime diagnostics SHALL expose additive adapter-health summary fields
Runtime diagnostics MUST expose additive adapter-health fields while preserving compatibility-window semantics (`additive + nullable + default`).

Minimum required fields:
- `adapter_health_status`
- `adapter_health_probe_total`
- `adapter_health_degraded_total`
- `adapter_health_unavailable_total`
- `adapter_health_primary_code`

Adapter-health diagnostics MUST preserve replay-idempotent logical aggregates.

#### Scenario: Consumer queries diagnostics after adapter-health evaluation
- **WHEN** run includes adapter-health probe and readiness mapping
- **THEN** diagnostics output includes additive adapter-health summary fields with canonical status and primary code

#### Scenario: Adapter-health events are replayed
- **WHEN** equivalent adapter-health events are replayed for one run
- **THEN** logical adapter-health aggregate counters remain stable after first ingestion
