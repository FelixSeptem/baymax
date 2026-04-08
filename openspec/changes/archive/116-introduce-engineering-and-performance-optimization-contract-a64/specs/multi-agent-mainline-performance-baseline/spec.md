## ADDED Requirements

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
