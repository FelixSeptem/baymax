# composite-continuation-restore Specification

## Purpose
This capability provides source-owned, bounded continuation material for
composite executions and makes child, team, mailbox, and resumed-plan support
independent, evidence-based admission decisions.

## Requirements

### Requirement: Composite continuation material SHALL be source-owned and bounded

Each supported composite source SHALL persist a bounded continuation envelope containing form and graph identity/version/digest, original parent/run/session/causation/authority/occurrence/effect identities, native child/member/step or message/delivery/lease identifiers, dependency state, pending and completed actions, consumed budget/deadline, completed effect references, pause boundary, terminal arbitration evidence, and predecessor/successor lineage. Payloads, credentials, callbacks, and unbounded application bodies MUST remain outside the public envelope.

Each supported team continuation SHALL persist a bounded source-owned envelope with team/graph identity, version and digest; original parent/run/session/causation/authority/plan/write-scope/effect/occurrence identities; strategy; member records and native identifiers; pending and completed members; completed effect references; vote/fanout aggregate state; pending action; consumed budget and original deadline; terminal arbitration; and predecessor/successor lineage.

Each supported mailbox host-plan continuation SHALL persist a bounded source-owned envelope with host-plan binding identity; original message, delivery, lease and attempt causation; source lifecycle state; original parent/run/host correlation/authority/plan/write-scope/effect/occurrence identities; pending action; completed effect references; consumed budget and original deadline; lifecycle/terminal arbitration; and predecessor/successor lineage. Public continuation material MUST exclude business payloads, credentials, callbacks and host business-commit assertions.

#### Scenario: Child continuation retains source identity
- **WHEN** a child pauses at a declared safe point
- **THEN** its persisted continuation retains the original parent/run/session,
  task/attempt/lease, causation, budget, and completed-effect identities

#### Scenario: Existing recovery material is not continuation material
- **WHEN** admission receives a scheduler snapshot, mailbox history, callback
  event, or application projection without a source continuation envelope
- **THEN** it returns `unsupported_capability` or typed unavailable before any
  new child, member, message, or plan activation

#### Scenario: Team pause captures deterministic pending work
- **WHEN** a supported team owner pauses before dispatching a pending member/effect
- **THEN** it persists settled member records, the deterministic pending set and aggregate evidence
- **AND** an ambiguous in-flight member causes typed `recovery-required` or unsupported admission rather than a fabricated checkpoint

#### Scenario: Mailbox pause captures delivery lineage
- **WHEN** a supported mailbox owner pauses a host-plan message before its pending action/effect dispatch
- **THEN** the continuation retains the original message, delivery, lease and attempt causation facts
- **AND** it does not replace mailbox lifecycle ownership or create a new delivery identity

### Requirement: Supported composite forms SHALL use atomic idempotent resume claims

A source owner SHALL validate schema, digest, ownership, expiry, graph/form
binding, authority/plan/write-scope/effect namespace, occurrence/causation,
current revision/budget facts, and predecessor/successor relation before an
atomic resume claim. Equivalent requests SHALL return the same retained
projection; conflicting, foreign, stale, expired, corrupt, unavailable, or
already-running requests SHALL fail closed.

#### Scenario: Duplicate child resume converges
- **WHEN** two equivalent requests race for one child continuation
- **THEN** exactly one source continuation is admitted and both callers observe
  the same claim or terminal projection

#### Scenario: Conflicting composite resume is rejected
- **WHEN** a request changes mode, form, graph digest, pause reason, causation,
  or resume identity
- **THEN** the source returns a typed conflict/in-progress result without
  consuming the continuation or starting another logical execution

### Requirement: Child and fanout restore SHALL not re-enqueue or rerun completed effects

When the child/fanout source advertises restore, a new Composer SHALL reopen the
configured source store, restore the same native child graph and scheduler
facts, skip completed members/effects, and continue only pending work. Restore
MUST NOT call a new parent Run, create a replacement task, enqueue a duplicate,
or use history replay as execution state.

#### Scenario: New Composer restores pending local child
- **WHEN** a local child continuation is persisted and a new Composer opens the
  same compatible store
- **THEN** it resumes the original task/attempt lineage and does not spawn a
  second task or repeat completed effect outcomes

#### Scenario: Child lease loss is source evidence
- **WHEN** a child lease expires before restore
- **THEN** admission returns the native lease/recovery outcome and does not
  silently reclaim, retry, or re-run the parent

### Requirement: Pause reasons SHALL be independently evidenced by form

