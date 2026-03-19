## ADDED Requirements

### Requirement: Runtime diagnostics SHALL expose additive delayed-dispatch summary fields
Run diagnostics MUST expose additive delayed-dispatch summary fields with compatibility-window semantics.

Required minimum additive fields:
- `scheduler_delayed_task_total`
- `scheduler_delayed_claim_total`
- `scheduler_delayed_wait_ms_p95`

#### Scenario: Consumer queries delayed-dispatch run summary
- **WHEN** run includes tasks with delayed dispatch semantics
- **THEN** diagnostics include delayed-dispatch additive fields without breaking existing consumers

### Requirement: Delayed-dispatch diagnostics SHALL remain replay-idempotent
Repeated ingestion/replay of equivalent delayed-dispatch events MUST NOT inflate logical delayed aggregates.

#### Scenario: Delayed-dispatch events are replayed
- **WHEN** same delayed-dispatch timeline records are replayed for one run
- **THEN** delayed aggregate counters remain stable after first logical ingestion
