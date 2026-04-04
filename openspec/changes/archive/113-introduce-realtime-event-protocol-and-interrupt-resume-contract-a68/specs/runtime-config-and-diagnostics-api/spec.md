## ADDED Requirements

### Requirement: Runtime Config SHALL Expose A68 Realtime Controls
Runtime configuration SHALL expose `runtime.realtime.protocol.*` and `runtime.realtime.interrupt_resume.*` with precedence `env > file > default`.

At minimum, controls MUST include:
- `runtime.realtime.protocol.enabled`
- `runtime.realtime.protocol.version`
- `runtime.realtime.protocol.max_buffered_events`
- `runtime.realtime.interrupt_resume.enabled`
- `runtime.realtime.interrupt_resume.resume_cursor_ttl_ms`
- `runtime.realtime.interrupt_resume.idempotency_window_ms`

Invalid enums, malformed bounds, or incompatible combinations MUST fail fast at startup and rollback atomically on hot reload.

#### Scenario: Env precedence over file for realtime controls
- **WHEN** realtime key is set by both env and file
- **THEN** effective value MUST resolve from env source

#### Scenario: Invalid realtime config fails startup
- **WHEN** startup config contains invalid realtime protocol version or invalid buffered-event bound
- **THEN** runtime initialization MUST fail fast with deterministic validation error

#### Scenario: Invalid interrupt/resume config rolls back on hot reload
- **WHEN** hot reload payload includes invalid `runtime.realtime.interrupt_resume.*` controls
- **THEN** runtime MUST preserve previous valid config snapshot and record reload failure

### Requirement: Runtime Diagnostics SHALL Expose A68 Additive Realtime Fields
Run diagnostics MUST expose additive A68 fields while preserving compatibility contract `additive + nullable + default`.

Minimum required fields:
- `realtime_protocol_version`
- `realtime_event_seq_max`
- `realtime_interrupt_total`
- `realtime_resume_total`
- `realtime_resume_source`
- `realtime_idempotency_dedup_total`
- `realtime_last_error_code`

All A68 fields MUST be emitted through `RuntimeRecorder` single-writer path and preserve replay-idempotent aggregate semantics.

#### Scenario: Consumer queries diagnostics for realtime-interrupted run
- **WHEN** run executes realtime interrupt/resume path
- **THEN** diagnostics include canonical A68 additive fields with deterministic semantics

#### Scenario: Consumer queries diagnostics for realtime-disabled run
- **WHEN** run executes with realtime features disabled
- **THEN** diagnostics remain schema-compatible with nullable/default A68 fields

