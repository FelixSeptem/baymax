## ADDED Requirements

### Requirement: Runtime config SHALL expose operation-profile timeout governance with deterministic precedence
Runtime configuration MUST expose operation-profile timeout governance under `runtime.operation_profiles.*` with precedence `env > file > default`.

Minimum required controls:
- `runtime.operation_profiles.default_profile`
- `runtime.operation_profiles.interactive.timeout`
- `runtime.operation_profiles.background.timeout`
- `runtime.operation_profiles.batch.timeout`
- `runtime.operation_profiles.legacy.timeout`

Default profile MUST be `legacy`, and profile timeout values MUST be strictly positive durations.

Invalid startup and hot-reload values MUST fail fast and MUST preserve previous valid snapshot.

#### Scenario: Runtime starts without explicit operation-profile overrides
- **WHEN** no operation-profile timeout fields are configured
- **THEN** runtime resolves `default_profile=legacy` and documented default timeout baselines

#### Scenario: Hot reload applies invalid operation-profile timeout
- **WHEN** hot reload sets non-positive profile timeout or unsupported default profile
- **THEN** runtime rejects update and keeps prior active configuration snapshot unchanged

### Requirement: Runtime diagnostics SHALL expose additive timeout-resolution observability fields
Runtime diagnostics MUST expose additive timeout-resolution fields while preserving compatibility-window semantics (`additive + nullable + default`).

Minimum required fields:
- `effective_operation_profile`
- `timeout_resolution_source`
- `timeout_resolution_trace`
- `timeout_parent_budget_clamp_total`
- `timeout_parent_budget_reject_total`

Timeout-resolution diagnostics MUST remain replay-idempotent for equivalent events.

#### Scenario: Consumer queries diagnostics after profile-based execution
- **WHEN** run includes operation-profile timeout resolution and parent-budget clamp
- **THEN** diagnostics include additive timeout-resolution fields and deterministic source classification

#### Scenario: Equivalent timeout-resolution events are replayed
- **WHEN** recorder ingests duplicate timeout-resolution events for one run
- **THEN** logical aggregate counters remain stable after first ingestion
