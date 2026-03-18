## ADDED Requirements

### Requirement: Boundary governance SHALL keep A2A delivery/version negotiation outside MCP scope
A2A delivery-mode negotiation and Agent Card version negotiation MUST be implemented in A2A module scope and MUST NOT be implemented in MCP transport packages.

#### Scenario: Contributor adds delivery negotiation logic
- **WHEN** a change introduces A2A delivery-mode negotiation
- **THEN** implementation resides in `a2a/*` scope and not in `mcp/http` or `mcp/stdio`

#### Scenario: Contributor adds version negotiation logic
- **WHEN** a change introduces Agent Card version compatibility checks
- **THEN** implementation resides in A2A module scope and not MCP transport scope

### Requirement: A2A delivery/version observability SHALL use shared single-writer path
A2A delivery/version observability writes MUST flow through `observability/event.RuntimeRecorder` and MUST NOT bypass runtime diagnostics single-writer path.

#### Scenario: A2A delivery fallback emits diagnostics
- **WHEN** runtime records fallback behavior for A2A delivery
- **THEN** records are ingested through shared runtime recorder path without direct diagnostics store writes

#### Scenario: A2A version mismatch emits diagnostics
- **WHEN** runtime records version mismatch outcome
- **THEN** records are ingested through shared runtime recorder path with replay-idempotent semantics

### Requirement: Shared contract gate SHALL validate A2A delivery/version naming consistency
Changes touching A2A delivery/version semantics MUST pass shared multi-agent contract gate for naming consistency.

Minimum checks for this milestone:
- reason namespace consistency (`a2a.*`),
- canonical peer-field naming consistency (`peer_id`),
- delivery/version field naming consistency (snake_case additive fields).

#### Scenario: Non-canonical peer field is introduced
- **WHEN** a change emits A2A peer field using key other than `peer_id`
- **THEN** shared contract gate fails and blocks merge

#### Scenario: Non-namespaced A2A reason is introduced
- **WHEN** a change emits A2A timeline reason without `a2a.*` namespace
- **THEN** shared contract gate fails and blocks merge
