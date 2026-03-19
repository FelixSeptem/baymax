## ADDED Requirements

### Requirement: Runtime SHALL negotiate requested adapter capabilities against declared adapter capabilities
Runtime adapter invocation MUST evaluate requested capabilities against adapter-declared capabilities before execution.

Negotiation input MUST support:
- requested required capabilities,
- requested optional capabilities,
- negotiation strategy (`fail_fast` or `best_effort`).

#### Scenario: Requested capabilities are fully satisfied
- **WHEN** all requested required capabilities are available from adapter declaration and runtime support
- **THEN** negotiation succeeds and execution proceeds without downgrade classification

#### Scenario: Requested required capability is missing
- **WHEN** one or more requested required capabilities are unavailable
- **THEN** negotiation fails fast with deterministic reason `adapter.capability.missing_required`

### Requirement: Negotiation strategy SHALL default to fail_fast and support request-level override
The default negotiation strategy MUST be `fail_fast`.
Runtime MAY accept request-level override to `best_effort` when explicitly configured.

#### Scenario: No override provided
- **WHEN** invocation request does not set negotiation strategy override
- **THEN** runtime applies default `fail_fast` strategy

#### Scenario: Request overrides to best_effort
- **WHEN** invocation request explicitly sets strategy to `best_effort`
- **THEN** runtime applies best-effort behavior and records override reason `adapter.capability.strategy_override_applied`

### Requirement: Missing optional capability SHALL downgrade deterministically
If optional capability is unavailable, runtime MUST execute deterministic downgrade behavior and emit reason `adapter.capability.optional_downgraded`.

#### Scenario: Optional capability unavailable under best_effort
- **WHEN** optional requested capability is not available and strategy allows downgrade
- **THEN** runtime proceeds with downgraded path and records deterministic downgrade reason

#### Scenario: Optional capability unavailable under fail_fast for required-only request
- **WHEN** request contains only required capabilities and all are satisfied
- **THEN** optional capability absence does not affect execution result

### Requirement: Negotiation semantics SHALL be equivalent for Run and Stream paths
Given same request context and adapter declaration, Run and Stream MUST produce semantically equivalent negotiation outcomes (accept/reject/downgrade classification).

#### Scenario: Run and Stream evaluate same missing required capability
- **WHEN** Run and Stream execute equivalent request with same missing required capability
- **THEN** both paths fail with same negotiation classification and reason taxonomy

#### Scenario: Run and Stream evaluate same optional downgrade
- **WHEN** Run and Stream execute equivalent request where optional capability is unavailable under downgrade-allowed strategy
- **THEN** both paths emit equivalent downgrade classification and reason taxonomy
