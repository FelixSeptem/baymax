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
After semantic candidates are filtered by confidence threshold and deterministically sorted, loader MUST apply `max_semantic_candidates` as top-k cap.

Candidates outside top-k MUST be pruned from semantic selection. Explicitly referenced skills MUST remain selectable and MUST NOT be dropped by semantic budget capping.

#### Scenario: Default top-k budget prunes low-ranked semantic candidates
- **WHEN** semantic candidates exceed default `max_semantic_candidates=3`
- **THEN** loader keeps top 3 semantic candidates and prunes the remaining candidates before compile assembly

#### Scenario: Explicit skill selection bypasses semantic budget pruning
- **WHEN** user input explicitly references a skill and semantic candidate count exceeds top-k
- **THEN** explicit skill selection remains effective while semantic pruning applies only to semantic candidates

### Requirement: Skill trigger observability SHALL include tokenizer and pruning signals
Skill diagnostics/events MUST include additive fields `tokenizer_mode` and `candidate_pruned_count` for each compile evaluation path.

`candidate_pruned_count` MUST represent the number of semantic candidates removed by top-k budget and MUST be `0` when no semantic candidate is pruned.

#### Scenario: No pruning path exposes zero pruned count
- **WHEN** semantic candidate count is less than or equal to configured top-k budget
- **THEN** diagnostics expose `candidate_pruned_count=0` with active `tokenizer_mode`

#### Scenario: Pruning path exposes non-zero pruned count
- **WHEN** semantic candidate count exceeds configured top-k budget
- **THEN** diagnostics expose active `tokenizer_mode` and `candidate_pruned_count` greater than `0`

### Requirement: Run and Stream SHALL preserve multilingual lexical-budget semantic equivalence
For equivalent input, equivalent effective configuration, and equivalent scorer behavior, Run and Stream MUST produce semantically equivalent skill selection outcomes with multilingual lexical tokenization and semantic budget pruning enabled.

#### Scenario: Equivalent multilingual lexical selection in Run and Stream
- **WHEN** equivalent Chinese or mixed Chinese-English requests execute under the same tokenizer mode and threshold settings
- **THEN** Run and Stream produce semantically equivalent selected skill set and deterministic order

#### Scenario: Equivalent top-k pruning in Run and Stream
- **WHEN** equivalent requests produce semantic candidates exceeding `max_semantic_candidates`
- **THEN** Run and Stream prune the same semantic candidates and expose semantically equivalent pruning diagnostics

