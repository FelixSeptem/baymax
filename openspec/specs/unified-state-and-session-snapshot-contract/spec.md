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
Restore flow MUST support `strict|compatible` policy modes and MUST remain idempotent across repeated imports of the same snapshot.

#### Scenario: Strict restore blocks incompatible payload
- **WHEN** restore mode is strict and payload contains incompatible schema/segment versions
- **THEN** restore MUST stop before state mutation and return canonical conflict code

#### Scenario: Compatible restore records downgrade action
- **WHEN** restore mode is compatible and payload is within configured compatibility window
- **THEN** restore MAY continue with bounded downgrade action and MUST record deterministic restore action metadata

#### Scenario: Repeated import is idempotent
- **WHEN** the same snapshot payload is imported multiple times with same operation identity
- **THEN** resulting runtime state and diagnostics aggregates MUST remain stable without inflation

