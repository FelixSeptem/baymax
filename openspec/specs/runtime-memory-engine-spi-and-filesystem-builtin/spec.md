# runtime-memory-engine-spi-and-filesystem-builtin Specification

## Purpose
TBD - created by archiving change introduce-memory-provider-spi-and-builtin-filesystem-engine-contract-a54. Update Purpose after archive.
## Requirements
### Requirement: Memory engine SPI SHALL expose canonical Query Upsert Delete contracts
Runtime memory integration MUST expose canonical SPI operations:
- `Query`
- `Upsert`
- `Delete`

Each operation MUST use normalized request and response schema with stable fields for:
- operation id,
- namespace or scope,
- record identifiers,
- timestamps,
- canonical reason codes on failure.

SPI errors MUST be mapped into canonical memory taxonomy and MUST NOT leak provider-specific raw error shapes to upper layers.

#### Scenario: Query operation succeeds through canonical SPI
- **WHEN** caller invokes `Query` with valid scope and filter conditions
- **THEN** SPI returns normalized result structure with deterministic metadata fields

#### Scenario: Provider returns non-canonical error payload
- **WHEN** underlying provider emits vendor-specific error structure
- **THEN** SPI returns canonical memory reason code and preserves raw detail only as bounded optional metadata

### Requirement: Memory runtime mode SHALL support external SPI and builtin filesystem with atomic switching
Runtime MUST support memory mode enum:
- `external_spi`
- `builtin_filesystem`

Mode switching at startup and hot reload MUST be validated fail-fast. Hot reload mode changes MUST be atomic and MUST rollback to previous valid snapshot on validation or activation failure.

#### Scenario: Hot reload switches mode with valid target backend
- **WHEN** active runtime applies valid memory mode change from `builtin_filesystem` to `external_spi`
- **THEN** runtime atomically activates new mode and publishes deterministic switch diagnostics

#### Scenario: Hot reload mode switch fails validation
- **WHEN** runtime receives invalid mode target configuration during hot reload
- **THEN** runtime rejects update and preserves previous active mode snapshot without partial activation

### Requirement: Builtin filesystem memory engine SHALL provide WAL compaction and crash-safe recovery
Builtin filesystem memory engine MUST implement:
- append-only write-ahead log,
- deterministic index snapshot,
- atomic compaction and snapshot replacement.

Crash recovery MUST reconstruct a consistent readable state from latest valid snapshot plus WAL tail without logical data corruption.

#### Scenario: Process crash occurs during compaction
- **WHEN** crash happens after new compacted artifact is generated but before full swap completion
- **THEN** next startup recovers using last valid atomic snapshot and WAL replay with deterministic state

#### Scenario: Concurrent read and write operations execute on builtin engine
- **WHEN** concurrent `Upsert` and `Query` operations run under configured concurrency
- **THEN** engine preserves deterministic visibility guarantees and does not return partially written record state

### Requirement: Memory provider profile pack SHALL include mainstream adapters and generic extension slot
The canonical memory profile pack for this milestone MUST include:
- `mem0`
- `zep`
- `openviking`
- `generic`

Each profile MUST declare required and optional operation capabilities plus canonical error-mapping behavior.

#### Scenario: Integrator selects canonical mem0 profile
- **WHEN** runtime config references `mem0` memory profile id
- **THEN** runtime resolves deterministic provider mapping and capability defaults from profile pack

#### Scenario: Runtime receives unknown profile id
- **WHEN** runtime config references unsupported memory profile id
- **THEN** activation fails fast with deterministic profile-unknown classification

### Requirement: External memory failure fallback SHALL be policy-driven and observable
Runtime MUST support fallback policy values:
- `fail_fast`
- `degrade_to_builtin`
- `degrade_without_memory`

Fallback policy execution MUST emit canonical reason codes and MUST preserve deterministic behavior under equivalent inputs.

#### Scenario: External provider fails and policy is degrade to builtin
- **WHEN** `external_spi` operation fails with recoverable error and fallback policy is `degrade_to_builtin`
- **THEN** runtime reroutes operation to builtin filesystem engine and records fallback-used diagnostics

#### Scenario: External provider fails and policy is fail fast
- **WHEN** `external_spi` operation fails and fallback policy is `fail_fast`
- **THEN** runtime returns blocking error with canonical reason code and does not perform implicit backend switch

### Requirement: Run and Stream memory behavior SHALL remain semantically equivalent
For equivalent input, effective config, and memory backend state, Run and Stream paths MUST produce semantically equivalent memory operation outcomes, allowing non-semantic event ordering differences.

#### Scenario: Equivalent operation sequence under Run and Stream
- **WHEN** Run and Stream execute equivalent memory `Query/Upsert/Delete` sequence with same effective mode and profile
- **THEN** both paths produce semantically equivalent operation result and fallback classification

#### Scenario: Equivalent failure path under Run and Stream
- **WHEN** Run and Stream encounter equivalent provider failure and fallback policy
- **THEN** both paths produce semantically equivalent canonical reason taxonomy and mode outcome

