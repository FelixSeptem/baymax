## ADDED Requirements

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
