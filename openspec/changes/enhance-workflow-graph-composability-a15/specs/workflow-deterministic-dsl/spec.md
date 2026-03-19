## ADDED Requirements

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
