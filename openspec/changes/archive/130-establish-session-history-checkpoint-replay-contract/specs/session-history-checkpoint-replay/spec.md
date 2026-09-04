# session-history-checkpoint-replay Specification

## Purpose

Define the library-first boundary between session message history, run checkpoints, snapshot restore, and offline diagnostics replay.

## ADDED Requirements

### Requirement: Session history SHALL be a bounded, source-owned reference projection

The runtime SHALL expose a bounded session-history projection containing a stable session identifier, optional history root and leaf references, parent reference, history position, and producing Run/Step correlation when known. The projection MUST NOT embed unbounded message bodies, prompts, provider objects, or create a second session repository.

#### Scenario: History leaf is projected with source correlation
- **WHEN** a source-owned history contains a valid leaf associated with a Run and Step
- **THEN** the projection preserves session, leaf, parent, position, and producing correlation references without copying message content

#### Scenario: Missing optional history is backward compatible
- **WHEN** a source Runtime exposes only an existing SessionRef
- **THEN** the history projection is absent or empty and existing protocol mappings remain valid

#### Scenario: Oversized history metadata is rejected
- **WHEN** a history projection exceeds configured reference, metadata, or serialized-size limits
- **THEN** validation fails with a deterministic history schema classification and leaves the source history unchanged

### Requirement: History chains SHALL preserve continuity and immutable leaves

History references MUST validate root/parent continuity, monotonic position, and stable identifiers. A terminal history leaf and its producing Run/Checkpoint MUST remain immutable; later branches reference the leaf rather than mutating it.

#### Scenario: Continuous history chain validates
- **WHEN** each history entry references an existing parent with the expected next position
- **THEN** validation succeeds with a deterministic normalized chain digest

#### Scenario: History gap fails fast
- **WHEN** a parent is missing, position regresses, or a chain contains conflicting identifiers
- **THEN** validation fails with `session.history_gap` or `session.history_conflict` and performs no state mutation

### Requirement: Branch and fork SHALL create distinct Run lineage

A branch or fork MUST reference an existing session history leaf and optional parent Run/Checkpoint, then produce a distinct Run correlation. The source owner remains responsible for admission, scheduling, and persistence; the protocol MUST NOT mutate the original terminal Run or synthesize a merge.

#### Scenario: Fork from a valid leaf preserves lineage
- **WHEN** a host requests a fork from a valid history leaf and the source creates a new Run
- **THEN** the normalized result includes the new Run, parent leaf, parent Run/Checkpoint, and deterministic branch identity

#### Scenario: Fork parent is missing
- **WHEN** a branch references a missing leaf, parent checkpoint, or required parent Run
- **THEN** validation fails with `session.branch_parent_missing` and the original Run and history remain unchanged

#### Scenario: Terminal Run is not mutated by a fork
- **WHEN** a new branch is created from a completed, failed, or canceled Run
- **THEN** the original terminal Run remains terminal and the branch receives a separate causal Run correlation

### Requirement: History and checkpoint association SHALL be mutually consistent

When both references are present, a checkpoint MUST identify the session and history leaf/position that produced it, and a history leaf MUST NOT claim a checkpoint from another session or incompatible Run lineage. Optional references may be omitted only when the source does not own them.

#### Scenario: Matching checkpoint and history validate
- **WHEN** session, Run, history leaf, checkpoint, and schema references agree
- **THEN** validation succeeds and emits a stable normalized association digest

#### Scenario: Cross-session association is rejected
- **WHEN** a checkpoint or history leaf references a different session or incompatible parent Run
- **THEN** validation fails with `session.checkpoint_association_mismatch` before restore or branch mutation

### Requirement: Restore SHALL validate history boundaries before source mutation

Snapshot restore MUST validate history/checkpoint associations, schema compatibility, and branch lineage before mutating source state. `strict` mode MUST reject incompatible or incomplete required references. `compatible` mode MAY continue only within the existing compatibility window and MUST record a bounded downgrade outcome.

#### Scenario: Strict restore rejects broken lineage
- **WHEN** strict restore receives a checkpoint with a missing history parent or incompatible schema
- **THEN** restore fails deterministically before mutation with the corresponding lineage or compatibility classification

#### Scenario: Compatible restore records bounded downgrade
- **WHEN** compatible restore receives a within-window schema with valid required lineage
- **THEN** the source restore proceeds under existing policy and the normalized result records a downgrade classification without inventing references

### Requirement: Restore and replay operations SHALL be idempotent

Repeated restore or replay requests with the same operation identity and normalized input MUST produce the same state, lineage, and diagnostic aggregate. Reusing an operation identity with different normalized history/checkpoint data MUST fail deterministically and MUST NOT create a second restore or branch.

#### Scenario: Duplicate restore is idempotent
- **WHEN** the same snapshot, history, checkpoint, and operation identity are restored repeatedly
- **THEN** source state and diagnostics remain stable without duplicate side effects

#### Scenario: Conflicting operation identity fails
- **WHEN** an existing restore or replay identity is submitted with different normalized lineage
- **THEN** validation fails with `session.replay_conflict` and no additional mutation occurs

### Requirement: Offline replay SHALL be read-only and side-effect-free

Diagnostics replay MUST validate versioned history/checkpoint fixtures using source-owned references, normalized digests, and expected terminal outcomes. Replay MUST NOT invoke providers or tools, mutate workspace or snapshot state, create branches, or write diagnostics through a second path.

#### Scenario: Valid history fixture replays without runtime connectivity
- **WHEN** replay receives a valid session/checkpoint fixture with expected normalized output
- **THEN** replay succeeds deterministically without live runtime or provider access

#### Scenario: Side-effect attempt is rejected
- **WHEN** replay processing attempts tool execution, provider access, workspace mutation, or restore mutation
- **THEN** replay fails with `session.replay_side_effect` and produces no side effect

### Requirement: Run and Stream SHALL preserve history and replay parity

Equivalent Run and Stream inputs with the same session history, checkpoint, restore mode, branch request, and source outcome MUST normalize to semantically equivalent lineage, conflict, restore, and replay results after permitted event-order normalization.

#### Scenario: Equivalent branch requests preserve parity
- **WHEN** equivalent Run and Stream paths fork from the same history leaf
- **THEN** both paths produce equivalent parent references, branch identity semantics, and terminal outcomes

#### Scenario: Restore drift is detected across paths
- **WHEN** Run and Stream normalize different history/checkpoint associations under equivalent input
- **THEN** the contract gate reports `run_stream_history_replay_parity_drift`

### Requirement: History and replay fields SHALL preserve compatibility and observability

All new protocol, diagnostics, and fixture fields MUST be additive, nullable or defaultable, and tied to an explicit profile or fixture version. Observable history, restore, branch, and replay decisions MUST use the existing `RuntimeRecorder` single-write path with bounded cardinality.

#### Scenario: Historical consumers parse a fixture without extensions
- **WHEN** an archived protocol, snapshot, or provenance fixture omits session-history fields
- **THEN** parsing and normalized validation succeed using documented defaults

#### Scenario: History drift is recorded through the canonical writer
- **WHEN** replay or runtime validation detects a history, branch, restore, or replay drift
- **THEN** bounded additive fields are recorded through `RuntimeRecorder` and no parallel diagnostics writer is used
