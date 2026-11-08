# source-history-failure-semantics Specification

## Purpose
Defines how source owners expose retained-history write failures so consumers can distinguish complete source evidence from best-effort observation without confusing replay with execution recovery.

## Requirements

### Requirement: Source owners SHALL expose history append health

An execution source that has explicitly bound a native history journal MUST expose failed appends as a typed source-history health outcome associated with that journal's source identity. It MUST NOT report the failed append as retained/replayable. This callback-history contract does not gate execution effects or make append failure an execution terminal. History remains opt-in; the source MUST NOT create a parallel journal implicitly.

#### Scenario: Callback history append fails
- **WHEN** an explicitly configured owner cannot persist a callback-projected source fact
- **THEN** its history health reports typed source-unavailable/degraded state for that source identity
- **AND** the failed fact is not represented as retained/replayable
- **AND** the owner does not claim that this callback append gated an effect

#### Scenario: Observation append fails after an effect boundary
- **WHEN** a non-authoritative observation append fails after the source may have performed an effect
- **THEN** the source exposes degraded/unavailable history health with the affected source identity
- **AND** the runtime does not convert the error into a fresh execution or claim an atomic consumer projection

### Requirement: History failure SHALL preserve execution uncertainty

When a source-owned continuation transition may have happened but its continuation result cannot be committed, that continuation owner MUST expose `recovery_required` or an equivalent typed outcome. Callback-history replay MUST NOT be used to infer or repair the uncertain effect. The owner MUST NOT retry, resume, or rerun the parent/child/step solely to reconstruct missing history. Consumer projection remains independent and MUST replay only committed source facts.

#### Scenario: Terminal history commit fails after an effect
- **WHEN** execution may have completed an effect but persistence of its terminal source fact fails
- **THEN** the caller receives a typed unavailable or recovery-required result
- **AND** a retry of the logical execution is not started automatically

#### Scenario: Consumer projection restarts
- **WHEN** source facts were committed but the consumer projection crashed before applying them
- **THEN** the consumer can read the same source-owned facts after its confirmed cursor
- **AND** replay performs no source execution or business commit

### Requirement: Owner-store failures SHALL preserve durable capability truth

When an explicitly bound source history store, coordinator, or projection ledger becomes unavailable, the source owner MUST downgrade its capability to unavailable, degraded, or recovery-required with a typed source identity and deployment scope. It MUST NOT continue advertising complete durable replay or projection recovery based on stale capability evidence.

#### Scenario: Coordinator outage is visible

- **WHEN** a coordinated source owner cannot read or validate its current coordinator revision
- **THEN** capability and append admission return a typed unavailable result
- **AND** the owner does not claim the source event is committed or replayable

#### Scenario: File store corruption is fail-closed

- **WHEN** a fresh process cannot validate the persisted source journal or projection ledger
- **THEN** opening or recovery returns `ErrSourceUnavailable` or `recovery_required`
- **AND** the owner does not start at the latest retained event or rerun uncertain execution

### Requirement: Recovery-required outcomes SHALL retain reconciliation identity without execution retry

If source append or projection persistence is uncertain, the owner MUST retain the source identity, cursor, and deployment evidence needed for bounded reconciliation when available. Recovery-required handling MUST remain read/reconcile-only and MUST NOT synthesize a missing source fact or retry the effect.

#### Scenario: Uncertain append is not rerun

- **WHEN** a source append may have committed but its result cannot be confirmed
- **THEN** the owner returns a typed recovery-required or unavailable result
- **AND** no parent, child, team member, workflow step, mailbox handler, tool, or model effect is automatically rerun

#### Scenario: Committed fact is reconciled after projection crash

- **WHEN** source append committed and the consumer projection crashed before applying the fact
- **THEN** a fresh process can reconcile the exact committed fact by source identity and cursor
- **AND** repeated reconciliation is idempotent and side-effect free
