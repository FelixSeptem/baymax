## ADDED Requirements

### Requirement: Extension activation SHALL be manifest-first and deterministic
The runtime MUST validate an extension descriptor before activation. The descriptor MUST expose a stable identity, kind, version, compatibility range, content digest, declared required/optional capabilities, and source reference.

#### Scenario: Valid descriptor proceeds to admission
- **WHEN** an extension descriptor contains all required identity, compatibility, digest, and capability fields
- **THEN** runtime proceeds to policy/readiness admission without executing extension code

#### Scenario: Invalid descriptor is rejected before execution
- **WHEN** an extension descriptor is missing a required field or contains malformed metadata
- **THEN** runtime rejects activation with deterministic field-level classification and does not execute extension code

### Requirement: Extension capability negotiation SHALL reuse canonical adapter semantics
Extension requested capabilities MUST be negotiated against declared capabilities using existing required/optional and `fail_fast|best_effort` semantics.

#### Scenario: Required extension capability is missing
- **WHEN** a requested required capability is absent from the descriptor or runtime support
- **THEN** activation fails with canonical missing-required capability reason

#### Scenario: Optional extension capability is unavailable
- **WHEN** an optional capability is unavailable and policy permits degradation
- **THEN** extension activates with deterministic downgrade marker and recorded reason

### Requirement: Extension activation SHALL pass readiness and policy admission
Runtime MUST perform manifest validation before readiness/policy admission, and MUST NOT activate blocked extensions. Degraded activation MUST follow configured admission policy.

#### Scenario: Blocked extension is not activated
- **WHEN** readiness or policy admission returns `deny`
- **THEN** extension code is not activated and no extension lifecycle side effect is committed

#### Scenario: Degraded extension follows policy
- **WHEN** admission returns `degraded` and configured policy is `allow_and_record`
- **THEN** extension activates and records degraded admission markers

### Requirement: Extension lifecycle SHALL enforce bounded execution and local failure isolation
Extension hooks and extension-provided tool actions MUST execute within configured timeout and resource bounds. Panic, timeout, invalid result, or finalize failure MUST be classified as extension failure and MUST NOT silently bypass policy or sandbox boundaries.

#### Scenario: Extension timeout is isolated
- **WHEN** an extension hook exceeds its configured timeout
- **THEN** runtime records deterministic timeout classification, releases extension-owned resources, and applies configured skip/deny/degrade behavior without hanging the Run

#### Scenario: Extension panic is isolated
- **WHEN** an extension panics during a lifecycle phase
- **THEN** runtime recovers the panic, records phase and extension identity, and prevents the panic from corrupting unrelated runtime state

### Requirement: Extension failures SHALL NOT rewrite the authoritative Run terminal outcome
Extension failure handling MUST preserve the existing Run/Stream terminal arbiter and MUST NOT create a second terminal state machine.

#### Scenario: Optional extension fails during a successful Run
- **WHEN** an optional extension fails under a continue/degrade policy while the core Run succeeds
- **THEN** Run terminal outcome remains core-runtime authoritative and extension failure is recorded separately

#### Scenario: Extension denial occurs before execution
- **WHEN** required extension admission is denied before core execution starts
- **THEN** runtime returns deterministic admission failure and performs no scheduler, mailbox, or tool lifecycle mutation

### Requirement: Reload SHALL isolate extension generations
Each successful reload MUST create a new activation generation. Previous generations MUST become stale and MUST NOT receive new events; in-flight actions MUST follow existing cancellation/completion semantics and MUST NOT be implicitly retried by reload.

#### Scenario: Reload replaces active generation
- **WHEN** resource reload completes successfully
- **THEN** new events are delivered only to the new generation and old generation handlers are inactive

#### Scenario: Reload fails atomically
- **WHEN** new resources fail validation or activation during reload
- **THEN** previous active generation remains effective and runtime records deterministic rollback reason

### Requirement: Extension observability SHALL use RuntimeRecorder and additive diagnostics
Extension discovery, admission, activation, hook execution, denial, degradation, failure, reload, and rollback events MUST be emitted through `observability/event.RuntimeRecorder`. New diagnostics fields MUST be additive, nullable, and default-compatible.

#### Scenario: Lifecycle event is recorded through single writer
- **WHEN** an extension lifecycle event occurs
- **THEN** diagnostics receives the event only through RuntimeRecorder with bounded extension identity and generation metadata

#### Scenario: Existing diagnostics consumer reads new records
- **WHEN** a consumer reads diagnostics without extension-aware fields
- **THEN** existing parsing and query behavior remains compatible through additive defaults

### Requirement: Extension decisions SHALL be replay-verifiable and Run/Stream equivalent
Given identical resource descriptors, configuration snapshots, requested capabilities, and admission inputs, Run and Stream MUST produce equivalent extension discovery, admission, activation, failure, and reload decisions. Decisions MUST be reproducible from versioned replay fixtures.

#### Scenario: Equivalent Run and Stream activation
- **WHEN** equivalent Run and Stream requests resolve the same extension inputs
- **THEN** both paths produce equivalent activation or denial classification and reason taxonomy

#### Scenario: Replay reproduces reload rollback
- **WHEN** replay evaluates a fixture containing invalid reload resources
- **THEN** replay reproduces stale-generation handling, restored previous generation, and rollback reason deterministically
