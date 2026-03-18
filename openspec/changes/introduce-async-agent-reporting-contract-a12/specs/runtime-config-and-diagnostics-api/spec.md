## ADDED Requirements

### Requirement: Runtime config SHALL expose async reporting controls with deterministic precedence
Runtime configuration MUST expose async reporting controls with precedence `env > file > default`, and async reporting MUST be disabled by default.

Required minimum config markers:
- `a2a.async_reporting.enabled`
- `a2a.async_reporting.sink`
- `a2a.async_reporting.retry.max_attempts`
- `a2a.async_reporting.retry.backoff_initial`
- `a2a.async_reporting.retry.backoff_max`

#### Scenario: Startup without async reporting overrides
- **WHEN** runtime starts with default configuration
- **THEN** async reporting is disabled and existing synchronous paths remain unchanged

#### Scenario: Invalid async retry settings on hot reload
- **WHEN** hot reload updates async reporting retry fields with invalid values
- **THEN** runtime rejects update and keeps previous valid snapshot

### Requirement: Runtime diagnostics SHALL expose additive async reporting summary fields
Run diagnostics MUST expose additive async reporting summary fields while preserving compatibility-window semantics.

Required minimum additive fields:
- `a2a_async_report_total`
- `a2a_async_report_failed`
- `a2a_async_report_retry_total`
- `a2a_async_report_dedup_total`

#### Scenario: Consumer queries diagnostics for async-enabled run
- **WHEN** async reporting is enabled and reports are delivered
- **THEN** diagnostics include additive async reporting summary fields without breaking legacy consumers

### Requirement: Async reporting diagnostics SHALL remain replay-idempotent
Repeated ingestion of equivalent async reporting events for one run MUST NOT inflate logical async aggregates.

#### Scenario: Duplicate async report events are replayed
- **WHEN** async reporting events are replayed for a completed run
- **THEN** diagnostics maintain stable async aggregate counters after first logical ingestion
