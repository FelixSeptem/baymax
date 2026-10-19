## MODIFIED Requirements

### Requirement: Orthogonal axes and conservative capabilities
Execution shape, delivery and trigger SHALL be separate dimensions. Supported shapes SHALL be direct, subagent, team and workflow, with explicit path constraints. The initial bridge SHALL support direct inline, inline teams parallel/vote, flat inline workflow and isolated exclusive local scheduler-backed child dispatch. Fanout SHALL use native teams/workflow composition. Mailbox delivery SHALL be supported only as `mailbox_async` when the capability manifest, mailbox contract version and source binding explicitly permit it. Resumed-trigger execution, native checkpoint resume, cross-owner durable continuation and shared-queue child dispatch MUST reject before dispatch unless a separately declared and compatible capability explicitly supports them. Manual/scheduled/callback labels MUST NOT create workers, schedules or authority.

#### Scenario: Scheduled direct execution
- **WHEN** a scheduled-labeled direct inline descriptor has a valid binding and host admission
- **THEN** the native direct owner executes
- **AND** the bridge creates no cron scheduler or additional authorization

#### Scenario: Manifest-approved mailbox request
- **WHEN** a descriptor requests `mailbox_async` delivery with a compatible manifest declaration and native mailbox binding
- **THEN** compilation and admission select the existing mailbox owner
- **AND** the bridge creates no queue, worker loop, scheduler or second mailbox owner

#### Scenario: Mailbox or resumed request
- **WHEN** a descriptor requests mailbox delivery without a compatible manifest and native mailbox binding, or requests resumed execution or native checkpoint recovery without a separately declared compatible capability
- **THEN** compilation returns typed unavailable or `unsupported_capability` before enqueue, publish or any new Run
- **AND** reusing a run ID or checkpoint reference does not override the rejection or cause inline/direct fallback

#### Scenario: Mailbox capability unavailable or incompatible
- **WHEN** a descriptor requests mailbox delivery but the manifest, mailbox contract version or source availability is absent or incompatible
- **THEN** compilation or admission returns typed unavailable or `unsupported_capability` before enqueue, publish or any new Run
- **AND** reusing a run ID or checkpoint reference does not override the rejection or cause inline/direct fallback

#### Scenario: Resumed request remains unsupported
- **WHEN** a descriptor requests resumed execution or native checkpoint recovery without a separately declared compatible capability
- **THEN** compilation returns `unsupported_capability` before enqueue, publish or any new Run
- **AND** reusing a run ID or checkpoint reference does not override the rejection

#### Scenario: Shared scheduler cannot dispatch a specific child
- **WHEN** the requested child binding uses a shared queue or cannot prove exclusive isolated ownership
- **THEN** the bridge rejects before enqueue or claim
- **AND** no unrelated queued task is claimed or mutated

#### Scenario: Local child source streaming limits
- **WHEN** an isolated local child is invoked through bridge Stream
- **THEN** only actual supported native source callbacks are forwarded
- **AND** token streaming or missing source capabilities are not fabricated from a synchronous child result

## ADDED Requirements

### Requirement: Mailbox plan binding SHALL preserve source-owned admission and identity

For an admitted `mailbox_async` plan, the bridge SHALL validate bounded mailbox configuration and contract version, source-owned message identity, parent run, causation, authority digest, plan digest, write scope, effect namespace and resource limits before delegating. It SHALL preserve the mailbox source's message ID, delivery/lease ID, original run/parent run, host correlation, attempt/retry causation, lifecycle outcome and terminal outcome. The bridge MUST NOT create a new logical parent Run, authorize a business write, or use host-created identifiers in place of required source identities.

#### Scenario: Complete mailbox binding is compiled without source activation
- **WHEN** a descriptor and mailbox binding agree on all required identity, digest, scope and resource facts
- **THEN** compilation produces a detached plan snapshot without publishing, claiming, acknowledging or executing a message
- **AND** later admission retains those facts for current source validation

#### Scenario: Binding mismatch fails before side effects
- **WHEN** message identity, parent run, causation, authority digest, plan digest, write scope, effect namespace or resource limit differs between the plan and binding
- **THEN** compilation or admission rejects with a bounded binding or invalid-plan error before source activation
- **AND** it does not publish a partial message or create a replacement Run

#### Scenario: Duplicate dispatch converges to one logical admission
- **WHEN** the same full logical mailbox identity is dispatched again with equivalent authority, occurrence and effect namespace
- **THEN** the bridge returns the existing logical admission or its idempotent projection
- **AND** it does not create a second logical execution or replace source correlation

#### Scenario: Non-equivalent identity reuse rejects
- **WHEN** a caller reuses message identity or duplicate identity with different authority, occurrence, parent/causation, write scope or effect namespace
- **THEN** admission fails closed before publish or execution
- **AND** it does not report duplicate success

### Requirement: Mailbox lifecycle outcomes SHALL remain source-owned

The bridge SHALL expose source-owned acknowledgement, negative acknowledgement, retry, DLQ, lease-loss, timeout and terminal/attempt evidence for an admitted mailbox plan. It MUST preserve the distinction between delivery lifecycle, source terminal outcome and host business commit. Lease loss, timeout, retry and DLQ MUST NOT be implemented by re-running a parent Run, fabricating a terminal result or changing the original authority, occurrence or effect namespace.

#### Scenario: Lease loss and retry retain causal evidence
- **WHEN** an admitted mailbox delivery loses its lease or is retried
- **THEN** the exposed outcome retains the original message/delivery/lease identity and attempt/retry causation
- **AND** the bridge does not start a replacement parent Run

#### Scenario: DLQ remains distinct from business commit
- **WHEN** a source mailbox delivery reaches its DLQ outcome
- **THEN** the bridge exposes the source DLQ classification and related attempt/terminal evidence
- **AND** it does not report a host business commit, successful direct Run or fabricated completion

### Requirement: Mailbox continuation SHALL preserve Run Stream terminal parity

Equivalent Run and Stream views of the same admitted mailbox continuation SHALL agree on normalized source terminal outcome, original authority/occurrence/effect namespace, source correlation and lifecycle facts within the source's declared event capability. Rejection, replay or outcome retrieval MUST NOT mutate delivery lifecycle, invoke a handler or create a second execution.

#### Scenario: Run Stream views agree for one logical message
- **WHEN** a single admitted mailbox continuation is observed through equivalent Run and Stream paths
- **THEN** both paths report equivalent normalized source terminal and correlation facts
- **AND** neither path creates another logical admission

#### Scenario: Rejected mailbox path has no source side effect
- **WHEN** a mailbox descriptor is rejected for unavailable capability, incompatible version or invalid binding
- **THEN** no message is published, claimed, acknowledged, retried or dead-lettered
- **AND** no handler, model, tool or parent Run is invoked
