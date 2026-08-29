## MODIFIED Requirements

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
