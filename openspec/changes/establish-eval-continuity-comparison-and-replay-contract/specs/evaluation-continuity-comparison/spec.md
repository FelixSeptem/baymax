## Purpose

Provides a bounded, reference-only contract for proving that source-owned runtime facts remain continuous across evaluation, compaction, handoff, snapshot restore, and recovery boundaries without persisting transcript or artifact bodies.

## ADDED Requirements

### Requirement: Continuity projections SHALL be versioned, bounded, and reference-only

The evaluation layer MUST accept baseline and candidate continuity projections with a supported schema version, run/session correlation, explicit phase, and a bounded list of fixed-axis references. Each reference MUST identify its semantic kind, source owner, stable identifier, and optional digest/version. Projections MUST NOT contain transcript text, reasoning bodies, artifact bodies, provider responses, tool output bodies, or unbounded maps.

#### Scenario: Valid projection is accepted
- **WHEN** a baseline or candidate projection contains supported version, run correlation, phase, and unique bounded references
- **THEN** normalization succeeds deterministically and produces a stable projection digest

#### Scenario: Body-bearing projection is rejected
- **WHEN** a projection includes raw transcript, reasoning, artifact, provider, or tool body content
- **THEN** validation fails fast with `continuity_privacy_violation` and no comparison result is emitted

#### Scenario: Unsupported schema is rejected
- **WHEN** a projection declares an unsupported major version or omits required identity
- **THEN** validation fails with deterministic `continuity_schema_drift`

### Requirement: Continuity comparison SHALL cover fixed source-owned axes

Comparison MUST normalize and compare identity (agent, role, team), objective, task association, attempt/lease, workspace binding, pending correlated request, checkpoint, and artifact references, including owner, identifier, version, and digest where present. The comparator MUST NOT infer facts from summaries or create a parallel source of truth.

#### Scenario: Equivalent handoff and snapshot projections compare equal
- **WHEN** baseline and candidate projections contain semantically equivalent normalized references across all required axes
- **THEN** comparison returns a deterministic pass with no continuity drift

#### Scenario: Objective or task association changes
- **WHEN** candidate objective or task association differs from baseline
- **THEN** comparison returns `continuity_objective_drift` or `continuity_task_association_drift` with the affected axis

#### Scenario: Attempt, lease, or workspace changes
- **WHEN** candidate attempt/lease or workspace binding is stale, replaced, or associated with a different source
- **THEN** comparison returns the corresponding bounded `continuity_attempt_lease_drift` or `continuity_workspace_binding_drift` classification without mutating runtime state

### Requirement: Comparison SHALL classify reference and parity drift deterministically

The comparator MUST distinguish missing, duplicate, owner, checkpoint, artifact-reference, pending-request, Run/Stream parity, and recovery-idempotency drift. Equivalent local/distributed or Run/Stream projections MUST normalize identically; duplicate references MUST NOT be resolved by last-write-wins behavior.

#### Scenario: Missing or duplicate reference is detected
- **WHEN** a required reference is missing or the same semantic key appears with conflicting data
- **THEN** comparison returns `continuity_reference_missing` or `continuity_duplicate_reference` deterministically

#### Scenario: Run and Stream projections are equivalent
- **WHEN** equivalent Run and Stream recovery inputs produce the same normalized continuity references
- **THEN** comparison reports parity success and identical comparison identity

#### Scenario: Recovery restore is not idempotent
- **WHEN** repeated restore projections with the same operation identity produce different normalized references
- **THEN** comparison returns `continuity_recovery_idempotency_drift`

### Requirement: Continuity comparison SHALL be offline and side-effect free

Evaluation continuity comparison MUST not call providers, tools, Git, workspace mutation, artifact/transcript resolvers, or runtime restore paths. It MUST use existing handoff, snapshot, checkpoint, workspace, task/attempt, and pending-request owners only to create bounded projections and MUST leave their state unchanged.

#### Scenario: Offline replay does not require live runtime connectivity
- **WHEN** a continuity fixture is evaluated from version-controlled reference data
- **THEN** the comparator returns a deterministic result without network, provider, tool, Git, or live runtime access

#### Scenario: Failed comparison leaves sources unchanged
- **WHEN** validation or drift detection fails
- **THEN** no source owner, snapshot, handoff, scheduler, workspace, or diagnostics state is mutated

