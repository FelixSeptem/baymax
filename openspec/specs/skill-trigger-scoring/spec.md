# skill-trigger-scoring Specification

## Purpose
TBD - created by archiving change introduce-skill-trigger-scoring-and-contract-tests-d1. Update Purpose after archive.
## Requirements
### Requirement: Skill loader SHALL use configurable lexical trigger scoring by default
The skill loader MUST evaluate semantic trigger candidates with a configurable trigger-scoring strategy. The default strategy MUST remain lexical weighted-keyword scoring, and runtime configuration MUST be able to switch to embedding-enhanced strategy without changing external APIs.

#### Scenario: Default strategy remains lexical
- **WHEN** runtime starts with default skill trigger scoring configuration
- **THEN** loader uses lexical weighted-keyword scoring and preserves existing trigger behavior baseline

#### Scenario: Runtime switches to embedding-enhanced strategy
- **WHEN** runtime configuration sets strategy to `lexical_plus_embedding`
- **THEN** loader evaluates candidates with lexical+embedding fusion and applies configured confidence threshold

### Requirement: Skill loader SHALL use highest-priority tie-break for equal scores
When two or more skill candidates have equal final score, the loader MUST deterministically select by `highest-priority` rule.

#### Scenario: Equal scores with different priorities
- **WHEN** two candidates produce the same score and one has higher configured priority
- **THEN** loader selects the higher-priority candidate

#### Scenario: Equal scores and equal priorities
- **WHEN** two candidates produce the same score and same priority
- **THEN** loader applies deterministic stable order and produces repeatable selection result

### Requirement: Low-confidence suppression SHALL be enabled by default
The runtime MUST enable low-confidence suppression by default so weak semantic matches do not trigger skill activation unless explicitly disabled.

#### Scenario: Default config without explicit suppression override
- **WHEN** runtime starts with default skill trigger scoring configuration
- **THEN** low-confidence suppression is enabled and below-threshold candidates are filtered out

#### Scenario: Explicit suppression disable
- **WHEN** runtime configuration explicitly disables low-confidence suppression
- **THEN** loader allows below-threshold candidates to continue according to configured fallback behavior

### Requirement: Skill trigger scoring architecture SHALL reserve scorer extension interface
The implementation MUST provide an internal scorer extension interface for embedding-based trigger scoring integration and MUST support host registration of embedding scorer implementation without changing public loader APIs.

#### Scenario: Host registers embedding scorer implementation
- **WHEN** runtime runs with `lexical_plus_embedding` and a valid embedding scorer is registered
- **THEN** loader invokes registered scorer and merges embedding signal into final score

#### Scenario: Runtime runs without embedding scorer registration
- **WHEN** runtime runs with `lexical_plus_embedding` but no embedding scorer is registered
- **THEN** loader falls back to lexical-only scoring path with normalized fallback reason and continues compile flow

### Requirement: Skill trigger scoring behavior SHALL be guarded by contract tests
Repository MUST include contract tests that verify threshold behavior, tie-break determinism, and low-confidence suppression defaults.

#### Scenario: Contract suite validates equal-score tie-break
- **WHEN** contract tests execute with equal-score candidate fixtures
- **THEN** tests assert deterministic `highest-priority` selection

#### Scenario: Contract suite validates suppression defaults
- **WHEN** contract tests execute with default configuration
- **THEN** below-threshold candidates are not activated and test fails on regression

### Requirement: Skill trigger scoring SHALL support linear weighted lexical-plus-embedding fusion
For `lexical_plus_embedding` strategy, loader MUST compute final candidate score with linear weighted fusion:
`final_score = lexical_weight * lexical_score + embedding_weight * embedding_score`.

Weight values MUST be runtime-configurable and validated before activation.

#### Scenario: Fusion score is computed from lexical and embedding signals
- **WHEN** loader evaluates a candidate under `lexical_plus_embedding` with valid weights and both scores available
- **THEN** candidate final score equals configured linear weighted result

