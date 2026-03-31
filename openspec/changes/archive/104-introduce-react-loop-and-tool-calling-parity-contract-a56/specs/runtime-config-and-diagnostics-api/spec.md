## ADDED Requirements

### Requirement: Runtime config SHALL expose ReAct loop controls with deterministic precedence
Runtime configuration MUST expose `runtime.react.*` controls with precedence `env > file > default`.

Minimum required controls for this milestone:
- `runtime.react.enabled`
- `runtime.react.max_iterations`
- `runtime.react.tool_call_limit`
- `runtime.react.stream_tool_dispatch_enabled`
- `runtime.react.on_budget_exhausted`

#### Scenario: Runtime starts with default ReAct configuration
- **WHEN** no explicit ReAct configuration override is provided
- **THEN** runtime resolves deterministic default values for `runtime.react.*` controls

#### Scenario: Environment variable overrides file-level ReAct config
- **WHEN** both YAML file and environment variables define `runtime.react.max_iterations`
- **THEN** effective configuration uses environment value according to `env > file > default`

### Requirement: Runtime SHALL validate ReAct configuration at startup and hot reload with fail-fast rollback
ReAct config fields MUST be validated at startup and during hot reload. Invalid values MUST fail fast and hot reload MUST rollback atomically to previous valid snapshot.

At minimum, validation MUST cover:
- positive bounds for `max_iterations` and `tool_call_limit`,
- enum validity for `on_budget_exhausted`,
- compatibility of `stream_tool_dispatch_enabled` with effective runtime mode.

#### Scenario: Startup with invalid tool-call limit
- **WHEN** `runtime.react.tool_call_limit` is non-positive
- **THEN** runtime initialization fails fast with validation error

#### Scenario: Hot reload introduces invalid ReAct enum
- **WHEN** hot reload payload sets unsupported `runtime.react.on_budget_exhausted`
- **THEN** runtime rejects update and keeps previous active snapshot unchanged

### Requirement: Runtime diagnostics SHALL expose additive ReAct loop fields with compatibility guarantees
Run diagnostics MUST expose ReAct additive fields and preserve compatibility contract `additive + nullable + default`.

Minimum required fields for this milestone:
- `react_enabled`
- `react_iteration_total`
- `react_tool_call_total`
- `react_tool_call_budget_hit_total`
- `react_iteration_budget_hit_total`
- `react_termination_reason`
- `react_stream_dispatch_enabled`

ReAct diagnostics fields MUST remain bounded-cardinality and replay-idempotent.

#### Scenario: Consumer queries diagnostics for ReAct-enabled run
- **WHEN** run executes one or more ReAct tool-loop iterations
- **THEN** diagnostics include canonical ReAct loop counters and terminal reason fields

#### Scenario: Consumer queries diagnostics for non-ReAct run
- **WHEN** run executes with ReAct disabled
- **THEN** diagnostics preserve schema compatibility through nullable or default ReAct additive fields

### Requirement: ReAct observability events SHALL flow through RuntimeRecorder single-writer path
ReAct loop counters, budget-hit markers, and termination classifications MUST be emitted through `RuntimeRecorder` single-writer path and MUST preserve idempotent aggregate semantics.

#### Scenario: Equivalent ReAct terminal event is replayed
- **WHEN** duplicate equivalent ReAct terminal events are ingested for the same run
- **THEN** logical aggregate counters remain stable after first ingestion

#### Scenario: Run and Stream emit equivalent ReAct completion events
- **WHEN** equivalent Run and Stream flows complete with the same ReAct outcome
- **THEN** recorder persists semantically equivalent ReAct aggregate semantics
