# engineering-and-performance-optimization-contract Specification

## Purpose
TBD - created by archiving change introduce-engineering-and-performance-optimization-contract-a64. Update Purpose after archive.
## Requirements
### Requirement: A64 Scope Mapping and Single-Track Absorption
The optimization program SHALL absorb performance and engineering changes through a fixed S1~S10 scope map, and any newly discovered hotspot MUST be mapped to an existing S item before implementation.

#### Scenario: Hotspot intake is mapped before coding
- **WHEN** a maintainer identifies a new optimization hotspot in `core/*`, `context/*`, `runtime/*`, `observability/*`, `mcp/*`, `model/*`, `skill/*`, `memory/*`, `orchestration/*`, or `tool/local/*`
- **THEN** the hotspot is recorded as an incremental task under one of A64-S1~S10 instead of creating a parallel performance proposal

#### Scenario: Scope map is treated as merge prerequisite
- **WHEN** a PR claims to implement an A64 optimization
- **THEN** review evidence includes explicit S-item mapping and impacted-contract suites selection

### Requirement: Semantic Stability Is a Hard Blocker
All A64 optimizations SHALL preserve existing external semantics, and contract or replay drift MUST block merge.

#### Scenario: Semantic drift blocks merge
- **WHEN** an optimization changes Run/Stream equivalence, reason taxonomy, diagnostics schema semantics, or replay idempotency
- **THEN** semantic-stability checks fail and the change is rejected

#### Scenario: No behavior change under optimization toggles
- **WHEN** optimization toggles are switched between baseline and optimized paths
- **THEN** externally observable contract behavior remains equivalent

#### Scenario: Hard semantic fields remain unchanged
- **WHEN** A64 optimizations touch runner, scheduler, MCP, memory, or observability hotpaths
- **THEN** `backpressure`、`fail_fast`、`timeout/cancel`、`reason taxonomy`、`decision trace` semantics remain equivalent

### Requirement: Optimization Paths Are Toggleable and Rollback-Safe
Each optimization path SHALL provide explicit enable/disable control and MUST support rollback without changing default behavior semantics.

#### Scenario: Toggle-off returns to baseline path
- **WHEN** an optimization toggle is disabled
- **THEN** execution falls back to baseline logic and keeps the same contract output semantics

#### Scenario: Invalid hot-update rollback is atomic
- **WHEN** an invalid runtime update targets optimization-related config
- **THEN** the runtime rejects the update fail-fast and atomically rolls back to the previous valid snapshot

### Requirement: Module-Scoped Benchmark Governance
A64 benchmark governance SHALL provide module-scoped benchmark entrypoints and enforce regression thresholds on `ns/op`, `allocs/op`, and `B/op`.

#### Scenario: Benchmarks are independently executable
- **WHEN** a maintainer validates an S-item optimization
- **THEN** the associated benchmark can be executed independently at module scope without running the entire benchmark suite

#### Scenario: Regression threshold blocks merge
- **WHEN** benchmark results exceed configured regression thresholds
- **THEN** performance-regression checks fail and merge is blocked

### Requirement: Existing Performance Baseline Suites Are Mandatory for Mapped Hotpaths
A64 optimization validation SHALL execute existing baseline performance suites for context, diagnostics query, and multi-agent concurrent paths according to S-item mapping.

#### Scenario: Context hotpath change executes context benchmark regression suite
- **WHEN** A64 changes are mapped to S1 context assembly/stage2 hotspots
- **THEN** `check-context-production-hardening-benchmark-regression.sh/.ps1` is executed as a blocking performance step

#### Scenario: Diagnostics hotpath change executes diagnostics benchmark regression suite
- **WHEN** A64 changes are mapped to S2 diagnostics/recorder query hotspots
- **THEN** `check-diagnostics-query-performance-regression.sh/.ps1` is executed as a blocking performance step

#### Scenario: Concurrent orchestration hotpath change executes multi-agent performance regression suite
- **WHEN** A64 changes are mapped to S3/S7 concurrent orchestration hotspots
- **THEN** `check-multi-agent-performance-regression.sh/.ps1` is executed as a blocking performance step

### Requirement: Impacted Contract Suites Mapping Is Explicit and Enforced
A64 validation SHALL define explicit S1~S10 impacted-contract suites mappings and enforce them as blocking checks.

#### Scenario: S-item contract suites are explicitly declared
- **WHEN** A64 gate wiring is reviewed
- **THEN** each S-item includes explicit minimum contract/replay suite commands with shell and PowerShell parity

#### Scenario: Missing S-item suite fails validation
- **WHEN** an A64 change is validated without required impacted suites for its mapped S-item
- **THEN** gate validation fails and blocks merge

### Requirement: Repository Hygiene Covers Untracked Temporary Artifacts
Repository hygiene checks SHALL detect both tracked and untracked temporary backup artifacts that match banned patterns.

