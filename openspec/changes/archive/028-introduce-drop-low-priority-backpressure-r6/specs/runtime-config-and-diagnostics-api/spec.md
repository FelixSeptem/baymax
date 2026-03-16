## ADDED Requirements

### Requirement: Runtime SHALL expose drop_low_priority policy rules and drop-set controls in config
Runtime configuration MUST expose policy fields for `drop_low_priority` behavior, including rule-based low-priority matching and configurable droppable-priority set controls.

#### Scenario: Startup with drop policy rules from file and env
- **WHEN** drop policy fields are set in YAML and overridden by environment variables
- **THEN** effective configuration resolves with precedence `env > file > default`

#### Scenario: Invalid drop policy config
- **WHEN** droppable-priority set or rule enum contains unsupported value
- **THEN** runtime fails fast and rejects startup/hot-reload activation

### Requirement: Runtime diagnostics SHALL expose drop_low_priority outcome semantics
Runtime diagnostics MUST expose backpressure drop outcomes with semantically consistent counters and reason mapping aligned to timeline events.

#### Scenario: Consumer inspects drop outcomes
- **WHEN** a run triggers low-priority drops under queue pressure
- **THEN** diagnostics include non-zero drop counters and timeline correlation with `backpressure.drop_low_priority`
