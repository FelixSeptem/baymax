## ADDED Requirements

### Requirement: Runtime config SHALL expose collaboration primitive controls with deterministic precedence
Runtime configuration MUST expose collaboration primitive controls with precedence `env > file > default` and default-disabled behavior.

Minimum required controls:
- `composer.collab.enabled`
- `composer.collab.default_aggregation` (`all_settled|first_success`)
- `composer.collab.failure_policy` (`fail_fast|best_effort`)
- `composer.collab.retry.enabled`

Default requirements:
- `composer.collab.enabled=false`
- `composer.collab.default_aggregation=all_settled`
- `composer.collab.failure_policy=fail_fast`
- `composer.collab.retry.enabled=false`

#### Scenario: Runtime starts with default configuration
- **WHEN** no collaboration primitive config overrides are provided
- **THEN** collaboration primitive capability remains disabled with documented default strategy and policy values

#### Scenario: Invalid collaboration config is loaded
- **WHEN** configuration provides unsupported aggregation strategy or failure policy value
- **THEN** runtime fails fast on startup/reload and keeps previous valid snapshot

### Requirement: Runtime diagnostics SHALL expose additive collaboration primitive summary fields
Run diagnostics MUST expose additive collaboration primitive summary fields while preserving compatibility-window semantics.

Minimum required fields:
- `collab_handoff_total`
- `collab_delegation_total`
- `collab_aggregation_total`
- `collab_aggregation_strategy`
- `collab_fail_fast_total`

#### Scenario: Consumer queries run summary with collaboration primitives
- **WHEN** collaboration primitive execution is enabled and used in a run
- **THEN** diagnostics include additive collaboration fields without breaking existing consumers

### Requirement: Collaboration diagnostics SHALL preserve additive nullable default compatibility
Collaboration additive fields MUST follow `additive + nullable + default` semantics and MUST NOT change pre-existing field meanings.

#### Scenario: Legacy consumer parses diagnostics after collaboration rollout
- **WHEN** legacy parser reads run summary containing new collaboration fields
- **THEN** parser remains compatible and pre-existing field semantics remain unchanged
