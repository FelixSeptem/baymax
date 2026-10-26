## ADDED Requirements

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
