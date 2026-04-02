## ADDED Requirements

### Requirement: State Session Snapshot Replay Fixture Support
Diagnostics replay tooling MUST support `state_session_snapshot.v1` fixture schema with deterministic normalization and mixed-version compatibility.

#### Scenario: Replay parses v1 fixture deterministically
- **WHEN** replay executes against valid `state_session_snapshot.v1` fixture input
- **THEN** normalized output MUST be deterministic across repeated executions

#### Scenario: Mixed fixture compatibility
- **WHEN** replay executes with historical fixtures and `state_session_snapshot.v1` together
- **THEN** parser MUST preserve backward compatibility and reject only true schema violations

### Requirement: Snapshot Drift Classification
Replay tooling MUST classify snapshot drifts using canonical taxonomy for schema, compatibility, restore semantics, and partial restore behavior.

#### Scenario: Schema drift classification
- **WHEN** required snapshot manifest fields drift from expected schema
- **THEN** replay MUST classify failure as `snapshot_schema_drift`

#### Scenario: Restore semantic drift classification
- **WHEN** restore action/conflict outcome differs under equivalent fixture input
- **THEN** replay MUST classify failure as `state_restore_semantic_drift`

#### Scenario: Compatibility window drift classification
- **WHEN** compatible/strict acceptance behavior differs for same version inputs
- **THEN** replay MUST classify failure as `snapshot_compat_window_drift`
