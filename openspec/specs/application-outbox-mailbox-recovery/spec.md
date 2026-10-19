# application-outbox-mailbox-recovery Specification

## Purpose
Defines a bounded, explicit host integration contract that dispatches a host-owned application outbox to existing mailbox or scheduler owners while preserving at-least-once recovery and host-owned business commit semantics.

## Requirements

### Requirement: Application outbox dispatch SHALL use a trusted bounded opt-in binding

The runtime SHALL expose an explicitly enabled application outbox dispatch contract using a versioned, bounded, reference-only descriptor and trusted host binding. The first profile SHALL be `application_outbox_dispatch.v1` and SHALL correlate application outbox identity, occurrence identity when present, business revision, authority reference, effect-idempotency reference, target binding, and existing run/message/task/attempt references without carrying application payloads, effect bodies, webhook bodies, credentials, raw callback errors, or an outbox store.

The descriptor SHALL declare exactly one target: `mailbox` or `scheduler`. It SHALL have finite documented limits for its serialized envelope, reference count, each reference and each digest. Unsupported versions, empty identities, duplicate/conflicting correlation, malformed target bindings, oversized fields, absent trusted host binding, or a host admission result that is invalid, canceled, failed, panics, or disagrees with current facts SHALL fail closed before publish, enqueue, source execution, acknowledgement, retry, or business effect.

An application-supplied digest SHALL prove only normalized descriptor/binding consistency. It SHALL NOT substitute for current host authority, revision, effect eligibility, outbox state, or a business commit.

#### Scenario: Valid trusted mailbox binding is admitted
- **WHEN** a bounded `application_outbox_dispatch.v1` descriptor has a matching trusted current host binding and selects a valid mailbox target
- **THEN** the bridge may delegate one logical delivery to the existing mailbox owner using the stable correlation
- **AND** it does not persist the host outbox body or authorize a business effect

#### Scenario: Invalid or stale binding fails before side effects
- **WHEN** a descriptor is malformed, oversized, unsupported, missing current admission, has authority/revision/effect drift, or conflicts with its trusted target binding
- **THEN** dispatch rejects with a finite classification before mailbox publish, scheduler enqueue, handler execution, acknowledgement, or business commit

#### Scenario: Caller-provided digest is not current authority
- **WHEN** a caller supplies a descriptor whose digest is syntactically valid but whose current host authority or effect eligibility is revoked
- **THEN** dispatch rejects and does not treat the digest as authorization

### Requirement: Dispatch SHALL delegate to one native target without creating a parallel lifecycle owner

For the `mailbox` target, the bridge SHALL delegate delivery lifecycle to the existing mailbox publish, claim, acknowledgement, negative acknowledgement, requeue, delay, lease reclaim and DLQ semantics. For the `scheduler` target, the bridge SHALL delegate task, claim, attempt, lease, requeue/retry, terminal and DLQ semantics to the existing scheduler owner.

The bridge SHALL preserve the selected target's source identity, lifecycle classifications and correlation. It SHALL NOT create a mailbox/scheduler store, consumer worker, task queue, lease manager, retry policy, DLQ policy, terminal reducer, budget ledger, outbox persistence layer, cross-owner recovery state machine, host wire command, or provider dependency. Direct mailbox and scheduler APIs SHALL remain compatible and SHALL not require the bridge.

#### Scenario: Mailbox target preserves mailbox lifecycle ownership
- **WHEN** an admitted descriptor selects `mailbox` and the existing mailbox subsequently retries, delays, reclaims, acknowledges, negatively acknowledges, or dead-letters the message
- **THEN** those source lifecycle semantics remain authoritative
- **AND** the bridge does not add a second delivery retry, lease, acknowledgement, or terminal state

#### Scenario: Scheduler target preserves task-attempt lifecycle ownership
- **WHEN** an admitted descriptor selects `scheduler` and the existing scheduler claims, rolls over a lease, retries, requeues, terminally commits, or dead-letters its task
- **THEN** those source task/attempt semantics remain authoritative
- **AND** the bridge does not create a second task, attempt, lease, retry, or terminal owner

#### Scenario: Unsupported target rejects before dispatch
- **WHEN** a descriptor selects an unknown, absent, disabled, or source-incompatible target
- **THEN** the bridge rejects before publish/enqueue and creates no Run, message, task, attempt, effect, or synthetic terminal result

### Requirement: Delivery, execution, and business commit SHALL remain separate facts

The bridge SHALL represent delivery accepted, scheduler task/execution terminal, host effect committed, and host outbox confirmed as distinct facts. Publish success, enqueue success, message acknowledgement, task terminal commit, Run/Stream terminal result, descriptor digest, replay success, or duplicate convergence SHALL NOT be inferred to mean that the host business effect was committed or that its outbox record may be deleted.

The host SHALL retain application outbox ownership, occurrence/revision policy, business authorization, effect key, idempotency/CAS and final effect/outbox commit. The bridge SHALL require host-owned effect idempotency/CAS evidence at the declared recovery boundary but SHALL NOT execute, persist, reverse, or synthesize the business effect.

#### Scenario: Mailbox acknowledgement is not a business commit
- **WHEN** a mailbox-target message reaches acknowledged state
- **THEN** the bridge reports the source acknowledgement separately from host effect/outbox facts
- **AND** it does not represent the acknowledgement as effect committed or outbox confirmed

