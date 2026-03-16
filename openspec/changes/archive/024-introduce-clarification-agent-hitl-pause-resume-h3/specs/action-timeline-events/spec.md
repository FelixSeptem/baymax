## ADDED Requirements

### Requirement: Action timeline SHALL encode clarification HITL lifecycle semantics
When clarification HITL is triggered, timeline events MUST expose normalized reason semantics for await/resume/cancel transitions.

#### Scenario: Timeline records await-user transition
- **WHEN** runner enters clarification waiting state
- **THEN** timeline event includes reason code `hitl.await_user`

#### Scenario: Timeline records resumed transition
- **WHEN** runner resumes after receiving clarification
- **THEN** timeline event includes reason code `hitl.resumed`

#### Scenario: Timeline records cancel-by-user transition
- **WHEN** clarification timeout policy resolves to cancel
- **THEN** timeline event includes reason code `hitl.canceled_by_user`

### Requirement: Clarification request event payload SHALL be structured
Clarification events MUST include a structured `clarification_request` payload for direct consumer rendering.

#### Scenario: Consumer reads clarification request event
- **WHEN** runtime emits clarification request event
- **THEN** payload includes at least `request_id`, `questions`, `context_summary`, and `timeout_ms`
