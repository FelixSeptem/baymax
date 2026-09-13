## Purpose

This capability defines the ownership seam between durable mailbox or scheduler completion and the existing source-owned runtime-input safe point. It ensures a background completion can inform a later model decision exactly once without creating a second pending queue or terminal state machine.

## ADDED Requirements

### Requirement: Completion promotion SHALL reuse existing correlation and safe-point owners

When a durable background operation completes, the completion path MUST retain task, attempt, lease, mailbox, Run, Session, and source-correlation references needed by the existing runtime-input safe point. Promotion MUST be performed by the source Runtime owner and MUST NOT inject content into an active provider, tool, or HITL operation.

#### Scenario: Completion reaches the next eligible decision boundary
- **WHEN** a correlated mailbox result is terminal and the owning Run reaches its next source-declared safe point
- **THEN** the source Runtime applies the bounded completion input at that boundary and preserves causal correlation to the originating task and attempt

#### Scenario: Completion cannot mutate an active atomic operation
- **WHEN** a mailbox result arrives while the owning Run is inside an atomic model, tool, or HITL operation
- **THEN** the result remains pending or is classified according to existing backpressure/late policy until the safe point, without provider or tool mid-operation mutation

### Requirement: Completion promotion SHALL be idempotent and terminal-safe

Duplicate completion, duplicate promotion, late completion after timeout, and terminal races MUST converge through existing scheduler terminal and runtime-input admission semantics. A terminal outcome MUST remain immutable, and no completion race may create a second terminal transition or duplicate model decision.

#### Scenario: Duplicate completion does not duplicate promotion
- **WHEN** the same task/attempt/mailbox completion is delivered more than once
- **THEN** the first accepted logical completion may promote once and subsequent deliveries are classified duplicate without a second promotion

#### Scenario: Late completion cannot overwrite timeout
- **WHEN** a completion arrives after the scheduler has committed an authoritative timeout terminal
- **THEN** the result is classified late or conflict according to existing policy and the timeout terminal remains authoritative

### Requirement: Disconnect and recovery SHALL reconcile completion without resurrection

Completion handling MUST distinguish disconnected, stale, and recovered owners. A disconnected or closed Run MUST NOT be implicitly resurrected by a late background completion; recovery MAY replay only bounded completion references that are not already terminally applied.

#### Scenario: Disconnected Run does not resume implicitly
- **WHEN** a completion arrives after the owning Run has closed or disconnected
- **THEN** admission records a bounded disconnected/stale classification and does not create a new active Run or mutate the closed Run

#### Scenario: Recovery replays pending completion once
- **WHEN** recovery restores a pending correlated completion that was not applied before interruption
- **THEN** reconciliation either applies it once at the next safe point or records a deterministic not-applied/late outcome, and repeated recovery does not duplicate the decision

### Requirement: Run and Stream completion promotion SHALL remain semantically equivalent

Equivalent Run and Stream completion inputs, effective policy, correlation, and recovery state MUST produce equivalent promotion, duplicate, late, disconnect, terminal, and diagnostics classifications after permitted event-order normalization. Diagnostics MUST remain bounded and use only `RuntimeRecorder`.

#### Scenario: Run and Stream use equivalent completion semantics
- **WHEN** equivalent completion sequences are delivered through Run and Stream paths
- **THEN** both paths produce equivalent safe-point ownership, terminal, idempotency, and recovery classifications apart from permitted event ordering

#### Scenario: Raw completion payload is excluded from diagnostics
- **WHEN** a completion is admitted, promoted, rejected, or classified as duplicate/late
- **THEN** diagnostics retain bounded identifiers and reason codes but do not store raw result bodies, reasoning content, credentials, or unbounded payloads
