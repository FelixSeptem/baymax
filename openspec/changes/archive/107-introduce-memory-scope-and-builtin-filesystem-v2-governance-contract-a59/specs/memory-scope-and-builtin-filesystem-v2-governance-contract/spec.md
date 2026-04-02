## ADDED Requirements

### Requirement: Runtime SHALL resolve memory scope deterministically with bounded injection budget
Runtime MUST resolve memory retrieval scope using canonical order and explicit override semantics:
- canonical order: `session -> project -> global`
- explicit override MAY narrow scope but MUST NOT bypass allowlist or namespace validation
- resolved scope MUST be recorded in diagnostics as `memory_scope_selected`

Runtime MUST enforce injection budget before context assembly and MUST expose consumed budget as `memory_budget_used`.

#### Scenario: Scope is not explicitly provided
- **WHEN** request omits scope override and all three scopes have available records
- **THEN** runtime resolves scope with canonical order and records deterministic `memory_scope_selected`

#### Scenario: Injection budget is exceeded
- **WHEN** retrieved candidates exceed configured injection budget
- **THEN** runtime truncates deterministically by configured policy and records bounded `memory_budget_used`

### Requirement: Runtime SHALL separate backend mode from write mode policy
Runtime MUST keep backend selection and write strategy as separate dimensions:
- backend selector: `runtime.memory.mode=external_spi|builtin_filesystem`
- write strategy: `runtime.memory.write_mode=automatic|agentic`

Invalid combinations MUST fail fast at startup and rollback atomically on hot reload.

#### Scenario: Valid backend and write mode combination
- **WHEN** runtime loads `runtime.memory.mode=external_spi` and `runtime.memory.write_mode=automatic`
- **THEN** memory writes follow automatic policy without changing backend selection semantics

#### Scenario: Unsupported write mode during hot reload
- **WHEN** hot reload sets unsupported `runtime.memory.write_mode`
- **THEN** runtime rejects update and preserves previous active snapshot

### Requirement: Runtime SHALL govern memory search pipeline with deterministic quality controls
Runtime MUST support search governance fields under `runtime.memory.search.*` including:
- hybrid retrieval controls (keyword/vector weighting)
- rerank controls
- temporal decay controls
- index update policy

Search pipeline outputs MUST expose additive diagnostics fields `memory_hits` and `memory_rerank_stats`.

#### Scenario: Hybrid retrieval with rerank is enabled
- **WHEN** runtime executes query with hybrid retrieval and rerank enabled
- **THEN** output includes deterministic top-k ordering and records `memory_hits` with `memory_rerank_stats`

#### Scenario: Search config is malformed
- **WHEN** runtime receives invalid hybrid/rerank/temporal configuration
- **THEN** startup or hot reload fails fast with deterministic validation classification

### Requirement: Runtime SHALL enforce memory lifecycle governance
Runtime MUST expose lifecycle policies under `runtime.memory.lifecycle.*` for at least:
- retention
- ttl
- forget

Lifecycle actions MUST be observable via additive diagnostics field `memory_lifecycle_action`.

#### Scenario: TTL expiration triggers lifecycle action
- **WHEN** records exceed configured TTL window
- **THEN** runtime applies configured lifecycle policy and records canonical `memory_lifecycle_action`

#### Scenario: Forget policy receives invalid target
- **WHEN** forget operation references malformed or unsupported target scope
- **THEN** runtime fails fast with deterministic lifecycle validation error
