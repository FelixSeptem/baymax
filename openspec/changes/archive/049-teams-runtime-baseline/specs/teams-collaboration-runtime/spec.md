## ADDED Requirements

### Requirement: Runtime SHALL provide first-class Teams collaboration primitives
The runtime MUST provide first-class Teams primitives to model agent roles and collaboration plans, including `leader`, `worker`, and `coordinator` roles, without requiring application-specific ad hoc orchestration code.

#### Scenario: Host creates a team execution plan
- **WHEN** the host submits a team plan with role assignments and tasks
- **THEN** runtime accepts a normalized team definition and starts execution through a stable orchestration API

### Requirement: Teams strategy execution SHALL be deterministic
Teams orchestration MUST support `serial`, `parallel`, and `vote` strategies with deterministic resolution semantics for equivalent input, configuration, and dependency graph.

#### Scenario: Equivalent team input produces stable result under vote strategy
- **WHEN** the same team plan is executed repeatedly with strategy `vote`
- **THEN** runtime returns the same winning decision under the configured tie-break rules

### Requirement: Teams task lifecycle SHALL be explicit and queryable
Each team task MUST move through normalized lifecycle statuses (`pending`, `running`, `succeeded`, `failed`, `skipped`, `canceled`) and expose terminal status for diagnostics and replay.

#### Scenario: Worker task times out
- **WHEN** a worker task exceeds configured timeout
- **THEN** runtime marks the task with terminal status `canceled` or `failed` according to policy and records the reason

### Requirement: Teams execution SHALL preserve Run and Stream semantic equivalence
For equivalent team plans, Run and Stream paths MUST preserve semantic equivalence for strategy decisions, lifecycle transitions, and terminal outcomes.

#### Scenario: Equivalent team plan via Run and Stream
- **WHEN** a team plan runs once through Run and once through Stream under the same policy
- **THEN** both paths expose semantically equivalent team lifecycle and terminal status outcomes

### Requirement: Teams orchestration SHALL align with existing cancellation and backpressure governance
Teams execution MUST honor existing runtime cancellation propagation and backpressure semantics and MUST NOT introduce conflicting policy behavior.

#### Scenario: Parent cancellation during parallel strategy
- **WHEN** parent context is canceled while parallel team tasks are inflight
- **THEN** runtime propagates cancellation across affected team tasks and records consistent cancellation reasons
