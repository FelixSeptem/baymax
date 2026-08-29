# evaluation-corpus-badcase-and-experiment-contract Specification

## Purpose
TBD - created by archiving change extend-runtime-otel-and-agent-eval-with-corpus-badcase-and-experiment-contract. Update Purpose after archive.
## Requirements
### Requirement: Evaluation corpus SHALL be versioned and reference-first
The runtime MUST accept an evaluation corpus with a stable corpus version and item identifier. Each item MUST declare a scenario and MAY reference input, tool, policy, and runtime snapshot artifacts by bounded URI or digest. Corpus normalization MUST be deterministic, and incompatible corpus versions MUST fail before evaluation.

#### Scenario: Versioned corpus item is accepted
- **WHEN** a corpus declares a supported version and a unique item with valid scenario and references
- **THEN** the runtime emits a deterministic corpus digest and accepts the item for evaluation

#### Scenario: Unsupported corpus version is rejected
- **WHEN** a corpus declares an unsupported major version
- **THEN** evaluation is rejected with a deterministic corpus version drift classification and no result is emitted

#### Scenario: Missing required corpus identity is rejected
- **WHEN** a corpus item omits its corpus version, item identifier, or scenario
- **THEN** validation fails fast with a stable malformed-corpus classification

### Requirement: Badcase records SHALL be replayable and correlated
The runtime MUST represent a Badcase with a stable identifier, bounded classification, reproduction reference, and available `run_id`, `step_id`, Trace, Artifact, or Checkpoint correlation. A Badcase MUST report whether reproduction is deterministic, unavailable, or drifted without mutating the source runtime state.

#### Scenario: Reproducible Badcase is recorded
- **WHEN** a failed corpus item has a valid reproduction reference and matching run correlation
- **THEN** the runtime records a Badcase with deterministic identity and replay status `reproducible`

#### Scenario: Badcase reproduction input is unavailable
- **WHEN** the referenced reproduction input cannot be resolved
- **THEN** the runtime records status `unavailable` with `corpus_reference_unavailable` and does not mark the case as passed

#### Scenario: Badcase replay drifts
- **WHEN** replay produces a result different from the recorded normalized outcome
- **THEN** replay reports a deterministic Badcase drift classification and preserves both expected and observed digests

### Requirement: Metric and rubric declarations SHALL detect semantic drift
Each evaluation result MUST carry a metric/rubric version and normalized declaration digest. A changed declaration with the same logical experiment identity MUST be classified as drift rather than silently compared.

#### Scenario: Matching rubric declaration is compared
- **WHEN** two results reference the same rubric version and declaration digest
- **THEN** the runtime permits comparison using the declared metric semantics

#### Scenario: Rubric declaration changes
- **WHEN** two results use the same metric name but different declaration digests
- **THEN** comparison is rejected or marked `metric_rubric_drift` deterministically

### Requirement: Experiment comparison SHALL be deterministic and idempotent
Experiment comparison MUST identify corpus version, rubric version, run batch, and execution mode. Aggregation MUST deduplicate retries by stable item/shard identity, preserve local/distributed parity, and classify conflicting digests without last-write-wins behavior.

#### Scenario: Local and distributed results are equivalent
- **WHEN** local and distributed executions evaluate the same corpus and rubric with equivalent item outcomes
- **THEN** comparison produces semantically equivalent normalized metrics and a stable comparison digest

#### Scenario: Duplicate shard submission is resumed
- **WHEN** a distributed shard is submitted more than once with the same identity and digest
- **THEN** the duplicate is ignored idempotently and the aggregate remains unchanged

#### Scenario: Conflicting duplicate is submitted
- **WHEN** the same shard identity is submitted with a different digest
- **THEN** aggregation returns a deterministic `experiment_aggregate_conflict` classification

### Requirement: Human feedback SHALL be an auditable recommendation only
Feedback recommendations MUST reference an experiment or Badcase, include reviewer identity and decision context, and use an explicit `pending|approved|rejected` status. Missing approval context MUST be classified as `approval_missing`; recommendations MUST NOT mutate prompt, tool, policy, memory, or runtime configuration.

#### Scenario: Approved recommendation is projected
- **WHEN** a reviewer approves a recommendation with valid experiment/Badcase references and context
- **THEN** the runtime emits a bounded, auditable feedback projection with status `approved`

#### Scenario: Approval context is missing
- **WHEN** a recommendation lacks reviewer identity or decision context
- **THEN** the runtime emits `approval_missing` and leaves the recommendation non-actionable

#### Scenario: Recommendation cannot change execution policy
- **WHEN** a feedback recommendation is present during a subsequent run
- **THEN** the runtime does not alter prompt, tool, policy, memory, or runtime configuration based on that recommendation

