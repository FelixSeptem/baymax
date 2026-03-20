## ADDED Requirements

### Requirement: Runtime config SHALL expose async-await lifecycle controls with deterministic precedence
Runtime configuration MUST expose async-await lifecycle controls under scheduler domain and resolve with precedence `env > file > default`.

Minimum required controls:
- `scheduler.async_await.report_timeout`
- `scheduler.async_await.late_report_policy`
- `scheduler.async_await.timeout_terminal`

Default values MUST be:
- `scheduler.async_await.report_timeout=15m`
- `scheduler.async_await.late_report_policy=drop_and_record`
- `scheduler.async_await.timeout_terminal=failed`

Invalid values in startup or hot reload MUST fail fast and keep last valid snapshot unchanged.

#### Scenario: Runtime starts with default async-await settings
- **WHEN** runtime loads default configuration without overrides
- **THEN** effective config resolves default async-await controls with documented values

#### Scenario: Hot reload provides invalid async-await policy
- **WHEN** hot reload updates async-await controls with unsupported enum or invalid timeout
- **THEN** runtime rejects update and keeps previous valid snapshot

### Requirement: Runtime diagnostics SHALL expose additive async-await lifecycle summary fields
Runtime diagnostics MUST expose additive async-await summary fields while preserving compatibility-window semantics (`additive + nullable + default`).

Minimum required additive fields:
- `async_await_total`
- `async_timeout_total`
- `async_late_report_total`
- `async_report_dedup_total`

#### Scenario: Consumer queries diagnostics for async-await run
- **WHEN** run contains async-await lifecycle transitions including timeout or late-report handling
- **THEN** diagnostics response includes additive async-await fields without changing legacy field meanings

### Requirement: Async-await diagnostics replay SHALL remain idempotent
Repeated ingestion or replay of equivalent async-await events for one run MUST NOT inflate logical aggregate counters.

#### Scenario: Replay submits equivalent timeout and late-report events
- **WHEN** diagnostics recorder replays equivalent async-await events for same run
- **THEN** async-await additive counters remain stable after first logical ingestion

