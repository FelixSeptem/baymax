## ADDED Requirements

### Requirement: Runtime config SHALL expose async-await reconcile controls with deterministic precedence
Runtime configuration MUST expose async-await reconcile controls under `scheduler.async_await.reconcile.*` and resolve effective values using `env > file > default`.

Minimum required controls:
- `scheduler.async_await.reconcile.enabled` (default `false`)
- `scheduler.async_await.reconcile.interval` (default `5s`)
- `scheduler.async_await.reconcile.batch_size` (default `64`)
- `scheduler.async_await.reconcile.jitter_ratio` (default `0.2`)
- `scheduler.async_await.reconcile.not_found_policy` (default `keep_until_timeout`)

Invalid values during startup or hot reload MUST fail fast and MUST keep previous valid snapshot unchanged.

#### Scenario: Runtime starts with default reconcile controls
- **WHEN** runtime loads configuration without explicit reconcile overrides
- **THEN** effective config resolves documented reconcile defaults with feature disabled

#### Scenario: Hot reload provides invalid reconcile controls
- **WHEN** hot reload updates reconcile interval, batch size, jitter ratio, or not-found policy with invalid value
- **THEN** runtime rejects update and preserves last valid active snapshot

### Requirement: Runtime diagnostics SHALL expose additive async-await reconcile aggregates
Runtime diagnostics MUST expose additive reconcile aggregates while preserving compatibility-window semantics (`additive + nullable + default`).

Minimum required additive fields:
- `async_reconcile_poll_total`
- `async_reconcile_terminal_by_poll_total`
- `async_reconcile_error_total`
- `async_terminal_conflict_total`

#### Scenario: Consumer queries diagnostics for reconcile-enabled run
- **WHEN** run executes async-await reconcile polling and terminal convergence
- **THEN** diagnostics response includes additive reconcile fields without changing pre-existing field semantics

### Requirement: Async-await reconcile diagnostics SHALL remain replay-idempotent
Repeated ingestion or replay of equivalent reconcile events for one run MUST NOT inflate logical aggregate counters.

#### Scenario: Reconcile events are replayed
- **WHEN** diagnostics recorder replays equivalent reconcile polling and conflict events for same run
- **THEN** reconcile additive counters remain stable after first logical ingestion

