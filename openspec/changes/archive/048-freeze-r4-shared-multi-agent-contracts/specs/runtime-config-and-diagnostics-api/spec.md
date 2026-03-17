## MODIFIED Requirements

### Requirement: Runtime SHALL load configuration with deterministic precedence
The runtime MUST load configuration from defaults, YAML file, and environment variables with precedence `env > file > default`.

For R4 multi-agent domains, config namespaces MUST be non-overlapping and domain-scoped, and shared keys MUST NOT carry conflicting semantics across domains.

Required domain scopes for this milestone:
- `teams.*`
- `workflow.*`
- `a2a.*`

#### Scenario: Multi-agent domains define overlapping semantic keys
- **WHEN** teams/workflow/a2a configs define similarly named keys
- **THEN** each key remains domain-scoped and no shared key changes meaning across domains

## ADDED Requirements

### Requirement: Runtime diagnostics SHALL expose canonical multi-agent naming with additive compatibility
Runtime diagnostics fields for multi-agent domains MUST follow canonical snake_case naming and remain additive to existing contracts.

Canonical shared naming constraints:
- identifier fields MUST align with `run_id/session_id/team_id/workflow_id/task_id/step_id/agent_id`.
- A2A remote peer identifier MUST use `peer_id`.
- lifecycle aggregates MUST preserve existing idempotent replay semantics.

#### Scenario: Consumer reads multi-agent diagnostics payload
- **WHEN** diagnostics include teams/workflow/a2a summary fields
- **THEN** field naming follows canonical snake_case and `peer_id` is used for A2A peer identity

#### Scenario: Legacy consumer ignores new multi-agent fields
- **WHEN** an existing diagnostics consumer parses only historical fields
- **THEN** the consumer remains compatible because multi-agent fields are additive only
