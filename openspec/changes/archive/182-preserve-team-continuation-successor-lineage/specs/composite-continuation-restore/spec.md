## MODIFIED Requirements

### Requirement: Supported composite forms SHALL use atomic idempotent resume claims

A source owner SHALL validate schema, digest, ownership, expiry, graph/form
binding, authority/plan/write-scope/effect namespace, occurrence/causation,
current revision/budget facts, and predecessor/successor relation before an
atomic resume claim. Equivalent requests SHALL return the same retained
projection; conflicting, foreign, stale, expired, corrupt, unavailable, or
already-running requests SHALL fail closed. When a resumed execution pauses
again, the owner SHALL atomically retain the successor checkpoint reference
and the predecessor resume projection, binding the successor digest, graph,
run/session/authority identities, and next generation. A successor SHALL be
claimable only through that retained source lineage; it MUST NOT be treated as
an unrelated generation-1 root.

#### Scenario: Duplicate child resume converges
- **WHEN** two equivalent requests race for one child continuation
- **THEN** exactly one source continuation is admitted and both callers observe
  the same claim or terminal projection

#### Scenario: Conflicting composite resume is rejected
- **WHEN** a request changes mode, form, graph digest, pause reason, causation,
  or resume identity
- **THEN** the source returns a typed conflict/in-progress result without
  consuming the continuation or starting another logical execution

#### Scenario: Repeated pause atomically publishes successor lineage
- **WHEN** a claimed composite continuation pauses again
- **THEN** its source owner commits the successor checkpoint reference and
  predecessor resume projection as one source-store transition
- **AND** the successor retains the original logical run and the next generation

#### Scenario: Successor cannot be claimed as a foreign root
- **WHEN** a resume request presents a successor whose digest, generation,
  predecessor, graph, run, session, or authority binding differs from the
  predecessor's committed resume projection
- **THEN** the source rejects before activating any member or step
