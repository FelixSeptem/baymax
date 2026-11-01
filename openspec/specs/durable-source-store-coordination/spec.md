# durable-source-store-coordination Specification

## Purpose
This capability gives durable source owners an explicit, testable coordination
contract for multi-writer state without promoting local persistence into
distributed recovery.

## Requirements

### Requirement: Coordinated source owners SHALL expose fenced revisioned CAS

A source owner that advertises multi-writer or multi-instance durable state
SHALL bind its source state to an explicit coordinator. Cross-machine support
MUST remain unsupported unless an implementation provides separate evidence
for its distributed lock and CAS semantics. The
coordinator MUST provide a source-owner namespace, opaque writer fence, lease
expiry, monotonically increasing revision and compare-and-swap publication of
bounded detached source facts. A missing, expired, foreign or stale fence, or a
revision mismatch, MUST fail closed with a typed unavailable, foreign, stale or
conflict result.

#### Scenario: Stale writer cannot replace a new owner state
- **WHEN** a writer lease is replaced and the prior writer attempts a CAS
- **THEN** the prior write fails with typed stale-writer evidence and leaves the
  current source revision unchanged

#### Scenario: CAS conflict preserves committed source facts
- **WHEN** two writers submit a mutation from the same revision
- **THEN** exactly one detached source state is committed and the other receives
  a typed conflict without replacing, merging or retrying source facts

### Requirement: Coordinated checkpoint claim SHALL remain source-owned and atomic

When a checkpoint store is coordinator-backed, checkpoint save, resume claim,
state transition and terminal/successor publication SHALL each publish through
the owner coordinator revision. Equivalent duplicate resume requests MUST return
the committed source projection; a conflict, stale writer or coordinator outage
MUST NOT allocate a replacement logical run, rerun an effect or downgrade to a
local claim.

#### Scenario: Competing resume claims converge on one logical attempt
- **WHEN** distinct writers claim the same checkpoint concurrently through a
  valid coordinator
- **THEN** one logical attempt is committed and the other receives its retained
  duplicate/in-progress projection or a typed conflict

#### Scenario: Coordinator outage rejects before restore activation
- **WHEN** a coordinator-backed checkpoint cannot read or publish current source
  facts
- **THEN** resume admission returns typed unavailable before provider, tool,
  handler or replacement Run activation

### Requirement: Coordinated history SHALL atomically advance source cursor facts

When a source journal is coordinator-backed, append and retention-floor/cursor
advance SHALL be one source-owner CAS transition. Event identity remains
source-local and monotonic; `ReadAfter` remains read-only and MUST NOT obtain a
writer lease, advance a cursor, call execution, or repair state.

#### Scenario: Concurrent append has one source order
- **WHEN** two valid writers append source facts through the same journal
- **THEN** retained events receive one contiguous source-local sequence with no
  duplicate event ID and a conflicting append fails closed or retries only its
  pure source-state CAS before an event is exposed

#### Scenario: Replay remains side-effect free
- **WHEN** a consumer reads a retained coordinated source cursor after restart
- **THEN** it receives only committed source facts and does not acquire a lease,
  mutate a revision, invoke Run, Stream, retry, resume, tool or model

### Requirement: Coordination capability SHALL be explicit and deployment-scoped

Capability manifests and owner projections SHALL advertise multi-writer or
cross-machine support only while a compatible coordinator is live and its
source-owner, fencing, transaction and evidence facts validate. Existing
same-host single-writer stores MUST retain their prior capability. A plan or
resume request whose requested profile exceeds the owner capability MUST reject
before any source activation.

#### Scenario: Local FileStore cannot claim distributed support
- **WHEN** a local FileStore is opened without a coordinator and a caller
  requests multi-writer or cross-machine restore/cursor behavior
- **THEN** the owner and planbridge return typed unsupported before publish,
  claim, enqueue, Run or Stream

#### Scenario: Coordinator evidence is unavailable
- **WHEN** an owner loses its coordinator lease or cannot validate current
  coordination evidence
- **THEN** its capability becomes unavailable and plan admission fails before
  source activation
