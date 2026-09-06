# unified-state-and-session-snapshot-contract Specification

## Purpose
TBD - created by archiving change introduce-unified-state-and-session-snapshot-contract-a66. Update Purpose after archive.
## Requirements
### Requirement: Unified Snapshot Manifest Contract
The runtime SHALL expose a unified state/session snapshot manifest with versioned schema, source metadata, module segment descriptors, and integrity checksum.

#### Scenario: Export includes canonical manifest fields
- **WHEN** a caller exports runtime state/session snapshot
- **THEN** output MUST include `schema_version`, `exported_at`, `source`, `segments`, and deterministic integrity digest

#### Scenario: Missing required manifest fields fails fast
- **WHEN** import payload is missing required manifest fields
- **THEN** import MUST fail fast with deterministic schema validation error

### Requirement: Segment-Based Snapshot Interoperability
Snapshot payload MUST preserve module segment boundaries for runner/session, scheduler/mailbox, composer recovery, and memory without rewriting underlying source-of-truth semantics.

#### Scenario: Segment passthrough preserves module semantics
- **WHEN** snapshot is exported and imported without mutation
- **THEN** each module segment MUST retain canonical semantics equivalent to module-native snapshot behavior

#### Scenario: Unsupported segment version in strict mode
- **WHEN** one segment version is outside compatibility window and restore mode is strict
- **THEN** restore MUST be rejected deterministically with compatibility mismatch reason

### Requirement: Restore Policy and Idempotency Contract

Restore flow MUST support `strict|compatible` policy modes and MUST remain idempotent across repeated imports of the same snapshot and operation identity. When session-history or checkpoint provenance is supplied, restore MUST validate those references before state mutation. `strict` MUST reject incompatible schema, broken lineage, or cross-session association; `compatible` MAY continue only within the configured compatibility window and MUST record a bounded downgrade action.

#### Scenario: Strict restore blocks incompatible payload
- **WHEN** restore mode is strict and payload contains incompatible schema/segment versions or invalid history/checkpoint lineage
- **THEN** restore MUST stop before state mutation and return canonical conflict code

#### Scenario: Compatible restore records downgrade action
- **WHEN** restore mode is compatible and payload is within the configured compatibility window with valid required lineage
- **THEN** restore MAY continue with bounded downgrade action and MUST record deterministic restore action metadata

#### Scenario: Repeated import is idempotent
- **WHEN** the same snapshot, history/checkpoint context, and operation identity are imported multiple times
- **THEN** resulting runtime state and diagnostics aggregates MUST remain stable without inflation

#### Scenario: Conflicting restore identity fails
- **WHEN** the same operation identity is reused with different normalized history or checkpoint data
- **THEN** restore MUST fail with deterministic replay conflict and MUST NOT mutate the snapshot fact source

### Requirement: Snapshot manifests SHALL provide canonical checkpoint references

The unified state/session snapshot contract MUST provide deterministic Agent Runtime Protocol checkpoint references derived from manifest source, schema version, segment metadata, and integrity digest. The projection MAY add optional checkpoint lineage, restore-source, replay identity, and workspace provenance references supplied by the source recovery context. This mapping MUST NOT alter manifest structure, source ownership, restore policy, or import idempotency.

#### Scenario: Export maps manifest and recovery context to checkpoint reference
- **WHEN** a caller exports a valid unified snapshot manifest with optional history and workspace context
- **THEN** runtime derives a checkpoint reference retaining canonical manifest fields and validated optional provenance

#### Scenario: Manifest remains the storage fact source
- **WHEN** lineage or workspace provenance is invalid during projection
- **THEN** projection fails before changing the manifest or restore store

#### Scenario: Strict and compatible restore semantics remain unchanged
- **WHEN** the same manifest is imported with strict or compatible mode before and after provenance projection
- **THEN** restore acceptance, conflict, downgrade, and idempotency outcomes remain equivalent

### Requirement: Snapshot manifests SHALL remain the handoff source of truth
Handoff records SHALL reference existing snapshot/checkpoint/session identifiers and SHALL NOT duplicate their authoritative state; restore SHALL verify manifest compatibility before state mutation.

#### Scenario: Compatible manifest
- **WHEN** a handoff references a valid snapshot manifest and checkpoint
- **THEN** restore resolves the existing sources and preserves their identifiers

#### Scenario: Incompatible manifest
- **WHEN** the referenced manifest version or boundary is incompatible
- **THEN** restore fails deterministically before mutating source state

