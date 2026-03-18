## ADDED Requirements

### Requirement: Boundary governance SHALL enforce orchestration-to-A2A integration ownership
Composed orchestration features MUST be implemented in orchestration and A2A modules through explicit interfaces, and MUST NOT move peer-collaboration semantics into MCP transport packages.

#### Scenario: Contributor adds workflow remote-step integration
- **WHEN** a change adds workflow remote execution via A2A
- **THEN** implementation remains in orchestration/A2A ownership scope and does not introduce MCP responsibility overlap

### Requirement: Boundary governance SHALL preserve single-writer diagnostics path in composed flows
Composed orchestration observability MUST continue to use `observability/event.RuntimeRecorder` as the only diagnostics write path.

#### Scenario: Composed feature emits new diagnostics fields
- **WHEN** workflow/teams/A2A integration emits additive diagnostics output
- **THEN** writes flow through single-writer recorder and do not directly mutate diagnostics store

### Requirement: Shared contract gate SHALL cover composed orchestration consistency checks
Shared multi-agent contract validation MUST include composed orchestration checks for identifier mapping, reason namespace compliance, and canonical `peer_id` naming.

#### Scenario: Composed change introduces non-canonical field naming
- **WHEN** a composed orchestration change emits non-`peer_id` peer field or non-namespaced reason
- **THEN** shared-contract gate fails and blocks merge
