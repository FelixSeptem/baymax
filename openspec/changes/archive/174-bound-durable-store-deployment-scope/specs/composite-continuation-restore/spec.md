## ADDED Requirements

### Requirement: Composite graph restore SHALL inherit source store scope

Child, team, workflow, and mailbox graph checkpoints MUST carry the deployment
profile, writer owner, fence, and claim transaction scope of every source store
that owns graph state. Composite restore MUST be rejected when any required
member store is only local-single-writer but the request claims concurrent,
cross-machine, or multi-instance recovery.

#### Scenario: Mixed graph ownership fails closed
- **WHEN** a composite checkpoint binds a local child store and a requested
  distributed team store without distributed evidence
- **THEN** admission returns a typed unsupported deployment result before any
  member or step is resumed
- **AND** it does not resume a partial graph

#### Scenario: Local graph reopen retains scope
- **WHEN** a new process reopens a fully persisted graph under the same writer
  ownership
- **THEN** it may restore the graph within the same-host sequential profile
- **AND** completed effects and the original parent/causation identities remain
  source-owned

### Requirement: Composite claim conflicts SHALL not be hidden by aggregation

An aggregate graph result MUST preserve child/member/step store ownership and
claim conflicts. It MUST NOT turn a stale writer, foreign owner, or unsupported
deployment cell into a successful aggregate, retry, or fresh parent run.

#### Scenario: Member claim conflict blocks aggregate restore
- **WHEN** one member checkpoint cannot be claimed under its declared fence
- **THEN** the graph restore returns the member conflict and no aggregate
  execution starts
- **AND** already completed members are not replayed
