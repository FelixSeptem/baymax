## ADDED Requirements

### Requirement: Runtime config SHALL expose cancel-storm and backpressure baseline controls
Runtime configuration MUST expose concurrency baseline controls for cancellation storm and backpressure behavior with deterministic precedence `env > file > default`. Default backpressure mode MUST be `block`.

This requirement MUST NOT introduce a new public API surface; configuration behavior MUST remain library-first through existing runtime config manager entry points.

#### Scenario: Startup with default cancel-storm controls
- **WHEN** runtime starts without explicit cancel-storm/backpressure overrides
- **THEN** effective configuration uses documented defaults including backpressure mode `block`

#### Scenario: Startup with environment overrides for concurrency controls
- **WHEN** YAML and environment variables both define cancel-storm/backpressure fields
- **THEN** effective values resolve by `env > file > default`

#### Scenario: Startup with invalid backpressure mode
- **WHEN** configuration provides unsupported backpressure mode value
- **THEN** runtime fails fast and rejects startup or hot-reload snapshot

### Requirement: Runtime diagnostics SHALL expose minimal cancel-storm and backpressure counters
Run diagnostics MUST expose the following minimum fields for concurrency-control observability: `cancel_propagated_count`, `backpressure_drop_count`, and `inflight_peak`.

`cancel_propagated_count` MUST be non-negative and count successful cancellation propagation actions.
`backpressure_drop_count` MUST be non-negative and MUST remain zero when active policy is `block`.
`inflight_peak` MUST be non-negative and represent the run-scoped peak in-flight work count.

#### Scenario: Consumer inspects diagnostics after canceled high-fanout run
- **WHEN** a run is canceled during high-fanout execution
- **THEN** diagnostics expose non-negative `cancel_propagated_count` and `inflight_peak`

#### Scenario: Consumer inspects diagnostics under default block policy
- **WHEN** active backpressure policy is `block`
- **THEN** diagnostics expose `backpressure_drop_count` as zero without breaking schema compatibility

#### Scenario: Consumer inspects diagnostics with no concurrency pressure
- **WHEN** run completes without cancellation and without backpressure pressure-hit
- **THEN** diagnostics still expose zero-valued baseline fields in a stable schema

### Requirement: Runtime performance baseline SHALL include p95 latency and goroutine peak gates
Quality validation for runtime concurrency baseline MUST include contract-level verification and benchmark/pressure checks for `p95 latency` and `goroutine peak`.

#### Scenario: Quality gate checks concurrency baseline
- **WHEN** maintainers run baseline quality checks for this capability
- **THEN** reported outputs include both `p95 latency` and `goroutine peak` signals for regression judgment
