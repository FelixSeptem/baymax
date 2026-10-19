## ADDED Requirements

### Requirement: Mailbox SHALL expose bounded host-plan binding and outcome facts

The mailbox public contract SHALL provide bounded binding and outcome facts for an admitted host execution plan without transferring mailbox lifecycle ownership. The binding SHALL carry compatible mailbox contract version, source message identity, parent run, causation, authority digest, plan digest, write scope, effect namespace and resource-limit references. The outcome SHALL preserve source message ID, delivery/lease ID, attempt/retry causation, acknowledgement/negative-acknowledgement/retry/DLQ result, original run/parent run, host correlation and terminal outcome. These facts MUST NOT carry business payloads, credentials, callback errors or a host business-commit assertion.

#### Scenario: Binding and outcome expose source correlation
- **WHEN** a valid host-plan mailbox binding is accepted by the existing mailbox owner
- **THEN** its public outcome identifies the source message and delivery/lease, original run/parent run, host correlation and lifecycle result
- **AND** it preserves attempt/retry causation and terminal outcome without exposing business payloads

#### Scenario: Invalid host-plan binding does not publish
- **WHEN** a binding is oversized, lacks required source identity/digest/scope facts, or has an incompatible contract version
- **THEN** the mailbox-facing admission returns a bounded validation or unavailable error before publish or delivery
- **AND** it does not create a partial envelope or lifecycle record

### Requirement: Host-plan duplicate admission SHALL converge without replacing mailbox identity

For the same full logical host-plan mailbox identity, repeated equivalent admission SHALL converge to one logical message and preserve the original authority, occurrence, effect namespace, source correlation and lifecycle history. A conflicting reuse MUST fail closed and MUST NOT overwrite an existing message, delivery, lease, attempt or outcome.

#### Scenario: Equivalent host-plan duplicate converges
- **WHEN** an admitted host plan repeats dispatch with the same logical message identity and equivalent binding facts
- **THEN** mailbox admission returns the original logical admission or idempotent projection
- **AND** logical message and execution counts do not increase

#### Scenario: Conflicting duplicate is rejected
- **WHEN** a duplicate identity differs in authority, occurrence, parent/causation, write scope or effect namespace
- **THEN** admission rejects before publish or execution
- **AND** the existing mailbox lifecycle record remains unchanged
