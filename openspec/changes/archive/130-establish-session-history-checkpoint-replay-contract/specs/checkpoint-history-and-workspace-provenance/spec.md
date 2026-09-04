## MODIFIED Requirements

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
