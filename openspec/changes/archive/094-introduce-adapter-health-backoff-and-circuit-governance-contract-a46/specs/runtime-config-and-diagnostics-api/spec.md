## ADDED Requirements

### Requirement: Runtime config SHALL expose adapter-health backoff and circuit controls with deterministic precedence
Runtime configuration MUST expose adapter-health governance controls under `adapter.health.backoff.*` and `adapter.health.circuit.*` with precedence `env > file > default`.

Minimum required controls and defaults:
- `adapter.health.backoff.enabled=true`
- `adapter.health.backoff.initial=200ms`
- `adapter.health.backoff.max=5s`
- `adapter.health.backoff.multiplier=2.0`
- `adapter.health.backoff.jitter_ratio=0.2`
- `adapter.health.circuit.enabled=true`
- `adapter.health.circuit.failure_threshold=3`
- `adapter.health.circuit.open_duration=30s`
- `adapter.health.circuit.half_open_max_probe=1`
- `adapter.health.circuit.half_open_success_threshold=2`

Invalid startup or hot-reload values MUST fail fast and MUST preserve previous active valid snapshot.

#### Scenario: Runtime starts with default adapter-health governance controls
- **WHEN** adapter-health governance fields are not explicitly configured
- **THEN** effective runtime config resolves documented default values for backoff and circuit domains

#### Scenario: Hot reload provides invalid circuit threshold
- **WHEN** hot reload sets non-positive `adapter.health.circuit.failure_threshold`
- **THEN** runtime rejects update and keeps previous active snapshot unchanged

### Requirement: Runtime diagnostics SHALL expose additive adapter-health governance observability fields
Runtime diagnostics MUST expose additive adapter-health governance fields while preserving compatibility-window semantics (`additive + nullable + default`).

Minimum required fields:
- `adapter_health_backoff_applied_total`
- `adapter_health_circuit_open_total`
- `adapter_health_circuit_half_open_total`
- `adapter_health_circuit_recover_total`
- `adapter_health_circuit_state`
- `adapter_health_governance_primary_code`

Governance observability fields MUST remain bounded-cardinality and replay-idempotent for equivalent events.

#### Scenario: Consumer queries diagnostics after governed probe execution
- **WHEN** adapter probe execution applies backoff and circuit transitions
- **THEN** diagnostics include additive governance counters and canonical state/code fields

#### Scenario: Equivalent governance events are replayed
- **WHEN** recorder ingests duplicate adapter governance events for one run/session
- **THEN** logical governance aggregate counters remain stable after first ingestion
