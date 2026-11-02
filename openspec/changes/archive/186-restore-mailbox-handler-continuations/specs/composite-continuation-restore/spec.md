## ADDED Requirements

### Requirement: Mailbox handler execution continuation SHALL not be inferred from lifecycle material

A mailbox continuation supports host-handler execution only when the mailbox
owner provides a message/delivery/lease/attempt-bound handler safe point,
persisted handler continuation material, atomic resume claim, registered
resumer, and terminal lifecycle evidence for the requested pause reason.
Lifecycle history, lifecycle-only continuation projections, callback replay,
identical run IDs, and an application pause marker MUST NOT be treated as
handler execution material.

#### Scenario: Lifecycle-only owner remains rejected
- **WHEN** a mailbox owner supplies host-plan admission and lifecycle
  continuation but no handler execution safe point
- **THEN** handler continuation admission returns `unsupported_capability`
  before publish, claim, handler dispatch, or source mutation
- **AND** it does not create a bridge executor or fallback direct Run

#### Scenario: Evidenced handler owner restores one logical message
- **WHEN** an owner supplies matching handler safe-point and pause-reason
  evidence for a compatible continuation request
- **THEN** it restores only the original pending handler boundary for that
  logical message/delivery lineage
- **AND** it retains completed-effect and terminal arbitration evidence without
  re-enqueueing or re-running the parent
