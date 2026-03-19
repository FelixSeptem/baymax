## ADDED Requirements

### Requirement: Workflow DSL SHALL support collaboration primitive markers for step-level delegation and handoff
Workflow DSL MUST support step-level collaboration primitive markers for delegation/handoff orchestration under deterministic planning semantics.

#### Scenario: Workflow step declares delegation primitive
- **WHEN** workflow planner parses a step configured for collaboration delegation
- **THEN** planner emits deterministic step scheduling and validation outcome without ambiguous execution semantics

### Requirement: Workflow aggregation primitive semantics SHALL preserve deterministic execution order
Workflow aggregation primitive execution MUST preserve deterministic ordering and terminal semantics for equivalent input and configuration.

#### Scenario: Workflow uses all_settled aggregation primitive
- **WHEN** equivalent workflow requests run repeatedly with `all_settled`
- **THEN** step execution order and aggregate terminal semantics remain deterministic

### Requirement: Workflow collaboration primitive execution SHALL preserve Run Stream equivalence
For equivalent workflow plans using collaboration primitives, Run and Stream paths MUST preserve semantic equivalence on step status and aggregate workflow fields.

#### Scenario: Equivalent workflow primitive plan via Run and Stream
- **WHEN** same workflow plan using delegation/handoff/aggregation runs through Run and Stream
- **THEN** workflow terminal status and additive aggregates remain semantically equivalent