The manifest SHALL declare `input_required`, `approval_required`, and
`budget_exhausted` independently for each composite form. A supported cell MUST
have a source-owned producer, persisted safe point, cross-process restore,
cancel/timeout/retry race tests, terminal evidence, and Run/Stream parity. A
missing cell SHALL return `unsupported_capability` before activation.

#### Scenario: Unsupported team pause is rejected
- **WHEN** a team plan requests approval continuation before team source evidence
  exists
- **THEN** admission rejects before member activation and does not borrow the
  workflow or child capability

#### Scenario: Child budget pause preserves consumed resources
- **WHEN** a child pauses for budget exhaustion
- **THEN** restore revalidates current source budget facts, preserves consumed
  resources and original deadline, and does not replenish or reset them

### Requirement: Composite lifecycle races SHALL fail closed

Cancellation, timeout, retry, lease loss, resume, claim/start, and terminal commit ordering SHALL be decided from source evidence. Accepted cancellation is not terminal cancellation until the source records it. Store or terminal commit failure MUST surface as typed unavailable/recovery-required and MUST NOT trigger automatic re-execution.

Team cancel, timeout, member failure, aggregate resolution, claim/start and terminal commit ordering SHALL be decided from team source evidence. Mailbox lease loss, heartbeat, ack, nack, requeue, retry, dead-letter, expiration, cancellation, timeout, claim/start and terminal ordering SHALL be decided from mailbox source evidence. Unavailable source storage, stale/foreign lease, incomplete claimed attempt or terminal persistence failure MUST return typed unavailable/conflict/recovery-required and MUST NOT automatically re-run members, re-consume messages, re-enqueue, retry or start a replacement parent Run.

#### Scenario: Crash after child claim
- **WHEN** a process crashes after a child claim but before successor or terminal
  commit
- **THEN** duplicate requests return in-progress or recovery-required, conflicting
  requests fail closed, and no continuation starts automatically

#### Scenario: Team claim crash requires recovery evidence
- **WHEN** a process exits after a team continuation claim but before a persisted successor or terminal result
- **THEN** duplicate resumes return the retained in-progress or `recovery-required` source projection
- **AND** no team member or aggregate execution starts automatically

#### Scenario: Mailbox lease loss prevents synthetic resume
- **WHEN** a mailbox continuation lease expires or is reclaimed after checkpoint persistence but before resume start
- **THEN** restore returns the current source lease/lifecycle recovery outcome
- **AND** it does not acknowledge, re-consume, requeue, publish or run the parent to reconstruct execution

#### Scenario: Retry and DLQ retain causation
- **WHEN** a resumed mailbox continuation nacks, retries or reaches DLQ
- **THEN** source evidence preserves message identity, delivery/lease identity and attempt/retry causation through the terminal projection
- **AND** terminal outcome is not presented as a host business commit

### Requirement: Team, mailbox, and resumed-plan cells SHALL remain explicit until evidenced

Team parallel/vote, mailbox host-plan continuation, and planbridge resumed execution SHALL remain `unsupported_capability` unless their own source owners provide equivalent graph/message/lease material, atomic claim, restore, terminal evidence, and capability references. Workflow or child support MUST NOT imply support for another form.

When a configured team source owner supplies bounded team continuation material and evidence for a specific pause-reason cell, team parallel/vote MAY be admitted only for that declared cell. When a configured mailbox source owner supplies bounded host-plan message/delivery/lease continuation material and evidence for a specific pause-reason cell, mailbox continuation MAY be admitted only for that declared cell. Resumed planbridge execution SHALL remain an independent unsupported cell until it owns separate source graph material; mailbox admission, lifecycle or continuation support MUST NOT be interpreted as resumed-plan support.

#### Scenario: Team continuation does not borrow workflow support
- **WHEN** a team plan requests continuation and only workflow or child source evidence is configured
- **THEN** admission rejects before member activation with `unsupported_capability`
- **AND** it does not reuse another form's checkpoint, history, scheduler snapshot or parent Run

#### Scenario: Resumed plan cannot borrow child support
- **WHEN** a resumed plan requests continuation while only child source evidence is configured
- **THEN** admission rejects the resumed plan with `unsupported_capability`
- **AND** it does not reuse child checkpoint material, history, scheduler state or parent Run

#### Scenario: Evidenced team resume preserves one graph
- **WHEN** an owner-configured team parallel or vote continuation is resumed by a compatible new process
- **THEN** it restores the original graph/member identities and only pending members
- **AND** completed members/effects and aggregate evidence are not executed a second time

