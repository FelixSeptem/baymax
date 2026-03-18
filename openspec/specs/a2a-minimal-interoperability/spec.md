# a2a-minimal-interoperability Specification

## Purpose
TBD - created by archiving change a2a-minimal-interoperability. Update Purpose after archive.
## Requirements
### Requirement: Runtime SHALL provide minimal A2A task lifecycle interoperability
The runtime MUST provide minimal A2A interoperability primitives for task submission, status query, and result return between peer agents.

#### Scenario: Agent submits task to peer agent
- **WHEN** an agent submits a valid A2A task request
- **THEN** peer agent acknowledges submission and returns a queryable task identifier

### Requirement: A2A lifecycle statuses SHALL be normalized and queryable
A2A task lifecycle statuses MUST be normalized to `submitted`, `running`, `succeeded`, `failed`, and `canceled`, and MUST be queryable until terminal state.

For cross-domain observability, A2A status `submitted` MUST map to unified semantic-layer status `pending` before timeline aggregation and run-level diagnostics summarization.

#### Scenario: Client polls task status
- **WHEN** client queries an in-progress A2A task
- **THEN** server returns a normalized status value and latest progress metadata

#### Scenario: Submitted state enters timeline aggregation
- **WHEN** an A2A task is in `submitted` lifecycle state
- **THEN** timeline and aggregate diagnostics treat it as normalized status `pending`

### Requirement: Runtime SHALL support Agent Card capability discovery for A2A routing
The runtime MUST support Agent Card capability discovery and use discovered capability metadata as routing input for A2A peer selection.

#### Scenario: Router selects peer by capability match
- **WHEN** multiple peer agents are available and capability requirements are provided
- **THEN** router selects peers using Agent Card capability metadata and deterministic selection rules

### Requirement: A2A error semantics SHALL map to runtime error taxonomy
A2A transport/protocol/semantic failures MUST map to normalized runtime error classes so operators can diagnose failures consistently across subsystems.

#### Scenario: Peer returns unsupported method
- **WHEN** A2A server rejects a method as unsupported
- **THEN** runtime classifies the failure using normalized protocol error mapping and records diagnostics

### Requirement: A2A interoperability SHALL preserve semantic equivalence across equivalent run modes
For equivalent A2A task interactions, runtime observability semantics MUST remain equivalent across non-streaming and streaming execution paths.

#### Scenario: Equivalent A2A call via Run and Stream
- **WHEN** equivalent A2A interaction is invoked through Run and Stream
- **THEN** both paths expose semantically equivalent lifecycle transitions and terminal status

### Requirement: A2A SHALL expose orchestration-consumable correlation contract
A2A execution used by orchestration modules MUST preserve and expose correlation metadata required by composed flows, including `workflow_id`, `team_id`, `step_id`, and `task_id` when provided.

#### Scenario: Orchestration passes cross-domain correlation metadata
- **WHEN** workflow or teams dispatches an A2A remote task with correlation metadata
- **THEN** A2A path preserves metadata for timeline and diagnostics mapping

### Requirement: A2A orchestration integration SHALL preserve normalized terminal semantics
When consumed by orchestration modules, A2A terminal outcomes MUST remain normalized and deterministic under retry, timeout, and cancellation paths.

#### Scenario: Remote call times out under orchestration path
- **WHEN** composed orchestration invokes A2A and remote call exceeds timeout budget
- **THEN** A2A returns normalized timeout-class outcome and deterministic error-layer mapping

### Requirement: A2A orchestration integration SHALL preserve MCP boundary separation
A2A orchestration integration MUST NOT redefine MCP tool-integration responsibilities and MUST keep peer collaboration semantics inside A2A domain.

#### Scenario: Composed path includes both remote collaboration and tool invocation
- **WHEN** one run includes A2A peer delegation and MCP tool calls
- **THEN** A2A handles peer lifecycle semantics while MCP handles tool semantics without namespace overlap

### Requirement: A2A interoperability SHALL support scheduler-managed dispatch lifecycle
A2A task dispatch MUST support scheduler-managed lifecycle transitions including queued, claimed, and terminal commit phases without breaking existing submit/status/result contract.

#### Scenario: A2A task is dispatched by scheduler worker
- **WHEN** scheduler worker claims a remote-collaboration task and dispatches through A2A
- **THEN** A2A lifecycle remains queryable and terminal status maps to normalized A2A semantics

### Requirement: A2A scheduler integration SHALL preserve idempotent terminal mapping
A2A terminal outcomes committed through scheduler retry/takeover paths MUST remain idempotent and deterministic.

#### Scenario: A2A terminal commit is replayed after takeover
- **WHEN** takeover worker replays terminal commit for already-completed task attempt
- **THEN** A2A summary fields remain stable and duplicate commit does not alter logical terminal state

### Requirement: A2A scheduler integration SHALL preserve normalized error-layer mapping
When A2A execution fails under scheduler-managed retries, transport/protocol/semantic mapping MUST remain normalized and stable.

#### Scenario: Scheduler retries after transport failure
- **WHEN** remote collaboration fails with retryable transport error and scheduler retries claim execution
- **THEN** resulting A2A error class and `a2a_error_layer` remain normalized and deterministic

### Requirement: A2A in-flight state SHALL be included in recovery model
A2A interoperability contract MUST include in-flight task state in composed recovery snapshots to preserve remote collaboration continuity.

#### Scenario: Recovery resumes with pending A2A task
- **WHEN** composed recovery snapshot contains A2A in-flight task not yet terminal
- **THEN** recovery restores A2A task correlation and continues terminal convergence without creating duplicate logical tasks

### Requirement: Recovered A2A replay SHALL preserve error-layer normalization
Recovered A2A task replay MUST preserve existing error-layer normalization and reason taxonomy semantics.

#### Scenario: Recovery replays failed A2A terminal state
- **WHEN** recovered A2A failure is replayed into composed runtime
- **THEN** error layer and canonical reason mapping remain consistent with non-recovery execution paths

### Requirement: A2A WaitResult SHALL align with shared synchronous invocation contract
A2A `WaitResult` behavior consumed by orchestration modules MUST align with shared synchronous invocation contract for terminal convergence, cancellation, and error normalization.

#### Scenario: Shared invocation consumes A2A WaitResult
- **WHEN** orchestration path invokes A2A through shared synchronous invocation
- **THEN** `WaitResult` participates in terminal-only completion semantics and normalized error mapping

### Requirement: A2A synchronous waiting SHALL preserve polling compatibility defaults
A2A synchronous waiting consumed via shared invocation MUST preserve compatibility defaults for polling interval when caller does not override it.

#### Scenario: Caller omits poll interval
- **WHEN** shared synchronous invocation is called without poll interval override
- **THEN** A2A waiting behavior uses the existing default polling compatibility value

