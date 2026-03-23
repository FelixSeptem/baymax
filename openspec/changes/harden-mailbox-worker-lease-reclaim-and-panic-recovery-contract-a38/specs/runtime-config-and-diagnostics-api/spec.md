## ADDED Requirements

### Requirement: Runtime SHALL expose mailbox worker lease and panic-recovery controls
Runtime configuration MUST expose mailbox worker lease/recovery controls under `mailbox.worker.*` with precedence `env > file > default`.

Minimum required fields:
- `mailbox.worker.inflight_timeout`
- `mailbox.worker.heartbeat_interval`
- `mailbox.worker.reclaim_on_consume`
- `mailbox.worker.panic_policy`

Default values MUST be:
- `mailbox.worker.inflight_timeout=30s`
- `mailbox.worker.heartbeat_interval=5s`
- `mailbox.worker.reclaim_on_consume=true`
- `mailbox.worker.panic_policy=follow_handler_error_policy`

Validation MUST enforce:
- `inflight_timeout > 0`
- `heartbeat_interval > 0`
- `heartbeat_interval < inflight_timeout`
- `panic_policy` within supported enum set

Invalid startup or hot reload values MUST fail fast and preserve previous valid snapshot.

#### Scenario: Effective mailbox worker lease config uses deterministic precedence
- **WHEN** file and env both configure mailbox worker lease/recovery fields
- **THEN** effective runtime config resolves fields with `env > file > default`

#### Scenario: Invalid hot reload for heartbeat and timeout
- **WHEN** hot reload sets `mailbox.worker.heartbeat_interval>=mailbox.worker.inflight_timeout`
- **THEN** runtime rejects reload and keeps previous valid snapshot unchanged

### Requirement: Runtime diagnostics SHALL expose mailbox reclaim and panic-recover observability
Mailbox diagnostics MUST include lifecycle observability for reclaim and panic-recover execution paths.

Required observability semantics:
- stale in-flight reclaim events are queryable and aggregated deterministically,
- reclaim outcomes preserve canonical reason taxonomy (including `lease_expired`),
- panic-recovered handler outcomes are queryable without changing existing additive compatibility guarantees.

#### Scenario: Consumer inspects reclaim diagnostics
- **WHEN** mailbox runtime reclaims stale in-flight messages
- **THEN** diagnostics query and aggregates include reclaim lifecycle records with canonical reason mapping

#### Scenario: Consumer inspects panic-recover diagnostics
- **WHEN** mailbox worker recovers handler panic and applies configured policy
- **THEN** diagnostics query and aggregates include deterministic lifecycle records for recovered path outcomes