#### Scenario: Invalid weight configuration is rejected
- **WHEN** runtime configuration sets invalid lexical/embedding weights
- **THEN** startup or hot reload fails fast and previous active configuration remains unchanged

### Requirement: Skill trigger scoring SHALL fallback to lexical path on embedding failure under best-effort policy
Embedding path failures (missing scorer, timeout, invocation error, invalid score) MUST NOT terminate skill selection flow. Loader MUST fallback to lexical scoring and emit normalized fallback observability fields.

#### Scenario: Embedding scorer timeout fallback
- **WHEN** embedding scorer invocation exceeds configured timeout
- **THEN** loader falls back to lexical scoring and skill compile continues without fail-fast termination

#### Scenario: Embedding scorer returns invalid score
- **WHEN** embedding scorer returns NaN, infinity, or out-of-range normalized score
- **THEN** loader falls back to lexical scoring and records normalized fallback reason

### Requirement: Skill trigger scoring outcomes SHALL remain semantically equivalent between Run and Stream
For equivalent inputs, equivalent effective configuration, and equivalent scorer behavior, Run and Stream MUST produce semantically equivalent skill-trigger outcomes (selected skills and deterministic ordering), allowing non-semantic event timing differences.

#### Scenario: Equivalent lexical-plus-embedding selection in Run and Stream
- **WHEN** equivalent requests execute with strategy `lexical_plus_embedding` and the same scorer outputs
- **THEN** Run and Stream produce semantically equivalent selected skill set and order

#### Scenario: Equivalent embedding fallback in Run and Stream
- **WHEN** equivalent requests execute with `lexical_plus_embedding` and embedding path fails with the same failure class
- **THEN** Run and Stream both fallback to lexical scoring and produce semantically equivalent selection outcome

### Requirement: Skill loader SHALL support deterministic mixed CJK-English lexical tokenization
Skill lexical trigger scoring MUST support `mixed_cjk_en` tokenization so that Chinese and mixed Chinese-English inputs can participate in lexical matching without changing external loader APIs.

`mixed_cjk_en` mode MUST remain deterministic for the same input and MUST preserve existing lexical weighted-keyword scoring pipeline semantics (weighting, thresholding, tie-break order).

#### Scenario: Chinese lexical trigger is recognized under default tokenizer mode
- **WHEN** runtime uses default skill trigger scoring configuration and user input is Chinese
- **THEN** loader produces non-empty lexical tokens and evaluates candidate skills through lexical weighted-keyword scoring

#### Scenario: Mixed Chinese-English lexical trigger is recognized
- **WHEN** user input contains both Chinese and English trigger phrases
- **THEN** loader evaluates mixed-language tokens in one lexical scoring pass and keeps deterministic candidate ordering

### Requirement: Skill trigger scoring SHALL enforce semantic candidate budget with top-k control
After semantic candidates are filtered by confidence threshold and deterministically sorted, loader MUST apply semantic candidate budget by configured budget mode.

Budget mode semantics:
- `fixed`: select top `max_semantic_candidates`.
- `adaptive`: select candidates deterministically within `[min_k, max_k]` using score-margin rule `min_score_margin`.

In adaptive mode, implementation MUST produce deterministic selection count for equivalent input/config/scorer outputs and MUST expose a normalized decision reason.

Candidates outside selected budget MUST be pruned from semantic selection. Explicitly referenced skills MUST remain selectable and MUST NOT be dropped by semantic budget capping.

#### Scenario: Default adaptive budget keeps minimum candidates when top score is clearly separated
- **WHEN** runtime uses default adaptive budget config and `top1-top2 >= min_score_margin`
- **THEN** loader selects `min_k` semantic candidates and prunes the remainder

#### Scenario: Adaptive budget expands candidates for close-score cluster
- **WHEN** runtime uses adaptive budget and top-ranked candidates have score margins below `min_score_margin`
- **THEN** loader expands selected semantic candidates up to `max_k` deterministically

#### Scenario: Fixed budget mode preserves top-k behavior
- **WHEN** runtime sets budget mode to `fixed` with `max_semantic_candidates=N`
- **THEN** loader selects exactly top `N` semantic candidates after sorting and threshold filtering

