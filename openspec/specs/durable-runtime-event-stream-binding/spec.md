# durable-runtime-event-stream-binding Specification

## Purpose
TBD - created by archiving change extend-realtime-event-protocol-with-durable-runtime-stream-binding. Update Purpose after archive.
## Requirements
### Requirement: Runtime SHALL expose a transport-neutral durable event-stream binding

The Runtime SHALL expose an additive, versioned, embeddable event-stream binding for a bounded subscription identity, source/session/run filter, start mode (`latest` or `after_cursor`), opaque cursor reference, and bounded delivery policy. The binding SHALL be transport-neutral and SHALL only project source-owned history, live events, retention, disconnect, and backpressure results. It MUST NOT create a listener, hosted event/session service, event store, scheduler, global queue, or control plane.

#### Scenario: Host starts a latest subscription
- **WHEN** a host supplies a valid bounded subscription for a source that supports live events
- **THEN** the binding returns a deterministic source-owned accepted/live outcome without creating transport or source state on behalf of the host

#### Scenario: Invalid subscription is rejected before source mutation
- **WHEN** a subscription has an invalid identifier, unbounded filter, unsupported start mode, or invalid delivery policy
- **THEN** binding validation returns a deterministic protocol classification and leaves source Runtime state unchanged

### Requirement: Cursor catch-up and live-tail handoff SHALL preserve source ordering semantics

For an `after_cursor` subscription, the binding SHALL request source-owned catch-up and SHALL represent `catching_up` followed by `live` only when the source supplies a valid handoff boundary. Catch-up/live overlap SHALL preserve existing Realtime event ID and deduplication semantics; a missing required sequence progression SHALL produce a deterministic gap classification. The binding SHALL NOT synthesize a cursor, history, sequence, or exactly-once guarantee.

#### Scenario: Catch-up hands off to live tail with bounded overlap
- **WHEN** a source supplies valid history after a cursor and the first live event overlaps the last catch-up event
- **THEN** normalized delivery preserves the existing deduplication behavior and reports a deterministic live handoff outcome

#### Scenario: Handoff gap is rejected
- **WHEN** source history and live events omit a required monotonic Realtime sequence between the reported handoff boundary and first live event
- **THEN** the binding returns a deterministic sequence-gap classification and does not claim a live subscription

### Requirement: Retention, disconnect, and backpressure outcomes SHALL remain source-owned and explicit

The binding SHALL represent source-owned cursor retention, disconnect recovery, and consumer backpressure with finite deterministic outcomes. A cursor outside source retention SHALL be reported as `expired` with a stable reason; unknown history availability SHALL be `unresolved`; a disconnected consumer SHALL be recoverable only when the source accepts its existing cursor; and backpressure policy/outcome pairs SHALL be validated without the binding queueing, dropping, or pausing Runtime work itself.

#### Scenario: Expired cursor does not silently start at latest
- **WHEN** a host requests catch-up from a cursor outside source retention
- **THEN** the binding returns an expired-cursor outcome and stable reason without delivering an unrelated latest stream

#### Scenario: Slow consumer is classified without a binding-owned queue
- **WHEN** a source reports a slow-consumer result under a compatible delivery policy
- **THEN** the binding exposes the source-owned backpressure outcome and reason without allocating a queue or altering Runner execution

#### Scenario: Incompatible delivery policy and outcome are rejected
- **WHEN** a mapping reports a dropped outcome for `reject` policy or a paused outcome for a source that did not declare pause support
- **THEN** binding validation fails deterministically and preserves source state

### Requirement: Durable event-stream binding SHALL be replayable and Run/Stream equivalent

The Runtime SHALL provide replay fixtures and contract-gate coverage for subscription validation, cursor catch-up, overlap deduplication, handoff gap, retention expiry, disconnect recovery, and backpressure classification. Equivalent Run and Stream inputs with the same source history, cursor, delivery policy, and source outcome SHALL produce semantically equivalent normalized binding decisions after permitted event-order normalization.

#### Scenario: Reconnect classification remains Run/Stream equivalent
- **WHEN** equivalent Run and Stream paths reconnect using the same valid source cursor and history
- **THEN** both paths expose equivalent catch-up, live-tail, correlation, and reason semantics

#### Scenario: Binding control-plane absence is enforced
- **WHEN** the durable event-stream binding contract gate runs
- **THEN** it fails if implementation introduces a transport gateway, hosted event/session service, external event store, or global binding queue

