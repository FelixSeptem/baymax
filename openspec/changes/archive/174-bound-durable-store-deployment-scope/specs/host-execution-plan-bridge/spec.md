## ADDED Requirements

### Requirement: Plan admission SHALL validate owner deployment scope

Before publishing, enqueueing, claiming, or restoring a plan, the bridge MUST
validate the bound source owner's deployment profile, writer fence, transaction
scope, and capability evidence against the plan's requested concurrency and
recovery mode. A plan requesting unsupported multi-writer, cross-machine, or
multi-instance behavior MUST return typed `unsupported_capability` or
`unavailable` before source activation.

#### Scenario: Local mailbox plan remains owner-scoped
- **WHEN** a mailbox plan binds a same-host single-writer owner with matching
  contract evidence
- **THEN** admission preserves that scope and delegates to the existing owner
- **AND** the bridge does not create a second queue, owner, or coordinator

#### Scenario: Distributed plan is rejected before side effects
- **WHEN** a plan asks for cross-machine resume or concurrent cursor/claim
  behavior from a local-only owner
- **THEN** compilation or admission returns typed unsupported before publish,
  enqueue, claim, Run, Stream, retry, or restore
- **AND** it does not degrade to direct execution

### Requirement: Plan replay SHALL not upgrade deployment guarantees

Reference-only plan replay and capability inspection MUST report the source
owner's declared deployment boundary without acquiring a writer, advancing a
cursor, claiming a checkpoint, or starting execution. Replay output MUST NOT
claim distributed support from a successful local reopen.

#### Scenario: Replay preserves local-only evidence
- **WHEN** an offline fixture contains a successful local reopen and a request
  for multi-instance recovery
- **THEN** replay reports the local success and an explicit unsupported result
  for the distributed request
- **AND** no owner method or side effect is invoked
