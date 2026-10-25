## ADDED Requirements

### Requirement: Mailbox host-plan continuation SHALL remain owned by the existing mailbox source

For an explicitly configured mailbox continuation owner, the mailbox public contract SHALL provide bounded host-plan continuation creation, inspection and restore admission using the original logical message and existing mailbox lifecycle owner. The continuation SHALL bind the normalized host-plan identity, message ID, delivery ID, lease ID, attempt causation, parent/run/host correlation, authority/plan/write-scope/effect/occurrence identity, pause reason, pending action/effect references, budget/deadline, expiry and predecessor/successor lineage. The continuation API MUST NOT create a second queue, worker, scheduler, message or execution loop.

#### Scenario: Equivalent mailbox continuation request converges
- **WHEN** equivalent continuation requests are made for one eligible host-plan delivery
- **THEN** mailbox retains one logical continuation and returns the original or duplicate source projection
- **AND** message, delivery and execution counts do not increase

#### Scenario: Invalid continuation admission has no lifecycle side effect
- **WHEN** a continuation request is malformed, foreign, stale, expired, conflicting, lease-invalid or unsupported
- **THEN** mailbox returns a typed error before action dispatch
- **AND** it does not consume, heartbeat, acknowledge, negative-acknowledge, requeue, dead-letter, publish or invoke a handler

### Requirement: Mailbox continuation restore SHALL validate current lifecycle facts

Mailbox continuation restore SHALL atomically claim only the original source continuation after validating schema/digest, owner namespace, deployment scope, full host-plan binding, message/delivery/lease/attempt causation, current lifecycle state, pause reason, expiry, budget/deadline, predecessor/successor lineage and resume identity. Exact duplicates SHALL return one retained claim/result projection. A stale lease, reclaimed delivery, missing source record, incomplete claimed attempt, store failure or incompatible deployment SHALL fail closed with typed source evidence.

#### Scenario: New process restores valid original delivery
- **WHEN** a compatible new process reopens the same mailbox continuation store and submits a valid resume for an active original delivery
- **THEN** it restores the pending source boundary for the same logical message and delivery lineage
- **AND** completed effects are not re-run and no replacement message is published

#### Scenario: Lease loss returns source recovery outcome
- **WHEN** the original delivery lease is no longer current at resume admission
- **THEN** mailbox returns a lease/lifecycle recovery outcome before action execution
- **AND** it does not claim a new delivery or silently retry the continuation

### Requirement: Mailbox continuation terminal projection SHALL preserve lifecycle semantics

Completion of mailbox continuation work SHALL be projected through the existing mailbox lifecycle owner. Ack, nack, requeue, retry, dead-letter, expiration, cancellation and timeout results SHALL preserve original message/delivery/lease/attempt causation and terminal evidence. The public terminal projection MUST remain distinct from a host business commit and MUST be semantically equivalent for Run and Stream host-plan surfaces.

#### Scenario: DLQ terminal remains correlated after continuation resume
- **WHEN** a mailbox continuation reaches retry exhaustion with DLQ enabled
- **THEN** the terminal projection contains original message identity, latest delivery/lease facts, attempt causation and dead-letter reason
- **AND** it does not claim that the host business effect committed
