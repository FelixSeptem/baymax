# workflow-graph-composability Specification

## Purpose
TBD - created by archiving change enhance-workflow-graph-composability-a15. Update Purpose after archive.

## Requirements

### Requirement: Workflow DSL SHALL support reusable subgraph definitions and references
Workflow DSL MUST support reusable subgraph declarations and `use_subgraph` references so hosts can compose larger workflow graphs from shared fragments.

#### Scenario: Workflow references reusable subgraph
- **WHEN** workflow definition includes `subgraphs` and one or more `use_subgraph` nodes
- **THEN** runtime compiler expands references into executable workflow steps deterministically

### Requirement: Subgraph expansion SHALL enforce bounded recursion depth
Workflow subgraph compilation MUST enforce maximum recursion depth of `3`.

#### Scenario: Subgraph expansion depth exceeds limit
- **WHEN** nested subgraph references exceed depth `3`
- **THEN** workflow compilation fails fast with deterministic validation error

### Requirement: Expanded step identifiers SHALL use canonical alias path format
Expanded step identifiers MUST use canonical format `<subgraph_alias>/<step_id>`.

#### Scenario: One subgraph is instantiated with alias
- **WHEN** compiler expands one subgraph instance with alias `prepare`
- **THEN** generated steps use stable IDs such as `prepare/fetch`, `prepare/validate`

### Requirement: Condition templates SHALL be supported with explicit variables only
Workflow DSL MUST support `condition_templates` and `template_vars`, and template expansion MUST apply to condition semantics only.

#### Scenario: Template is applied to condition
- **WHEN** workflow step references a condition template with complete variable bindings
- **THEN** compiler resolves condition expression deterministically before planning

#### Scenario: Template is used outside condition scope
- **WHEN** workflow definition tries to apply condition template to payload or non-condition field
- **THEN** workflow compilation fails fast with scope violation error

### Requirement: Subgraph override policy SHALL permit retry timeout and reject kind override
Subgraph instance overrides MUST allow `retry` and `timeout` overrides, and MUST reject `kind` override.

#### Scenario: Subgraph instance overrides timeout
- **WHEN** workflow instance overrides `timeout` for one subgraph step
- **THEN** compiler accepts override and preserves other subgraph semantics

#### Scenario: Subgraph instance overrides kind
- **WHEN** workflow instance attempts to override step `kind`
- **THEN** workflow compilation fails fast with unsupported-override error

### Requirement: Graph composability feature SHALL be default disabled
Runtime MUST keep workflow graph composability capability disabled by default and require explicit enablement.

#### Scenario: Runtime uses default configuration
- **WHEN** runtime starts without graph composability enablement
- **THEN** legacy flat workflow DSL remains default execution path

### Requirement: Workflow graph checkpoints SHALL preserve composability material

A durable continuation for a composable workflow SHALL retain the normalized expanded graph identity/version/digest, canonical alias-path step identities, dependency completion state, pending and completed step state, applicable retry/timeout facts, budget consumption, and aggregate/terminal arbitration evidence. Restore SHALL verify that this material corresponds to the graph currently being resumed and SHALL reject graph-version or digest drift before scheduling.

#### Scenario: Graph digest drift rejects restore
- **WHEN** a checkpoint graph digest differs from the normalized graph bound to the resume request
- **THEN** restore fails with a typed incompatible or corrupt result before any expanded step runs

### Requirement: Composite graph restore SHALL preserve causal lineage

Workflow checkpoint and restore projections SHALL retain original parent/run/session/causation identities and predecessor/successor checkpoint relationship for each paused graph. A resumed graph SHALL associate later pauses and terminal evidence with that lineage and SHALL NOT create a parallel graph or reinterpret a history cursor as graph state.

#### Scenario: Successor pause preserves predecessor identity
- **WHEN** a restored workflow reaches another durable pause
- **THEN** the successor checkpoint refers to its consumed predecessor and preserves the original graph lineage
