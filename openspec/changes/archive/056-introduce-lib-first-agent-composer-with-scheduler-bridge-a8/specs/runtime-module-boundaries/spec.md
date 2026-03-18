## ADDED Requirements

### Requirement: Composer module SHALL remain orchestration-layer glue only
The `orchestration/composer` module MUST remain an orchestration-layer composition boundary and MUST NOT absorb provider protocol logic, diagnostics storage writes, or MCP transport internals.

#### Scenario: Composer integrates runner and scheduler
- **WHEN** composer wires runner and scheduler for a composed execution path
- **THEN** composer only coordinates module integration and does not bypass existing ownership boundaries

### Requirement: Composer and scheduler SHALL use RuntimeRecorder as single diagnostics write path
Composer- and scheduler-related observability data MUST be emitted as standard events and MUST be persisted through `observability/event.RuntimeRecorder` single-writer path.

#### Scenario: Composer-managed run emits completion summary
- **WHEN** composer-managed execution completes
- **THEN** diagnostics persistence occurs through RuntimeRecorder event ingestion and not through direct `runtime/diagnostics` writes from orchestration modules
