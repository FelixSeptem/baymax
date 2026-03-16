## ADDED Requirements

### Requirement: Action timeline SHALL emit unified drop_low_priority reason across dispatch phases
When low-priority backpressure is triggered, action timeline events MUST use `backpressure.drop_low_priority` as reason consistently across `tool`, `mcp`, and `skill` phases.

#### Scenario: Low-priority drop occurs in mcp phase
- **WHEN** mcp dispatch sheds a droppable call due to backpressure
- **THEN** timeline event uses reason `backpressure.drop_low_priority`

#### Scenario: Low-priority drop occurs in skill phase
- **WHEN** skill dispatch sheds a droppable call due to backpressure
- **THEN** timeline event uses reason `backpressure.drop_low_priority`

#### Scenario: Run and stream paths observe drop-low-priority reason
- **WHEN** equivalent workloads are executed via Run and Stream
- **THEN** both paths emit semantically equivalent timeline reason and phase status transitions
