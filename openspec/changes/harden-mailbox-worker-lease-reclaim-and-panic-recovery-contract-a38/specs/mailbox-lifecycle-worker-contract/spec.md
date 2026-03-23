## MODIFIED Requirements

### Requirement: Runtime SHALL provide mailbox lifecycle worker primitive
Runtime MUST provide a library-level mailbox lifecycle worker primitive that executes:
`consume -> handler -> ack|nack|requeue` under one deterministic loop contract.

Worker primitive MUST be optional and MUST NOT require platform-side control plane dependencies.

Worker primitive MUST additionally support:
- handler panic recovery under configured panic policy,
- heartbeat renewal for active in-flight processing,
- stale in-flight reclaim under lease-timeout controls.

#### Scenario: Host starts worker and handler succeeds
- **WHEN** mailbox worker consumes a command envelope and handler returns success
- **THEN** runtime acknowledges the message and records lifecycle diagnostics for consume and ack

#### Scenario: Host starts worker and handler returns error
- **WHEN** mailbox worker consumes a command envelope and handler returns error
- **THEN** runtime applies configured handler-error policy and records lifecycle diagnostics including reason code

#### Scenario: Host starts worker and handler panics
- **WHEN** mailbox worker consumes a command envelope and handler panics
- **THEN** runtime recovers panic, maps to configured policy outcome, and records lifecycle diagnostics without leaving unbounded stuck state

### Requirement: Mailbox worker defaults SHALL remain conservative and deterministic
Default mailbox worker behavior MUST be:
- `enabled=false`
- `poll_interval=100ms`
- `handler_error_policy=requeue`
- `inflight_timeout=30s`
- `heartbeat_interval=5s`
- `reclaim_on_consume=true`
- `panic_policy=follow_handler_error_policy`

`poll_interval` MUST be strictly greater than zero, `heartbeat_interval` MUST be strictly greater than zero, and `heartbeat_interval` MUST be strictly less than `inflight_timeout`.

Invalid values in startup or hot reload MUST fail fast with rollback semantics.

#### Scenario: Runtime uses default worker config
- **WHEN** mailbox worker config is omitted
- **THEN** runtime resolves defaults (`enabled=false`, `poll_interval=100ms`, `handler_error_policy=requeue`, `inflight_timeout=30s`, `heartbeat_interval=5s`, `reclaim_on_consume=true`, `panic_policy=follow_handler_error_policy`)

#### Scenario: Hot reload sets invalid poll interval
- **WHEN** hot reload sets `mailbox.worker.poll_interval<=0`
- **THEN** runtime rejects reload and keeps previous valid snapshot

#### Scenario: Hot reload sets invalid lease controls
- **WHEN** hot reload sets `mailbox.worker.heartbeat_interval>=mailbox.worker.inflight_timeout`
- **THEN** runtime rejects reload and keeps previous valid snapshot

### Requirement: Mailbox lifecycle reason taxonomy SHALL be frozen by contract
Mailbox lifecycle diagnostics MUST use a canonical reason taxonomy for lifecycle transitions.

Minimum required reason set MUST include:
- `retry_exhausted`
- `expired`
- `consumer_mismatch`
- `message_not_found`
- `handler_error`
- `lease_expired`

Reason taxonomy extensions MUST be additive and MUST be synchronized with contract and gate updates.

#### Scenario: Lifecycle records use canonical reason taxonomy
- **WHEN** mailbox worker emits lifecycle diagnostics for nack/requeue/dead-letter/expired/reclaim flows
- **THEN** reason codes belong to canonical taxonomy or approved additive extension set

#### Scenario: Change introduces non-canonical lifecycle reason
- **WHEN** repository change emits lifecycle reason outside canonical taxonomy without synchronized contract update
- **THEN** contract validation fails and blocks completion
