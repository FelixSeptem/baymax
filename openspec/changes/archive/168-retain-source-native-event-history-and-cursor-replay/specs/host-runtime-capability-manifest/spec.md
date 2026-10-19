## ADDED Requirements

### Requirement: History declarations SHALL be source-specific and auditable

For every execution form, the capability manifest SHALL distinguish real-time
native events from retained source history and after-cursor replay. When either
history dimension is supported, the manifest SHALL expose a bounded evidence
reference to the source capability declaration and its retention policy. When it
is unsupported or unavailable, the manifest SHALL expose a stable reason that
identifies the owning source limitation. A form-level declaration MUST NOT
upgrade an undeclared runner, child, team, workflow or mailbox source instance
to history support.

#### Scenario: Mailbox form advertises its actual retained history
- **WHEN** a planner reads the mailbox_async manifest entry for a mailbox owner
  with a compatible retained source journal
- **THEN** durable history and after-cursor replay are declared with the
  source's evidence and retention policy
- **AND** the declaration does not imply checkpoint restore or pause support

#### Scenario: Other forms remain explicitly unsupported
- **WHEN** a planner reads direct, child, team or workflow entries whose source
  owners have not declared retained history
- **THEN** each history/replay dimension remains unsupported with a stable
  source-owned reason
- **AND** the planner cannot infer history from native events or a public DTO
