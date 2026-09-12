## Purpose

定义嵌入式宿主在 active Run 中提交 steering 或 follow-up 输入时的有界、可关联、可回放和 Run/Stream 对等行为，同时保持 Runner、Session history、Checkpoint、Realtime 和终态事实源的既有所有权。

## ADDED Requirements

### Requirement: Runtime SHALL distinguish steering from follow-up input

The runtime MUST expose a versioned input envelope with an explicit input kind of `steering` or `follow_up`, bounded message content, stable input identity, `session_id`, `run_id`, request correlation, causation, and source correlation when known. Steering MUST mean input intended to affect the next eligible decision boundary of an active Run; follow-up MUST mean input that is retained for execution only after the current Run reaches its declared idle or terminal boundary.

#### Scenario: Valid steering input is correlated to an active Run
- **WHEN** a host submits a bounded steering envelope for an active Run with matching Session and request correlation
- **THEN** the source returns an accepted admission outcome retaining the input identity and correlation without claiming that the model step has already consumed the input

#### Scenario: Follow-up input is not mistaken for steering
- **WHEN** a host submits a valid follow-up envelope for an active Run
- **THEN** the source records a follow-up admission outcome and MUST NOT apply it to the current model or tool step as steering

#### Scenario: Input content exceeds the bounded profile
- **WHEN** a steering or follow-up envelope exceeds the effective payload or serialized-size limit
- **THEN** validation rejects it before source mutation with a deterministic size classification

### Requirement: Input admission SHALL remain source-owned, bounded, and fail-fast

Input admission MUST be evaluated by the source Runtime using the existing Session/Run correlation, readiness, policy, sandbox, authorization, terminal, and cancellation owners. The source MUST maintain a bounded per-Run input boundary; a host adapter MUST NOT own a global input queue, mutate `RunRequest` directly, or block indefinitely while waiting for capacity.

#### Scenario: Unknown or mismatched target is rejected
- **WHEN** an input targets an unknown Run, an inactive Run, or a different Session
- **THEN** admission returns a deterministic rejection and performs no input or lifecycle mutation

#### Scenario: Per-Run input capacity is full
- **WHEN** a valid input arrives after the source-owned bounded input boundary reaches capacity
- **THEN** admission fails fast with an explicit backpressure outcome and does not silently drop or reorder an accepted input

#### Scenario: Host disconnect does not own input state
- **WHEN** the host connection closes after an input admission response is emitted
- **THEN** the source-owned admission and any already accepted input retain their declared outcome, while the adapter does not create or cancel a separate input queue

### Requirement: Steering SHALL apply only at a source-owned safe point

The runtime MUST NOT inject steering content into a Provider call, tool execution, or already-running HITL resolver. An accepted steering input MUST become eligible only at a source-owned safe point after the current atomic model/tool/HITL operation has reached its contract boundary. Applying steering MUST preserve the existing event ordering, Realtime sequence, terminal arbiter, and failure taxonomy.

#### Scenario: Steering waits for an in-flight model call
- **WHEN** steering is accepted while a model call is in progress
- **THEN** the current model call completes, fails, or is canceled according to its existing policy and the steering is considered only at the next eligible decision boundary

#### Scenario: Steering does not interrupt a tool execution
- **WHEN** steering is accepted while a tool call is executing
- **THEN** the tool owner completes or fails the existing call under its current policy before steering can affect the next Runner decision

#### Scenario: Steering during HITL remains separate from the resolver
- **WHEN** steering arrives while clarification or Action Gate resolution is pending
- **THEN** the pending resolver keeps its existing RequestID, timeout, cancellation, and fail-closed semantics, and steering is not applied until the resolver boundary is settled

### Requirement: Follow-up SHALL use an explicit idle or terminal boundary

Follow-up input MUST remain pending until the source declares the current Run eligible for follow-up execution through an explicit idle or terminal boundary. A follow-up MUST NOT mutate a completed, failed, or canceled Run back to `working`; when a new Run is required, the source MUST create a distinct causal Run or attempt under existing Session admission and retry/branch ownership.

#### Scenario: Follow-up starts after an idle boundary
- **WHEN** an active Run reaches the declared idle boundary and has an accepted follow-up
- **THEN** the source starts the follow-up through the existing Run admission path with preserved Session and causation references

