## ADDED Requirements

### Requirement: Observability export SHALL support OTel tracing collector interoperability
Runtime observability export contract MUST support OTel tracing collector interoperability under canonical OTLP tracing configuration.

Tracing export behavior MUST be backend-agnostic and MUST preserve canonical semantic mapping independent of collector implementation details.

#### Scenario: Runtime exports tracing to local collector
- **WHEN** tracing config targets a local OTLP-compatible collector endpoint
- **THEN** runtime exports canonical tracing payloads and reports deterministic export status

#### Scenario: Runtime exports tracing to remote collector
- **WHEN** tracing config targets a remote OTLP-compatible collector endpoint
- **THEN** runtime preserves canonical tracing semantics without backend-specific field forks

### Requirement: OTel tracing export failure handling SHALL map to canonical policy semantics
Tracing export failures MUST map to canonical reason taxonomy and MUST preserve configured on-error behavior (`fail_fast` or degradation semantics where applicable).

Failure handling MUST remain machine-assertable and replay-stable.

#### Scenario: Collector is unavailable during trace export
- **WHEN** tracing export cannot reach collector endpoint
- **THEN** runtime emits deterministic failure classification and applies configured on-error behavior

#### Scenario: Equivalent trace export failure repeats
- **WHEN** equivalent failure condition occurs under unchanged effective config
- **THEN** runtime emits semantically equivalent failure classification and status output

### Requirement: Tracing export path SHALL preserve Run Stream semantic equivalence
For equivalent input, effective tracing config, and dependency state, Run and Stream paths MUST produce semantically equivalent tracing export outcomes.

Diagnostics bundle metadata MUST include trace schema compatibility metadata when tracing is enabled.

#### Scenario: Equivalent request is executed through Run and Stream
- **WHEN** equivalent requests execute with tracing export enabled
- **THEN** Run and Stream produce semantically equivalent tracing export status classifications

#### Scenario: Diagnostics bundle generated after tracing-enabled run
- **WHEN** diagnostics bundle generation runs after tracing-enabled execution
- **THEN** bundle metadata preserves canonical trace schema compatibility markers for replay consumers
