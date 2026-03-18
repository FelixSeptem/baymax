## ADDED Requirements

### Requirement: Scheduler module SHALL preserve boundary ownership separation
Distributed scheduler implementation MUST reside in dedicated scheduler/orchestration ownership scope and MUST NOT move peer-collaboration semantics into MCP transport packages.

#### Scenario: Contributor adds lease-claim logic
- **WHEN** a change introduces lease claim and takeover behavior
- **THEN** implementation is placed in scheduler/orchestration scope and not in `mcp/http` or `mcp/stdio`

### Requirement: Scheduler observability SHALL use shared single-writer diagnostics path
Scheduler and subagent observability output MUST enter diagnostics through `observability/event.RuntimeRecorder` single-writer path only.

#### Scenario: Scheduler emits queue and lease metrics
- **WHEN** scheduler emits queue/lease lifecycle events
- **THEN** diagnostics ingestion occurs through RuntimeRecorder and not via direct diagnostics store writes

### Requirement: Shared contract governance SHALL include scheduler/subagent consistency checks
Shared multi-agent contract gate MUST validate scheduler/subagent identifier mapping and reason namespace compliance in addition to existing team/workflow/a2a checks.

#### Scenario: Change introduces non-namespaced scheduler reason
- **WHEN** scheduler/subagent change emits reason outside approved namespace conventions
- **THEN** shared-contract gate fails and blocks merge