#### Scenario: Follow-up cannot reopen a terminal Run
- **WHEN** a follow-up is submitted after the source has published a terminal outcome
- **THEN** the source rejects or maps it to a distinct causal Run according to the declared policy, and the original terminal Run remains immutable

#### Scenario: Follow-up ordering is deterministic
- **WHEN** multiple follow-ups are accepted for the same idle boundary
- **THEN** the source applies the documented stable ordering and preserves each input identity without duplicate logical execution

### Requirement: Input races SHALL preserve terminal, cancel, retry, and Realtime ownership

Input admission MUST be arbitrated against terminal publication, cancel, retry, Realtime interrupt/resume, and source shutdown by existing owners. The first authoritative terminal outcome MUST remain immutable; an input race MUST produce an admission classification rather than a second terminal transition or a parallel cursor/state machine.

#### Scenario: Input races with terminal publication
- **WHEN** steering or follow-up admission races with source terminal publication
- **THEN** exactly one deterministic result is returned for the input and the source publishes exactly one authoritative terminal outcome

#### Scenario: Cancel wins before steering is applied
- **WHEN** cancel is accepted before an admitted steering reaches its safe point
- **THEN** cancellation follows the existing idempotent cancel path and the steering is rejected or marked not-applied without changing the canceled terminal classification

#### Scenario: Retry does not reuse the prior input identity
- **WHEN** a failed or canceled Run is retried and the prior Run had accepted inputs
- **THEN** the retry receives a new causal Run identity and does not silently replay prior input identities as new logical inputs

#### Scenario: Realtime interrupt remains authoritative
- **WHEN** Realtime interrupt/resume and steering arrive near the same decision boundary
- **THEN** Realtime keeps ownership of sequence, cursor, freeze, and resume semantics, while steering is admitted or deferred according to the source boundary without rewriting Realtime state

### Requirement: Pending input SHALL have deterministic disconnect and replay behavior

The runtime MUST define whether each input admission outcome is `accepted`, `rejected`, `duplicate`, `stale`, `terminal`, `backpressure`, `disconnected`, or `not_applied`, and MUST make that outcome replayable from bounded transcript data. Replay MUST compare input identity, kind, correlation, admission, apply boundary, and terminal normalization without invoking a Provider or tool.

#### Scenario: Duplicate input replay is idempotent
- **WHEN** the same input identity and normalized payload are submitted or replayed more than once
- **THEN** only the first logical admission/apply is counted and later submissions return the documented duplicate outcome

#### Scenario: Reusing an identity with different content fails
- **WHEN** an existing input identity is submitted with different normalized content or target correlation
- **THEN** the source returns a deterministic replay-conflict classification and performs no second mutation

#### Scenario: Disconnect leaves source outcome observable
- **WHEN** a host disconnects before an accepted input is applied
- **THEN** the source retains the declared source outcome and a later host can observe it through existing event/cursor or terminal recovery paths without adapter-owned replay state

### Requirement: Steering and follow-up SHALL preserve Run/Stream parity and bounded observability

Equivalent Run and Stream inputs, effective policy, queue limits, source boundaries, and host commands MUST produce semantically equivalent input admission, apply/not-applied, cancellation, terminal, and causal outcomes after permitted event-order normalization. Input diagnostics MUST be additive, bounded, and written only through `observability/event.RuntimeRecorder`; raw input content MUST NOT become a high-cardinality diagnostic or OTel attribute.

#### Scenario: Equivalent Run and Stream steering remains equivalent
- **WHEN** equivalent steering commands reach equivalent Run and Stream decision boundaries
- **THEN** normalized admission, apply boundary, terminal classification, and causation are equivalent

#### Scenario: Queue pressure diagnostics remain bounded
- **WHEN** input admission is rejected because of queue pressure, stale state, or terminal race
- **THEN** diagnostics record bounded reason, kind, queue outcome, and correlation fields through `RuntimeRecorder` without recording raw prompt content

#### Scenario: Parity drift fails the contract gate
- **WHEN** replay observes a Run/Stream difference outside permitted event ordering
- **THEN** the steering/follow-up contract gate returns a deterministic parity-drift classification and fails
