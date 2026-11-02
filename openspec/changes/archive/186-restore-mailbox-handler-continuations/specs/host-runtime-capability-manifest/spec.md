## ADDED Requirements

### Requirement: Mailbox handler continuation capability SHALL be owner- and reason-evidenced

The `mailbox_async` capability entry SHALL distinguish lifecycle continuation
from host-handler execution continuation. It SHALL mark
`input_required`, `approval_required`, and `budget_exhausted` supported only
when the exact mailbox owner publishes a compatible handler safe point, durable
checkpoint/restore evidence, deployment scope, resumer/revalidation evidence,
and terminal/Run-Stream evidence for that reason. The entry SHALL retain a
stable unsupported or unavailable reason for every missing dimension.

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
