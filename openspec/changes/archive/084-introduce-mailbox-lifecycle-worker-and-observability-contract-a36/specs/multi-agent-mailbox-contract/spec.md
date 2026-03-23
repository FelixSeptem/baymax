## MODIFIED Requirements

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
