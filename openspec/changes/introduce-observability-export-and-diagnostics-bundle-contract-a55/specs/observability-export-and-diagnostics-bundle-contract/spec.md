## ADDED Requirements

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
