## ADDED Requirements

### Requirement: Runtime config SHALL expose A2A delivery and version-negotiation controls
Runtime configuration MUST expose A2A delivery and card-version-negotiation controls with deterministic precedence `env > file > default`.

At minimum, runtime MUST support:
- `a2a.delivery.mode` (`callback|sse`)
- `a2a.delivery.fallback_mode` (`callback|sse`)
- `a2a.delivery.callback_retry.max_attempts`
- `a2a.delivery.sse_reconnect.max_attempts`
- `a2a.card.version_policy.mode` (`strict_major`)
- `a2a.card.version_policy.min_supported_minor`

#### Scenario: Environment override takes precedence for A2A delivery controls
- **WHEN** both environment and YAML define A2A delivery controls
- **THEN** effective runtime config resolves A2A delivery fields by `env > file > default`

#### Scenario: Startup applies default A2A delivery and version policy
- **WHEN** runtime starts without explicit A2A delivery/version controls
- **THEN** effective config uses default delivery mode and version policy values

### Requirement: Runtime SHALL fail fast on invalid A2A delivery/version configuration
Runtime startup and hot reload MUST validate A2A delivery/version controls before activation.

Validation MUST reject:
- unsupported `a2a.delivery.mode` or `a2a.delivery.fallback_mode`,
- non-positive callback retry or SSE reconnect attempts,
- unsupported `a2a.card.version_policy.mode`,
- negative `a2a.card.version_policy.min_supported_minor`.

Invalid updates MUST NOT replace active configuration snapshot.

#### Scenario: Invalid delivery mode fails startup
- **WHEN** runtime config sets unsupported A2A delivery mode
- **THEN** startup fails fast with validation error

#### Scenario: Invalid reconnect budget fails hot reload and rolls back
- **WHEN** hot reload applies non-positive SSE reconnect max attempts
- **THEN** reload is rejected and runtime keeps previous valid snapshot

### Requirement: Runtime diagnostics SHALL expose additive A2A delivery/version fields
Runtime diagnostics MUST include additive A2A delivery/version fields in run and skill-adjacent summaries where applicable.

At minimum, diagnostics MUST support:
- `a2a_delivery_mode`
- `a2a_delivery_fallback_used`
- `a2a_delivery_fallback_reason`
- `a2a_version_local`
- `a2a_version_peer`
- `a2a_version_negotiation_result`

These fields MUST be backward-compatible and MUST NOT alter existing run summary semantics.

#### Scenario: Consumer inspects successful A2A negotiation
- **WHEN** diagnostics are queried for a run with successful A2A delivery/version negotiation
- **THEN** diagnostics include additive A2A delivery/version fields with success semantics

#### Scenario: Consumer inspects failed A2A version negotiation
- **WHEN** diagnostics are queried for a run that failed due to version mismatch
- **THEN** diagnostics include local/peer versions and normalized negotiation result fields

### Requirement: A2A delivery/version aggregates SHALL remain replay-idempotent
Repeated ingestion of identical A2A delivery/version events for the same run MUST NOT inflate logical counters or trend aggregates.

#### Scenario: Duplicate A2A delivery events are replayed
- **WHEN** identical delivery/negotiation events are replayed for a completed run
- **THEN** diagnostics keep stable A2A delivery/version aggregate counters after first logical write
