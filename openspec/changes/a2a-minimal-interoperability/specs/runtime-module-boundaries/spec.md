## ADDED Requirements

### Requirement: Boundary governance SHALL enforce A2A and MCP responsibility separation
Architecture boundaries MUST enforce that A2A modules own inter-agent collaboration semantics while MCP modules own tool-integration semantics, with no semantic overlap in responsibility.

#### Scenario: Contributor adds cross-agent request feature
- **WHEN** a feature implements peer-agent task lifecycle operations
- **THEN** implementation is placed in A2A module scope and not in MCP transport packages

### Requirement: A2A modules SHALL consume runtime observability through the shared single-writer path
A2A modules MUST emit events and diagnostics through the same `observability/event.RuntimeRecorder` single-writer path used by existing runtime components.

#### Scenario: A2A module adds diagnostic output
- **WHEN** A2A module records lifecycle outcomes
- **THEN** diagnostics flow through the shared event recorder path and do not introduce direct diagnostics store writes
