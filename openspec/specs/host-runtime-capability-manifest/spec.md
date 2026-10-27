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

### Requirement: Capability entries SHALL declare deployment and writer scope

Every source-history, checkpoint, continuation, and mailbox capability entry
MUST declare a bounded deployment profile, writer ownership mode, fencing
version, and transaction scope. The default supported profile SHALL be
`same_host_single_writer_sequential_reopen`; multi-writer, cross-machine,
distributed-journal, and multi-instance resume claims MUST be explicit cells
and MUST remain unsupported when no source evidence exists.

#### Scenario: Planner distinguishes sequential reopen from distributed restore
- **WHEN** a planner reads a FileStore-backed continuation capability
- **THEN** it receives the same-host single-writer profile and its fencing and
  transaction scope
- **AND** it does not infer cross-machine or concurrent restore support

#### Scenario: Unsupported deployment scope is explicit
- **WHEN** a host requests a multi-writer or cross-machine capability without
  source-owned evidence
- **THEN** the manifest returns `unsupported_capability` with a stable reason
- **AND** admission does not enqueue, publish, claim, resume, or replay

### Requirement: Capability evidence SHALL identify ownership and fencing

A supported deployment declaration MUST identify the source owner, writer
ownership token or equivalent bounded fence reference, snapshot/journal
transaction scope, and the evidence revision that proves the declaration.
These references MUST be detached and MUST NOT contain payloads, credentials,
or host business authority.

#### Scenario: Stale writer is rejected before activation
- **WHEN** a request presents a writer fence older than the source's current
  owner fence
- **THEN** admission returns a typed stale-owner or unavailable result
- **AND** no source lifecycle or restore transition is performed

### Requirement: Workflow continuation capability SHALL require an active owner fence

The workflow continuation capability manifest MUST advertise same-host single-writer sequential reopen only while the workflow source store owns its process-wide writer fence and can validate incomplete-claim recovery semantics. A failed or absent fence MUST report typed unavailable/unsupported before workflow resume admission.

#### Scenario: Capability requested without an acquired fence
- **WHEN** a workflow Engine requests continuation capabilities but its FileStore has no valid writer ownership
- **THEN** capability lookup returns typed unavailable/unsupported
- **AND** the engine does not claim cross-process restore availability

#### Scenario: Capability requested by the fenced owner
- **WHEN** a workflow Engine is bound to a valid fenced FileStore with supported continuation contracts
- **THEN** it reports only same-host single-writer sequential-reopen support
- **AND** it does not advertise multi-writer or cross-machine recovery

### Requirement: Manifest SHALL declare source commit projection recovery separately

For a source with durable history, the capability manifest MUST separately
declare whether source commit/projection recovery is supported, unavailable, or
unsupported. A supported declaration MUST reference the source-owned commit
contract, its recovery evidence, and deployment scope. History replay, a file
reopen, or a continuation DTO MUST NOT imply this support.

#### Scenario: Durable history does not imply transactional recovery
- **WHEN** a source entry has a retained journal but no declared commit and
  projection recovery capability
- **THEN** the manifest reports the recovery dimension unavailable or unsupported
- **AND** the planner cannot admit a transaction-dependent projection path

#### Scenario: Supported recovery shows its boundary
- **WHEN** a source owner provides committed source records and recovery reads
- **THEN** the manifest includes its evidence reference and same-host or
  distributed deployment scope
- **AND** it does not claim an external business database is atomically committed

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

### Requirement: Manifest SHALL declare composite context assembly and restore independently

The versioned capability manifest SHALL declare host context projection assembly and correlated handoff/restore independently for runner/direct, child/fanout, team, workflow, and mailbox forms. Each supported cell MUST include source owner/namespace, supported native-node scope, projection and continuation contract/API versions, build/module revision, deployment profile, resolver reauthorization requirement, retention/expiry policy, and a concrete fixture or contract-test evidence reference. Missing or incompatible evidence MUST remain unsupported or unavailable with a stable reason. A supported context cell MUST NOT imply execution resume, source history replay, or business authorization/commit.

#### Scenario: Manifest enables only an evidenced source form
- **WHEN** a planner reads a manifest with valid context binding evidence for team but not mailbox
- **THEN** it may admit the declared team context path while mailbox context assembly remains unsupported/unavailable

#### Scenario: Context manifest does not imply execution restore
- **WHEN** a form supports context projection but lacks matching continuation restore evidence
- **THEN** the manifest reports context assembly separately and rejects cross-process context restore before checkpoint claim or source activation

#### Scenario: Lifecycle-only mailbox evidence cannot promote context cells
- **WHEN** mailbox evidence proves delivery, lease, retry, DLQ, or terminal lifecycle projection but does not expose a context-aware handler safe point
- **THEN** `host_context_assembly` and `host_context_handoff_restore` remain unsupported for `mailbox_async`
- **AND** the manifest MUST NOT infer context support from mailbox continuation, history, or planbridge package presence

#### Scenario: Deployment scope remains bounded
- **WHEN** the only configured context/continuation evidence is a same-host single-writer FileStore
- **THEN** the manifest declares sequential reopen scope and does not advertise multi-writer or cross-machine projection restore

### Requirement: Capability manifests SHALL separate history, commit, recovery, and acknowledgement

For every runner, child, team, workflow, and mailbox form, the capability
manifest MUST independently declare retained history, source commit evidence,
projection reconciliation, and consumer acknowledgement. A supported
projection-recovery cell MUST reference an explicitly injected matching journal
and ledger, evidence version, and deployment scope. Durable history or a DTO
alone MUST NOT promote the cell.

#### Scenario: History-only owner is not transactionally recoverable

- **WHEN** a source has a retained file journal but no matching durable
  projection ledger
- **THEN** the manifest reports history support and projection recovery as
  unsupported or unavailable with stable reasons
- **AND** plan admission cannot infer transactional projection

#### Scenario: Reconciliation evidence is bounded to local deployment

- **WHEN** a source owner supplies matching file-backed journal and ledger
  evidence
- **THEN** the manifest may report projection recovery for same-host,
  single-writer, sequential reopen
- **AND** it does not advertise multi-writer CAS, cross-machine coordination,
  or cross-database exactly-once semantics

### Requirement: Capability promotion SHALL be independent per source form

Evidence for one source form MUST NOT promote another form's projection
recovery or acknowledgement cell. Mailbox lifecycle history MUST NOT promote
runner, child, team, or workflow projection recovery, and direct runner
capability MUST NOT promote composite forms.

#### Scenario: Foreign owner evidence is rejected

- **WHEN** a capability source reports a ledger namespace or form that does not
  match the requested source form
- **THEN** capability construction returns a typed foreign or unsupported
  outcome before admission
- **AND** no journal, ledger, execution, or retry is created
