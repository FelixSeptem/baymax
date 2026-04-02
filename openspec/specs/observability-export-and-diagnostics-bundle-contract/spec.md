# observability-export-and-diagnostics-bundle-contract Specification

## Purpose
TBD - created by archiving change introduce-observability-export-and-diagnostics-bundle-contract-a55. Update Purpose after archive.
## Requirements
### Requirement: Runtime SHALL provide profile-driven observability export contract
Runtime MUST provide a profile-driven observability export contract with canonical profile values:
- `none`
- `otlp`
- `langfuse`
- `custom`

Exporters MUST consume normalized runtime event payloads from the `RuntimeRecorder` single-writer path and MUST NOT bypass recorder semantics.

Exporter failures MUST map to canonical reason codes and MUST be machine-assertable.

#### Scenario: Runtime enables otlp export profile
- **WHEN** effective configuration sets `runtime.observability.export.profile=otlp`
- **THEN** runtime resolves canonical exporter behavior and emits deterministic export status metadata

#### Scenario: Exporter returns non-canonical error
- **WHEN** exporter implementation returns backend-specific error payload
- **THEN** runtime maps the error into canonical export reason code taxonomy

### Requirement: Diagnostics bundle SHALL be versioned deterministic and redaction-safe
Runtime MUST support deterministic diagnostics bundle generation with versioned manifest schema.

Bundle payload MUST include at minimum:
- manifest metadata (`schema_version`, generation timestamp, runtime metadata),
- timeline window,
- diagnostics snapshots,
- redacted effective configuration,
- replay hints,
- gate fingerprint metadata.

Bundle generation MUST apply repository redaction policy before persistence or export.

#### Scenario: Runtime generates diagnostics bundle successfully
- **WHEN** host triggers bundle generation under valid configuration and writable target path
- **THEN** runtime produces deterministic versioned bundle artifacts including all required sections

#### Scenario: Bundle includes sensitive fields before redaction
- **WHEN** diagnostics and effective config contain secret-like keys
- **THEN** generated bundle persists only redacted representations and never stores raw secret values

### Requirement: Export and bundle paths SHALL preserve Run Stream semantic equivalence
For equivalent input, effective configuration, and dependency state, Run and Stream paths MUST produce semantically equivalent export/bundle classification outcomes, allowing non-semantic ordering differences.

#### Scenario: Equivalent run emits exportable events via Run and Stream
- **WHEN** equivalent requests are executed through Run and Stream with the same export profile
- **THEN** both paths produce semantically equivalent export success or degradation classifications

#### Scenario: Equivalent bundle generation failure via Run and Stream
- **WHEN** equivalent requests trigger the same bundle output-path failure condition
- **THEN** both paths emit semantically equivalent bundle failure reason taxonomy

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

