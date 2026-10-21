## ADDED Requirements

### Requirement: Manifest SHALL expose independent composite continuation cells

Capability documents SHALL independently declare checkpoint creation,
cross-process restore, each pause reason, cancellation, timeout, causal retry,
retention/expiry, evidence reference, and deployment limitation for child/fanout,
team, workflow, and mailbox. Resumed plan support SHALL be a separate bridge
binding cell. A missing or evidence-free cell is unsupported.

#### Scenario: Child evidence does not enable team
- **WHEN** child continuation cells are supported but team evidence is absent
- **THEN** the manifest reports child support and team `unsupported_capability`
  independently
