## Modified Requirements

### Requirement: Mailbox handler continuation capability SHALL be owner- and reason-evidenced

The capability manifest SHALL promote mailbox handler
`input_required`, `approval_required`, and `budget_exhausted` cells only when
the exact configured mailbox owner reports durable handler safe-point and
cross-process restore evidence for each cell. Evidence MUST include the
same-host/single-writer/sequential-reopen deployment scope, fresh-process
restore, revalidation and lease fencing, terminal/retry/DLQ projection,
duplicate claim behavior, and completed-effect non-replay. A lifecycle-only
mailbox owner, host-plan history, package presence, or planbridge binding MUST
not promote these cells.

#### Scenario: Capability promotion preserves the owner deployment boundary

- **WHEN** the configured mailbox owner reports complete handler evidence but
  only a same-host single-writer sequential-reopen scope
- **THEN** the manifest promotes the three handler pause cells with that exact
  scope
- **AND** a multi-writer or cross-machine request remains unsupported before
  publish, consume, claim, or handler execution

#### Scenario: Missing one pause evidence keeps that cell unsupported

- **WHEN** the mailbox owner reports handler restore evidence for input and
  approval but not budget pause
- **THEN** only input and approval cells may be supported
- **AND** budget remains unsupported with a stable owner evidence reason

#### Scenario: Configured owner promotes only proven pause cells

- **WHEN** a mailbox owner is configured with durable handler continuation and
  evidence for input and approval but not budget pause
- **THEN** the manifest marks only input and approval supported
- **AND** the budget cell remains unsupported with its owner-local reason

#### Scenario: Lifecycle evidence cannot promote handler support

- **WHEN** the source exposes retained lifecycle history or a lifecycle-only
  continuation projection but no configured handler resumer
- **THEN** all mailbox host-handler pause cells remain unsupported
- **AND** planners cannot infer support from package presence, DTOs, or another
  form's evidence
