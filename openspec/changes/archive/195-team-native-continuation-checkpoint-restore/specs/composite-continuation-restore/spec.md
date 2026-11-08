## MODIFIED Requirements

### Requirement: Composite continuation material SHALL be source-owned and bounded

Each supported composite source SHALL persist a bounded continuation envelope containing form and graph identity/version/digest, original parent/run/session/causation/authority/occurrence/effect identities, native child/member/step or message/delivery/lease identifiers, dependency state, pending and completed actions, consumed budget/deadline, completed effect references, pause boundary, terminal arbitration evidence, and predecessor/successor lineage. Payloads, credentials, callbacks, and unbounded application bodies MUST remain outside the public envelope.

Each supported team continuation SHALL additionally bind every normalized plan and aggregation input that can affect member dispatch or terminal aggregation, including strategy, participant/member identity, failure/backpressure policy and vote tie-break. A source owner MUST reject a resume whose current graph or aggregation policy does not match the sealed continuation digest.

#### Scenario: Team policy is part of the source identity
- **WHEN** a paused team is resumed with a changed vote tie-break, failure policy, backpressure policy, or participant identity
- **THEN** the source rejects before claim or member activation with a typed conflict or unsupported result
- **AND** it does not treat the changed plan as the original continuation

#### Scenario: Child continuation retains source identity
- **WHEN** a child pauses at a declared safe point
- **THEN** its persisted continuation retains the original parent/run/session, task/attempt/lease, causation, budget, and completed-effect identities

#### Scenario: Existing recovery material is not continuation material
- **WHEN** admission receives a scheduler snapshot, mailbox history, callback event, or application projection without a source continuation envelope
- **THEN** it returns `unsupported_capability` or typed unavailable before any new child, member, message, or plan activation

#### Scenario: Team pause captures deterministic pending work
- **WHEN** a supported team owner pauses before dispatching a pending member/effect
- **THEN** it persists settled member records, the deterministic pending set and aggregate evidence
- **AND** an ambiguous in-flight member causes typed `recovery-required` or unsupported admission rather than a fabricated checkpoint

#### Scenario: Mailbox pause captures delivery lineage
- **WHEN** a supported mailbox owner pauses a host-plan message before its pending action/effect dispatch
- **THEN** the continuation retains the original message, delivery, lease and attempt causation facts
- **AND** it does not replace mailbox lifecycle ownership or create a new delivery identity
