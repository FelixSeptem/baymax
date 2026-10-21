## ADDED Requirements

### Requirement: Workflow graph capability SHALL not imply other composite restore

Workflow graph continuation evidence SHALL remain scoped to workflow graph
identity, step state, and workflow source ownership. It MUST NOT be used as
evidence for child/fanout, team, mailbox, or resumed-plan continuation.

#### Scenario: Workflow evidence is not transferable
- **WHEN** a planner sees a valid workflow continuation manifest entry and asks
  for team restore
- **THEN** the team request remains explicitly unsupported until team source
  evidence is independently declared
