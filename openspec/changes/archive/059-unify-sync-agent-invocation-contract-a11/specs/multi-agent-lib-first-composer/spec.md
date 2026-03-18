## ADDED Requirements

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
