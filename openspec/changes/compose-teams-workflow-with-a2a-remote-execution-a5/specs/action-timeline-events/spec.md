## ADDED Requirements

### Requirement: Action timeline SHALL encode composed orchestration reason semantics
Action timeline MUST include normalized reason semantics for composed orchestration transitions in approved namespaces (`workflow.*`, `team.*`, `a2a.*`).

For this milestone, timeline reasons MUST additionally include at minimum:
- `workflow.dispatch_a2a`
- `team.dispatch_remote`
- `team.collect_remote`

#### Scenario: Workflow dispatches remote step through A2A
- **WHEN** workflow scheduler dispatches an A2A remote step
- **THEN** timeline event uses reason `workflow.dispatch_a2a`

#### Scenario: Teams dispatches and collects remote worker result
- **WHEN** teams orchestration dispatches a remote worker and later collects its result
- **THEN** timeline events use reasons `team.dispatch_remote` and `team.collect_remote`

### Requirement: Action timeline SHALL carry composed correlation metadata
Timeline events emitted on composed orchestration paths MUST carry available cross-domain correlation metadata for traceability.

#### Scenario: Composed path emits timeline events
- **WHEN** one run includes workflow step execution, team dispatch, and A2A remote interaction
- **THEN** timeline events carry available `workflow_id/team_id/step_id/task_id/agent_id/peer_id` metadata consistently

### Requirement: Composed timeline semantics SHALL preserve Run and Stream equivalence
For equivalent composed requests, Run and Stream MUST emit semantically equivalent timeline phase/status/reason outcomes.

#### Scenario: Equivalent composed request via Run and Stream
- **WHEN** equivalent composed orchestration requests run through Run and Stream
- **THEN** timeline semantics remain equivalent across phase transitions and reason categories