#### Scenario: Mailbox continuation does not imply resumed planbridge support
- **WHEN** a host requests resumed planbridge execution after a mailbox host-plan continuation is enabled
- **THEN** planbridge rejects the resumed plan before publish, enqueue, Run, Stream or fallback
- **AND** it does not create a replacement message or parent Run

#### Scenario: Evidenced mailbox resume preserves one logical message
- **WHEN** a mailbox host-plan continuation is restored by a compatible new process with the original message, delivery and lease facts
- **THEN** it resumes only the source-owned pending boundary for that logical message
- **AND** it does not re-consume, re-enqueue or publish a replacement message

### Requirement: Composite graph restore SHALL inherit source store scope

Child, team, workflow, and mailbox graph checkpoints MUST carry the deployment
profile, writer owner, fence, and claim transaction scope of every source store
that owns graph state. Composite restore MUST be rejected when any required
member store is only local-single-writer but the request claims concurrent,
cross-machine, or multi-instance recovery.

#### Scenario: Mixed graph ownership fails closed
- **WHEN** a composite checkpoint binds a local child store and a requested
  distributed team store without distributed evidence
- **THEN** admission returns a typed unsupported deployment result before any
  member or step is resumed
- **AND** it does not resume a partial graph

#### Scenario: Local graph reopen retains scope
- **WHEN** a new process reopens a fully persisted graph under the same writer
  ownership
- **THEN** it may restore the graph within the same-host sequential profile
- **AND** completed effects and the original parent/causation identities remain
  source-owned

### Requirement: Composite claim conflicts SHALL not be hidden by aggregation

An aggregate graph result MUST preserve child/member/step store ownership and
claim conflicts. It MUST NOT turn a stale writer, foreign owner, or unsupported
deployment cell into a successful aggregate, retry, or fresh parent run.

#### Scenario: Member claim conflict blocks aggregate restore
- **WHEN** one member checkpoint cannot be claimed under its declared fence
- **THEN** the graph restore returns the member conflict and no aggregate
  execution starts
- **AND** already completed members are not replayed

### Requirement: Composite continuation material SHALL bind host context projection identity

Each supported composite source SHALL persist a bounded continuation envelope containing form and graph identity/version/digest, original parent/run/session/causation/authority/occurrence/effect identities, native child/member/step or message/delivery/lease identifiers, dependency state, pending and completed actions, consumed budget/deadline, completed effect references, pause boundary, terminal arbitration evidence, and predecessor/successor lineage. Payloads, credentials, callbacks, resolved host context bodies, and unbounded application bodies MUST remain outside the public envelope.

Each supported team continuation SHALL persist a bounded source-owned envelope with team/graph identity, version and digest; original parent/run/session/causation/authority/plan/write-scope/effect/occurrence identities; strategy; member records and native identifiers; pending and completed members; completed effect references; vote/fanout aggregate state; pending action; consumed budget and original deadline; terminal arbitration; and predecessor/successor lineage.

Each supported mailbox host-plan continuation SHALL persist a bounded source-owned envelope with host-plan binding identity; original message, delivery, lease and attempt causation; source lifecycle state; original parent/run/host correlation/authority/plan/write-scope/effect/occurrence identities; pending action; completed effect references; consumed budget and original deadline; lifecycle/terminal arbitration; and predecessor/successor lineage.

When a source owner supports host context projection, its continuation SHALL bind only the detached projection identity/version/digest, capability digest, source-reference digests, source form, exact native node identity, and projection/continuation generation. The binding MUST be included in checkpoint digest and predecessor/successor validation, and MUST NOT contain resolved source content or imply current authorization.

The continuation identity in that binding SHALL be an owner-preexisting logical node, attempt, step, member, message or delivery identity plus generation. An owner SHALL validate its actual checkpoint reference and digest at the existing load/claim boundary. It MUST NOT embed a digest derived from the same checkpoint envelope back into that envelope's projection binding, create a fixed-point hash, or introduce a second checkpoint identifier allocator merely to support host context correlation.

#### Scenario: Child continuation retains source identity
- **WHEN** a child pauses at a declared safe point
- **THEN** its persisted continuation retains the original parent/run/session, task/attempt/lease, causation, budget, and completed-effect identities

#### Scenario: Existing recovery material is not continuation material
- **WHEN** admission receives a scheduler snapshot, mailbox history, callback event, or application projection without a source continuation envelope
- **THEN** it returns `unsupported_capability` or typed unavailable before any new child, member, message, or plan activation

