## MODIFIED Requirements

### Requirement: Teams continuation restore SHALL be atomic and preserve Run/Stream parity

Team continuation restore SHALL validate schema, digest, graph/form/strategy
binding, source owner, expiry, deployment scope, parent/run/session/causation/
authority/plan/write-scope/effect/occurrence identity, pause reason, budget/
deadline and predecessor/successor lineage before atomic claim. Exact duplicate
resumes SHALL return the retained claim or terminal projection; conflicts and
incomplete claims SHALL fail closed. A new process SHALL execute only pending
members and preserve equivalent Run/Stream lifecycle, aggregate and terminal
semantics. When a resumed team pauses again, the owner SHALL persist the next
generation and predecessor identity and atomically commit its successor
checkpoint reference with the predecessor's retained resume projection. A
successor MUST NOT be restored as an unrelated generation-1 checkpoint.

#### Scenario: Duplicate team resume converges
- **WHEN** two equivalent resumes race for one team continuation
- **THEN** one logical team continuation is admitted and both callers receive the same retained projection
- **AND** pending members are not duplicated

#### Scenario: Team resume conflict blocks aggregate activation
- **WHEN** a request changes graph digest, strategy, pause reason, authority, occurrence, causation or resume identity
- **THEN** team restore rejects before member activation
- **AND** it does not create a replacement team plan or parent Run

#### Scenario: Run and Stream restore remain equivalent
- **WHEN** equivalent paused team plans are restored through Run and Stream surfaces
- **THEN** member lifecycle facts, vote/aggregate state and terminal classification are semantically equivalent apart from transport projection

#### Scenario: Repeated team pause preserves the source checkpoint chain
- **WHEN** a team resumes from one checkpoint and pauses at another declared safe point
- **THEN** the successor retains the original team/run/session/causation identities, increments generation, and names the consumed checkpoint as predecessor
- **AND** the predecessor's retained resume projection names the successor checkpoint and digest

#### Scenario: Fresh owner resumes the committed team successor
- **WHEN** a new team engine opens the compatible store after the successor publication
- **THEN** it validates the committed lineage and resumes only the successor's pending members
- **AND** settled member effects are not dispatched again

#### Scenario: Conflicting successor publication fails closed
- **WHEN** a resume attempts to publish a successor with a mismatched predecessor, generation, graph digest or authority binding
- **THEN** the owner returns a typed conflict or recovery-required result without replacing the retained predecessor projection
- **AND** it does not activate another team member