#### Scenario: Scheduler terminal result is not a business commit
- **WHEN** a scheduler-target task reaches a native terminal outcome
- **THEN** its task/attempt/result facts remain distinct from the host effect commit gate
- **AND** the bridge does not infer business success, delete an outbox record, or fabricate host confirmation

#### Scenario: Host reports an idempotently committed effect
- **WHEN** the host effect store proves the effect key was already committed for the matching trusted binding during recovery
- **THEN** the bridge preserves that host fact for reconciliation
- **AND** it does not repeat the effect or treat it as a source delivery acknowledgement

### Requirement: Crash recovery SHALL converge at-least-once without retry amplification

The bridge SHALL preserve at-least-once recovery using a stable full logical dispatch identity including target, outbox identity, occurrence identity when present, trusted revision/effect binding, source correlation and idempotency key. An exact duplicate SHALL converge to its existing logical dispatch result without a new target publish/enqueue, effect attempt, or lifecycle inflation. A non-equivalent reuse of identity SHALL reject rather than receive duplicate success.

The bridge SHALL classify and test at least: target dispatch accepted before host outbox confirmation; host effect committed before target acknowledgement/terminal reconciliation; duplicate dispatch; stale attempt; lease rollover; delayed delivery; cancellation; recovery after interruption; and DLQ. Delivery retry SHALL remain mailbox-owned, execution retry SHALL remain scheduler-owned, and business commit retry SHALL remain host-owned; the bridge SHALL not multiply retries across those scopes.

#### Scenario: Publish occurs before outbox confirmation
- **WHEN** a mailbox publish or scheduler enqueue succeeds but the host crashes before confirming its outbox record
- **THEN** retrying the same full logical dispatch identity converges without a second logical target delivery or duplicate business effect
- **AND** the host remains responsible for deciding when to confirm its outbox

#### Scenario: Effect commits before acknowledgement reconciliation
- **WHEN** the host commits an idempotent effect but crashes before target acknowledgement or terminal reconciliation completes
- **THEN** recovery revalidates current binding and effect idempotency before any further action
- **AND** it does not repeat the effect or infer that the target already acknowledged it

#### Scenario: Stale attempt cannot commit a newer occurrence
- **WHEN** a scheduler-target attempt becomes stale after lease rollover or its occurrence/revision binding is replaced
- **THEN** the stale attempt is rejected or follows the existing scheduler stale classification
- **AND** it cannot confirm the newer occurrence, effect, or outbox record

#### Scenario: Delivery exhausts into DLQ
- **WHEN** a target-specific delivery or execution lifecycle reaches its configured DLQ outcome
- **THEN** the native owner records its deterministic DLQ classification
- **AND** the bridge preserves the pending host reconciliation boundary without declaring business commit or silently rerouting retries

### Requirement: Dispatch conformance SHALL preserve source execution semantics and offline safety

For any bridge path that activates an existing Run or Stream source, equivalent supported Run and Stream executions SHALL preserve source-owned normalized outcome, cancellation/pause semantics, correlation and target facts. A mailbox acceptance, scheduler enqueue, replay, or recovery label SHALL not create a new Run, resume a metadata-only runner, fabricate token/events, or synthesize a terminal outcome.

The delivered contract SHALL include positive, negative, boundary and concurrency tests; both target integrations using real native owners with fake host/outbox/effect collaborators; versioned reference-only replay fixtures; equivalent shell/PowerShell contract gates; contribution checks; and an offline example. Replay SHALL not invoke host callbacks, execute effects, publish/enqueue a target, mutate an outbox, or restore runtime state.

#### Scenario: Supported source Run and Stream stay equivalent
- **WHEN** equivalent admitted dispatches activate a supported native source through Run and Stream
- **THEN** their normalized source outcome, correlation and target facts agree according to the source contract
- **AND** neither path turns delivery acceptance into a business terminal result

#### Scenario: Replay detects non-equivalent crash recovery evidence
- **WHEN** an offline fixture reports duplicate target activation, repeated effect, target/host fact conflation, or Run/Stream divergence
- **THEN** replay returns a stable contract-drift classification without invoking a host callback or source side effect

### Requirement: Public semantic documentation and example baseline SHALL remain synchronized

The proposal, design and tasks SHALL declare identical public semantic documentation commitments and the implementation SHALL synchronize public API, limits, failure taxonomy, target support matrix, ownership boundaries, delivery/execution/business-commit separation, compatibility and rollback in maintained documentation. The offline example README SHALL be established before example code and identify semantic anchors, runtime paths, expected markers and rollback notes.

Implementation completion and archive SHALL be blocked while required public documentation, example baseline, tests, replay evidence, or shell/PowerShell gate evidence is missing.

#### Scenario: Documentation omits a public boundary
- **WHEN** the implementation changes a public bridge target, limit, error, commit boundary, compatibility guarantee, or rollback instruction without synchronized documentation
- **THEN** documentation governance fails and the change cannot be completed or archived

#### Scenario: Offline example distinguishes target and business facts
- **WHEN** the delivered offline example exercises mailbox and scheduler targets across both crash windows
- **THEN** its README and expected markers distinguish delivery/execution facts from host effect/outbox confirmation
- **AND** it requires no live provider, business database, or hosted control plane
