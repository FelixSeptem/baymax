## ADDED Requirements

### Requirement: Team and mailbox continuation capability SHALL be owner-evidenced per pause reason

The capability manifest SHALL declare checkpoint creation, cross-process restore, `input_required`, `approval_required`, `budget_exhausted`, cancellation, timeout, causal retry and mailbox continuation independently for team and mailbox forms. A supported team or mailbox cell MUST include the matching source continuation capability, source namespace, same deployment scope, retention/recovery boundary and concrete evidence reference. Default or unrelated form capabilities MUST NOT promote that cell.

#### Scenario: Team source promotes only team cells
- **WHEN** a valid team continuation capability source is configured
- **THEN** the manifest promotes only supported team checkpoint/restore/pause cells with its evidence and deployment scope
- **AND** child, workflow, mailbox and resumed-plan cells remain unchanged unless their own owners are configured

#### Scenario: Mailbox source promotes only mailbox cells
- **WHEN** a valid mailbox host-plan continuation capability source is configured
- **THEN** the manifest promotes only supported mailbox checkpoint/restore/pause, causal-retry and mailbox-continuation cells with its evidence and deployment scope
- **AND** it does not imply resumed planbridge execution

#### Scenario: Missing owner evidence remains rejectable
- **WHEN** a planner asks for a team or mailbox continuation cell without a valid owner capability source, version, namespace or evidence reference
- **THEN** the manifest reports `unsupported` or `unavailable` with a bounded reason
- **AND** admission rejects before member activation, publish, consume or resume
