## ADDED Requirements

### Requirement: Runtime Config SHALL Expose A67-CTX JIT Context Controls
Runtime configuration SHALL expose `runtime.context.jit.*` controls with precedence `env > file > default`.

At minimum, controls MUST include:
- `runtime.context.jit.reference_first.enabled`
- `runtime.context.jit.reference_first.max_refs`
- `runtime.context.jit.reference_first.max_resolve_tokens`
- `runtime.context.jit.isolate_handoff.enabled`
- `runtime.context.jit.isolate_handoff.default_ttl_ms`
- `runtime.context.jit.isolate_handoff.min_confidence`
- `runtime.context.jit.edit_gate.enabled`
- `runtime.context.jit.edit_gate.clear_at_least_tokens`
- `runtime.context.jit.edit_gate.min_gain_ratio`
- `runtime.context.jit.swap_back.enabled`
- `runtime.context.jit.swap_back.min_relevance_score`
- `runtime.context.jit.lifecycle_tiering.enabled`
- `runtime.context.jit.lifecycle_tiering.hot_ttl_ms`
- `runtime.context.jit.lifecycle_tiering.warm_ttl_ms`
- `runtime.context.jit.lifecycle_tiering.cold_ttl_ms`

Invalid enums, malformed bounds, or incompatible combinations MUST fail fast at startup and rollback atomically on hot reload.

#### Scenario: Env precedence over file for JIT context controls
- **WHEN** the same `runtime.context.jit.*` key is set by both env and file
- **THEN** runtime MUST resolve effective value from env source

#### Scenario: Invalid JIT context config fails startup
- **WHEN** startup config contains invalid `min_gain_ratio` or inconsistent tier TTL ordering
- **THEN** runtime initialization MUST fail fast with deterministic validation classification

#### Scenario: Invalid hot reload payload rolls back atomically
- **WHEN** hot reload includes invalid `runtime.context.jit.*` values
- **THEN** runtime MUST preserve previous valid snapshot and record deterministic reload failure

### Requirement: Runtime Diagnostics SHALL Expose A67-CTX Additive Context Fields
Run diagnostics MUST expose additive A67-CTX fields while preserving compatibility contract `additive + nullable + default`.

Minimum required fields:
- `context_ref_discover_count`
- `context_ref_resolve_count`
- `context_edit_estimated_saved_tokens`
- `context_edit_gate_decision`
- `context_swapback_relevance_score`
- `context_lifecycle_tier_stats`
- `context_recap_source`

All A67-CTX fields MUST be emitted through `RuntimeRecorder` single-writer path and preserve replay-idempotent aggregate semantics.

#### Scenario: Consumer queries diagnostics for JIT-context-enabled run
- **WHEN** run executes reference-first discovery, edit gate, swap-back, and tiering flow
- **THEN** diagnostics MUST include canonical A67-CTX additive fields with deterministic semantics

#### Scenario: Consumer queries diagnostics for JIT-context-disabled run
- **WHEN** run executes with JIT context organization disabled
- **THEN** diagnostics MUST remain schema-compatible with nullable/default A67-CTX fields
