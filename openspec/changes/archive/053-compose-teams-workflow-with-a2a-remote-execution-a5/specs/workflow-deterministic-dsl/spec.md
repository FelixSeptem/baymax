## ADDED Requirements

### Requirement: Workflow DSL SHALL support A2A remote step kind
Workflow DSL MUST support an A2A remote step kind under the existing deterministic scheduling model.

#### Scenario: Valid workflow with A2A remote step
- **WHEN** workflow definition contains a valid remote step with required A2A fields
- **THEN** planner accepts the definition and scheduler executes it through the workflow step adapter

#### Scenario: Invalid A2A remote step definition
- **WHEN** workflow definition omits required remote-step fields or uses unsupported values
- **THEN** validation fails fast before execution with normalized validation error

### Requirement: Workflow A2A steps SHALL preserve bounded retry and timeout semantics
A2A remote workflow steps MUST obey workflow retry and timeout controls, and exhaustion MUST result in explicit terminal status.

#### Scenario: Remote workflow step exhausts retry budget
- **WHEN** A2A remote step fails repeatedly beyond configured retry budget
- **THEN** workflow marks the step terminal and records deterministic failure reason

### Requirement: Workflow A2A execution SHALL preserve Run and Stream semantic equivalence
Workflow runs containing A2A remote steps MUST preserve semantic equivalence between Run and Stream for execution order, terminal state, and aggregate workflow fields.

#### Scenario: Equivalent workflow with remote step via Run and Stream
- **WHEN** the same workflow containing A2A remote step is executed through Run and Stream
- **THEN** both paths expose semantically equivalent workflow result and step terminal statuses
