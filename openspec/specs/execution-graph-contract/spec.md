# execution-graph-contract Specification

## Purpose
Provide a bounded, source-neutral representation for comparing execution topology across existing orchestration forms without transferring their execution, persistence, or restore ownership.

## Requirements

### Requirement: Execution topology SHALL have a bounded canonical representation
The runtime MUST provide a versioned graph definition containing a graph identifier, graph version, node identifiers, and dependency edges. The graph contract MUST enforce documented limits for graph identifiers, node count, edge count, and identifier byte length before canonicalization.

#### Scenario: Equivalent definitions normalize identically
- **WHEN** callers provide equivalent node and dependency sets in different input orders
- **THEN** the graph contract returns the same normalized representation, stable topological order, and topology digest

#### Scenario: Graph exceeds a declared structural bound
- **WHEN** a graph exceeds an identifier, node, or edge limit
- **THEN** compilation fails with a typed limit error before returning a digest or activating an owner

### Requirement: Graph compilation SHALL reject invalid topology deterministically
Compilation MUST reject missing or malformed graph identity, empty or duplicate node identifiers, missing dependencies, self-dependencies, and dependency cycles with stable typed validation errors. Duplicate dependency declarations MUST be normalized consistently with the workflow contract.

#### Scenario: Graph contains a missing dependency or cycle
- **WHEN** compilation receives a missing edge target, self-edge, or cycle
- **THEN** compilation returns the corresponding typed error and no partial executable projection

#### Scenario: Graph has more than one valid topological order
- **WHEN** multiple nodes are ready at the same scheduling boundary
- **THEN** the returned projection uses a deterministic node-identifier tie-break without claiming a global order across independently executing sources

### Requirement: Owner projections SHALL preserve source-specific execution ownership
Workflow, team, and composer child owners MUST expose projections of only the topology their native request can represent. Workflow dependencies, team strategy ordering, and one-child composer dispatch MUST map deterministically to the common graph form. Projection MUST NOT create or execute work, and MUST NOT add unsupported dependencies or imply a complete parent fanout graph from one child dispatch.

#### Scenario: Workflow and team describe equivalent serial topology
- **WHEN** a workflow dependency chain and a serial team plan contain the same graph identity and ordered node set
- **THEN** their projected topology digests and dependency order are equal while their native execution owners remain distinct

#### Scenario: Workflow and parallel team describe equivalent independent topology
- **WHEN** a workflow with independent steps and a parallel or vote team contain the same graph identity and node set
- **THEN** their projected topology digests are equal and the projection does not impose task execution order

#### Scenario: Child projection is not a parent fanout projection
- **WHEN** composer projects a single child dispatch
- **THEN** the result contains only that child node and MUST NOT claim sibling dependencies or parent graph completeness

### Requirement: Topology digest SHALL remain distinct from authorization and restore bindings
The topology digest MUST bind only the canonical topology identity and edges. It MUST NOT replace owner plan digests, authority/write-scope/effect bindings, checkpoint digests, source identity, or owner-specific restore validation. Existing persisted owner checkpoint digest semantics MUST remain unchanged.

#### Scenario: Topology comparison cannot authorize restore
- **WHEN** a caller presents a matching topology digest with a foreign or changed owner checkpoint binding
- **THEN** the source owner continues to reject restore using its existing binding and claim rules

#### Scenario: Node execution metadata changes without topology change
- **WHEN** an owner-specific task payload or execution policy changes but graph node identities and edges do not
- **THEN** the topology digest remains a topology-only comparison fact and the owner-specific digest detects the relevant execution change
