## ADDED Requirements

### Requirement: Runtime SHALL expose cross-run timeline trend configuration with deterministic precedence
Runtime configuration MUST expose cross-run Action Timeline trend aggregation controls with precedence `env > file > default`.

The minimum configuration set MUST include:
- enable switch (default enabled),
- `last_n_runs` window size (default `100`),
- `time_window` duration (default `15m`).

Invalid values MUST fail fast during startup and hot reload.

#### Scenario: Startup with default trend configuration
- **WHEN** runtime starts without trend-specific overrides
- **THEN** cross-run trend aggregation is enabled with `last_n_runs=100` and `time_window=15m`

#### Scenario: Startup with file and environment trend overrides
- **WHEN** trend fields are configured in both YAML and environment variables
- **THEN** effective trend settings resolve with `env > file > default`

#### Scenario: Invalid trend window configuration
- **WHEN** `last_n_runs` is non-positive or `time_window` is invalid
- **THEN** runtime rejects startup or hot reload snapshot with fail-fast validation error

### Requirement: Runtime diagnostics SHALL expose cross-run timeline trend aggregates
Runtime diagnostics API MUST expose cross-run timeline trend aggregates using both `last_n_runs` and `time_window` selection modes.

Trend aggregates MUST support `phase + status` dimensions and MUST include at least:
- `count_total`
- `failed_total`
- `canceled_total`
- `skipped_total`
- `latency_avg_ms`
- `latency_p95_ms`
- `window_start`
- `window_end`

The capability MUST be additive and MUST NOT break existing run-level diagnostics consumers.

#### Scenario: Consumer queries trends with last_n_runs mode
- **WHEN** application queries trend diagnostics using `last_n_runs`
- **THEN** runtime returns bounded `phase + status` aggregates over the most recent N runs with required metric fields

#### Scenario: Consumer queries trends with time_window mode
- **WHEN** application queries trend diagnostics using `time_window`
- **THEN** runtime returns bounded `phase + status` aggregates over runs inside the time window with required metric fields

#### Scenario: Consumer reads existing run-level diagnostics only
- **WHEN** existing integrations continue reading legacy run-level fields
- **THEN** diagnostics remain backward-compatible without requiring consumer changes

#### Scenario: Consumer queries trends for empty window
- **WHEN** selected window has no eligible run samples
- **THEN** runtime returns an empty aggregate set and does not fabricate metrics
