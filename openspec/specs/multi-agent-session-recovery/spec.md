# multi-agent-session-recovery Specification

## Purpose
TBD - created by archiving change harden-composed-session-recovery-and-deterministic-replay-a9. Update Purpose after archive.
## Requirements
### Requirement: Multi-agent runtime SHALL provide composer-level session recovery
The runtime MUST provide a composer-level recovery contract that restores multi-agent execution context across process restarts for workflow, scheduler, and A2A composed paths.

#### Scenario: Process restarts during composed execution
- **WHEN** a composed multi-agent execution is interrupted by process restart
- **THEN** recovery resumes from persisted recovery state instead of restarting from an empty orchestration state

### Requirement: Recovery store SHALL support memory and file backends
The recovery contract MUST support `memory` and `file` backends through a unified storage abstraction.

#### Scenario: Host configures recovery backend
- **WHEN** host config selects `memory` or `file` backend
- **THEN** recovery runtime loads the selected backend through the same contract surface

### Requirement: Recovery conflict handling SHALL fail fast
If persisted recovery snapshot and runtime reconciliation state conflict, recovery MUST fail fast with deterministic conflict classification.

#### Scenario: Recovery snapshot conflicts with runtime attempt state
- **WHEN** restore detects mismatch in required state version/cursor/attempt correlation
- **THEN** recovery terminates immediately with conflict error and does not continue best-effort merge

### Requirement: Recovery replay SHALL remain idempotent
Recovery replay MUST preserve idempotency for terminal outcomes and aggregate counters under repeated restore/replay attempts.

#### Scenario: Same recovery batch is replayed twice
- **WHEN** the same recovered events and terminal commits are applied repeatedly
- **THEN** logical terminal outcomes remain singular and additive diagnostics do not inflate

### Requirement: Session recovery SHALL apply explicit resume boundary policy for long-running execution
Session recovery contract MUST apply explicit resume boundary policy and reject ambiguous recovery continuation paths.

#### Scenario: Recovery request lacks deterministic continuation boundary
- **WHEN** restored state cannot be mapped to deterministic next-attempt continuation
- **THEN** recovery fails fast with explicit boundary conflict classification

### Requirement: Session recovery timeout reentry budget SHALL be deterministic and bounded
Session recovery MUST track timeout reentry budget per task and enforce configured maximum reentry count.

#### Scenario: Recovery resumes timeout-sensitive task
- **WHEN** recovered task exceeds timeout reentry budget
- **THEN** runtime converges task to deterministic terminal failure with bounded reentry counter

### Requirement: Session recovery SHALL preserve replay-idempotent outcomes under boundary enforcement
Recovery boundary enforcement MUST preserve replay idempotency for terminal outcomes and additive counters under repeated replay attempts.

#### Scenario: Same recovery batch is replayed with boundary checks enabled
- **WHEN** identical recovery snapshot and events are replayed multiple times
- **THEN** logical terminal outcomes and recovery counters remain stable after first logical ingestion

### Requirement: Recovery Import via Unified Snapshot Contract
Composer and scheduler recovery paths MUST accept unified snapshot manifest input and MUST apply restore policy deterministically.

#### Scenario: Recovery strict mode boundary enforcement
- **WHEN** imported snapshot violates recovery boundary under strict restore mode
- **THEN** recovery MUST fail fast before mutating scheduler/composer runtime state

#### Scenario: Recovery compatible mode deterministic action
- **WHEN** compatible mode accepts an in-window snapshot
- **THEN** recovery MUST emit deterministic restore action and preserve canonical terminal arbitration semantics

### Requirement: Cross-Module Recovery Consistency
Recovery from unified snapshots MUST preserve Run/Stream semantic equivalence and memory/file backend parity.

#### Scenario: Run/Stream equivalence after restore
- **WHEN** the same snapshot is restored and resumed through `Run` and `Stream`
- **THEN** resulting terminal classification and recovery aggregates MUST be equivalent

#### Scenario: Backend parity after restore
- **WHEN** recovery is executed against memory and file scheduler backends from equivalent snapshots
- **THEN** restored task/session semantics MUST remain equivalent modulo additive metadata ordering

