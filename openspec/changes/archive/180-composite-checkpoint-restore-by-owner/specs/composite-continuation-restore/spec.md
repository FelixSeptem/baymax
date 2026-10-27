## ADDED Requirements

### Requirement: Plan bridge continuation restore SHALL delegate to the matching source owner

For every execution form that advertises cross-process continuation restore, the host plan bridge SHALL require an explicit resume request and the matching source owner/store binding, validate the form-specific capability evidence and deployment scope before activation, and delegate claim/restore exactly once to that owner. The bridge MUST preserve the owner's native result, identity, and terminal evidence. A normal Run/Stream request, another form's checkpoint, a history cursor, or an application projection MUST NOT be used to synthesize restore.

#### Scenario: Workflow plan restores through a configured owner
- **WHEN** a workflow plan contains a valid owner-bound resume request and the configured workflow store supports the declared deployment profile
- **THEN** the bridge delegates resume to that workflow owner and returns its native result
- **AND** it does not call workflow Run/Stream to recreate the graph

#### Scenario: Missing or incompatible workflow owner fails before activation
- **WHEN** a resumed workflow plan has no compatible continuation store, mismatched capability version, or unsupported deployment scope
- **THEN** admission returns typed unavailable, unsupported, or version error before any step dispatch or checkpoint claim

#### Scenario: Manifest does not borrow another form's evidence
- **WHEN** a plan requests continuation for a form whose configured source/store lacks the requested restore or pause-reason evidence
- **THEN** the bridge rejects before owner activation with `unsupported_capability` or typed unavailable
- **AND** it does not infer support from another form, package presence, or DTO compilation

#### Scenario: Unsupported mailbox handler continuation remains rejected
- **WHEN** a mailbox plan requests handler continuation but the mailbox owner exposes lifecycle projection without a handler execution safe point
- **THEN** the bridge returns `unsupported_capability` before publish, consume, claim, or handler dispatch

#### Scenario: Run and Stream use the same native restore semantics
- **WHEN** an equivalent owner-bound workflow resume is invoked through Run or Stream
- **THEN** both use the same owner claim/restore and terminal result semantics
- **AND** Stream only adds native event delivery without changing execution or terminal decisions
