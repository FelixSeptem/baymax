## ADDED Requirements

### Requirement: Runtime config SHALL expose memory scope and write/search/lifecycle governance fields
Runtime configuration MUST expose memory governance fields with precedence `env > file > default`:
- `runtime.memory.scope.*`
- `runtime.memory.write_mode.*`
- `runtime.memory.injection_budget.*`
- `runtime.memory.lifecycle.*`
- `runtime.memory.search.*`

Runtime MUST preserve existing `runtime.memory.mode=external_spi|builtin_filesystem` backend semantics while validating new fields independently.

#### Scenario: Startup resolves effective memory governance config
- **WHEN** file and env both provide memory governance fields
- **THEN** runtime resolves effective values with `env > file > default` and keeps backend selector unchanged

#### Scenario: Invalid memory governance config during hot reload
- **WHEN** hot reload payload includes invalid scope or write/search/lifecycle values
- **THEN** runtime rejects update and rolls back atomically to previous active snapshot

### Requirement: Runtime diagnostics SHALL expose additive memory governance fields
Run diagnostics MUST expose additive memory fields:
- `memory_scope_selected`
- `memory_budget_used`
- `memory_hits`
- `memory_rerank_stats`
- `memory_lifecycle_action`

These fields MUST remain compatible under `additive + nullable + default` contract and MUST be written through `RuntimeRecorder` single-writer path.

#### Scenario: Consumer inspects memory-enriched run diagnostics
- **WHEN** run executes memory retrieval and lifecycle actions
- **THEN** diagnostics include the canonical memory additive fields with deterministic semantics

#### Scenario: Run has no memory lifecycle action
- **WHEN** run does not trigger retention/ttl/forget path
- **THEN** diagnostics remain compatible with nullable/default memory lifecycle fields
