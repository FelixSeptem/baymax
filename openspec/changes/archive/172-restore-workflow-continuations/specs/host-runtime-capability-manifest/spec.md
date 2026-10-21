## ADDED Requirements

### Requirement: Manifest SHALL distinguish workflow continuation support from other composite forms

The capability manifest SHALL declare checkpoint creation, cross-process restore, `input_required`, `approval_required`, `budget_exhausted`, cancellation, timeout, causal retry, retention/expiry, and evidence reference independently for the workflow form. A supported workflow cell SHALL identify the compatible contract/API version and storage deployment limitation. Child/fanout, team parallel/vote, mailbox host plan, and planbridge resumed cells SHALL remain explicitly unsupported until each source owner supplies its own evidence.

#### Scenario: Planner admits only an evidenced workflow pause
- **WHEN** a planner requires workflow cross-process restore for a declared pause reason with compatible source evidence
- **THEN** it can admit that workflow form without inferring support for any other composite form

#### Scenario: Unsupported composite form cannot borrow workflow capability
- **WHEN** a planner requests team or mailbox restore after workflow support is enabled
- **THEN** the manifest returns the corresponding unsupported limitation before admission

### Requirement: Workflow admission SHALL validate manifest compatibility before restore

Workflow restore admission SHALL verify manifest contract/API compatibility, evidence availability, and declared source store scope before claiming the checkpoint. An unavailable owner, incompatible contract, or unsupported deployment profile SHALL return a typed unavailable or unsupported result before restore side effects.

#### Scenario: Incompatible workflow manifest rejects before claim
- **WHEN** a resume request requires a workflow capability version not provided by the current source owner
- **THEN** admission fails before claiming the checkpoint or starting a workflow step
