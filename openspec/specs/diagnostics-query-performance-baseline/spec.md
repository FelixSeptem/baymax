# diagnostics-query-performance-baseline Specification

## Purpose
TBD - created by archiving change introduce-diagnostics-query-performance-baseline-and-regression-gate-a42. Update Purpose after archive.
## Requirements
### Requirement: Repository SHALL define diagnostics-query benchmark matrix
The repository MUST define benchmark coverage for diagnostics query mainline surfaces:
- `QueryRuns`
- `QueryMailbox`
- `MailboxAggregates`

Each benchmark MUST be runnable via `go test ./integration -run '^$' -bench ...` without external network dependency.

#### Scenario: Contributor executes diagnostics-query benchmark suite
- **WHEN** contributor runs the documented diagnostics-query benchmark command
- **THEN** benchmark output includes all required query categories (`QueryRuns`, `QueryMailbox`, `MailboxAggregates`)

#### Scenario: CI executes diagnostics-query benchmark suite
- **WHEN** CI runs diagnostics-query benchmark suite in standard validation flow
- **THEN** suite completes without external API credentials or network calls

### Requirement: Diagnostics-query baseline SHALL use deterministic dataset and parameters
The diagnostics-query regression workflow MUST use repository-owned deterministic dataset generation and default run parameters.

Default parameters MUST be:
- `benchtime=200ms`
- `count=5`

The comparison MUST evaluate relative degradation for:
- `ns/op`
- `p95-ns/op`
- `allocs/op`

#### Scenario: Regression check runs without explicit overrides
- **WHEN** environment variables are not provided
- **THEN** workflow uses deterministic default dataset and documented default parameters

#### Scenario: Regression check computes query metric deltas
- **WHEN** benchmark candidate results are produced
- **THEN** workflow compares candidate vs baseline on `ns/op`, `p95-ns/op`, and `allocs/op`

### Requirement: Diagnostics-query regression gate SHALL fail fast on invalid input or parse failure
If baseline values are missing/non-numeric, threshold parameters are invalid, or benchmark output lacks required metrics, the diagnostics-query regression gate MUST fail fast and return non-zero status.

#### Scenario: Baseline lacks required metric for one query category
- **WHEN** baseline input misses any required metric for `QueryRuns` or `QueryMailbox` or `MailboxAggregates`
- **THEN** gate exits non-zero with explicit missing-baseline error

#### Scenario: Benchmark output omits required metric token
- **WHEN** benchmark output cannot be parsed for `ns/op`, `p95-ns/op`, or `allocs/op`
- **THEN** gate exits non-zero with explicit parse-failure classification

### Requirement: Diagnostics-query thresholds SHALL be configurable with strict defaults
The diagnostics-query regression gate MUST expose threshold configuration via environment variables and MUST apply strict defaults when variables are absent.

Default threshold values MUST be:
- max `ns/op` degradation `12%`
- max `p95-ns/op` degradation `15%`
- max `allocs/op` degradation `12%`

#### Scenario: Candidate exceeds diagnostics-query p95 threshold
- **WHEN** one query benchmark category exceeds default `p95-ns/op` threshold
- **THEN** diagnostics-query gate fails and blocks validation

#### Scenario: Maintainer overrides diagnostics-query thresholds
- **WHEN** maintainer provides explicit threshold environment variables for one run
- **THEN** gate uses overridden values while preserving remaining default semantics

### Requirement: Diagnostics-query benchmark matrix SHALL cover sandbox-enriched run summaries
Diagnostics-query performance baseline MUST include sandbox-enriched run-summary dataset coverage to detect regression introduced by sandbox additive fields.

Sandbox-enriched coverage MUST include at minimum:
- sandbox decision fields populated,
- sandbox fallback markers populated,
- sandbox failure counters populated.

#### Scenario: QueryRuns benchmark executes sandbox-enriched dataset
- **WHEN** diagnostics-query benchmark suite runs with default dataset generator
- **THEN** QueryRuns benchmark includes sandbox-enriched records in deterministic workload composition

#### Scenario: Sandbox field growth causes threshold breach
- **WHEN** sandbox-related additive fields introduce measurable regression beyond configured threshold
- **THEN** diagnostics-query regression gate fails and blocks validation

### Requirement: Diagnostics-query benchmark SHALL include sandbox rollout-enriched dataset profile
Diagnostics-query benchmark matrix MUST include sandbox rollout-enriched dataset profile that covers additive fields introduced by rollout governance.

At minimum, dataset profile MUST include records with:
- mixed rollout phases (`observe|canary|baseline|full|frozen`)
- mixed capacity actions (`allow|throttle|deny`)
- mixed health budget states (`within_budget|near_budget|breached`)

#### Scenario: Contributor executes diagnostics-query benchmark with sandbox-enriched profile
- **WHEN** benchmark suite runs with default diagnostics profile set
- **THEN** output includes sandbox rollout-enriched query categories and deterministic metric collection

#### Scenario: CI executes sandbox-enriched diagnostics benchmark
- **WHEN** quality gate runs diagnostics performance regression checks
- **THEN** sandbox rollout-enriched profile is included without external dependency requirements

### Requirement: Diagnostics-query regression gate SHALL enforce thresholds for sandbox-enriched query paths
Regression gate MUST enforce documented relative-threshold policy for sandbox-enriched query paths using the same deterministic baseline comparison semantics as existing diagnostics-query checks.

#### Scenario: Sandbox-enriched query p95 regression exceeds threshold
- **WHEN** one or more sandbox-enriched query paths exceed configured `p95-ns/op` regression threshold
- **THEN** diagnostics-query regression gate fails and blocks validation

#### Scenario: Sandbox-enriched query metrics remain within thresholds
- **WHEN** sandbox-enriched query candidate metrics stay within configured thresholds
- **THEN** diagnostics-query regression gate passes without blocking validation

