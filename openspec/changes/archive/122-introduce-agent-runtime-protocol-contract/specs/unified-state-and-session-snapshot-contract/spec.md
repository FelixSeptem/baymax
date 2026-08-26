## ADDED Requirements

### Requirement: Snapshot manifests SHALL provide canonical checkpoint references
The unified state/session snapshot contract MUST provide deterministic Agent Runtime Protocol checkpoint references derived from manifest source, schema version, segment metadata, and integrity digest. This mapping MUST NOT alter manifest structure, source ownership, restore policy, or import idempotency.

#### Scenario: Export maps manifest to checkpoint reference
- **WHEN** a caller exports a valid unified state/session snapshot manifest
- **THEN** runtime can derive a checkpoint reference with source component, schema version, digest, and available Run/Session correlation
