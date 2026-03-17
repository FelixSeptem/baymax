## ADDED Requirements

### Requirement: Action timeline SHALL include A2A interaction correlation metadata
Timeline events for A2A interactions MUST include correlation metadata for peer interaction context, including `task_id`, `agent_id`, and remote peer identifier when available.

#### Scenario: A2A task submission emits timeline event
- **WHEN** runtime submits an A2A task to a peer agent
- **THEN** timeline event includes A2A correlation metadata sufficient for end-to-end tracing

### Requirement: Action timeline SHALL normalize A2A reason semantics
Timeline reason semantics for A2A interactions MUST include normalized reason codes for submission, status polling, callback delivery, and terminal resolution.

#### Scenario: A2A callback retry occurs
- **WHEN** result callback delivery fails and enters bounded retry
- **THEN** timeline events expose a normalized callback-retry reason code
