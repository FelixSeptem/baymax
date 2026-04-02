## ADDED Requirements

### Requirement: Runtime config SHALL expose OTel tracing and eval controls with deterministic precedence
Runtime configuration MUST expose tracing and eval controls with precedence `env > file > default`.

Minimum tracing controls:
- `runtime.observability.tracing.otel.*`

Minimum eval controls:
- `runtime.eval.agent.*`
- `runtime.eval.execution.*`

`runtime.eval.execution.mode` MUST support `local|distributed`.
`runtime.eval.execution.*` MUST include at minimum controls for:
- shard
- retry
- resume
- aggregation

Invalid enum values, malformed endpoint/protocol fields, invalid sampling or timeout bounds, and invalid eval execution controls MUST fail fast at startup and rollback atomically on hot reload.

#### Scenario: Startup resolves tracing and eval config precedence
- **WHEN** tracing and eval fields are provided by both environment and YAML
- **THEN** effective config is resolved deterministically by `env > file > default`

#### Scenario: Hot reload receives invalid eval execution mode
- **WHEN** hot reload sets unsupported `runtime.eval.execution.mode`
- **THEN** runtime rejects update and preserves previous active config snapshot

#### Scenario: Hot reload receives invalid eval execution shard or aggregation controls
- **WHEN** hot reload sets malformed shard, retry, resume, or aggregation controls under `runtime.eval.execution.*`
- **THEN** runtime rejects update and preserves previous active config snapshot

### Requirement: Runtime diagnostics SHALL expose additive tracing and eval fields
Run diagnostics MUST expose additive tracing and eval fields while preserving compatibility contract `additive + nullable + default`.

Minimum tracing additive fields:
- `trace_export_status`
- `trace_schema_version`

Minimum eval additive fields:
- `eval_suite_id`
- `eval_summary`
- `eval_execution_mode`
- `eval_job_id`
- `eval_shard_total`
- `eval_resume_count`

All fields MUST be emitted through `RuntimeRecorder` single-writer path and MUST preserve bounded-cardinality behavior.

#### Scenario: Consumer inspects run with tracing and eval enabled
- **WHEN** runtime executes with OTel tracing and eval enabled
- **THEN** diagnostics include canonical tracing and eval additive fields with deterministic semantics

#### Scenario: Consumer inspects run with tracing or eval disabled
- **WHEN** effective configuration disables tracing or eval paths
- **THEN** diagnostics remain schema-compatible using nullable/default values for corresponding additive fields
