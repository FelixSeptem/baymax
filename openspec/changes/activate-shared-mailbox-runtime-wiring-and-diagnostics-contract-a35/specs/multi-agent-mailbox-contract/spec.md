## MODIFIED Requirements

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