#### Scenario: Explicit skill selection bypasses semantic budget pruning
- **WHEN** user input explicitly references a skill and semantic candidate count exceeds selected budget
- **THEN** explicit skill selection remains effective while budget pruning applies only to semantic candidates

### Requirement: Skill trigger observability SHALL include tokenizer and pruning signals
Skill diagnostics/events MUST include additive lexical-budget fields for each compile evaluation path:
- `tokenizer_mode`
- `candidate_pruned_count`
- `budget_mode`
- `selected_semantic_count`
- `score_margin_top1_top2`
- `budget_decision_reason`

`candidate_pruned_count` MUST represent the number of semantic candidates removed by selected budget and MUST be `0` when no semantic candidate is pruned.

`score_margin_top1_top2` MUST be normalized when at least two semantic candidates exist.

#### Scenario: Adaptive minimum-selection path exposes decision fields
- **WHEN** adaptive budget keeps minimum candidates due to clear top margin
- **THEN** diagnostics include `budget_mode=adaptive`, selected count, top1-top2 margin, and normalized decision reason

#### Scenario: Fixed-mode path exposes deterministic budget fields
- **WHEN** fixed mode applies top-k selection
- **THEN** diagnostics include `budget_mode=fixed`, selected count, and pruning count consistent with fixed top-k result

### Requirement: Run and Stream SHALL preserve multilingual lexical-budget semantic equivalence
For equivalent input, equivalent effective configuration, and equivalent scorer behavior, Run and Stream MUST produce semantically equivalent skill selection outcomes with multilingual lexical tokenization and semantic budget control enabled.

#### Scenario: Equivalent adaptive-budget selection in Run and Stream
- **WHEN** equivalent Chinese or mixed Chinese-English requests execute under the same adaptive budget configuration
- **THEN** Run and Stream produce semantically equivalent selected skill set, order, and selected semantic candidate count

#### Scenario: Equivalent fixed-budget selection in Run and Stream
- **WHEN** equivalent requests execute under fixed budget mode with the same top-k setting
- **THEN** Run and Stream prune the same semantic candidates and expose semantically equivalent lexical-budget diagnostics

### Requirement: Multi-Source Skill Discovery Determinism
Skill discovery across `agents_md|folder|hybrid` modes MUST produce deterministic merge order and duplicate resolution under identical inputs.

#### Scenario: Hybrid mode deterministic merge
- **WHEN** `runtime.skill.discovery.mode=hybrid` with fixed source roots and `AGENTS.md`
- **THEN** discovered skill order and selected candidate set MUST be deterministic across repeated runs

#### Scenario: Duplicate skill resolution stability
- **WHEN** the same skill identifier appears from multiple discovery sources
- **THEN** duplicate resolution MUST follow configured deterministic policy and record selected source

### Requirement: Skill Preprocess and Trigger Scoring Consistency
Skill preprocess output MUST remain consistent with trigger scoring input expectations and MUST NOT bypass configured scoring budget and thresholds.

#### Scenario: Discover-only preprocess consistency
- **WHEN** preprocess runs in discover-only mode
- **THEN** downstream trigger scoring inputs MUST match deterministic discovered set with no hidden compile-side effects

#### Scenario: Discover+compile preprocess consistency
- **WHEN** preprocess runs in discover+compile mode
- **THEN** scoring pipeline MUST use compiled metadata consistently without changing configured budget semantics

### Requirement: SkillBundle Mapping Contract Stability
`SkillBundle` mapping to prompt augmentation and tool whitelist MUST follow explicit mapping modes and conflict policy with deterministic outcomes.

#### Scenario: Prompt mapping determinism
- **WHEN** multiple skill bundles contribute prompt augmentation content
- **THEN** final prompt augmentation MUST follow configured ordering and conflict policy deterministically

#### Scenario: Whitelist mapping upper-bound
- **WHEN** mapping mode proposes tools outside security governance boundary
- **THEN** effective whitelist MUST remain bounded by sandbox/allowlist upper-bound and record conflict reason

