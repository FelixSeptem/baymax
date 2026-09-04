## MODIFIED Requirements

### Requirement: Runtime SHALL expose artifact and checkpoint lineage as references

Runtime MUST expose `ArtifactRef` and `CheckpointRef` as reference-only protocol objects. An Artifact reference MUST retain `id`, `type`, `locator`, optional digest, and producing Run/Step correlation when known. A Checkpoint reference MUST retain identifier, schema version, source component, optional Run/Session correlation, and integrity reference, and MAY include additive parent, branch, history, restore-source, replay, and workspace provenance references. When a session-history projection is available, the checkpoint reference MUST preserve the associated history leaf, position, and branch/fork lineage without embedding message bodies or creating a history store. Existing snapshot manifest segment ownership and restore semantics MUST remain unchanged.

#### Scenario: Snapshot checkpoint exposes optional lineage
- **WHEN** a valid snapshot manifest is projected with root, derived, branch, replay, and session-history context
- **THEN** the checkpoint reference preserves manifest fields and validates optional lineage without changing the manifest

#### Scenario: Missing lineage parent is rejected
- **WHEN** a derived or branch checkpoint lacks a valid parent reference or history leaf
- **THEN** protocol validation returns deterministic lineage classification and does not mutate source state

#### Scenario: Existing reference remains backward compatible
- **WHEN** an existing `agent_runtime_protocol.v1` mapping contains only the original checkpoint fields
- **THEN** validation and replay succeed with all new history and branch fields omitted or defaulted

#### Scenario: Checkpoint/history association mismatch is rejected
- **WHEN** a checkpoint references a history leaf from another session or incompatible Run lineage
- **THEN** validation fails with `session.checkpoint_association_mismatch` before source restore mutation
