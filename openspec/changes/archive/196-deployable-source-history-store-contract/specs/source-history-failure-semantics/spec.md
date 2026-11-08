## ADDED Requirements

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
