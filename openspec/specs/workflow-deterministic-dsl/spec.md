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

