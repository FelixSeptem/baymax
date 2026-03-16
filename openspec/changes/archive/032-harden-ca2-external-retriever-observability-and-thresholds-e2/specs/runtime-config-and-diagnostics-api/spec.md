## ADDED Requirements

### Requirement: Runtime SHALL expose CA2 external retriever observability config with deterministic precedence
Runtime configuration MUST expose CA2 external retriever observability settings with precedence `env > file > default`.

The minimum set MUST include:
- observability window duration (default `15m`),
- static threshold `p95_latency_ms`,
- static threshold `error_rate`,
- static threshold `hit_rate`.

Invalid window or threshold values MUST fail fast during startup and hot reload.

#### Scenario: Startup with default CA2 external observability config
- **WHEN** runtime starts without explicit CA2 external observability overrides
- **THEN** effective config uses default window `15m` with valid threshold defaults

#### Scenario: Startup with env and file overrides
- **WHEN** observability window and thresholds are set in both YAML and environment variables
- **THEN** effective values resolve by `env > file > default`

#### Scenario: Invalid threshold or window config
- **WHEN** runtime receives out-of-range threshold values or non-positive window duration
- **THEN** startup or hot reload is rejected with fail-fast validation error

### Requirement: Runtime diagnostics SHALL expose provider-scoped CA2 external trend aggregates
Runtime diagnostics API MUST expose CA2 external retriever trend aggregates grouped by provider and window.

The minimum output fields MUST include:
- `provider`
- `window_start`
- `window_end`
- `p95_latency_ms`
- `error_rate`
- `hit_rate`

Trend outputs MUST be additive and MUST NOT break existing diagnostics consumers.

#### Scenario: Consumer queries CA2 external trends in default window
- **WHEN** application queries CA2 external trend diagnostics without explicit window override
- **THEN** runtime returns provider-scoped aggregates for default window `15m` with required fields

#### Scenario: Consumer queries CA2 external trends with custom window
- **WHEN** application queries CA2 external trend diagnostics with explicit window parameter
- **THEN** runtime returns provider-scoped aggregates for the requested window

#### Scenario: Consumer reads existing diagnostics only
- **WHEN** existing integration reads only legacy run-level fields
- **THEN** diagnostics remain backward-compatible without requiring consumer changes

### Requirement: Runtime diagnostics SHALL emit threshold-hit signals without automatic strategy actions
Runtime MUST evaluate CA2 external trend aggregates against static thresholds and emit normalized threshold-hit signals for observability and operator workflows.

Threshold-hit signals MUST NOT trigger automatic provider switching, routing changes, or policy mutation in this milestone.

#### Scenario: p95 latency threshold is exceeded
- **WHEN** provider trend `p95_latency_ms` exceeds configured threshold
- **THEN** diagnostics include threshold-hit signal for `p95_latency_ms` and runtime behavior remains unchanged

#### Scenario: error-rate threshold is exceeded
- **WHEN** provider trend `error_rate` exceeds configured threshold
- **THEN** diagnostics include threshold-hit signal for `error_rate` and runtime behavior remains unchanged

#### Scenario: hit-rate threshold is under target
- **WHEN** provider trend `hit_rate` is below configured threshold
- **THEN** diagnostics include threshold-hit signal for `hit_rate` and runtime behavior remains unchanged
