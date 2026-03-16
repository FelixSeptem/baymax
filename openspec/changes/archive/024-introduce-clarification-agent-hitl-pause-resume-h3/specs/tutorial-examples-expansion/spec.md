## ADDED Requirements

### Requirement: Multi-agent tutorials SHALL include clarification HITL path
At least one multi-agent tutorial MUST demonstrate clarification HITL interaction where an agent requests user clarification and resumes execution with the returned answer.

#### Scenario: User runs multi-agent clarification tutorial
- **WHEN** user executes the selected multi-agent example
- **THEN** example emits a structured clarification request, accepts simulated/user clarification input, and continues the workflow

#### Scenario: User inspects tutorial output events
- **WHEN** user observes runtime events for the tutorial run
- **THEN** output contains structured clarification lifecycle events for await/resume or await/cancel path
