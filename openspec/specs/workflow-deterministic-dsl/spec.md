# workflow-deterministic-dsl Specification

## Purpose
TBD - created by archiving change workflow-dsl-baseline. Update Purpose after archive.
## Requirements
### Requirement: Runtime SHALL accept a normalized Workflow DSL schema
The runtime MUST accept a normalized Workflow DSL in YAML or JSON form with minimum fields `workflow_id`, `steps`, `depends_on`, `condition`, `retry`, and `timeout`.

#### Scenario: Valid workflow DSL is loaded
- **WHEN** host submits a workflow definition that satisfies schema constraints
- **THEN** runtime parses and normalizes the workflow plan without ambiguity

### Requirement: Workflow plan validation SHALL fail fast on structural errors
Workflow planning MUST fail fast on invalid DAG structure, duplicate step identifiers, missing dependencies, and unsupported field values.

#### Scenario: Workflow contains dependency cycle
- **WHEN** a workflow plan includes cyclic dependencies
- **THEN** runtime rejects the plan before execution and returns a normalized validation error

### Requirement: Workflow execution SHALL be deterministic for equivalent inputs
For equivalent workflow plan, configuration, and inputs, execution order and terminal statuses MUST be deterministic, including stable ordering of concurrently ready steps.

#### Scenario: Equivalent workflow plan executes twice
- **WHEN** the same workflow plan runs twice under identical inputs
- **THEN** runtime produces semantically equivalent step order and terminal outcomes

### Requirement: Workflow engine SHALL support bounded retry and timeout semantics per step
Workflow execution MUST honor per-step retry and timeout settings with bounded attempts and explicit terminal status on exhaustion.

#### Scenario: Step retries are exhausted
- **WHEN** a step fails repeatedly beyond configured retry budget
- **THEN** workflow marks the step terminal and applies configured failure-branch semantics

### Requirement: Workflow engine SHALL support minimal checkpoint and resume semantics
Workflow execution MUST persist minimal checkpoint state sufficient to resume from the next eligible step after interruption.

#### Scenario: Workflow resumes after interruption
- **WHEN** workflow restarts with a valid checkpoint snapshot
- **THEN** runtime resumes from remaining eligible steps without re-running already terminal steps

### Requirement: Workflow execution SHALL preserve Run and Stream semantic equivalence
Run and Stream paths MUST preserve semantic equivalence for workflow step status transitions and final workflow outcome for equivalent inputs.

#### Scenario: Equivalent workflow execution via Run and Stream
- **WHEN** the same workflow plan is executed through Run and Stream
- **THEN** both paths expose semantically equivalent workflow step states and final result

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

### Requirement: Workflow checkpoint recovery SHALL compose with scheduler and A2A restore
Workflow resume semantics MUST compose with scheduler and A2A recovery so checkpoint-based continuation remains deterministic in composed runs.

#### Scenario: Workflow resume follows scheduler and A2A restore
- **WHEN** workflow resumes from checkpoint after composed recovery initialization
- **THEN** step scheduling and dependency execution remain deterministic and aligned with restored scheduler/A2A state

### Requirement: Workflow recovery replay SHALL preserve deterministic execution order
Recovered workflow execution MUST preserve deterministic ordering guarantees for resumed steps under equivalent inputs and effective config.

#### Scenario: Equivalent recovery replay is executed twice
- **WHEN** the same workflow recovery snapshot is replayed under equivalent conditions
- **THEN** resumed execution order and terminal category remain semantically equivalent

### Requirement: Workflow planner SHALL compile subgraphs and templates before DAG ordering
Workflow deterministic planner MUST compile `use_subgraph` references and `condition_templates` into a flat normalized definition before topological ordering.

#### Scenario: Workflow plan is generated from composable DSL
- **WHEN** planner receives workflow definition with subgraphs and condition templates
- **THEN** planner first emits deterministic expanded definition and then computes deterministic DAG order

### Requirement: Workflow deterministic semantics SHALL remain stable after graph expansion
For equivalent composable workflow input and effective configuration, expanded step order and terminal status semantics MUST remain deterministic across repeated runs.

#### Scenario: Same composable workflow is planned twice
- **WHEN** one workflow definition with same aliases and template bindings is planned multiple times
- **THEN** expanded step ID set and execution order remain semantically equivalent

### Requirement: Workflow validation SHALL fail fast on composability compile errors
Workflow validation MUST fail fast on subgraph cycle, alias collision, step ID collision, missing template, or missing template variable.

#### Scenario: Workflow contains missing template variable
- **WHEN** step references template variable that is not provided in `template_vars`
- **THEN** workflow validation returns deterministic error and execution does not start

### Requirement: Workflow checkpoint resume SHALL preserve expanded graph semantics
Checkpoint and resume behavior MUST preserve deterministic continuation semantics on expanded subgraph steps and MUST NOT re-run already terminal expanded steps.

#### Scenario: Resume after partial completion on expanded subgraph
- **WHEN** workflow resumes from checkpoint after some expanded subgraph steps are already terminal
- **THEN** engine continues from next eligible expanded steps only and preserves deterministic outcome

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

