# evaluation-corpus-badcase-and-experiment-contract Specification

## Purpose
Define versioned, reference-first evaluation corpora, reproducible Badcases, and fixed-identity experiment comparisons. Preserve baseline eligibility, isolate candidate selection from final scoring, account for attempt costs and evidence gaps, and keep offline replay and recommendations read-only and review-only.

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

### Requirement: Experiment evaluation SHALL expose additive continuity comparison

Evaluation comparison MUST be able to associate a bounded continuity comparison with a corpus item or experiment result by stable run/session and checkpoint/reference identity. This association MUST remain nullable and reference-only, MUST NOT change existing corpus or experiment aggregation semantics, and MUST classify continuity drift independently from metric or rubric drift.

#### Scenario: Experiment includes a valid continuity association
- **WHEN** an experiment result references a valid continuity comparison for the same run and corpus item
- **THEN** the result preserves existing metric comparison behavior and exposes the normalized continuity outcome

#### Scenario: Continuity drift does not mutate experiment aggregates
- **WHEN** a continuity comparison reports drift for an otherwise valid experiment shard
- **THEN** the aggregate retains its existing deterministic semantics and reports continuity drift as an additive classification

### Requirement: Badcase and experiment records SHALL expose additive first-error attribution association

Badcase and experiment records MUST be able to associate a versioned first-error attribution by bounded reference and stable corpus item, Badcase, run, step, or experiment correlation. The association MUST remain additive, nullable, and reference-only; historical payloads without attribution MUST retain existing defaults, and attribution drift MUST NOT change existing reproduction status, metric/rubric comparison, experiment aggregation, or continuity comparison semantics.

#### Scenario: Badcase references a valid attribution
- **WHEN** a replayable Badcase has a valid first-error attribution for the same corpus item and run correlation
- **THEN** the normalized Badcase exposes the attribution reference while preserving its existing reproduction status and digest behavior

#### Scenario: Historical Badcase omits attribution
- **WHEN** a historical Badcase or experiment payload contains no first-error attribution fields
- **THEN** normalization and comparison retain existing behavior using documented nullable defaults

#### Scenario: Attribution correlation conflicts
- **WHEN** a referenced attribution identifies a different corpus item, Badcase, run, or first-error step than the owning evaluation record
- **THEN** evaluation reports deterministic attribution-correlation drift and does not rewrite either record

### Requirement: Evidence-linked feedback SHALL remain review-only

Feedback recommendations MAY reference a validated first-error attribution and bounded evidence references, but MUST preserve explicit reviewer identity, decision context, and `pending|approved|rejected` status. Attribution-linked feedback MUST NOT automatically modify prompt, Skill, tool, policy, memory, runtime configuration, code, tests, gates, repository state, or execution decisions.

#### Scenario: Reviewer approves an evidence-linked recommendation
- **WHEN** a reviewer approves a recommendation with valid Badcase or experiment correlation, first-error attribution, bounded evidence, and decision context
- **THEN** evaluation emits an auditable review-only recommendation without applying it to runtime or repository behavior

#### Scenario: Recommendation lacks validated evidence context
- **WHEN** a recommendation cites a missing, conflicting, body-bearing, or unvalidated attribution or evidence reference
- **THEN** validation leaves the recommendation non-actionable and reports deterministic approval or evidence classification

#### Scenario: Approved recommendation is observed by a later run
- **WHEN** a later run can read an approved evidence-linked recommendation
- **THEN** prompt, Skill, tool, policy, memory, runtime configuration, code, tests, gates, and execution decisions remain unchanged unless a separate reviewed change explicitly implements it

### Requirement: Bounded harness candidate comparison SHALL preserve a fixed baseline and experiment identity
Candidate comparison MUST retain an immutable baseline and identify the corpus/Badcase and rubric versions, model, authority, data snapshot, budget, network policy, execution mode, and candidate asset identities and digests. Candidates MUST be limited to explicitly declared asset paths, size bounds, and static validation rules. The baseline MUST remain an eligible selection outcome.

