## ADDED Requirements

### Requirement: Sandbox diagnostics ingestion SHALL preserve single-writer and idempotent semantics
Sandbox-related diagnostics events MUST flow through the existing single-writer path and MUST preserve idempotent aggregation semantics under retry and replay.

This applies at minimum to:
- sandbox decision fields,
- sandbox fallback markers,
- sandbox timeout and launch-failure counters.

#### Scenario: Duplicate sandbox decision events are replayed
- **WHEN** equivalent sandbox decision events for the same run are ingested multiple times
- **THEN** diagnostics keep one logical contribution for sandbox decision aggregates

#### Scenario: Concurrent duplicate sandbox failure events are recorded
- **WHEN** concurrent goroutines submit duplicate sandbox timeout or launch-failure events for the same run
- **THEN** diagnostics counters remain idempotent and do not inflate logical totals

