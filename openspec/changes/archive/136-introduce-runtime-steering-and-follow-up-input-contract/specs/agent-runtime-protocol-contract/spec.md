## MODIFIED Requirements

### Requirement: Runtime SHALL provide a deterministic minimal Run lifecycle state machine
Runtime MUST expose the canonical Run states `submitted`, `working`, `input_required`, `completed`, `failed`, and `canceled`. A Run MUST enter terminal state only through `completed`, `failed`, or `canceled`; cancel MUST be idempotent; resume MUST be accepted only from `input_required`; retry MUST preserve causal association without mutating a terminal Run into `working`. Accepted steering MUST remain a source-owned non-terminal input admission until an eligible decision boundary, and accepted follow-up MUST either remain a bounded pending outcome or create a distinct causal Run under existing admission ownership.

#### Scenario: Realtime interrupt maps to input-required
- **WHEN** an active Run accepts a valid realtime interrupt under the existing realtime contract
- **THEN** its protocol lifecycle maps to `input_required` and includes the existing resumable-cursor correlation

#### Scenario: Steering admission does not imply lifecycle completion
- **WHEN** an active Run accepts a steering input before its next decision boundary
- **THEN** the protocol exposes the input admission correlation while the Run remains in its source-owned non-terminal lifecycle state until the existing boundary resolves

#### Scenario: Follow-up does not mutate a terminal Run
- **WHEN** a follow-up is submitted after a Run is completed, failed, or canceled
- **THEN** the protocol either reports a deterministic terminal/stale rejection or exposes a distinct causally related Run, and the original terminal Run remains unchanged

#### Scenario: Invalid terminal resume is rejected
- **WHEN** a caller requests protocol resume for a completed, failed, or canceled Run
- **THEN** validation rejects the transition with deterministic protocol classification and does not modify source Runtime state

### Requirement: Protocol actions SHALL execute only through source-owned controls
When a host submits a lifecycle action or steering/follow-up input advertised by a `ProtocolDescriptor`, the protocol layer MUST validate profile, capability, input kind, correlation, lifecycle state, readiness, and authorization before delegating to a source-owned control or input admission. The protocol layer MUST expose the source admission outcome and MUST NOT perform a terminal transition, create a retry attempt, apply input content, or mutate Realtime state by itself.

Cancel delegation MUST be idempotent. Resume MUST remain valid only for a source Run in `input_required` with valid source-owned resume correlation. Retry MUST create a causally related source Run or attempt when supported and MUST NOT mutate a terminal Run back to `working`. Steering MUST be applied only at a source-owned safe point; follow-up MUST remain pending or create a distinct causal Run according to source policy.

#### Scenario: Advertised steering delegates to source input admission
- **WHEN** an authorized host submits steering for a Run whose descriptor advertises steering support
- **THEN** the source input owner returns the admission result and the protocol does not apply the content directly

#### Scenario: Advertised cancel delegates idempotently
- **WHEN** an authorized host repeats cancel for the same active Run
- **THEN** the source control observes at most one effective cancellation while every command receives a deterministic original-or-duplicate admission response

#### Scenario: Unsupported source action is rejected
- **WHEN** a descriptor advertises no executable source control for the requested action
- **THEN** protocol validation rejects the action before source mutation even if the action name is part of the canonical vocabulary

#### Scenario: Unsupported source input is rejected
- **WHEN** a descriptor advertises no executable source input control for the requested kind
- **THEN** protocol validation rejects the input before source mutation even if the input kind is part of the canonical vocabulary

#### Scenario: Retry preserves terminal history
- **WHEN** a source supports retry for a terminal Run and accepts an authorized retry command
- **THEN** the source creates a new causally related Run or attempt and the original terminal Run remains immutable

### Requirement: Host-controlled Run and Stream SHALL remain semantically equivalent
Equivalent host commands targeting Run and Stream with the same effective configuration MUST preserve equivalent action and input admission, authorization, steering/follow-up ordering and apply-boundary semantics, cancellation, retry causation, Realtime control, error classification, and terminal outcome semantics. Transport event timing MAY differ, but the protocol MUST NOT create a Stream-only or Run-only control or input state machine.

#### Scenario: Equivalent steering has equivalent outcome
- **WHEN** equivalent active Run and Stream executions receive the same authorized steering command at the same semantic boundary
- **THEN** their normalized input admission, apply/not-applied classification, and authoritative terminal classifications are equivalent

#### Scenario: Equivalent follow-up preserves causal parity
- **WHEN** equivalent Run and Stream executions reach the same idle boundary with the same accepted follow-up sequence
- **THEN** both paths preserve equivalent follow-up ordering, causal Run references, and terminal outcomes

#### Scenario: Equivalent cancel has equivalent outcome
- **WHEN** equivalent active Run and Stream executions receive the same authorized cancel command at the same semantic boundary
- **THEN** their normalized command admission and authoritative terminal classifications are equivalent
