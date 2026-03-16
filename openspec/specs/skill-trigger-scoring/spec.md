# skill-trigger-scoring Specification

## Purpose
TBD - created by archiving change introduce-skill-trigger-scoring-and-contract-tests-d1. Update Purpose after archive.
## Requirements
### Requirement: Skill loader SHALL use configurable lexical trigger scoring by default
The skill loader MUST evaluate semantic trigger candidates with a lexical weighted-keyword scoring strategy and MUST apply a configurable confidence threshold before selecting semantic matches.

#### Scenario: Candidate reaches threshold
- **WHEN** user input contains weighted keywords and a skill candidate score is greater than or equal to configured threshold
- **THEN** the skill candidate is considered matched for semantic trigger selection

#### Scenario: Candidate below threshold
- **WHEN** user input does not reach configured confidence threshold for a skill candidate
- **THEN** the skill candidate is not selected and no low-confidence semantic trigger is emitted

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
The implementation MUST provide an internal scorer extension interface for future embedding-based scorer integration without changing existing external APIs.

#### Scenario: Maintainer adds non-default scorer
- **WHEN** maintainer introduces a new internal scorer implementation
- **THEN** loader wiring can switch scorer implementation without changing public interfaces

#### Scenario: Current milestone uses default scorer only
- **WHEN** system runs under this milestone baseline
- **THEN** default lexical scorer is active and embedding scorer remains documented as TODO/not-enabled

### Requirement: Skill trigger scoring behavior SHALL be guarded by contract tests
Repository MUST include contract tests that verify threshold behavior, tie-break determinism, and low-confidence suppression defaults.

#### Scenario: Contract suite validates equal-score tie-break
- **WHEN** contract tests execute with equal-score candidate fixtures
- **THEN** tests assert deterministic `highest-priority` selection

#### Scenario: Contract suite validates suppression defaults
- **WHEN** contract tests execute with default configuration
- **THEN** below-threshold candidates are not activated and test fails on regression

