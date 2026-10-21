## ADDED Requirements

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
