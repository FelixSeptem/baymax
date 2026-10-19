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
