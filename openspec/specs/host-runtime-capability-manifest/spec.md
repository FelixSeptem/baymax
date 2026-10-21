# host-runtime-capability-manifest Specification

## Purpose
Defines a versioned, evidence-linked public capability contract so planners and hosts can determine source-owned runtime support before admission without inferring behavior from package presence, DTO compilation, or historical defaults.

## Requirements

### Requirement: Capability manifest SHALL declare execution support by form

The runtime SHALL expose a versioned capability document or API whose entries are keyed by declared execution form. Each entry MUST explicitly declare admission, native events, durable history, after-cursor replay, checkpoint creation, cross-process restore, `input_required`, `approval_required`, `budget_exhausted`, cancellation, timeout, causal retry, mailbox continuation and retention policy. A missing value MUST NOT be interpreted as support.

#### Scenario: Planner reads a declared mailbox form
- **WHEN** a planner requests capabilities for `mailbox_async`
- **THEN** it receives an explicit value for every required capability dimension
- **AND** it can determine whether the form is admissible without dispatching a source

#### Scenario: Undeclared form is not inferred from an API
- **WHEN** a package or DTO exists but its execution form is absent from the manifest
- **THEN** the caller receives an explicit unsupported result
- **AND** the runtime does not infer support from that implementation surface

### Requirement: Capability manifest SHALL be versioned and traceable

Every manifest SHALL identify its Baymax contract version, build or module revision, public API version and an evidence reference for each declared capability. Unsupported or unavailable entries MUST include a stable reason or limitation. Manifest identity and evidence references MUST be bounded and safe to expose without application payloads, credentials or callback errors.

#### Scenario: Capability declaration exposes provenance
- **WHEN** a host reads a supported capability entry
- **THEN** the entry includes contract version, build or module revision, public API version and contract-test or fixture evidence reference
- **AND** the evidence reference does not confer business authority

#### Scenario: Unsupported declaration names its limitation
- **WHEN** a form does not support durable history or cross-process restore
- **THEN** the manifest marks that dimension unsupported or unavailable with its limitation
- **AND** it does not advertise a future roadmap item as current support

### Requirement: Admission SHALL require compatible declared capability

An execution admission that depends on a manifest capability SHALL validate contract/API compatibility and current source availability before side effects. An incompatible manifest version, unavailable source, unsupported form or unsupported required dimension MUST produce a typed unavailable or unsupported result before enqueue, publish, Run, Stream, retry or restore. The runtime MUST NOT silently substitute a different execution form.

#### Scenario: Version-incompatible mailbox capability rejects before delivery
- **WHEN** a mailbox plan presents a manifest or mailbox contract version incompatible with the current source
- **THEN** admission returns a typed unavailable or unsupported result before a message is published or a Run is created
- **AND** it does not fall back to inline/direct execution

#### Scenario: Required pause capability is unsupported
- **WHEN** a plan requires `approval_required` for a form whose manifest marks it unsupported
- **THEN** admission rejects before source activation
- **AND** the planner can distinguish the rejection from a transient source failure

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

### Requirement: Manifest SHALL expose checkpoint and pause support by form

Capability entries SHALL separately declare checkpoint creation, cross-process restore, `input_required`, `approval_required`, and `budget_exhausted` for direct, child/fanout, team, workflow, and mailbox forms. Each supported cell MUST reference source evidence, contract version, build/module revision, public API version, and retention/expiry limitation; absent evidence is unsupported.

#### Scenario: Direct durable cell is source-evidenced
- **WHEN** a durable direct store and restore path are configured
- **THEN** the manifest marks only the verified pause cells supported and identifies their evidence

#### Scenario: Composite cells remain unsupported
- **WHEN** a child/team/workflow/mailbox owner has no graph restore evidence
- **THEN** each corresponding cell is explicitly `unsupported_capability` with a stable limitation

### Requirement: Admission SHALL reject unsupported restore before side effects

Planner and host admission MUST validate the required form, pause reason, contract version, and current source availability before enqueue, publish, Run, Stream, retry, or restore. Manifest values MUST NOT be inferred from package presence, DTO compilation, terminal-only behavior, or historical event replay.

#### Scenario: Required unsupported cell fails closed
- **WHEN** a plan requires `approval_required` on a form whose manifest marks it unsupported
- **THEN** admission returns typed `unsupported_capability` before source activation and does not substitute direct execution

### Requirement: Manifest expansion SHALL keep history evidence source-specific

The versioned host runtime capability manifest SHALL expose durable history and
after-cursor replay independently for runner, isolated child, team, workflow,
and mailbox forms. Each value SHALL include the source namespace, retention and
cursor boundary, availability, unsupported reason when false, and a reference
to fixture or contract evidence. Planner and admission MUST NOT infer support
from a form type, import, DTO, callback adapter, or mailbox capability.

#### Scenario: Unsupported composite history is rejected before read
- **WHEN** a plan asks for history for a child, team, or workflow source whose
  owner has not supplied retained evidence
- **THEN** the manifest reports unsupported with a stable reason
- **AND** admission does not create a cursor, subscribe to callbacks, or run
  the source

#### Scenario: Available source evidence is independently reported
- **WHEN** one source form has a retained journal and another source form does
  not
- **THEN** the manifest reports their history cells independently
- **AND** support is not borrowed across forms

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
