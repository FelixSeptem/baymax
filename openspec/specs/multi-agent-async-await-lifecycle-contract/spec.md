# multi-agent-async-await-lifecycle-contract Specification

## Purpose
TBD - created by archiving change harden-async-subagent-lifecycle-and-await-report-contract-a31. Update Purpose after archive.
## Requirements
### Requirement: Runtime SHALL expose explicit awaiting-report lifecycle state for async subagent tasks
For async subagent dispatch, runtime MUST use an explicit intermediate lifecycle state `awaiting_report` after async submit is accepted and before terminal report commit is finalized.

#### Scenario: Async submit is accepted
- **WHEN** scheduler-managed async child dispatch returns accepted acknowledgement
- **THEN** task lifecycle transitions from `running` to `awaiting_report` and remains queryable by this state

### Requirement: Async-await lifecycle SHALL enforce deterministic report-timeout terminalization
Runtime MUST enforce bounded wait semantics for async reports using configured timeout and deterministic terminal classification.

Default behavior MUST use:
- `report_timeout=15m`
- `timeout_terminal=failed`

When dead-letter policy is enabled and timeout exhaustion enters dead-letter path, terminal classification MUST converge to `dead_letter`.

#### Scenario: Report timeout is reached before terminal report arrives
- **WHEN** task remains in `awaiting_report` beyond configured `report_timeout`
- **THEN** runtime converges task to deterministic terminal outcome (`failed` by default, `dead_letter` when configured policy applies)

### Requirement: Late async reports SHALL use drop-and-record policy
After task has already reached terminal state, any later async terminal report MUST NOT mutate business terminal outcome and MUST be recorded as late-report diagnostics/timeline evidence.

#### Scenario: Late report arrives after timeout terminalization
- **WHEN** async terminal report arrives for a task already finalized by timeout path
- **THEN** runtime keeps existing terminal business status unchanged and records one late-report event under `drop_and_record` policy

### Requirement: Async-await lifecycle SHALL preserve idempotent convergence for duplicate report delivery and replay
Duplicate terminal reports and replayed equivalent reports for the same task/attempt MUST converge to one logical terminal outcome and MUST NOT inflate additive aggregates.

#### Scenario: Duplicate report replay for same task and attempt
- **WHEN** runtime receives repeated equivalent terminal reports for same task and attempt identity
- **THEN** logical terminal state remains unchanged and additive counters remain stable after first logical ingestion

### Requirement: Async-await lifecycle SHALL preserve Run Stream semantic equivalence
For equivalent logical requests and effective configuration, Run and Stream paths MUST preserve semantic equivalence for async-await terminal category and additive counters.

#### Scenario: Equivalent async-await flow in Run and Stream
- **WHEN** equivalent async subagent request executes through Run and Stream paths
- **THEN** terminal category and async-await additive summaries remain semantically equivalent

