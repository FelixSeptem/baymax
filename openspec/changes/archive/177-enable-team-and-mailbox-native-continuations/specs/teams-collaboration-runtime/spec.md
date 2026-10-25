## ADDED Requirements

### Requirement: Teams parallel and vote execution SHALL support source-owned continuation at declared safe points

When a team owner is explicitly configured with compatible continuation storage, parallel and vote plans SHALL support bounded `input_required`, `approval_required` and `budget_exhausted` pause boundaries before dispatch of unstarted member work. The persisted continuation SHALL preserve team/plan identity and digest, original execution identity/bindings, strategy, member status/output/vote records, deterministic pending member set, completed effect references, aggregate/vote state, pending action, consumed budget, original deadline, terminal arbitration and lineage. Existing unconfigured team engines SHALL retain terminal-only behavior and advertise unsupported continuation.

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

Team continuation restore SHALL validate schema, digest, graph/form/strategy binding, source owner, expiry, deployment scope, parent/run/session/causation/authority/plan/write-scope/effect/occurrence identity, pause reason, budget/deadline and predecessor/successor lineage before atomic claim. Exact duplicate resumes SHALL return the retained claim or terminal projection; conflicts and incomplete claims SHALL fail closed. A new process SHALL execute only pending members and preserve equivalent Run/Stream lifecycle, aggregate and terminal semantics.

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