#### Scenario: Untracked backup artifacts are blocked
- **WHEN** the workspace contains untracked files matching `*.go.<digits>`, `*.tmp`, `*.bak`, or `*~`
- **THEN** repository hygiene checks fail and report the violating paths

### Requirement: Inferential Feedback Sensors SHALL Stay Advisory in A64
Inferential feedback signals introduced by A64 SHALL be advisory-only and MUST NOT directly change existing readiness/admission deny semantics.

#### Scenario: Inferential feedback is emitted without decision drift
- **WHEN** A64 integrates `runtime.eval.*` and runtime quality signals into diagnostics/recorder hotpaths
- **THEN** runtime emits advisory observability fields while preserving readiness/admission deny outcomes under equivalent policy and budget inputs

#### Scenario: Inferential feedback drift is detected by replay
- **WHEN** replay fixtures evaluate inferential feedback outputs across equivalent inputs
- **THEN** drift classification MUST fail validation if advisory payload semantics become non-deterministic

### Requirement: Interaction-State Recovery SHALL Reuse Unified Snapshot Contract
A64 interaction-state recovery optimization SHALL reuse unified state/session snapshot contract extensions and MUST NOT introduce a second source-of-truth for realtime/handoff recovery.

#### Scenario: Realtime cursor and isolate-handoff state recover after restart
- **WHEN** runtime restarts after interrupt/resume or isolate-handoff progression
- **THEN** recovered interaction-state remains semantically equivalent through unified snapshot import/restore path

#### Scenario: Recovery optimization does not alter existing contract semantics
- **WHEN** A64 extends interaction-state persistence boundaries
- **THEN** A66/A67-CTX/A68 contract outputs remain semantically equivalent under Run/Stream and replay

### Requirement: Snapshot Entropy Governance SHALL Be Bounded and Rollback-Safe
A64 snapshot entropy governance SHALL provide bounded `retention/quota/cleanup` controls with fail-fast validation and atomic hot-update rollback while keeping default behavior unchanged.

#### Scenario: Invalid entropy governance update is rolled back atomically
- **WHEN** hot-update applies invalid snapshot entropy configuration values
- **THEN** runtime rejects update fail-fast and atomically restores previous valid snapshot configuration

#### Scenario: Entropy cleanup keeps restore semantics stable
- **WHEN** snapshot cleanup runs within configured entropy budget
- **THEN** strict/compatible restore semantics and replay outputs remain contract-compatible

### Requirement: Multi-Agent Emergent Behavior Governance SHALL Be Deterministic and Blocking
A64 optimizations touching concurrent orchestration paths SHALL include deterministic multi-agent behavior matrix validation, and emergent drift MUST block merge.

#### Scenario: Concurrency-sensitive optimization is validated with emergent matrix
- **WHEN** an A64 optimization modifies scheduler/mailbox/composer, runner dispatch, or observability fanout hotpaths
- **THEN** validation includes multi-agent matrix suites that cover parallel, interleaving, retry, and replay cases

#### Scenario: Emergent drift blocks merge
- **WHEN** matrix suites detect cascade amplification, ordering instability, or non-deterministic terminal outcomes
- **THEN** validation fails and merge is blocked until drift is resolved

### Requirement: Harness ROI and Adaptive Depth Governance SHALL Prevent Over-Engineering
A64 harness governance SHALL quantify token/latency/quality tradeoffs and enforce adaptive depth policies so harness overhead does not exceed configured ROI thresholds.

#### Scenario: ROI threshold breach triggers governance action
- **WHEN** measured harness overhead exceeds configured ROI thresholds for the selected complexity tier
- **THEN** validation requires depth downgrade or explicit threshold update evidence before merge

#### Scenario: Adaptive depth keeps semantics stable
- **WHEN** runtime switches between lightweight/standard/enhanced harness depth profiles
- **THEN** external contract semantics remain equivalent while governance metrics record the selected profile and rationale

### Requirement: Gate Selection SHALL Be Impact-Aware Without Skipping Mandatory Suites
A64 validation SHALL select `fast/full` gate execution based on changed-files impact mapping, and MUST always include mandatory contract/performance suites for touched S-items.

#### Scenario: Fast mode only trims unrelated suites
- **WHEN** changed-files are mapped to a subset of A64 S-items
- **THEN** validation MAY run `fast` mode that trims unrelated suites but still executes all mandatory suites for mapped S-items

#### Scenario: Missing mandatory suite blocks merge
- **WHEN** impacted-gate selection omits a mandatory suite for a mapped S-item
- **THEN** validation fails and merge is blocked

### Requirement: Gate Latency Budget SHALL Be Measured and Governed
A64 validation SHALL emit step-level gate latency metrics and enforce configured latency regression thresholds with auditable baseline update process.

#### Scenario: Step-level latency report is produced
- **WHEN** A64 gate pipeline completes
- **THEN** report output includes per-step duration metrics in machine-readable form

#### Scenario: Latency regression threshold breach blocks merge
- **WHEN** gate step duration exceeds configured budget threshold without approved baseline update
- **THEN** validation fails and merge is blocked

