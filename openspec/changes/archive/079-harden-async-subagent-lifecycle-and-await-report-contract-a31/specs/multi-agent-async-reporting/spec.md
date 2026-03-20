## ADDED Requirements

### Requirement: Async reporting SHALL converge through awaiting-report lifecycle boundary
Async terminal reports MUST be committed only against scheduler attempts in `awaiting_report` lifecycle boundary and MUST preserve existing terminal commit idempotency contract.

#### Scenario: Report arrives for awaiting-report attempt
- **WHEN** async report is delivered for a task currently in `awaiting_report`
- **THEN** runtime commits terminal outcome through shared commit contract and keeps idempotent convergence behavior

### Requirement: Async reporting SHALL enforce late-report drop-and-record policy
When async report arrives after task has already converged to terminal state, runtime MUST not mutate terminal business outcome and MUST record late-report diagnostics.

#### Scenario: Report arrives after task already terminal
- **WHEN** async report is delivered for task no longer in `awaiting_report`
- **THEN** runtime treats the report as late, keeps terminal business status unchanged, and records drop-and-record diagnostics marker

### Requirement: Async reporting replay SHALL keep additive counters stable
Replayed equivalent async reports for the same task and attempt MUST keep async summary aggregates stable after first logical ingestion.

#### Scenario: Duplicate async report replay under recovery
- **WHEN** recovery path replays equivalent async reports
- **THEN** runtime keeps additive async-report counters and terminal summaries stable without inflation

