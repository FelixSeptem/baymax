# multi-agent-task-board-control-contract Specification

## Purpose
TBD - created by archiving change introduce-task-board-control-and-manual-recovery-contract-a39. Update Purpose after archive.
## Requirements
### Requirement: Task Board control API SHALL provide bounded manual control actions
Runtime MUST provide a library-level Task Board control API that supports bounded manual actions:
- `cancel`
- `retry_terminal`

The API MUST remain library-first and MUST NOT require platform-side control plane dependencies.

#### Scenario: Host submits supported control action
- **WHEN** caller sends Task Board control request with `action=cancel` or `action=retry_terminal`
- **THEN** runtime validates and executes the action through deterministic scheduler control path

#### Scenario: Host submits unsupported control action
- **WHEN** caller sends action outside supported set
- **THEN** runtime fails fast with validation error and no partial mutation

### Requirement: Task Board control API SHALL enforce deterministic state transition matrix
Control actions MUST enforce deterministic state constraints:
- `cancel` MUST be allowed only for `queued` and `awaiting_report`,
- `cancel` on `running` MUST fail fast,
- `retry_terminal` MUST be allowed only for `failed` and `dead_letter`,
- successful `retry_terminal` MUST transition task back to `queued`.

#### Scenario: Cancel queued task succeeds
- **WHEN** caller executes `cancel` for a queued task
- **THEN** task transitions to terminal canceled/failed-classified state as defined by scheduler contract and becomes non-claimable

#### Scenario: Cancel running task fails fast
- **WHEN** caller executes `cancel` for a running task
- **THEN** runtime returns fail-fast state-conflict error and task lease/execution state remains unchanged

#### Scenario: Retry dead-letter task succeeds
- **WHEN** caller executes `retry_terminal` for a dead-letter task within manual retry budget
- **THEN** task transitions to `queued` and is claimable under existing claim gating semantics

### Requirement: Task Board control API SHALL be idempotent by operation identifier
Each control request MUST include non-empty `operation_id`.

Runtime MUST deduplicate repeated requests with the same `operation_id` and preserve one logical control outcome without inflating counters.

#### Scenario: Duplicate manual control request is replayed
- **WHEN** runtime receives same `operation_id` more than once for equivalent action and target
- **THEN** runtime returns idempotent outcome and does not reapply state mutation

#### Scenario: Manual control request missing operation_id
- **WHEN** caller omits `operation_id`
- **THEN** runtime rejects request with fail-fast validation error

### Requirement: Task Board control semantics SHALL preserve backend parity and mode equivalence
For equivalent snapshots and control requests, manual control outcomes MUST be semantically equivalent across memory/file scheduler backends and across Run/Stream orchestration paths.

#### Scenario: Equivalent control request on memory and file backends
- **WHEN** same task snapshot and control request are applied to memory and file backends
- **THEN** terminal classification, queue visibility, and idempotent replay behavior remain semantically equivalent

#### Scenario: Equivalent managed request through Run and Stream
- **WHEN** equivalent orchestration context triggers same manual control action under Run and Stream paths
- **THEN** control terminal semantics and additive counters remain semantically equivalent

