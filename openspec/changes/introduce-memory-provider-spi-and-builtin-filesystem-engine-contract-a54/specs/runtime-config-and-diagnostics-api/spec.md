## ADDED Requirements

### Requirement: Runtime config SHALL expose memory mode provider and fallback controls with deterministic precedence
Runtime configuration MUST expose `runtime.memory.*` controls with precedence `env > file > default`.

Minimum required fields for this milestone:
- `runtime.memory.mode` (`external_spi|builtin_filesystem`)
- `runtime.memory.external.provider`
- `runtime.memory.external.profile`
- `runtime.memory.external.contract_version`
- `runtime.memory.builtin.root_dir`
- `runtime.memory.builtin.compaction`
- `runtime.memory.fallback.policy` (`fail_fast|degrade_to_builtin|degrade_without_memory`)

Invalid enum values, malformed paths, unsupported contract version, or illegal mode-provider combinations MUST fail fast at startup and hot reload.

#### Scenario: Runtime starts with valid memory config from env and file
- **WHEN** memory settings are provided by both YAML and environment variables
- **THEN** effective values resolve deterministically by `env > file > default`

#### Scenario: Hot reload receives invalid fallback policy
- **WHEN** hot reload sets unsupported `runtime.memory.fallback.policy`
- **THEN** runtime rejects update and keeps previous active config snapshot unchanged

### Requirement: Runtime diagnostics SHALL include additive memory observability fields
Runtime diagnostics MUST expose additive memory fields while preserving compatibility contract `additive + nullable + default`.

Minimum required fields:
- `memory_mode`
- `memory_provider`
- `memory_profile`
- `memory_contract_version`
- `memory_query_total`
- `memory_upsert_total`
- `memory_delete_total`
- `memory_error_total`
- `memory_fallback_total`
- `memory_fallback_reason_code`
- `memory_latency_ms_p95`

Memory diagnostics fields MUST remain bounded-cardinality and replay-idempotent.

#### Scenario: Consumer queries diagnostics for memory-enabled run
- **WHEN** run executes memory operations
- **THEN** diagnostics include additive memory fields with deterministic semantics

#### Scenario: Consumer queries diagnostics for run without memory activity
- **WHEN** run does not execute memory operations
- **THEN** diagnostics preserve schema compatibility with nullable or default memory fields

### Requirement: Memory mode switching diagnostics SHALL be traceable through single-writer recorder
Memory mode switches, fallback executions, and memory error classifications MUST be emitted through the runtime single-writer recorder path with deterministic event semantics.

#### Scenario: Mode switch succeeds during hot reload
- **WHEN** runtime activates valid memory mode switch
- **THEN** recorder stores switch event with previous and effective mode metadata

#### Scenario: Mode switch fails and rolls back
- **WHEN** memory mode activation fails after validation pass
- **THEN** recorder stores rollback event with canonical failure reason and active mode remains unchanged
