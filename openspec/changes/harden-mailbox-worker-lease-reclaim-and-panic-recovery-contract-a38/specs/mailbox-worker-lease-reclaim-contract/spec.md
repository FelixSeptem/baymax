## ADDED Requirements

### Requirement: Mailbox worker SHALL reclaim stale in-flight messages deterministically
Mailbox worker runtime MUST support stale `in_flight` reclaim based on lease timeout semantics.

When an in-flight message exceeds configured timeout without heartbeat renewal, runtime MUST reclaim it deterministically and route it through configured lifecycle policy.

#### Scenario: Stale in-flight message is reclaimed on consume path
- **WHEN** worker process crashes after consume and message remains `in_flight` beyond `inflight_timeout`
- **THEN** next consume cycle reclaims the stale message and transitions it through deterministic requeue/dead-letter semantics

#### Scenario: Active long-running message is not reclaimed before timeout
- **WHEN** worker keeps renewing heartbeat for a long-running handler
- **THEN** runtime keeps message in active processing state and MUST NOT reclaim it prematurely

### Requirement: Mailbox worker SHALL recover handler panic with deterministic policy mapping
Mailbox worker MUST recover handler panic and MUST NOT terminate process loop with unbounded message loss.

Recovered panic MUST map to deterministic handler-error policy outcomes (`requeue` or `nack`) according to effective worker configuration.

#### Scenario: Panic recovered under default policy
- **WHEN** worker handler panics and effective policy is default (`requeue`)
- **THEN** runtime recovers panic, records lifecycle diagnostics, and performs requeue transition

#### Scenario: Panic recovered under nack policy
- **WHEN** worker handler panics and effective policy is `nack`
- **THEN** runtime recovers panic, records lifecycle diagnostics, and performs nack transition with deterministic retry/DLQ follow-up

### Requirement: Mailbox worker lease controls SHALL define conservative defaults
Mailbox worker lease/recover controls MUST resolve conservative defaults:
- `inflight_timeout=30s`
- `heartbeat_interval=5s`
- `reclaim_on_consume=true`
- `panic_policy=follow_handler_error_policy`

`heartbeat_interval` MUST be strictly greater than zero and MUST be strictly less than `inflight_timeout`.

#### Scenario: Runtime uses default lease controls
- **WHEN** mailbox worker lease controls are omitted
- **THEN** runtime resolves conservative defaults and keeps worker behavior deterministic

#### Scenario: Invalid lease controls are configured
- **WHEN** runtime receives invalid lease controls (for example `heartbeat_interval>=inflight_timeout`)
- **THEN** startup or hot reload fails fast and previous valid snapshot remains active
