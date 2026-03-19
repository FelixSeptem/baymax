# multi-agent-lib-first-composer Specification

## Purpose
TBD - created by archiving change introduce-lib-first-agent-composer-with-scheduler-bridge-a8. Update Purpose after archive.
## Requirements
### Requirement: Composer SHALL provide a library-first unified orchestration entrypoint
The runtime MUST provide a dedicated `orchestration/composer` package that composes runner, workflow, teams, A2A, and scheduler capabilities behind a single library-first entrypoint, so hosts no longer need manual multi-module stitching.

#### Scenario: Host initializes composed runtime through composer package
- **WHEN** host code constructs and executes a multi-agent run through `orchestration/composer`
- **THEN** the composed path executes without requiring host-side manual wiring of workflow/teams/a2a/scheduler internals

### Requirement: Composer SHALL support scheduler-managed local and A2A child execution
Composer-managed subagent execution MUST support both local child-run and A2A child-run targets, and MUST converge both targets through scheduler terminal commit semantics.

#### Scenario: Parent run dispatches mixed child targets
- **WHEN** one composed run dispatches child tasks to both local and A2A targets under scheduler management
- **THEN** both targets produce normalized task terminal states and idempotent scheduler commits through the same convergence contract

### Requirement: Composer SHALL preserve Run/Stream semantic equivalence
For equivalent requests and effective configuration, composer-managed Run and Stream paths MUST preserve semantically equivalent terminal status category and additive aggregate summaries.

#### Scenario: Equivalent composed request through Run and Stream
- **WHEN** an equivalent composer-managed request executes once with Run and once with Stream
- **THEN** terminal status category and required additive summary counters remain semantically equivalent

### Requirement: Composer A2A child dispatch SHALL consume shared synchronous invocation contract
Composer `ChildTargetA2A` dispatch MUST consume shared synchronous invocation contract and MUST NOT maintain an incompatible module-local synchronous remote execution flow.

#### Scenario: Composer dispatches A2A child task
- **WHEN** composer dispatches child task to A2A target
- **THEN** child execution uses shared synchronous invocation and returns normalized terminal outcome

### Requirement: Composer terminal commit mapping SHALL stay deterministic under shared invocation
Composer child terminal commit mapping produced from shared synchronous invocation MUST remain deterministic for equivalent transport/protocol/semantic failure classes.

#### Scenario: Composer receives transport-layer failure from shared invocation
- **WHEN** shared synchronous invocation classifies child execution failure as transport-layer
- **THEN** composer keeps deterministic commit mapping and retryability semantics for downstream scheduler handling

### Requirement: Composer SHALL support async child dispatch report sinks
Composer MUST support async child dispatch where child terminal outcomes are converged by report sink instead of mandatory synchronous wait.

#### Scenario: Composer dispatches child with async mode enabled
- **WHEN** composer dispatches a child task in async mode
- **THEN** composer returns accepted dispatch result and tracks child terminal through report sink updates

### Requirement: Composer async child reporting SHALL preserve scheduler terminal idempotency
Composer async child reporting integration MUST preserve scheduler terminal idempotency semantics for duplicate report deliveries.

#### Scenario: Duplicate async child terminal reports arrive
- **WHEN** same child terminal report is delivered more than once
- **THEN** composer/scheduler convergence keeps one logical terminal result and additive counters do not inflate

### Requirement: Composer SHALL expose delayed child dispatch contract
Composer child dispatch contract MUST allow passing delayed dispatch intent (`not_before`) to scheduler-managed child tasks.

#### Scenario: Host dispatches child task with delayed execution
- **WHEN** host submits composer child dispatch request with future `not_before`
- **THEN** composer enqueues child task with delayed semantics and no premature claim occurs

### Requirement: Composer delayed child execution SHALL preserve Run/Stream semantic equivalence
For equivalent delayed child requests, composer-managed Run and Stream paths MUST preserve semantic equivalence of terminal category and additive counters.

#### Scenario: Equivalent delayed child workflow via Run and Stream
- **WHEN** equivalent delayed child dispatch is exercised through Run and Stream
- **THEN** terminal category and delayed-related additive summaries remain semantically equivalent

