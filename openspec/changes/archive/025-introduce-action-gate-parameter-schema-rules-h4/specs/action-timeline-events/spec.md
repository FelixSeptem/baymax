## MODIFIED Requirements

### Requirement: Action timeline SHALL encode Action Gate reason semantics
When Action Gate is evaluated for tool execution, timeline events MUST expose normalized reason codes for gate control outcomes. At minimum, reason codes MUST include `gate.rule_match`, `gate.require_confirm`, `gate.denied`, and `gate.timeout`.

#### Scenario: Timeline records parameter-rule match reason
- **WHEN** runner hits a parameter-level Action Gate rule
- **THEN** corresponding timeline event includes reason code `gate.rule_match`

#### Scenario: Timeline records confirmation-required reason
- **WHEN** runner marks a tool action as `require_confirm`
- **THEN** corresponding timeline event includes reason code `gate.require_confirm`

#### Scenario: Timeline records denied reason
- **WHEN** gate outcome denies tool execution
- **THEN** corresponding timeline event includes reason code `gate.denied`

#### Scenario: Timeline records timeout reason
- **WHEN** confirmation resolver times out and execution is denied
- **THEN** corresponding timeline event includes reason code `gate.timeout`
