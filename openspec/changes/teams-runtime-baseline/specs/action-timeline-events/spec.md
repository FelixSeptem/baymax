## ADDED Requirements

### Requirement: Action timeline SHALL carry Teams correlation metadata
Action Timeline events emitted by Teams orchestration MUST include normalized correlation metadata for team execution context, including `team_id`, `agent_id`, and `task_id` when available.

#### Scenario: Team dispatch emits timeline event
- **WHEN** coordinator dispatches a worker task inside a team run
- **THEN** emitted timeline event contains `team_id`, `agent_id`, and `task_id` fields for correlation

### Requirement: Action timeline SHALL normalize Teams orchestration reason codes
Timeline reason semantics for Teams orchestration MUST include normalized codes for dispatch, collect, and resolution paths.

#### Scenario: Team collect path is observed
- **WHEN** coordinator collects worker results
- **THEN** timeline events expose a stable collect reason code that can be aggregated consistently across runs
