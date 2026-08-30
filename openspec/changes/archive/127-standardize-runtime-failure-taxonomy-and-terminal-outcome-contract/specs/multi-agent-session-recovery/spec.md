## MODIFIED Requirements

### Requirement: Recovery conflict handling SHALL fail fast
If persisted recovery snapshot and runtime reconciliation state conflict, recovery MUST fail fast with deterministic conflict classification. The normalized terminal projection MUST classify this as `recovery_conflict` in phase `pre_execution` when continuation has not begun, or as a post-start recovery conflict when an attempt was already active. Recovery MUST NOT silently merge conflicting terminal outcomes.

#### Scenario: Recovery snapshot conflicts with runtime attempt state
- **WHEN** restore detects mismatch in required state version/cursor/attempt correlation
- **THEN** recovery terminates immediately with conflict error, normalized family `recovery_conflict`, and does not continue best-effort merge

#### Scenario: Late recovery report conflicts with an existing terminal outcome
- **WHEN** recovery reconciliation reports a terminal state that conflicts with an already committed terminal outcome
- **THEN** the first terminal outcome remains authoritative, the conflict is recorded deterministically, and replay does not inflate terminal counters

### Requirement: Recovery replay SHALL remain idempotent
Recovery replay MUST preserve idempotency for terminal outcomes and aggregate counters under repeated restore/replay attempts. The terminal projection MUST apply first-terminal-wins semantics and retain conflicting late reports as diagnostics without changing the source recovery owner.

#### Scenario: Same recovery batch is replayed twice
- **WHEN** the same recovered events and terminal commits are applied repeatedly
- **THEN** logical terminal outcomes remain singular, conflict records remain deduplicated, and additive diagnostics do not inflate

### Requirement: Cross-Module Recovery Consistency
Recovery from unified snapshots MUST preserve Run/Stream semantic equivalence and memory/file backend parity, including normalized terminal family, terminal state, recovery conflict classification, and bounded attempt metadata.

#### Scenario: Run/Stream equivalence after restore
- **WHEN** the same snapshot is restored and resumed through `Run` and `Stream`
- **THEN** resulting terminal classification, normalized family, recovery aggregates, and conflict handling remain equivalent

#### Scenario: Backend parity after restore
- **WHEN** recovery is executed against memory and file scheduler backends from equivalent snapshots
- **THEN** restored task/session semantics and normalized terminal outcomes remain equivalent modulo additive metadata ordering
