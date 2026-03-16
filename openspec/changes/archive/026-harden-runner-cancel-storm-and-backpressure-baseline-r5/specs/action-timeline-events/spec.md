## ADDED Requirements

### Requirement: Action timeline SHALL expose cancellation-propagation reason semantics
When cancellation storm controls are triggered, action timeline events MUST expose normalized reason semantics indicating cancellation propagation outcomes across execution phases.

#### Scenario: Timeline records cancellation propagation during tool phase
- **WHEN** runner propagates parent cancellation while tool fanout is active
- **THEN** corresponding timeline event includes cancellation-propagation reason semantics and terminal status consistency

#### Scenario: Timeline records cancellation propagation during mcp or skill phase
- **WHEN** runner propagates parent cancellation while mcp or skill work is active
- **THEN** corresponding timeline event includes cancellation-propagation reason semantics aligned with run terminal classification

### Requirement: Action timeline SHALL preserve backpressure observability consistency with diagnostics
Timeline and diagnostics outputs MUST remain semantically consistent for backpressure and cancellation outcomes in the same run.

#### Scenario: Consumer correlates timeline and diagnostics under block policy
- **WHEN** a high-fanout run triggers backpressure with policy `block`
- **THEN** timeline events and run diagnostics present non-conflicting outcome semantics, and `backpressure_drop_count` remains zero

#### Scenario: Consumer correlates timeline and diagnostics under canceled run
- **WHEN** a run is canceled and cancellation is propagated across branches
- **THEN** timeline terminal semantics match diagnostics counters and final run status category
