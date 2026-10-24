## Purpose

Defines how source owners expose retained-history write failures so consumers can distinguish complete source evidence from best-effort observation without confusing replay with execution recovery.

## ADDED Requirements

### Requirement: Source owners SHALL expose history append health

An execution source that has explicitly bound a native history journal MUST expose failed appends as a typed source-history health outcome associated with that journal's source identity. It MUST NOT report the failed append as retained/replayable. This callback-history contract does not gate execution effects or make append failure an execution terminal. History remains opt-in; the source MUST NOT create a parallel journal implicitly.

#### Scenario: Callback history append fails
- **WHEN** an explicitly configured owner cannot persist a callback-projected source fact
- **THEN** its history health reports typed source-unavailable/degraded state for that source identity
- **AND** the failed fact is not represented as retained/replayable
- **AND** the owner does not claim that this callback append gated an effect

#### Scenario: Observation append fails after an effect boundary
- **WHEN** a non-authoritative observation append fails after the source may have performed an effect
- **THEN** the source exposes degraded/unavailable history health with the affected source identity
- **AND** the runtime does not convert the error into a fresh execution or claim an atomic consumer projection

### Requirement: History failure SHALL preserve execution uncertainty

When a source-owned continuation transition may have happened but its continuation result cannot be committed, that continuation owner MUST expose `recovery_required` or an equivalent typed outcome. Callback-history replay MUST NOT be used to infer or repair the uncertain effect. The owner MUST NOT retry, resume, or rerun the parent/child/step solely to reconstruct missing history. Consumer projection remains independent and MUST replay only committed source facts.

#### Scenario: Terminal history commit fails after an effect
- **WHEN** execution may have completed an effect but persistence of its terminal source fact fails
- **THEN** the caller receives a typed unavailable or recovery-required result
- **AND** a retry of the logical execution is not started automatically

#### Scenario: Consumer projection restarts
- **WHEN** source facts were committed but the consumer projection crashed before applying them
- **THEN** the consumer can read the same source-owned facts after its confirmed cursor
- **AND** replay performs no source execution or business commit
