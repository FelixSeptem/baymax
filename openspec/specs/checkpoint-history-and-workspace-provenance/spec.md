# checkpoint-history-and-workspace-provenance Specification

## Purpose
TBD - created by archiving change extend-agent-runtime-protocol-with-checkpoint-history-and-workspace-provenance. Update Purpose after archive.
## Requirements
### Requirement: Checkpoint history SHALL be a validated reference-only projection

The runtime SHALL expose checkpoint relation, parent, branch, history index, restore source, replay identity, and optional session-history leaf references as additive optional references. The projection MUST validate root/derived/branch/replay relationships, ordered history continuity, session association, schema compatibility, and replay idempotency without mutating snapshot state or creating a history store.

#### Scenario: Root checkpoint is projected without a parent
- **WHEN** a valid snapshot manifest is projected as a root checkpoint without session-history context
- **THEN** the result retains manifest schema/source/digest and omits parent, branch, and history references

#### Scenario: Derived checkpoint requires an existing parent
- **WHEN** a derived, branch, or replay checkpoint omits its parent or references a missing history entry or session leaf
- **THEN** validation fails with deterministic lineage classification and snapshot state is unchanged

#### Scenario: Conflicting replay key fails deterministically
- **WHEN** the same replay key is projected with different normalized checkpoint or history data
- **THEN** validation fails with `checkpoint.replay_conflict` and does not create a second checkpoint

#### Scenario: History/checkpoint session mismatch fails
- **WHEN** a checkpoint and history leaf identify different sessions or incompatible producing Runs
- **THEN** validation fails with `session.checkpoint_association_mismatch` before restore mutation

### Requirement: Workspace provenance SHALL expose bounded integrity references

The runtime SHALL expose workspace and change-set identity, before/after integrity references, producing Run/Step correlation, and optional checkpoint correlation without embedding workspace contents or introducing a workspace store, ACL, or policy owner.

#### Scenario: Workspace change-set is associated with a checkpoint
- **WHEN** a source workspace owner supplies valid change-set and integrity references for a producing Run/Step
- **THEN** the protocol projection preserves those references and the checkpoint association

#### Scenario: Workspace integrity drift is rejected
- **WHEN** observed pre-restore workspace integrity differs from the provenance `before_integrity`
- **THEN** projection returns `workspace.integrity_drift` and performs no workspace mutation or rollback

#### Scenario: Missing required provenance association fails fast
- **WHEN** a provenance record has a change-set or integrity reference but no required workspace identity or producing correlation
- **THEN** validation returns `workspace.provenance_missing` or `workspace.association_mismatch`

### Requirement: Checkpoint and workspace provenance SHALL preserve Run/Stream and restore parity

Equivalent Run and Stream inputs MUST use the same provenance normalization and produce equivalent lineage, restore-source, replay, and workspace drift classifications. Strict and compatible restore behavior MUST remain owned by the snapshot/composer source paths.

#### Scenario: Run and Stream normalize equivalent provenance
- **WHEN** equivalent Run and Stream recoveries use the same manifest, history, workspace provenance, and restore mode
- **THEN** normalized protocol references and classifications are semantically equivalent

#### Scenario: Strict restore blocks incompatible provenance
- **WHEN** strict restore encounters schema incompatibility, disconnected lineage, or workspace drift
- **THEN** validation fails before source restore mutation with a deterministic classification

#### Scenario: Compatible restore preserves bounded downgrade
- **WHEN** compatible restore accepts a within-window schema and valid lineage
- **THEN** source restore proceeds under existing bounded compatibility behavior and the projection records the restore source without inventing lineage

