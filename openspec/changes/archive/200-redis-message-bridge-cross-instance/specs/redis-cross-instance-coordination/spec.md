## ADDED Requirements

### Requirement: Redis coordinator SHALL implement the existing source-owner Coordinator contract
Redis MUST implement `Acquire`, `Read`, `CompareAndSwap`, `ValidateLease` and `DeploymentScope` with the same error taxonomy as the memory and file backends (`ErrForeignWriter`, `ErrStaleWriter`, `ErrDeploymentConflict`, `ErrOwnerUnavailable`), without introducing parallel fencing semantics.

#### Scenario: two writers contend for the same namespace
- **WHEN** writer A holds an unexpired lease and writer B calls `Acquire`
- **THEN** B receives `ErrOwnerUnavailable` and A's lease stays valid

#### Scenario: stale fence is rejected
- **WHEN** a writer presents a lease whose fence is no longer current
- **THEN** `CompareAndSwap` returns `ErrStaleWriter` and the stored revision is unchanged

#### Scenario: revision conflict is rejected atomically
- **WHEN** two writers holding valid leases attempt `CompareAndSwap` with the same expected revision
- **THEN** exactly one succeeds and the other receives `ErrDeploymentConflict`, with no partial write

### Requirement: cross_machine deployment scope SHALL be evidence-gated
`cross_machine` MUST only be reported as supported when a distributed coordinator is explicitly configured and connectivity plus lease readability are verified; otherwise the scope MUST stay fail-closed as unsupported.

#### Scenario: distributed coordinator is reachable
- **WHEN** the Redis coordinator answers ping and its lease key is readable
- **THEN** `DeploymentScope()` returns `profile=cross_machine`, `fencing_scope=distributed`, `transaction_scope=distributed`, `supported=true`

#### Scenario: coordinator is not configured
- **WHEN** `runtime.coordination.backend` is `memory` or `file`
- **THEN** `DeploymentScope()` returns an unsupported scope with a bounded reason and `cross_machine` requests are rejected with `ErrDeploymentUnsupported`

#### Scenario: cross-machine profile requires distributed fencing
- **WHEN** a scope declares `profile=cross_machine` but `fencing_scope=source_owner`
- **THEN** `Validate()` returns `ErrDeploymentUnsupported`

### Requirement: Redis mailbox store SHALL preserve existing delivery semantics
The Redis backend MUST implement the full `mailbox.Store` surface with the same publish idempotency, inflight ownership, timeout reclaim, ack/nack/requeue and snapshot/restore semantics as the file backend.

#### Scenario: duplicate publish stays idempotent
- **WHEN** the same message id is published twice
- **THEN** the second call returns the existing record and the ready queue holds a single entry

#### Scenario: expired inflight message is reclaimed
- **WHEN** a consumer's inflight window expires without heartbeat
- **THEN** a later consume with reclaim enabled returns that message and increments the reclaimed counter

#### Scenario: consumer mismatch is rejected
- **WHEN** a consumer acks a message owned by another consumer
- **THEN** the store returns `ErrConsumerMismatch` and leaves the record untouched

### Requirement: realtime cursor storage SHALL be injectable with unchanged default behavior
`core/runner` MUST load and save realtime cursors through a `CursorStore` abstraction; the default implementation MUST keep the current in-process map plus TTL semantics, and an injected store failure MUST NOT block Run/Stream.

#### Scenario: default backend keeps process-local behavior
- **WHEN** no cursor store is injected
- **THEN** cursors live in process memory with the existing TTL expiry behavior

#### Scenario: injected store is unavailable
- **WHEN** an injected cursor store returns an error on load or save
- **THEN** the runtime continues the stream and records a diagnostic instead of failing the run

### Requirement: instance registry SHALL expose affinity records for cross-instance routing
The registry MUST support instance heartbeats with TTL, active instance enumeration, and session-to-instance binding with lookup, so hosts can route sticky traffic without the runtime shipping an HTTP load balancer.

#### Scenario: expired instance is not routable
- **WHEN** an instance stops heartbeating past its TTL
- **THEN** it no longer appears in the active instance list

#### Scenario: affinity target disappeared
- **WHEN** a session is bound to an instance that is no longer active
- **THEN** resolution returns `ErrAffinityLost` so the host can replay or reject instead of silently rerouting

### Requirement: coordination configuration SHALL fail fast and fail closed
Invalid coordination configuration MUST fail startup and hot reload atomically; runtime unavailability of the configured backend MUST surface as an error rather than a silent fallback to process-local state.

#### Scenario: redis backend without address
- **WHEN** `runtime.coordination.backend=redis` and `redis.addr` is empty
- **THEN** configuration validation fails with an error naming `runtime.coordination.redis.addr`

#### Scenario: mailbox backend without coordinator
- **WHEN** `runtime.coordination.mailbox_backend=redis` while `backend` is not `redis`
- **THEN** validation fails fast instead of producing a half-distributed deployment

#### Scenario: backend outage does not degrade silently
- **WHEN** the configured Redis backend becomes unreachable during runtime
- **THEN** lease and publish operations return an unavailable error and the deployment scope is no longer advertised as supported

### Requirement: Redis coordination SHALL be verified by a parity gate
The gate MUST run the same contract suite across memory, file and Redis backends, and MUST cover the fail-closed paths; shell and PowerShell entry points MUST stay at parity.

#### Scenario: gate blocks on semantic drift
- **WHEN** a backend disagrees with the shared contract assertions
- **THEN** `scripts/check-redis-coordination-contract` fails

#### Scenario: live Redis is not configured
- **WHEN** `BAYMAX_TEST_REDIS_ADDR` is unset
- **THEN** real-Redis integration cases report `skipped-unconfigured` rather than pass or fail
