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

### Requirement: Composer SHALL expose collaboration primitive entrypoints
Composer MUST expose library-first entrypoints to run collaboration primitives (`handoff`, `delegation`, `aggregation`) through unified contracts.

#### Scenario: Host dispatches collaboration primitive through composer
- **WHEN** host invokes composer collaboration primitive API
- **THEN** runtime executes primitive through unified contract and returns normalized terminal outcome

### Requirement: Composer collaboration primitive execution SHALL compose with sync async delayed modes
Composer collaboration primitive execution MUST compose with synchronous invocation, async reporting, and delayed dispatch paths.

#### Scenario: Composer delegation uses delayed dispatch and async terminal reporting
- **WHEN** composer executes delegation primitive with delayed scheduling and async reporting enabled
- **THEN** terminal convergence remains deterministic and additive counters stay replay-idempotent

### Requirement: Composer collaboration primitive execution SHALL preserve Run Stream semantic equivalence
For equivalent collaboration primitive requests, composer Run and Stream paths MUST preserve semantically equivalent terminal category and additive summary fields.

#### Scenario: Equivalent collaboration primitive request via Run and Stream
- **WHEN** same collaboration primitive request is executed once via Run and once via Stream
- **THEN** summary semantics and terminal category remain semantically equivalent

### Requirement: Composer recovery SHALL enforce next_attempt_only resume boundary
Composer recovery flow MUST enforce next-attempt-only boundary when restoring scheduler/workflow/A2A state.

#### Scenario: Composer restores run with active in-flight child attempts
- **WHEN** composer recovery resumes run containing in-flight child attempts
- **THEN** current attempt semantics remain stable and updated controls apply on next attempt boundary

### Requirement: Composer recovery SHALL preserve no_rewind semantics for completed child tasks
Composer recovery MUST not dispatch already terminal child tasks again after restore.

#### Scenario: Restored snapshot contains completed child tasks
- **WHEN** composer finishes recovery initialization and resumes orchestration
- **THEN** completed child tasks remain terminal and are excluded from new dispatch

### Requirement: Composer timeout reentry semantics SHALL remain bounded and deterministic
Composer-managed timeout reentry after restore MUST follow bounded single reentry policy and deterministic fail convergence.

#### Scenario: Restored child execution times out beyond reentry budget
- **WHEN** child task exceeds configured timeout reentry budget after recovery
- **THEN** composer emits deterministic terminal failure and no further reentry is attempted

### Requirement: Composer SHALL expose runtime readiness passthrough for managed runtime path
When composer uses managed runtime components, it MUST expose a library-level readiness passthrough entrypoint that returns runtime readiness summary without mutating scheduling or run state.

Readiness passthrough MUST preserve runtime result semantics (`ready|degraded|blocked`) and MUST NOT invent composer-local status taxonomy.

#### Scenario: Host queries composer readiness on managed runtime
- **WHEN** application calls composer readiness API and composer uses managed runtime manager
- **THEN** returned readiness status and findings are semantically equivalent to runtime readiness preflight result

#### Scenario: Readiness query does not mutate orchestration state
- **WHEN** application queries composer readiness while scheduler has queued tasks
- **THEN** query path is read-only and does not mutate task lifecycle state

### Requirement: Composer readiness semantics SHALL remain mode-independent
For equivalent effective configuration and dependency states, readiness result exposed by composer MUST remain semantically equivalent regardless of Run or Stream usage path.

#### Scenario: Equivalent config used by Run and Stream entrypoints
- **WHEN** host queries composer readiness before equivalent Run and Stream calls
- **THEN** readiness status and finding classifications remain semantically equivalent

### Requirement: Composer SHALL expose operation-profile selection in managed orchestration requests
Composer MUST allow managed orchestration requests to specify operation profile selection and request-level timeout overrides through library-level API.

Composer MUST validate profile selection against canonical profile set before dispatching to scheduler.

#### Scenario: Host submits managed request with explicit profile
- **WHEN** caller invokes composer with `operation_profile=interactive`
- **THEN** composer accepts request, validates profile, and forwards resolved timeout context to scheduler

#### Scenario: Host submits managed request with unsupported profile
- **WHEN** caller invokes composer with non-canonical profile value
- **THEN** composer fails fast and does not create child dispatch attempt

### Requirement: Composer SHALL propagate timeout-resolution summary as additive diagnostics context
Composer run summary MUST include additive timeout-resolution context sufficient to explain effective profile and parent-child convergence outcome without breaking existing diagnostics consumers.

Minimum summary context:
- effective operation profile
- final child timeout budget
- resolution source classification

#### Scenario: Consumer inspects composer diagnostics after child dispatch
- **WHEN** managed run performs child dispatch with profile-based timeout resolution
- **THEN** composer diagnostics include additive timeout-resolution summary fields

#### Scenario: Equivalent Run and Stream execution paths
- **WHEN** equivalent inputs execute through Run and Stream with same profile and overrides
- **THEN** composer timeout-resolution summary semantics remain equivalent across modes

