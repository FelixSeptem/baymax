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
