## ADDED Requirements

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
