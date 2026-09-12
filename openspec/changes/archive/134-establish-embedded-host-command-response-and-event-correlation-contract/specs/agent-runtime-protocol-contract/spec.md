## ADDED Requirements

### Requirement: Protocol actions SHALL execute only through source-owned controls
When a host submits a lifecycle action advertised by a `ProtocolDescriptor`, the protocol layer MUST validate profile, action availability, correlation, lifecycle state, readiness, and authorization before delegating to a source-owned control. The protocol layer MUST expose the source admission outcome and MUST NOT perform a terminal transition, create a retry attempt, or mutate Realtime state by itself.

Cancel delegation MUST be idempotent. Resume MUST remain valid only for a source Run in `input_required` with valid source-owned resume correlation. Retry MUST create a causally related source Run or attempt when supported and MUST NOT mutate a terminal Run back to `working`.

#### Scenario: Advertised cancel delegates idempotently
- **WHEN** an authorized host repeats cancel for the same active Run
- **THEN** the source control observes at most one effective cancellation while every command receives a deterministic original-or-duplicate admission response

#### Scenario: Unsupported source action is rejected
- **WHEN** a descriptor advertises no executable source control for the requested action
- **THEN** protocol validation rejects the action before source mutation even if the action name is part of the canonical vocabulary

#### Scenario: Retry preserves terminal history
- **WHEN** a source supports retry for a terminal Run and accepts an authorized retry command
- **THEN** the source creates a new causally related Run or attempt and the original terminal Run remains immutable

### Requirement: Active Run control lifecycle SHALL be bounded and race safe
An Engine exposing host controls MUST register a unique source control before an admitted Run becomes externally controllable and MUST remove the active control after source terminal publication. Duplicate active `run_id` registration MUST fail fast. A control racing with terminal publication MUST produce one deterministic admission result and MUST NOT overwrite, duplicate, or delay the first authoritative terminal outcome.

Active control state MUST be process-local and bounded by active Runs. Runtime diagnostics and `QueryRuns` MUST NOT become the active registry or lifecycle owner.

#### Scenario: Duplicate active Run ID fails fast
- **WHEN** a second Run attempts to register a `run_id` already owned by an active Run in the same Engine
- **THEN** admission fails before executing model or tool work and the first Run's control remains unchanged

#### Scenario: Cancel races with completion
- **WHEN** cancel admission and source completion occur concurrently
- **THEN** the source publishes exactly one authoritative terminal outcome and the command reports whether cancellation was accepted, already terminal, or duplicate

### Requirement: Host-controlled Run and Stream SHALL remain semantically equivalent
Equivalent host commands targeting Run and Stream with the same effective configuration MUST preserve equivalent action admission, authorization, cancellation, retry causation, Realtime control, error classification, and terminal outcome semantics. Transport event timing MAY differ, but the protocol MUST NOT create a Stream-only or Run-only control state machine.

#### Scenario: Equivalent cancel has equivalent outcome
- **WHEN** equivalent active Run and Stream executions receive the same authorized cancel command at the same semantic boundary
- **THEN** their normalized command admission and authoritative terminal classifications are equivalent

