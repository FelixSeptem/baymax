# multi-agent-mailbox-contract Specification

## Purpose
TBD - created by archiving change introduce-unified-mailbox-coordination-contract-a30. Update Purpose after archive.
## Requirements
### Requirement: Runtime SHALL provide unified mailbox envelope contract for multi-agent coordination
The runtime MUST provide a mailbox envelope contract with canonical fields:
`message_id`, `idempotency_key`, `correlation_id`, `kind`, `from_agent`, `to_agent`, `task_id`, `run_id`, `payload`, `not_before`, `expire_at`, and `attempt`.

The envelope `kind` MUST support at least `command`, `event`, and `result`.

#### Scenario: Publish command envelope
- **WHEN** host publishes a valid `command` envelope
- **THEN** mailbox accepts and persists the envelope with normalized metadata

#### Scenario: Publish invalid envelope
- **WHEN** envelope is missing required identifiers or has unsupported kind
- **THEN** mailbox returns validation error immediately and does not persist partial data

### Requirement: Mailbox delivery SHALL be at-least-once with idempotent convergence
Mailbox delivery MUST provide at-least-once semantics.

Duplicate publishes or retries with the same `idempotency_key` MUST converge to one logical message outcome.

#### Scenario: Duplicate publish with same idempotency key
- **WHEN** the same logical envelope is published multiple times with identical `idempotency_key`
- **THEN** mailbox converges duplicates and does not inflate logical outcome counts

#### Scenario: Retry delivery after transient consume failure
- **WHEN** consumer fails transiently and message is retried
- **THEN** mailbox redelivers according to retry policy while preserving logical dedup semantics

### Requirement: Mailbox lifecycle SHALL support ack, nack, retry, ttl, and dlq semantics
Mailbox MUST support `Ack`, `Nack`, and `Requeue` semantics for consumed messages.

Mailbox MUST enforce TTL and expiration behavior, and expired or retry-exhausted messages MUST follow configured drop/DLQ policy.

Mailbox lifecycle semantics MUST be executable through runtime mailbox worker primitive (when enabled), and handler error paths MUST map to deterministic lifecycle transitions according to configured handler-error policy.

#### Scenario: Consumer acknowledges message
- **WHEN** consumer calls `Ack` on delivered envelope
- **THEN** message transitions to terminal acknowledged state and is not redelivered

#### Scenario: Message exceeds retry budget with DLQ enabled
- **WHEN** message retries exceed configured limit and DLQ is enabled
- **THEN** mailbox moves the message to dead-letter state with deterministic reason metadata

#### Scenario: Worker handler error applies default requeue policy
- **WHEN** mailbox worker is enabled with default policy and handler returns error for an in-flight message
- **THEN** runtime performs requeue transition before retry budget exhaustion and preserves deterministic backoff/retry lifecycle behavior

### Requirement: Mailbox SHALL compose sync async delayed flows under one contract
Mailbox contract MUST support:
- synchronous command->result wait semantics,
- asynchronous publish and later result report semantics,
- delayed dispatch semantics via `not_before`.

Runtime-managed orchestration paths MUST route these sync/async/delayed flows through a shared mailbox runtime instance for the effective configuration snapshot, and MUST NOT rely on per-call ephemeral mailbox bridge state.

When configured persistent mailbox backend initialization fails, runtime MUST fallback to memory backend and preserve deterministic fallback reason metadata for diagnostics.

#### Scenario: Synchronous flow waits for result envelope
- **WHEN** caller uses mailbox sync invocation path
- **THEN** runtime returns only terminal `result` envelope or explicit context/error termination

#### Scenario: Delayed message with not_before
- **WHEN** envelope has `not_before` later than current runtime time
- **THEN** mailbox keeps it non-consumable until eligibility time is reached

#### Scenario: Managed runtime reuses shared mailbox instance
- **WHEN** equivalent orchestration requests are executed under one effective runtime configuration snapshot
- **THEN** mailbox command/result publication converges through the same shared runtime mailbox instance

#### Scenario: File backend init fails and runtime falls back
- **WHEN** managed runtime config requests `mailbox.backend=file` and initialization fails
- **THEN** runtime falls back to memory backend and records deterministic fallback reason metadata

### Requirement: Mailbox query API SHALL provide deterministic read-only retrieval
Mailbox MUST provide a read-only query API with canonical filtering, sorting, pagination, and opaque cursor traversal.

Default query behavior MUST use `page_size=50`, `page_size<=200`, and `updated_at desc` when not specified.

#### Scenario: Query uses filters and default pagination
- **WHEN** caller queries mailbox with `run_id` and `kind` filters without pagination fields
- **THEN** runtime applies `page_size=50`, `updated_at desc`, and returns only matching messages

#### Scenario: Query uses invalid cursor
- **WHEN** caller provides malformed or boundary-mismatched cursor
- **THEN** runtime fails fast with validation error

### Requirement: Mailbox SHALL preserve backend semantic parity
For equivalent mailbox state snapshots, `memory` and `file` backends MUST return semantically equivalent delivery and query outcomes.

#### Scenario: Equivalent state on memory and file backend
- **WHEN** same envelope set is restored on both backends and queried with identical request
- **THEN** result set, ordering, and cursor traversal semantics remain equivalent