#### Scenario: Team pause captures deterministic pending work
- **WHEN** a supported team owner pauses before dispatching a pending member/effect
- **THEN** it persists settled member records, the deterministic pending set and aggregate evidence
- **AND** an ambiguous in-flight member causes typed `recovery-required` or unsupported admission rather than a fabricated checkpoint

#### Scenario: Mailbox pause captures delivery lineage
- **WHEN** a supported mailbox owner pauses a host-plan message before its pending action/effect dispatch
- **THEN** the continuation retains the original message, delivery, lease and attempt causation facts
- **AND** it does not replace mailbox lifecycle ownership or create a new delivery identity

#### Scenario: Lifecycle-only mailbox continuation cannot bind host context
- **WHEN** the mailbox owner exposes only lifecycle outcome projection and no message/delivery/lease/attempt-bound handler execution safe point
- **THEN** it MUST NOT persist or resolve a host context binding for that continuation
- **AND** admission remains typed unsupported rather than treating lifecycle replay, duplicate projection, or a new planbridge executor as execution restore

#### Scenario: Context binding follows only its native continuation lineage
- **WHEN** a source owner persists a continuation with a host projection binding
- **THEN** the checkpoint digest and successor lineage retain that binding's detached identity and native node scope without storing resolved context bodies

### Requirement: Composite context restore SHALL revalidate projection and native source binding

For each composite form that advertises context restore, the source owner MUST validate the checkpoint and native graph/message identity before context resolution, then validate the bound projection identity/digest/capability subset and invoke the current host resolver before assembly or injection. A missing, foreign, stale, conflicting, expired, or corrupt binding MUST fail closed without consuming the continuation or changing source state. A valid projection is limited to its bound member, step, child attempt, message, delivery, and lease generation; it MUST NOT transfer to siblings or successor deliveries unless the source owner explicitly persists a compatible successor binding.

#### Scenario: Foreign node projection is rejected before resolution
- **WHEN** a restore request presents a projection bound to another member, step, child attempt, message, delivery, or lease generation
- **THEN** the owner returns a typed conflict/unsupported result before resolver invocation and does not claim or consume the checkpoint

#### Scenario: Current authorization is checked after native restore validation
- **WHEN** the native checkpoint and projection binding are valid but the host resolver denies a source under current authorization
- **THEN** restore returns a bounded authorization failure without injecting context, continuing execution, or altering source terminal state

#### Scenario: Unsupported owner does not synthesize context restore
- **WHEN** an owner has native continuation support but no context binding capability/evidence
- **THEN** it rejects the context restore before checkpoint claim and does not derive support from direct assembly, source history, or another composite owner

### Requirement: Plan bridge continuation restore SHALL delegate to the matching source owner

For every execution form that advertises cross-process continuation restore, the host plan bridge SHALL require an explicit resume request and the matching source owner/store binding, validate the form-specific capability evidence and deployment scope before activation, and delegate claim/restore exactly once to that owner. The bridge MUST preserve the owner's native result, identity, and terminal evidence. A normal Run/Stream request, another form's checkpoint, a history cursor, or an application projection MUST NOT be used to synthesize restore.

#### Scenario: Workflow plan restores through a configured owner
- **WHEN** a workflow plan contains a valid owner-bound resume request and the configured workflow store supports the declared deployment profile
- **THEN** the bridge delegates resume to that workflow owner and returns its native result
- **AND** it does not call workflow Run/Stream to recreate the graph

#### Scenario: Missing or incompatible workflow owner fails before activation
- **WHEN** a resumed workflow plan has no compatible continuation store, mismatched capability version, or unsupported deployment scope
- **THEN** admission returns typed unavailable, unsupported, or version error before any step dispatch or checkpoint claim

#### Scenario: Manifest does not borrow another form's evidence
- **WHEN** a plan requests continuation for a form whose configured source/store lacks the requested restore or pause-reason evidence
- **THEN** the bridge rejects before owner activation with `unsupported_capability` or typed unavailable
- **AND** it does not infer support from another form, package presence, or DTO compilation

#### Scenario: Unsupported mailbox handler continuation remains rejected
- **WHEN** a mailbox plan requests handler continuation but the mailbox owner exposes lifecycle projection without a handler execution safe point
- **THEN** the bridge returns `unsupported_capability` before publish, consume, claim, or handler dispatch

#### Scenario: Run and Stream use the same native restore semantics
- **WHEN** an equivalent owner-bound workflow resume is invoked through Run or Stream
- **THEN** both use the same owner claim/restore and terminal result semantics
- **AND** Stream only adds native event delivery without changing execution or terminal decisions
