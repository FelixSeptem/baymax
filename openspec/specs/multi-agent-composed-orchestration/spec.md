# multi-agent-composed-orchestration Specification

## Purpose
TBD - created by archiving change compose-teams-workflow-with-a2a-remote-execution-a5. Update Purpose after archive.
## Requirements
### Requirement: Runtime SHALL support composed orchestration across Workflow, Teams, and A2A
The runtime MUST provide a composed orchestration path where Workflow and Teams can invoke A2A remote execution as first-class steps without requiring business-side manual stitching.

#### Scenario: Workflow step delegates to remote peer through A2A
- **WHEN** workflow executes a step declared for remote execution
- **THEN** runtime dispatches through A2A and returns normalized terminal status and correlation metadata

### Requirement: Composed orchestration SHALL preserve deterministic identifier propagation
Composed execution MUST preserve stable identifier propagation and mapping for `run_id`, `workflow_id`, `team_id`, `step_id`, `task_id`, `agent_id`, and `peer_id` across retries and replay.

#### Scenario: Composed execution is replayed
- **WHEN** the same composed run is replayed with equivalent inputs
- **THEN** identifier mapping remains stable and diagnostics aggregates do not inflate

### Requirement: Composed orchestration SHALL preserve Run and Stream semantic equivalence
For equivalent requests and effective configuration, Run and Stream paths MUST expose semantically equivalent composed terminal outcomes and additive diagnostics fields.

#### Scenario: Equivalent composed request via Run and Stream
- **WHEN** the same composed orchestration request executes once via Run and once via Stream
- **THEN** both paths produce semantically equivalent terminal status, reason category, and aggregate counters

### Requirement: Composed orchestration SHALL expose a composer-first integration path
Composed orchestration MUST define a first-class composer integration contract so workflow, teams, and A2A cooperation can be consumed through a single runtime entrypoint instead of host-side manual composition.

#### Scenario: Workflow and Teams run through composer entrypoint
- **WHEN** host invokes composed orchestration through the composer package
- **THEN** workflow and teams orchestration semantics remain available without requiring custom host glue code

### Requirement: Composed orchestration SHALL preserve existing reason namespace contract
Composer-managed composed flows MUST continue using existing namespaced timeline reasons (`team.*`, `workflow.*`, `a2a.*`, `scheduler.*`, `subagent.*`) and MUST NOT introduce non-namespaced reasons in multi-agent paths.

#### Scenario: Composer emits timeline events in composed path
- **WHEN** composed orchestration emits timeline events under composer management
- **THEN** each multi-agent reason remains in the existing canonical namespace set and remains correlation-compatible with shared contract checks

