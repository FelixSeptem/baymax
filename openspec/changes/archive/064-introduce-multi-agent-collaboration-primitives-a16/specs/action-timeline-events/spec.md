## ADDED Requirements

### Requirement: Action timeline SHALL encode collaboration primitive reasons in existing canonical namespaces
Action timeline MUST encode collaboration primitive reasons using existing canonical namespace prefixes and MUST NOT introduce new top-level namespace.

Minimum collaboration reason set MUST include:
- `team.handoff`
- `team.delegation`
- `team.aggregation`
- `workflow.handoff`
- `workflow.delegation`
- `workflow.aggregation`

#### Scenario: Collaboration timeline event is emitted
- **WHEN** runtime emits timeline for collaboration primitive transitions
- **THEN** reason codes use existing canonical namespace prefixes and no `collab.*` top-level namespace appears

### Requirement: Collaboration timeline events SHALL preserve required correlation metadata
Timeline events for collaboration primitive transitions MUST preserve required correlation fields (`run_id`, `team_id`, `workflow_id`, `step_id`, `task_id`, `agent_id`, `peer_id`) where applicable.

#### Scenario: Workflow delegation to remote peer is traced
- **WHEN** workflow delegation primitive dispatches remote execution
- **THEN** timeline events include required correlation fields for deterministic cross-domain tracing

### Requirement: Collaboration timeline semantics SHALL preserve Run Stream equivalence
For equivalent collaboration primitive executions, Run and Stream timeline reason and status semantics MUST remain equivalent.

#### Scenario: Equivalent collaboration request executes via Run and Stream
- **WHEN** same collaboration primitive scenario executes in both modes
- **THEN** timeline reason taxonomy and terminal status semantics remain equivalent
