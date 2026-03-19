## ADDED Requirements

### Requirement: Composed orchestration SHALL support workflow subgraph-expanded remote steps
Composer-managed composed orchestration MUST support workflow runs where A2A remote steps originate from subgraph expansion.

#### Scenario: Expanded workflow subgraph includes remote step
- **WHEN** workflow subgraph expansion produces A2A remote step under composer-managed execution
- **THEN** composed orchestration dispatches remote step with existing normalized identifier propagation semantics

### Requirement: Composed orchestration SHALL preserve Run Stream equivalence with graph composability
For equivalent composed requests using workflow graph composability, Run and Stream paths MUST remain semantically equivalent in terminal category and additive aggregates.

#### Scenario: Equivalent composed graph-composable request via Run and Stream
- **WHEN** same composed workflow with subgraphs executes once via Run and once via Stream
- **THEN** terminal status category and required composed aggregate fields remain semantically equivalent

### Requirement: Composed orchestration SHALL preserve fail-fast compile boundary
Composer-managed workflow execution MUST fail before dispatch when workflow composability compilation fails.

#### Scenario: Composed run submits invalid subgraph override
- **WHEN** workflow definition contains invalid `kind` override in subgraph instance
- **THEN** composed orchestration returns compile validation error and does not dispatch child execution
