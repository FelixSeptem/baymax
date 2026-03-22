## MODIFIED Requirements

### Requirement: Mailbox SHALL compose sync async delayed flows under one contract
Mailbox contract MUST support:
- synchronous command->result wait semantics,
- asynchronous publish and later result report semantics,
- delayed dispatch semantics via `not_before`.

Mailbox contract entrypoints MUST be the canonical orchestration invoke surface for sync/async/delayed coordination, and multi-agent modules MUST NOT depend on legacy direct invoke public APIs.

#### Scenario: Synchronous flow waits for result envelope
- **WHEN** caller uses mailbox sync invocation path
- **THEN** runtime returns only terminal `result` envelope or explicit context/error termination

#### Scenario: Delayed message with not_before
- **WHEN** envelope has `not_before` later than current runtime time
- **THEN** mailbox keeps it non-consumable until eligibility time is reached

#### Scenario: Maintainer audits invoke entrypoint consistency
- **WHEN** maintainer reviews multi-agent invoke API usage across orchestration modules
- **THEN** sync/async/delayed flows all converge through mailbox-backed entrypoints
