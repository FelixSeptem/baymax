## MODIFIED Requirements

### Requirement: Runtime config SHALL define Action Gate defaults and policy fields
Runtime configuration MUST support Action Gate policy fields with deterministic precedence `env > file > default`. Default policy MUST be `require_confirm`. Runtime MUST provide timeout configuration for confirmation resolution, with timeout outcome interpreted as deny.

Runtime configuration MUST additionally support parameter-rule fields for Action Gate, including rule identifiers, condition trees (`and`/`or`), operators, optional per-rule action override, and evaluation priority semantics.

#### Scenario: Startup with no Action Gate override
- **WHEN** runtime starts without Action Gate config overrides
- **THEN** effective Action Gate policy is `require_confirm` and timeout-deny behavior is enabled

#### Scenario: Startup with Action Gate overrides
- **WHEN** Action Gate fields are provided in both YAML and environment variables
- **THEN** effective Action Gate settings resolve by `env > file > default`

#### Scenario: Startup with invalid parameter-rule config
- **WHEN** Action Gate parameter-rule config contains malformed condition tree or unsupported operator
- **THEN** runtime fails fast and rejects startup or hot-reload snapshot

### Requirement: Runtime diagnostics SHALL expose minimal Action Gate counters
Run diagnostics MUST expose minimal Action Gate counters including `gate_checks`, `gate_denied_count`, and `gate_timeout_count`.

Run diagnostics MUST additionally expose minimal parameter-rule counters/metadata fields including `gate_rule_hit_count` and `gate_rule_last_id`.

#### Scenario: Consumer inspects run diagnostics with gated actions
- **WHEN** a run performs Action Gate checks for one or more tool actions
- **THEN** diagnostics include non-negative values for `gate_checks`, `gate_denied_count`, and `gate_timeout_count`

#### Scenario: Consumer inspects run diagnostics with parameter-rule hit
- **WHEN** a run triggers at least one parameter-level rule match
- **THEN** diagnostics include non-negative `gate_rule_hit_count` and a stable `gate_rule_last_id` value

#### Scenario: Consumer inspects run diagnostics without gate activity
- **WHEN** a run does not trigger any Action Gate check
- **THEN** diagnostics expose zero-value counters without breaking existing diagnostics schema compatibility
