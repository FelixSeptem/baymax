## ADDED Requirements

### Requirement: Composer SHALL expose runtime readiness passthrough for managed runtime path
When composer uses managed runtime components, it MUST expose a library-level readiness passthrough entrypoint that returns runtime readiness summary without mutating scheduling or run state.

Readiness passthrough MUST preserve runtime result semantics (`ready|degraded|blocked`) and MUST NOT invent composer-local status taxonomy.

#### Scenario: Host queries composer readiness on managed runtime
- **WHEN** application calls composer readiness API and composer uses managed runtime manager
- **THEN** returned readiness status and findings are semantically equivalent to runtime readiness preflight result

#### Scenario: Readiness query does not mutate orchestration state
- **WHEN** application queries composer readiness while scheduler has queued tasks
- **THEN** query path is read-only and does not mutate task lifecycle state

### Requirement: Composer readiness semantics SHALL remain mode-independent
For equivalent effective configuration and dependency states, readiness result exposed by composer MUST remain semantically equivalent regardless of Run or Stream usage path.

#### Scenario: Equivalent config used by Run and Stream entrypoints
- **WHEN** host queries composer readiness before equivalent Run and Stream calls
- **THEN** readiness status and finding classifications remain semantically equivalent
