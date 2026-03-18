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

