## ADDED Requirements

### Requirement: Tutorials SHALL include Action Gate parameter-rule minimal demonstration
At least one tutorial example MUST demonstrate Action Gate parameter-rule matching behavior, including a rule-hit path and resulting gate decision outcome.

#### Scenario: User runs parameter-rule tutorial path
- **WHEN** user executes the selected tutorial example
- **THEN** output includes a parameter-rule match signal and the corresponding gate decision behavior

#### Scenario: User inspects tutorial event output
- **WHEN** user observes runtime events for the tutorial run
- **THEN** timeline includes `gate.rule_match` reason semantics for matched parameter rules
