## Purpose

This delta makes team continuation restore the exact aggregation contract that produced the paused source state, including vote tie-break and participant provenance, so a fresh owner cannot silently change the terminal decision.

## MODIFIED Requirements

### Requirement: Teams parallel and vote execution SHALL support source-owned continuation at declared safe points

When a team owner is explicitly configured with compatible continuation storage, parallel and vote plans SHALL support bounded `input_required`, `approval_required` and `budget_exhausted` pause boundaries before dispatch of unstarted member work. The persisted continuation SHALL preserve team/plan identity and digest, original execution identity/bindings, strategy, normalized aggregation policy including vote tie-break, member status/output/vote records, deterministic pending member set, completed effect references, aggregate/vote state, pending action, consumed budget, original deadline, terminal arbitration and lineage. Existing unconfigured team engines SHALL retain terminal-only behavior and advertise unsupported continuation.

#### Scenario: Vote policy drift is rejected before member activation
- **WHEN** a resume request presents a plan whose normalized vote tie-break or aggregation policy differs from the paused team graph digest
- **THEN** team restore returns a typed conflict or unsupported result before claiming or activating any member
- **AND** the source does not recompute a winner under the changed policy

#### Scenario: Vote pause preserves participant and tie-break provenance
- **WHEN** an owner-configured vote team pauses with settled and pending members
- **THEN** the continuation retains the settled member identities, votes, participant boundary, and normalized tie-break policy
- **AND** fresh-process restore resolves the pending member using the original policy

#### Scenario: Team capability remains deployment-bound
- **WHEN** a team owner is process-local, absent, or reports an unsupported deployment scope
- **THEN** its cross-process and pause capability cells remain unsupported or unavailable with a bounded reason
- **AND** child, workflow, mailbox, and resumed-plan capability cells are not promoted

#### Scenario: Parallel pause preserves settled members
- **WHEN** an owner-configured parallel team pauses before an unstarted member effect
- **THEN** settled member outcomes remain persisted and only deterministic pending members remain eligible after restore
- **AND** settled effects are not dispatched again

#### Scenario: Vote pause preserves aggregate evidence
- **WHEN** an owner-configured vote team pauses before unresolved member dispatch or vote resolution
- **THEN** persisted continuation retains completed votes and tie-break inputs
- **AND** restored resolution uses the original strategy and deterministic aggregate semantics

#### Scenario: Unsafe in-flight member prevents fabricated pause
- **WHEN** a parallel team has an in-flight member without restorable source evidence at a requested pause boundary
- **THEN** the team returns typed `recovery-required` or rejects continuation
- **AND** it does not advertise a resumeable team checkpoint

### Requirement: Teams continuation restore SHALL be atomic and preserve Run/Stream parity

Team continuation restore SHALL validate schema, digest, graph/form/strategy and all normalized aggregation-policy inputs, source owner, expiry, deployment scope, parent/run/session/causation/authority/plan/write-scope/effect/occurrence identity, pause reason, budget/deadline and predecessor/successor lineage before atomic claim. Exact duplicate resumes SHALL return the retained claim or terminal projection; conflicts and incomplete claims SHALL fail closed. A new process SHALL execute only pending members and preserve equivalent Run/Stream lifecycle, aggregate, participant provenance, tie-break and terminal semantics. When a resumed team pauses again, the owner SHALL persist the next generation and predecessor identity and atomically commit its successor checkpoint reference with the predecessor's retained resume projection. A successor MUST NOT be restored as an unrelated generation-1 checkpoint.

#### Scenario: Run and Stream restore preserve vote semantics
- **WHEN** equivalent paused vote plans are restored through Run and Stream surfaces in fresh team owners
- **THEN** completed votes, participant boundaries, tie-break policy, winner, lifecycle facts, and terminal classification are semantically equivalent apart from transport projection

#### Scenario: Ambiguous in-flight member fails closed
- **WHEN** a team source cannot prove whether an in-flight parallel or vote member committed its effect before a pause or crash
- **THEN** restore returns `recovery-required` or `unsupported_capability`
- **AND** it does not rerun that member, recompute aggregate state, or allocate a replacement team run

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
