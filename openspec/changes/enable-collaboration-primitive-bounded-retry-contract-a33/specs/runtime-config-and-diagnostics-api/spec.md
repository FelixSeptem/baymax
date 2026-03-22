## ADDED Requirements

### Requirement: Runtime config SHALL expose collaboration retry governance controls with deterministic precedence
Runtime configuration MUST expose collaboration retry governance fields under `composer.collab.retry.*` with precedence `env > file > default`.

Minimum required fields and defaults:
- `composer.collab.retry.enabled=false`
- `composer.collab.retry.max_attempts=3`
- `composer.collab.retry.backoff_initial=100ms`
- `composer.collab.retry.backoff_max=2s`
- `composer.collab.retry.multiplier=2.0`
- `composer.collab.retry.jitter_ratio=0.2`
- `composer.collab.retry.retry_on=transport_only`

Invalid startup or hot-reload values MUST fail fast and MUST keep previous active snapshot unchanged.

#### Scenario: Runtime starts without collaboration retry overrides
- **WHEN** runtime loads default configuration
- **THEN** collaboration retry remains disabled and all retry governance fields resolve to documented default values

#### Scenario: Hot reload provides invalid collaboration retry bounds
- **WHEN** hot reload sets invalid retry configuration (for example `max_attempts<=0`, `backoff_max<backoff_initial`, `multiplier<=1`, or `jitter_ratio` outside `[0,1]`)
- **THEN** runtime rejects update and keeps last valid active configuration snapshot

### Requirement: Runtime diagnostics SHALL expose additive collaboration retry summary fields
Run diagnostics MUST expose additive collaboration retry summary fields while preserving compatibility-window semantics (`additive + nullable + default`).

Minimum required fields:
- `collab_retry_total`
- `collab_retry_success_total`
- `collab_retry_exhausted_total`

#### Scenario: Consumer queries diagnostics for retry-enabled collaboration run
- **WHEN** collaboration primitive retries are executed during a run
- **THEN** diagnostics response includes additive collaboration retry fields without changing pre-existing field semantics

### Requirement: Collaboration retry diagnostics SHALL remain replay-idempotent
Repeated ingestion or replay of equivalent collaboration retry events for one run MUST NOT inflate logical retry aggregates.

#### Scenario: Retry events are replayed for one completed run
- **WHEN** recorder replays equivalent collaboration retry events for the same run
- **THEN** collaboration retry aggregate counters remain stable after first logical ingestion

