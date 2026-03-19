## ADDED Requirements

### Requirement: Runtime SHALL provide unified collaboration primitives for multi-agent orchestration
Runtime MUST provide unified collaboration primitives in `orchestration/collab` for `handoff`, `delegation`, and `aggregation`.

#### Scenario: Host uses collab primitives through library API
- **WHEN** host initiates a composed task using collab primitive contracts
- **THEN** runtime executes collaboration flow without requiring module-specific custom glue code

### Requirement: Collaboration aggregation strategy SHALL support all_settled and first_success
Collaboration primitive contract MUST support `all_settled` and `first_success` aggregation strategies, and default strategy MUST be `all_settled`.

#### Scenario: Aggregation strategy is omitted in request
- **WHEN** host submits collaboration request without explicit aggregation strategy
- **THEN** runtime resolves strategy to `all_settled` deterministically

#### Scenario: first_success short-circuits on first terminal success
- **WHEN** collaboration request runs with `first_success` and one branch succeeds
- **THEN** runtime returns successful aggregate outcome without waiting for remaining non-required branches

### Requirement: Collaboration failure policy SHALL default to fail_fast
Collaboration primitive execution MUST default to `fail_fast` failure policy unless explicitly overridden by valid configuration.

#### Scenario: One delegated branch fails under default policy
- **WHEN** collaboration request executes with default policy and one required branch fails
- **THEN** runtime terminates aggregate flow with fail-fast semantics and deterministic terminal classification

### Requirement: Collaboration primitive retries SHALL be disabled by default
Collaboration primitive layer MUST keep retry disabled by default and MUST rely on existing scheduler/retry governance for retry behavior.

#### Scenario: Delegation fails with retryable transport error
- **WHEN** collaboration primitive receives retryable error and primitive-level retry is disabled
- **THEN** runtime does not perform extra primitive-level retry and preserves downstream retry governance semantics

### Requirement: Collaboration primitives SHALL compose with sync async delayed execution paths
Collaboration primitive contract MUST compose with existing synchronous, async-reporting, and delayed-dispatch execution semantics.

#### Scenario: Delegation uses delayed dispatch and async reporting
- **WHEN** collaboration request uses delayed scheduling and async terminal convergence
- **THEN** runtime preserves deterministic handoff/delegation/aggregation semantics and replay-idempotent aggregates
