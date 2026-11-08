## MODIFIED Requirements

### Requirement: Team and mailbox continuation capability SHALL be owner-evidenced per pause reason

The public continuation capability document SHALL expose bounded, detached evidence for each canonical pause reason. Each entry MUST declare whether it is supported and, respectively, a valid evidence reference or a stable unsupported reason. Validation MUST reject missing, duplicate, foreign, package-only, or unbounded evidence. Capability manifest promotion MUST use only the configured source owner's entry for the requested execution form and deployment scope. Team promotion MUST also require evidence that the owner binds normalized aggregation policy, participant provenance, duplicate claim behavior, ambiguous in-flight recovery, and Run/Stream parity; package presence or child/workflow evidence MUST NOT promote team cells.

#### Scenario: Team promotion requires policy and restore evidence
- **WHEN** a configured team owner reports pause and restore cells without evidence for aggregation-policy binding or ambiguous in-flight recovery
- **THEN** team capability promotion remains unsupported or unavailable with a bounded reason
- **AND** plan admission rejects before member activation

#### Scenario: Team evidence does not promote sibling forms
- **WHEN** a valid team source capability is configured
- **THEN** only team checkpoint/restore/pause cells are promoted
- **AND** child, workflow, mailbox, and resumed-plan cells retain their own independent states

#### Scenario: Partial source evidence is representable
- **WHEN** a source supports input pause but not approval or budget pause
- **THEN** validation succeeds with two explicit unsupported entries
- **AND** only the input cell is promoted in the manifest

#### Scenario: Package presence cannot promote capability
- **WHEN** a source returns a pause reason name without source evidence
- **THEN** the corresponding cell remains unsupported or unavailable
- **AND** admission does not activate the source

#### Scenario: Lifecycle-only mailbox evidence is bounded
- **WHEN** a mailbox source reports lifecycle continuation without handler safe-point evidence
- **THEN** the manifest preserves lifecycle facts but leaves handler pause cells and host context restore unsupported
- **AND** each unsupported handler pause cell uses the stable reason `mailbox_handler_safe_point_unsupported`

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
