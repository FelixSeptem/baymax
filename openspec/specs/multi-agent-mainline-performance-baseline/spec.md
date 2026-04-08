# multi-agent-mainline-performance-baseline Specification

## Purpose
TBD - created by archiving change introduce-multi-agent-mainline-performance-baseline-gate-a19. Update Purpose after archive.
## Requirements
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

### Requirement: A64 Hotpath Benchmark Matrix Coverage
The mainline performance baseline SHALL include benchmark coverage for A64 hotspot groups (S1~S10) with stable benchmark naming and ownership mapping.

#### Scenario: Benchmark matrix includes all hotspot groups
- **WHEN** the A64 baseline is assembled
- **THEN** benchmark coverage exists for context assembly, diagnostics/recorder, scheduler/mailbox persistence, MCP invoke path, skill loader, memory filesystem, runner/local dispatch, provider decode, runtime config resolve, and observability pipeline

#### Scenario: Benchmark naming is stable and mappable
- **WHEN** benchmark suites are published for A64
- **THEN** benchmark names and docs provide deterministic mapping to S1~S10 ownership

### Requirement: Module-Level Execution Entry for A64 Benchmarks
A64 benchmark suites SHALL provide module-level execution entrypoints so maintainers can validate one hotspot group without running the full benchmark corpus.

#### Scenario: Single-module benchmark execution
- **WHEN** a maintainer runs A64 validation for a specific S-item
- **THEN** benchmark commands exist to run only that module's benchmark set

#### Scenario: Module benchmark output is threshold-evaluable
- **WHEN** module-level benchmark execution completes
- **THEN** output includes metrics consumable by the A64 regression threshold checker

### Requirement: Baseline Update Governance Is Explicit
Baseline refresh for A64 benchmarks SHALL require explicit baseline artifacts and documented reason before threshold adjustments are accepted.

#### Scenario: Threshold adjustment requires baseline evidence
- **WHEN** a contributor proposes changing A64 benchmark thresholds
- **THEN** the change includes updated baseline evidence and rationale in the same proposal/PR context

### Requirement: Multi-Agent Performance Regression Suite Is Mandatory for A64
The multi-agent performance regression suite MUST be executed as a required gate step for A64 changes that touch concurrent orchestration hotpaths.

#### Scenario: Multi-agent performance suite is wired as blocker
- **WHEN** A64 validation includes S3/S7 concurrent orchestration optimizations
- **THEN** `check-multi-agent-performance-regression.sh/.ps1` executes as a blocking step

