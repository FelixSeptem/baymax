## ADDED Requirements

### Requirement: Workflow engine SHALL remain decoupled from runner core state machine
Workflow orchestration MUST be implemented in a dedicated module and MUST consume runner/tool/mcp/skill capabilities through interfaces, without embedding workflow dependency-graph logic directly in `core/runner`.

#### Scenario: Contributor adds workflow orchestration support
- **WHEN** contributor implements workflow planning and scheduling behavior
- **THEN** dependency-graph and scheduling logic are placed in the workflow module, and `core/runner` retains single-run loop responsibility

### Requirement: Boundary governance SHALL verify workflow ownership and diagnostics write path
Boundary checks MUST verify workflow ownership and ensure workflow observability writes continue to use `observability/event.RuntimeRecorder` as the single diagnostics write entry.

Workflow change implementation MUST pass shared multi-agent contract gate before merge, including:
- status mapping consistency (with unified semantic layer),
- reason namespace consistency (`team.*|workflow.*|a2a.*`),
- canonical peer-field naming consistency (`peer_id` for A2A-related references).

#### Scenario: Workflow observability is added
- **WHEN** workflow implementation emits run and step observability data
- **THEN** data is recorded through the single-writer event path and not by direct diagnostics store mutation
