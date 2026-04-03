## ADDED Requirements

### Requirement: Snapshot Config Governance
Runtime configuration SHALL include `runtime.state.snapshot.*` and `runtime.session.state.*` with `env > file > default` precedence and fail-fast validation for invalid values.

#### Scenario: Env precedence over file
- **WHEN** snapshot config key is set by both env and file
- **THEN** effective value MUST resolve from env source

#### Scenario: Invalid snapshot config fails startup
- **WHEN** snapshot mode, compatibility window, or restore policy is invalid
- **THEN** runtime startup MUST fail fast with deterministic validation error

#### Scenario: Invalid hot reload rolls back atomically
- **WHEN** hot reload receives invalid snapshot/session config
- **THEN** runtime MUST preserve previous valid config snapshot and record reload failure

### Requirement: Snapshot Diagnostics Additive Fields
Diagnostics outputs MUST expose additive snapshot restore fields without breaking existing parser compatibility.

#### Scenario: QueryRuns additive compatibility
- **WHEN** snapshot restore metadata is present in run diagnostics
- **THEN** existing consumers MUST continue parsing legacy fields unchanged while new fields remain optional

#### Scenario: Canonical restore metadata projection
- **WHEN** a restore operation succeeds or fails
- **THEN** diagnostics MUST include deterministic `state_snapshot_version`, `state_restore_action`, `state_restore_conflict_code`, and `state_restore_source`
