# teams-collaboration-runtime Specification

## Purpose
TBD - created by archiving change teams-runtime-baseline. Update Purpose after archive.

## Requirements

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

### Requirement: Teams runtime SHALL consume unified delegation primitive for remote workers
Teams orchestration MUST consume unified collaboration delegation primitive for remote-worker execution instead of module-local divergent delegation semantics.

#### Scenario: Team plan includes remote worker tasks
- **WHEN** teams engine dispatches remote worker tasks
- **THEN** remote dispatch path uses shared delegation primitive semantics and preserves normalized lifecycle outcomes

### Requirement: Teams runtime SHALL support handoff primitive across role transitions
Teams orchestration MUST support explicit handoff primitive semantics for role transition flows between leader/coordinator/worker.

#### Scenario: Coordinator hands off unresolved task to another worker
- **WHEN** team coordination requires handoff from current owner to another worker
- **THEN** runtime records deterministic task ownership transition and preserves terminal convergence semantics

### Requirement: Teams aggregation semantics SHALL align with collaboration primitive strategies
Teams aggregation behavior for remote/local mixed tasks MUST align with collab strategy semantics (`all_settled` and `first_success`) while preserving Run/Stream equivalence.

#### Scenario: Mixed local remote team run with first_success strategy
- **WHEN** teams engine runs mixed targets using `first_success` aggregation mode
- **THEN** aggregate team terminal classification remains semantically equivalent across Run and Stream

### Requirement: Teams parallel and vote execution SHALL support source-owned continuation at declared safe points

When a team owner is explicitly configured with compatible continuation storage, parallel and vote plans SHALL support bounded `input_required`, `approval_required` and `budget_exhausted` pause boundaries before dispatch of unstarted member work. The persisted continuation SHALL preserve team/plan identity and digest, original execution identity/bindings, strategy, member status/output/vote records, deterministic pending member set, completed effect references, aggregate/vote state, pending action, consumed budget, original deadline, terminal arbitration and lineage. Existing unconfigured team engines SHALL retain terminal-only behavior and advertise unsupported continuation.

#### Scenario: Parallel pause preserves settled members
- **WHEN** an owner-configured parallel team pauses before an unstarted member effect
- **THEN** settled member outcomes remain persisted and only deterministic pending members remain eligible after restore
- **AND** settled effects are not dispatched again

#### Scenario: Vote pause preserves aggregate evidence
- **WHEN** an owner-configured vote team pauses before unresolved member dispatch or vote resolution
- **THEN** persisted continuation retains completed votes and tie-break inputs
- **AND** restored resolution uses the original strategy and deterministic aggregate semantics

#### Scenario: Unsafe in-flight member prevents fabricated pause
- **WHEN** a parallel team has an in-flight member without restorable source evidence at a requested pause boundary
- **THEN** the team returns typed `recovery-required` or rejects continuation
- **AND** it does not advertise a resumeable team checkpoint

### Requirement: Teams continuation restore SHALL be atomic and preserve Run/Stream parity

Team continuation restore SHALL validate schema, digest, graph/form/strategy
binding, source owner, expiry, deployment scope, parent/run/session/causation/
authority/plan/write-scope/effect/occurrence identity, pause reason, budget/
deadline and predecessor/successor lineage before atomic claim. Exact duplicate
resumes SHALL return the retained claim or terminal projection; conflicts and
incomplete claims SHALL fail closed. A new process SHALL execute only pending
members and preserve equivalent Run/Stream lifecycle, aggregate and terminal
semantics. When a resumed team pauses again, the owner SHALL persist the next
generation and predecessor identity and atomically commit its successor
checkpoint reference with the predecessor's retained resume projection. A
successor MUST NOT be restored as an unrelated generation-1 checkpoint.

#### Scenario: Duplicate team resume converges
- **WHEN** two equivalent resumes race for one team continuation
- **THEN** one logical team continuation is admitted and both callers receive the same retained projection
- **AND** pending members are not duplicated

#### Scenario: Team resume conflict blocks aggregate activation
- **WHEN** a request changes graph digest, strategy, pause reason, authority, occurrence, causation or resume identity
- **THEN** team restore rejects before member activation
- **AND** it does not create a replacement team plan or parent Run

#### Scenario: Run and Stream restore remain equivalent
- **WHEN** equivalent paused team plans are restored through Run and Stream surfaces
- **THEN** member lifecycle facts, vote/aggregate state and terminal classification are semantically equivalent apart from transport projection

#### Scenario: Repeated team pause preserves the source checkpoint chain
- **WHEN** a team resumes from one checkpoint and pauses at another declared safe point
- **THEN** the successor retains the original team/run/session/causation identities, increments generation, and names the consumed checkpoint as predecessor
- **AND** the predecessor's retained resume projection names the successor checkpoint and digest

#### Scenario: Fresh owner resumes the committed team successor
- **WHEN** a new team engine opens the compatible store after the successor publication
- **THEN** it validates the committed lineage and resumes only the successor's pending members
- **AND** settled member effects are not dispatched again

#### Scenario: Conflicting successor publication fails closed
- **WHEN** a resume attempts to publish a successor with a mismatched predecessor, generation, graph digest or authority binding
- **THEN** the owner returns a typed conflict or recovery-required result without replacing the retained predecessor projection
- **AND** it does not activate another team member
