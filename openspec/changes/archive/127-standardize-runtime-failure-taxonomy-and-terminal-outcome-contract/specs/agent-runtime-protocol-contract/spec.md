## ADDED Requirements

### Requirement: Runtime SHALL expose an idempotent source-owned terminal outcome projection
For every mapped Run, the runtime MAY expose an additive terminal outcome containing normalized failure family, terminal Run state, phase, source reason, retryable/resumable flags, and bounded causal/attempt metadata. The projection MUST NOT synthesize queueing, retry, cancellation, resume, or branching. The first valid terminal outcome MUST win; repeated identical publication MUST be an idempotent no-op; a conflicting later publication MUST be recorded as a conflict without overwriting the business terminal state.

#### Scenario: Successful completion publishes a terminal outcome
- **WHEN** a source Run completes without failure
- **THEN** the projection exposes terminal state `completed`, normalized family `none`, and a single terminal publication

#### Scenario: Repeated terminal publication is idempotent
- **WHEN** the same terminal outcome is projected more than once
- **THEN** the first publication remains authoritative and repeated projections do not inflate counters or create duplicate logical outcomes

#### Scenario: Conflicting late terminal report is retained without overwrite
- **WHEN** a later callback reports a terminal result that conflicts with an already published terminal outcome
- **THEN** the runtime preserves the first terminal business state, records a deterministic conflict classification, and does not transition the Run back to `working`

## MODIFIED Requirements

### Requirement: Runtime SHALL preserve fail-fast boundaries while representing recoverable outcomes
Protocol mapping MAY represent a recoverable tool or business failure as a failed Step outcome or error Event only when the owning Runtime path already allows recovery. Configuration validation, security/permission denial, protocol validation, snapshot compatibility conflict, and module-boundary violations MUST retain their existing fail-fast behavior and deterministic classifications. The normalized terminal projection MUST additionally identify whether the failure occurred before execution or after work started, while preserving the source `ErrorClass` and reason code.

#### Scenario: Recoverable tool failure is visible to a Run consumer
- **WHEN** a tool failure is classified as recoverable by the owning Runner policy
- **THEN** the mapped Step exposes failed outcome data with normalized family `runtime_failed`, phase `post_start`, and the owner-defined retry or continuation semantics

#### Scenario: Invalid configuration remains fail-fast
- **WHEN** protocol mapping is requested under an invalid effective configuration
- **THEN** runtime rejects the operation through existing configuration validation, classifies it as a pre-execution failure, and MUST NOT convert the violation into a recoverable Step outcome

#### Scenario: Policy denial remains distinct from runtime failure
- **WHEN** an action is rejected by policy or authorization before execution
- **THEN** the projection records normalized family `policy_denied`, phase `pre_execution`, the source deny reason, and no claim that the tool or model started

### Requirement: Runtime SHALL preserve partial valid facts on post-start failure
When a Provider stream, tool, or composed execution emits valid text deltas, complete tool calls, artifacts, or events before failure, the protocol projection MUST retain those facts and append the normalized terminal failure. It MUST NOT re-execute consumed work as part of projection.

#### Scenario: Provider fails after emitting partial output
- **WHEN** a Provider stream emits valid output and then terminates with an unrecoverable error
- **THEN** mapped events retain the valid output, the terminal projection uses normalized family `runtime_failed` or the source-specific mapped family, and no automatic duplicate retry is synthesized

#### Scenario: Tool failure preserves prior tool facts
- **WHEN** one tool call fails after earlier calls in the same Run produced valid outcomes
- **THEN** prior call outcomes remain correlated by call/step identifiers and the terminal projection does not erase them

### Requirement: Protocol terminal projection SHALL preserve Run/Stream semantic equivalence
For equivalent input, effective configuration, and dependency state, Run and Stream MUST expose semantically equivalent normalized terminal family, terminal state, phase, source reason, retryable/resumable flags, causal association, and bounded attempt aggregates. Event delivery order MAY differ only where permitted by the existing source/realtime contract.

#### Scenario: Equivalent Run and Stream complete successfully
- **WHEN** equivalent Run and Stream requests reach successful completion
- **THEN** both projections expose `completed` with normalized family `none` and equivalent causal/attempt metadata

#### Scenario: Equivalent Run and Stream are canceled
- **WHEN** equivalent Run and Stream requests are canceled at the same logical boundary
- **THEN** both projections expose terminal state `canceled`, normalized family `canceled`, and semantically equivalent cancellation reason

#### Scenario: Equivalent Run and Stream hit provider failure
- **WHEN** equivalent Run and Stream requests encounter the same provider failure phase
- **THEN** both projections expose equivalent normalized family, phase, source reason, and partial-fact preservation semantics

### Requirement: Protocol terminal projection SHALL remain additive and replay-compatible
Normalized terminal fields MUST be nullable, defaultable, and associated with an explicit profile or fixture version when emitted. Legacy protocol mappings without the fields MUST remain valid. Diagnostics and OTel projections MUST use `observability/event.RuntimeRecorder` and MUST exclude unbounded payloads and high-cardinality raw event bodies.

#### Scenario: Legacy fixture omits terminal projection fields
- **WHEN** an existing `agent_runtime_protocol.v1` fixture has no normalized terminal fields
- **THEN** replay succeeds using existing states and reasons, with new fields absent or defaulted

#### Scenario: Terminal conflict is replayed twice
- **WHEN** the same first-terminal and conflicting-late-terminal events are replayed repeatedly
- **THEN** the business terminal outcome remains singular and conflict diagnostics remain logically idempotent
