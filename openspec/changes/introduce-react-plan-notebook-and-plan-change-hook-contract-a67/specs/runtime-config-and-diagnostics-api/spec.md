## ADDED Requirements

### Requirement: Runtime Config SHALL Expose A67 Plan Notebook Controls
Runtime configuration SHALL expose `runtime.react.plan_notebook.*` and `runtime.react.plan_change_hook.*` with precedence `env > file > default`.

At minimum, controls MUST include:
- `runtime.react.plan_notebook.enabled`
- `runtime.react.plan_notebook.max_history`
- `runtime.react.plan_notebook.on_recover_conflict`
- `runtime.react.plan_change_hook.enabled`
- `runtime.react.plan_change_hook.fail_mode`
- `runtime.react.plan_change_hook.timeout_ms`

Invalid enums, malformed bounds, or incompatible combination values MUST fail fast at startup and rollback atomically on hot reload.

#### Scenario: Env precedence over file for notebook controls
- **WHEN** notebook key is set by both env and file
- **THEN** effective value MUST resolve from env source

#### Scenario: Invalid notebook config fails startup
- **WHEN** startup config contains invalid `on_recover_conflict` value or invalid `max_history`
- **THEN** runtime initialization MUST fail fast with deterministic validation error

#### Scenario: Invalid hook config rolls back on hot reload
- **WHEN** hot reload payload includes invalid `runtime.react.plan_change_hook.*` controls
- **THEN** runtime MUST preserve previous valid config snapshot and record reload failure

### Requirement: Runtime Diagnostics SHALL Expose A67 Additive Plan Fields
Run diagnostics MUST expose additive A67 fields while preserving compatibility contract `additive + nullable + default`.

Minimum required fields:
- `react_plan_id`
- `react_plan_version`
- `react_plan_change_total`
- `react_plan_last_action`
- `react_plan_change_reason`
- `react_plan_recover_count`
- `react_plan_hook_status`

All A67 fields MUST be emitted through `RuntimeRecorder` single-writer path and preserve replay-idempotent aggregate semantics.

#### Scenario: Consumer queries diagnostics for plan-enabled run
- **WHEN** run executes notebook actions and plan-change hooks
- **THEN** diagnostics include canonical A67 additive fields with deterministic semantics

#### Scenario: Consumer queries diagnostics for plan-disabled run
- **WHEN** run executes with plan notebook disabled
- **THEN** diagnostics remain schema-compatible with nullable/default A67 fields