#### Scenario: Candidate experiment uses a stable identity
- **WHEN** a comparison is admitted with valid baseline, candidate scope, and fixed experiment configuration
- **THEN** each result is bound to the normalized experiment identity and candidate asset digests, and baseline remains selectable

#### Scenario: Candidate exceeds declared asset scope
- **WHEN** a candidate modifies an undeclared asset, exceeds its size bound, or fails static validation
- **THEN** admission rejects that candidate with a stable scope or validation classification before probe execution

#### Scenario: Experiment configuration drifts under the same identity
- **WHEN** a result reuses a logical experiment identity with a different model, authority, data, budget, network policy, rubric, or asset digest
- **THEN** comparison reports deterministic experiment identity drift and does not merge the result

### Requirement: Candidate selection evidence SHALL be isolated from final scoring
Candidate selection MUST use only bounded, permitted candidate artifacts and selection evidence that excludes hidden rubric answers and final scoring data. Final scoring MUST occur independently after selection is committed. Unavailable or insufficient evidence MUST remain explicitly classified and MUST NOT be converted to a zero score or treated as evidence that a candidate is worse or better.

#### Scenario: Candidate selection is committed before final scoring
- **WHEN** baseline and candidate probe evidence is available for selection
- **THEN** the selected candidate or baseline and its bounded rationale are committed before the final scorer can access scoring-only data

#### Scenario: Final scoring evidence is unavailable
- **WHEN** a selected result cannot be scored or has insufficient valid evidence
- **THEN** the result is classified `unavailable` or `insufficient-evidence`, with no synthetic zero and no automatic candidate adoption

#### Scenario: Selection attempts to access scoring-only data
- **WHEN** the selection stage requests hidden rubric answers or other scoring-only data
- **THEN** the experiment rejects or records an isolation violation and does not accept that selection as valid evidence

### Requirement: Candidate experiments SHALL enforce isolation and produce cost and reliability evidence
Candidate comparisons MUST bind a declared egress/network policy and scoring-data visibility policy to the experiment identity. Reports MUST distinguish delivery quality, runtime reliability, contract regression, latency, and total cost, and MUST account for repeated executions, retries, cache hits, model calls, and incomplete outcomes without silently mixing configurations.

#### Scenario: Comparable runs use fixed isolation and budget
- **WHEN** baseline and candidates are probed for the same logical task
- **THEN** the experiment verifies matching declared model, authority, input data, budget, and network/egress policy before comparing outcomes

#### Scenario: Cost and outcome evidence is complete
- **WHEN** experiment executions finish or terminate incompletely
- **THEN** the report preserves separate quality, reliability, contract-regression, latency, and cost evidence with execution status and accounted attempts

#### Scenario: Network isolation cannot be verified
- **WHEN** the experiment cannot establish its declared egress or scoring-data isolation
- **THEN** it fails closed with an isolation-unavailable classification and does not publish a valid comparison

### Requirement: Harness improvement results SHALL remain review-only
Candidate comparison output and derived experience MUST be evidence-linked, bounded, and non-actionable. The experiment MUST NOT publish or activate candidate assets, mutate prompt/Skill/tool/policy/memory/configuration/code/tests/gates, create an automatic long-term experience library, or act as a production executor. Any newly discovered reproducible owner gap MUST be routed to a separately reviewed change.

#### Scenario: Candidate appears better than baseline
- **WHEN** candidate evidence is stronger than baseline under valid scoring
- **THEN** the comparison emits a review-only recommendation and does not publish, activate, or mutate any candidate asset

#### Scenario: New reproducible owner gap is found
- **WHEN** an experiment identifies a repeatable defect outside the existing evaluation contract
- **THEN** the result records bounded evidence and owner routing, while implementation requires a separate reviewed OpenSpec change

#### Scenario: Experiment completion is mistaken for runtime recovery
- **WHEN** an experiment is resumed or replayed after interruption
- **THEN** the operation remains read-only evaluation bookkeeping and does not claim native checkpoint restore, rerun a production parent, or invoke a production retry/recovery path
