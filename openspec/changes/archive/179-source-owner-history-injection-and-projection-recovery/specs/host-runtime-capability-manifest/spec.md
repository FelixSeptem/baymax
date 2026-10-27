## ADDED Requirements

### Requirement: Capability manifests SHALL separate history, commit, recovery, and acknowledgement

For every runner, child, team, workflow, and mailbox form, the capability
manifest MUST independently declare retained history, source commit evidence,
projection reconciliation, and consumer acknowledgement. A supported
projection-recovery cell MUST reference an explicitly injected matching journal
and ledger, evidence version, and deployment scope. Durable history or a DTO
alone MUST NOT promote the cell.

#### Scenario: History-only owner is not transactionally recoverable

- **WHEN** a source has a retained file journal but no matching durable
  projection ledger
- **THEN** the manifest reports history support and projection recovery as
  unsupported or unavailable with stable reasons
- **AND** plan admission cannot infer transactional projection

#### Scenario: Reconciliation evidence is bounded to local deployment

- **WHEN** a source owner supplies matching file-backed journal and ledger
  evidence
- **THEN** the manifest may report projection recovery for same-host,
  single-writer, sequential reopen
- **AND** it does not advertise multi-writer CAS, cross-machine coordination,
  or cross-database exactly-once semantics

### Requirement: Capability promotion SHALL be independent per source form

Evidence for one source form MUST NOT promote another form's projection
recovery or acknowledgement cell. Mailbox lifecycle history MUST NOT promote
runner, child, team, or workflow projection recovery, and direct runner
capability MUST NOT promote composite forms.

#### Scenario: Foreign owner evidence is rejected

- **WHEN** a capability source reports a ledger namespace or form that does not
  match the requested source form
- **THEN** capability construction returns a typed foreign or unsupported
  outcome before admission
- **AND** no journal, ledger, execution, or retry is created
