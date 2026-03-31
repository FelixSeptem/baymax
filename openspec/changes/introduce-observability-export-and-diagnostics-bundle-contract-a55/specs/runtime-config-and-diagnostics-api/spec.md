## ADDED Requirements

### Requirement: Runtime config SHALL expose observability export and diagnostics bundle controls
Runtime configuration MUST expose `runtime.observability.export.*` and `runtime.diagnostics.bundle.*` fields with deterministic precedence `env > file > default`.

Minimum required controls for this milestone:
- `runtime.observability.export.enabled`
- `runtime.observability.export.profile` (`none|otlp|langfuse|custom`)
- `runtime.observability.export.endpoint`
- `runtime.observability.export.queue_capacity`
- `runtime.observability.export.on_error` (`fail_fast|degrade_and_record`)
- `runtime.diagnostics.bundle.enabled`
- `runtime.diagnostics.bundle.output_dir`
- `runtime.diagnostics.bundle.max_size_mb`
- `runtime.diagnostics.bundle.include_sections`

Invalid enum values, malformed path inputs, or unsupported profile combinations MUST fail fast at startup and hot reload. Hot-reload activation failure MUST keep previous active snapshot unchanged.

#### Scenario: Runtime resolves export and bundle config precedence
- **WHEN** export and bundle fields are provided by both YAML and environment variables
- **THEN** runtime resolves effective configuration deterministically by `env > file > default`

#### Scenario: Hot reload receives invalid export profile
- **WHEN** hot reload sets unsupported `runtime.observability.export.profile`
- **THEN** runtime rejects the update and keeps previous active snapshot unchanged

### Requirement: Runtime diagnostics SHALL expose additive export and bundle observability fields
Runtime diagnostics MUST expose additive export and bundle observability fields while preserving compatibility contract `additive + nullable + default`.

Minimum required fields:
- `observability_export_profile`
- `observability_export_status`
- `observability_export_error_total`
- `observability_export_drop_total`
- `observability_export_queue_depth_peak`
- `diagnostics_bundle_total`
- `diagnostics_bundle_last_status`
- `diagnostics_bundle_last_reason_code`
- `diagnostics_bundle_last_schema_version`

All fields MUST remain bounded-cardinality and replay-idempotent.

#### Scenario: Consumer queries diagnostics after export and bundle operations
- **WHEN** run executes with export enabled and bundle generation triggered
- **THEN** diagnostics include additive export and bundle fields with deterministic semantics

#### Scenario: Consumer queries diagnostics with export disabled
- **WHEN** effective configuration keeps export and bundle disabled
- **THEN** diagnostics preserve schema compatibility with nullable or default export and bundle fields
