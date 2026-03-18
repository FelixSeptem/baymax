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

