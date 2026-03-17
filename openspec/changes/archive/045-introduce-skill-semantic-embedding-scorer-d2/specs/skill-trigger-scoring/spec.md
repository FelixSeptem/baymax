## MODIFIED Requirements

### Requirement: Skill loader SHALL use configurable lexical trigger scoring by default
The skill loader MUST evaluate semantic trigger candidates with a configurable trigger-scoring strategy. The default strategy MUST remain lexical weighted-keyword scoring, and runtime configuration MUST be able to switch to embedding-enhanced strategy without changing external APIs.

#### Scenario: Default strategy remains lexical
- **WHEN** runtime starts with default skill trigger scoring configuration
- **THEN** loader uses lexical weighted-keyword scoring and preserves existing trigger behavior baseline

#### Scenario: Runtime switches to embedding-enhanced strategy
- **WHEN** runtime configuration sets strategy to `lexical_plus_embedding`
- **THEN** loader evaluates candidates with lexical+embedding fusion and applies configured confidence threshold

### Requirement: Skill trigger scoring architecture SHALL reserve scorer extension interface
The implementation MUST provide an internal scorer extension interface for embedding-based trigger scoring integration and MUST support host registration of embedding scorer implementation without changing public loader APIs.

#### Scenario: Host registers embedding scorer implementation
- **WHEN** runtime runs with `lexical_plus_embedding` and a valid embedding scorer is registered
- **THEN** loader invokes registered scorer and merges embedding signal into final score

#### Scenario: Runtime runs without embedding scorer registration
- **WHEN** runtime runs with `lexical_plus_embedding` but no embedding scorer is registered
- **THEN** loader falls back to lexical-only scoring path with normalized fallback reason and continues compile flow

## ADDED Requirements

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
