## ADDED Requirements

### Requirement: Teams runtime SHALL support mixed local and remote worker execution
Teams orchestration MUST support mixed execution where worker tasks can run locally or be delegated to remote peers via A2A under a unified task lifecycle.

#### Scenario: Parallel team mixes local and remote workers
- **WHEN** a team plan contains both local worker tasks and remote worker tasks
- **THEN** runtime executes both within the same orchestration pass and emits unified lifecycle semantics

### Requirement: Teams mixed execution SHALL preserve deterministic failure and cancellation convergence
Teams mixed local/remote execution MUST preserve deterministic failure and cancellation convergence semantics under configured strategy and failure policy.

#### Scenario: Parent cancellation occurs during mixed execution
- **WHEN** parent context is canceled while local and remote workers are inflight
- **THEN** runtime propagates cancellation consistently and converges team terminal summary deterministically

### Requirement: Teams mixed execution SHALL preserve Run and Stream semantic equivalence
For equivalent team plans containing remote workers, Run and Stream MUST preserve semantic equivalence for lifecycle transitions and terminal aggregates.

#### Scenario: Equivalent mixed team plan via Run and Stream
- **WHEN** the same mixed team plan runs through Run and Stream
- **THEN** both paths expose semantically equivalent task status distribution and team aggregate fields
