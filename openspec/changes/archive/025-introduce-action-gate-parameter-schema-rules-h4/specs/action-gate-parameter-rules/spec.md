## ADDED Requirements

### Requirement: Action Gate SHALL support parameter-level rule evaluation
The runtime MUST support parameter-level Action Gate rules evaluated against tool call arguments using declared `path`, `operator`, and optional `expected` values.

#### Scenario: Rule matches by parameter path and operator
- **WHEN** a tool call argument at configured path satisfies the declared operator condition
- **THEN** runtime marks the rule as matched and uses rule action resolution flow

#### Scenario: Rule does not match
- **WHEN** all configured parameter rules evaluate to false for a tool call
- **THEN** runtime continues with non-parameter Action Gate evaluation paths

### Requirement: Parameter rules SHALL support composite conditions
The runtime MUST support composite condition trees with `and` and `or` semantics so multiple leaf conditions can be combined deterministically.

#### Scenario: AND composite condition
- **WHEN** all child conditions of an `and` node evaluate true
- **THEN** the composite condition evaluates true

#### Scenario: OR composite condition
- **WHEN** at least one child condition of an `or` node evaluates true
- **THEN** the composite condition evaluates true

### Requirement: Parameter rules SHALL support baseline operator set
The runtime MUST support at least the following operators: `eq`, `ne`, `contains`, `regex`, `in`, `not_in`, `gt`, `gte`, `lt`, `lte`, and `exists`.

#### Scenario: Numeric comparison operator
- **WHEN** a numeric field is evaluated by `gt`/`gte`/`lt`/`lte`
- **THEN** runtime applies numeric comparison with deterministic type handling and returns a boolean result

#### Scenario: Membership operator
- **WHEN** a field is evaluated with `in` or `not_in`
- **THEN** runtime evaluates membership against configured candidate values and returns a boolean result

### Requirement: Parameter rule action SHALL allow per-rule override with policy inheritance
A parameter rule MAY define an explicit action in `allow|require_confirm|deny`. If action is omitted, runtime MUST inherit the global `action_gate.policy` value.

#### Scenario: Rule with explicit action
- **WHEN** a matched parameter rule defines action `deny`
- **THEN** runtime enforces deny regardless of global default policy

#### Scenario: Rule without explicit action
- **WHEN** a matched parameter rule omits action
- **THEN** runtime resolves action from global `action_gate.policy`

### Requirement: Parameter rule evaluation SHALL be fail-fast on invalid configuration
Runtime startup and hot reload MUST reject invalid parameter rule configuration, including malformed condition trees, unsupported operators, and missing required fields.

#### Scenario: Invalid operator in rule config
- **WHEN** configuration contains a parameter rule with unsupported operator
- **THEN** runtime fails fast and does not activate the invalid snapshot

#### Scenario: Malformed composite condition
- **WHEN** configuration contains an `and`/`or` condition without valid child nodes
- **THEN** runtime fails fast with validation error
