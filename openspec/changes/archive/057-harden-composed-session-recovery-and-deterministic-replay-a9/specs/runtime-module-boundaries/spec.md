## ADDED Requirements

### Requirement: Recovery orchestration SHALL remain in orchestration and runtime domains
Recovery orchestration logic MUST remain within orchestration/runtime composition boundaries and MUST NOT introduce forbidden dependencies on MCP internal packages or direct diagnostics storage writes.

#### Scenario: Recovery feature is integrated across modules
- **WHEN** recovery integration updates orchestration and runtime packages
- **THEN** dependency direction and single-writer diagnostics boundary constraints remain satisfied

### Requirement: Recovery persistence SHALL preserve single-writer diagnostics discipline
Recovery execution and replay observability MUST be emitted as standard events and persisted only through RuntimeRecorder single-writer ingestion.

Required observability contract:
- recovery timeline reason namespace uses `recovery.*`
- recovery additive run summary fields are emitted via `run.finished` payload only

#### Scenario: Recovery replay updates run observability
- **WHEN** recovery replay path records timeline and run summary signals
- **THEN** persistence happens through event ingestion and not direct writes from recovery orchestrators
