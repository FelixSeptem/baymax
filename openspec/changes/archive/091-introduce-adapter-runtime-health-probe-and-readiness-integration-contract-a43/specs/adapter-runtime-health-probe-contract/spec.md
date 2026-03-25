## ADDED Requirements

### Requirement: Runtime SHALL expose library-level adapter health probe contract
Runtime MUST expose a library-level adapter health probe contract that can evaluate runtime adapter availability without requiring platform control-plane dependencies.

Probe result MUST include canonical fields:
- `status` (`healthy|degraded|unavailable`)
- `code`
- `message`
- `metadata`
- `checked_at`

#### Scenario: Host probes registered adapter health
- **WHEN** host invokes adapter health probe for a registered adapter
- **THEN** runtime returns deterministic health result with canonical result fields

#### Scenario: Host probes unknown adapter target
- **WHEN** host invokes health probe for adapter that is not registered
- **THEN** runtime returns deterministic `unavailable` result with canonical not-found code

### Requirement: Adapter health probe SHALL enforce timeout and cached-result semantics
Adapter health probe execution MUST enforce configured timeout and MAY reuse cached probe result within configured TTL.

Timeout and cache-hit behaviors MUST be observable through deterministic status and reason code outputs.

#### Scenario: Probe exceeds timeout budget
- **WHEN** probe execution exceeds configured timeout
- **THEN** runtime classifies result as `unavailable` with canonical timeout reason code

#### Scenario: Probe result is reused within cache TTL
- **WHEN** repeated probe is requested within configured cache TTL
- **THEN** runtime reuses cached result and preserves semantically equivalent health classification

### Requirement: Adapter health semantics SHALL preserve mode equivalence and replay stability
For equivalent adapter state and effective configuration, health classification MUST remain semantically equivalent across Run and Stream paths.

Equivalent health events replayed into diagnostics MUST remain logically idempotent.

#### Scenario: Equivalent Run and Stream health evaluation
- **WHEN** equivalent requests evaluate same adapter health under Run and Stream
- **THEN** health status and canonical reason classification remain semantically equivalent

#### Scenario: Health events are replayed
- **WHEN** equivalent adapter-health events are replayed for one run
- **THEN** logical health aggregates remain stable after first ingestion
