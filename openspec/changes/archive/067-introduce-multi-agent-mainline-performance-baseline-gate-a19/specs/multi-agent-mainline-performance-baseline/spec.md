## ADDED Requirements

### Requirement: Repository SHALL define multi-agent mainline benchmark matrix
The repository MUST define benchmark coverage for multi-agent mainline execution paths, including synchronous invocation, async reporting, delayed dispatch, and recovery replay paths.

Each benchmark entry MUST be runnable via `go test ./integration -run '^$' -bench ...` without requiring external services.

#### Scenario: Contributor executes multi-agent benchmark suite
- **WHEN** contributor runs the documented benchmark commands
- **THEN** benchmark results include all required mainline categories (sync, async, delayed, recovery)

#### Scenario: CI executes benchmark suite on clean workspace
- **WHEN** CI runs multi-agent benchmark suite in standard validation flow
- **THEN** suite completes without external network dependencies

### Requirement: Multi-agent performance baseline SHALL use deterministic comparison inputs
The benchmark regression workflow MUST provide repository-owned baseline values and deterministic default run parameters.

Default parameters MUST be:
- `benchtime=200ms`
- `count=5`

The comparison MUST evaluate relative degradation for `ns/op`, `p95-ns/op`, and `allocs/op`.

#### Scenario: Regression check runs without explicit overrides
- **WHEN** environment does not override benchmark parameters
- **THEN** workflow uses default `benchtime=200ms` and `count=5`

#### Scenario: Regression check computes metric deltas
- **WHEN** benchmark candidate results are produced
- **THEN** workflow compares candidate vs baseline on `ns/op`, `p95-ns/op`, and `allocs/op`

### Requirement: Multi-agent regression gate SHALL fail fast on invalid baseline or invalid parameters
If baseline values are missing, non-numeric, or threshold parameters are invalid, the regression gate MUST fail fast and return non-zero exit status.

If benchmark output cannot be parsed into required metrics, the regression gate MUST fail fast and MUST NOT silently pass.

#### Scenario: Baseline file misses required metric
- **WHEN** baseline input lacks required `ns/op`, `p95-ns/op`, or `allocs/op` values
- **THEN** gate exits non-zero with explicit missing-baseline error

#### Scenario: Benchmark output format is incomplete
- **WHEN** benchmark output does not include one of required metrics
- **THEN** gate exits non-zero with explicit parse-failure classification

### Requirement: Multi-agent performance thresholds SHALL be configurable with strict defaults
The regression gate MUST expose threshold configuration through environment variables and MUST apply strict defaults when variables are not provided.

Default threshold values MUST be:
- max `ns/op` degradation `8%`
- max `p95-ns/op` degradation `12%`
- max `allocs/op` degradation `10%`

#### Scenario: Candidate exceeds p95 threshold
- **WHEN** candidate `p95-ns/op` degradation is greater than `12%` under default settings
- **THEN** gate fails and blocks validation

#### Scenario: Maintainer overrides threshold for controlled rollout
- **WHEN** maintainer provides explicit threshold environment variables
- **THEN** gate uses overridden values for that run and keeps remaining defaults unchanged

