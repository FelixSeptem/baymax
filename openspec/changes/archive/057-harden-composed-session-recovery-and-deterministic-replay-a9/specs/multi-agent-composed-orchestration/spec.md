## ADDED Requirements

### Requirement: Composed orchestration SHALL expose resume and recover entrypoints
Composed orchestration MUST provide explicit resume/recover entrypoints so hosts can restore interrupted multi-agent executions through library interfaces.

#### Scenario: Host invokes composed recover API
- **WHEN** host calls the composed recovery entrypoint with a persisted run context
- **THEN** workflow, teams, scheduler, and A2A paths are resumed under one composed recovery contract

### Requirement: Recovery SHALL be default-disabled unless explicitly enabled
Composed recovery behavior MUST remain disabled by default and MUST require explicit runtime configuration enablement.

#### Scenario: Recovery flag is not enabled
- **WHEN** runtime starts with default configuration
- **THEN** composed runtime does not activate recovery flow and preserves existing non-recovery behavior
