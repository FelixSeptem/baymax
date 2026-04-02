## ADDED Requirements

### Requirement: Builtin filesystem engine SHALL provide v2 index update and drift-detection governance
Builtin filesystem memory engine MUST provide deterministic index update governance for:
- incremental updates on file-level mutations,
- full rebuild trigger on profile/model/index schema compatibility changes,
- checksum-based drift detection between snapshot, WAL tail, and index artifacts.

When drift is detected, runtime MUST emit canonical drift classification and execute configured recovery path without partial state exposure.

#### Scenario: Incremental index update on append-only write
- **WHEN** new records are appended under stable profile and schema
- **THEN** engine updates index incrementally and keeps query visibility deterministic

#### Scenario: Drift detected between snapshot and index
- **WHEN** engine startup detects checksum mismatch between snapshot and index artifacts
- **THEN** engine classifies `recovery_consistency_drift` and performs deterministic rebuild policy before serving queries

### Requirement: Memory fallback behavior SHALL preserve parity across external SPI and builtin filesystem paths
For equivalent requests and effective configuration, fallback outcomes between external SPI and builtin filesystem MUST remain semantically equivalent, including canonical reason mapping and mode metadata.

#### Scenario: External SPI recoverable failure degrades to builtin
- **WHEN** external SPI query fails with recoverable classification and fallback policy is `degrade_to_builtin`
- **THEN** runtime reroutes to builtin filesystem and preserves canonical fallback reason metadata

#### Scenario: Equivalent failure path under Run and Stream
- **WHEN** Run and Stream encounter equivalent external failure and fallback policy
- **THEN** both modes produce semantically equivalent fallback outcome and reason taxonomy
