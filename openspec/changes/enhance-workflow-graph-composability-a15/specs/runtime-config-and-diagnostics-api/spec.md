## ADDED Requirements

### Requirement: Runtime config SHALL expose workflow graph composability feature flag with deterministic precedence
Runtime configuration MUST expose `workflow.graph_composability.enabled` with precedence `env > file > default`, and default value MUST be `false`.

#### Scenario: Runtime starts with default config
- **WHEN** no explicit graph composability config is provided
- **THEN** effective configuration resolves `workflow.graph_composability.enabled=false`

#### Scenario: Environment enables graph composability
- **WHEN** YAML sets disabled and environment sets enabled
- **THEN** effective config resolves enabled by `env > file > default`

### Requirement: Runtime diagnostics SHALL expose additive workflow graph compilation summary fields
Run diagnostics MUST expose additive graph-composability summary fields with compatibility-window semantics.

Minimum required fields:
- `workflow_subgraph_expansion_total`
- `workflow_condition_template_total`
- `workflow_graph_compile_failed`

#### Scenario: Consumer queries diagnostics for composable workflow run
- **WHEN** workflow run uses subgraph expansion and condition templates
- **THEN** diagnostics include additive workflow graph summary fields without breaking existing consumers

### Requirement: Workflow graph diagnostics SHALL preserve additive nullable default compatibility
New workflow graph diagnostics fields MUST follow `additive + nullable + default` compatibility semantics.

#### Scenario: Legacy consumer parses diagnostics after A15 rollout
- **WHEN** legacy parser reads run diagnostics containing new workflow graph fields
- **THEN** existing field semantics remain unchanged and new fields are safely optional
