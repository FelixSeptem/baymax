## ADDED Requirements

### Requirement: A64 Semantic Stability Gate Is Mandatory
The quality gate SHALL execute an A64 semantic stability contract step that blocks merge on behavior drift.

#### Scenario: Semantic stability gate runs in required pipeline
- **WHEN** A64-related code changes are validated
- **THEN** `check-a64-semantic-stability-contract.sh/.ps1` is executed as a required blocking step

#### Scenario: Drift fails the quality gate
- **WHEN** semantic stability checks detect Run/Stream non-equivalence, diagnostics schema drift, or replay idempotency drift
- **THEN** the quality gate exits non-zero and blocks merge

### Requirement: A64 Performance Regression Gate Is Mandatory
The quality gate SHALL execute an A64 performance regression step with threshold enforcement on `ns/op`, `allocs/op`, and `B/op`.

#### Scenario: Performance gate runs in required pipeline
- **WHEN** A64-related optimizations are validated
- **THEN** `check-a64-performance-regression.sh/.ps1` is executed as a required blocking step

#### Scenario: Threshold breach fails gate
- **WHEN** benchmark results exceed configured A64 thresholds
- **THEN** quality gate validation fails and merge is blocked

### Requirement: Impacted Contract Suites Are Enforced per S Mapping
The quality gate SHALL enforce A64 impacted-contract suites according to S1~S10 module mapping, and missing required suites MUST fail validation.

#### Scenario: Missing impacted suite is rejected
- **WHEN** a PR modifies modules mapped to an A64 S-item but omits required impacted suites
- **THEN** gate validation fails with explicit missing-suite diagnostics

#### Scenario: Shell and PowerShell parity is preserved
- **WHEN** A64 gate steps run on shell and PowerShell environments
- **THEN** pass/fail semantics remain equivalent for the same input state

#### Scenario: S-matrix command set is complete
- **WHEN** quality-gate mapping for A64 is audited
- **THEN** S1~S10 and cross-cutting fallback suites are all present with explicit command definitions

### Requirement: Repo Hygiene Includes Untracked Artifact Scanning
Repository hygiene in the quality gate SHALL scan both tracked and untracked files for banned temporary artifact patterns.

#### Scenario: Untracked banned files fail hygiene
- **WHEN** `git ls-files --others --exclude-standard` returns files matching banned patterns
- **THEN** repository hygiene fails and blocks the quality gate

### Requirement: A64 Harnessability Scorecard Gate Is Mandatory
The quality gate SHALL execute an A64 harnessability scorecard step that emits machine-readable governance output and blocks merge when configured thresholds are violated.

#### Scenario: Harnessability scorecard runs in required pipeline
- **WHEN** A64-related changes are validated
- **THEN** `check-a64-harnessability-scorecard.sh/.ps1` is executed as a required blocking step

#### Scenario: Scorecard threshold breach fails gate
- **WHEN** scorecard metrics for contract coverage, replay drift health, gate coverage, or docs consistency violate configured thresholds
- **THEN** quality gate validation fails and merge is blocked

#### Scenario: Scorecard report is machine-readable
- **WHEN** harnessability scorecard execution completes
- **THEN** output includes machine-readable report artifacts consumable by CI/PR checks

### Requirement: Multi-Agent Emergent Matrix Validation Is Mandatory for Concurrent Hotpaths
The quality gate SHALL execute deterministic multi-agent emergent matrix suites for concurrency-sensitive A64 changes.

#### Scenario: Concurrent hotpath change requires emergent suites
- **WHEN** A64-related changes touch scheduler/mailbox/composer orchestration, runner dispatch concurrency, or observability fanout paths
- **THEN** quality gate executes the required multi-agent emergent matrix suites and blocks merge on drift

#### Scenario: Emergent drift fails gate
- **WHEN** matrix validation reports non-deterministic ordering, cascade amplification, or terminal-state instability
- **THEN** quality gate exits non-zero and blocks merge

### Requirement: Harness ROI and Adaptive Depth Metrics Are Mandatory in Scorecard
The quality gate SHALL require harness ROI/depth metrics in A64 scorecard output and block merge when configured overhead thresholds are violated.

#### Scenario: ROI/depth metrics are emitted
- **WHEN** A64 harnessability scorecard runs
- **THEN** machine-readable output includes token/latency/quality ROI metrics and selected depth profile metadata

#### Scenario: ROI threshold breach fails gate
- **WHEN** measured harness overhead exceeds configured ROI thresholds for the declared complexity tier
- **THEN** quality gate validation fails unless approved baseline update evidence is provided

### Requirement: Computational-First Sensor Hierarchy Is Mandatory
The quality gate SHALL enforce computational checks as objective blocking baseline, and inferential checks SHALL remain supplementary for subjective quality domains.

#### Scenario: Objective correctness cannot be gated by inferential-only signal
- **WHEN** validating contract/replay/schema/taxonomy correctness domains
- **THEN** gate blocking decision MUST come from computational suites, and inferential checks MUST NOT be the sole blocker

#### Scenario: Inferential findings include structured evidence
- **WHEN** inferential checks are used for subjective quality domains
- **THEN** results include structured evidence payloads (input snapshot, prompt/version metadata, scoring summary) for auditability

### Requirement: A64 Performance Regression Gate SHALL Aggregate Existing Performance Baseline Suites
The A64 performance regression gate MUST explicitly execute existing context, diagnostics-query, and multi-agent performance regression suites as mandatory sub-steps.

#### Scenario: Context performance suite is required for A64 hotpaths
- **WHEN** A64 validation covers S1 context assembly or stage2 hotspots
- **THEN** `check-context-production-hardening-benchmark-regression.sh/.ps1` is executed and failure blocks merge

#### Scenario: Diagnostics-query performance suite is required for A64 hotpaths
- **WHEN** A64 validation covers S2 diagnostics/recorder query hotspots
- **THEN** `check-diagnostics-query-performance-regression.sh/.ps1` is executed and failure blocks merge

#### Scenario: Multi-agent performance suite is required for A64 concurrent hotpaths
- **WHEN** A64 validation covers S3/S7 concurrency-sensitive orchestration hotpaths
- **THEN** `check-multi-agent-performance-regression.sh/.ps1` is executed and failure blocks merge

### Requirement: A64 Impact-Aware Gate Selection Is Mandatory
The quality gate SHALL execute an A64 impacted-gate selection step that maps changed files to S1~S10 suites and validates `fast/full` selection correctness.

#### Scenario: Impact map drives fast/full selection
- **WHEN** A64 validation starts
- **THEN** `check-a64-impacted-gate-selection.sh/.ps1` determines impacted suites and enforces mandatory suite completeness for mapped S-items

#### Scenario: Invalid fast mode selection blocks gate
- **WHEN** fast mode attempts to skip mandatory suites for touched S-items
- **THEN** quality gate exits non-zero and blocks merge

### Requirement: A64 Gate Latency Budget Check Is Mandatory
The quality gate SHALL execute an A64 gate latency budget step with step-level metrics and regression threshold enforcement.

#### Scenario: Gate latency budget step runs in required pipeline
- **WHEN** A64-related changes are validated
- **THEN** `check-a64-gate-latency-budget.sh/.ps1` runs as a blocking step and emits machine-readable step-duration metrics

#### Scenario: Latency regression breach blocks gate
- **WHEN** measured gate step duration exceeds configured budget thresholds without approved baseline update
- **THEN** quality gate validation fails and merge is blocked
