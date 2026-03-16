## ADDED Requirements

### Requirement: Runtime SHALL expose skill trigger scoring policy in YAML and environment overrides
Runtime configuration MUST expose skill trigger scoring policy fields through YAML and environment variables, and MUST resolve effective values with precedence `env > file > default`.

The policy MUST include at least:
- scorer strategy selector (default lexical weighted-keyword)
- confidence threshold
- tie-break mode (default `highest-priority`)
- low-confidence suppression toggle (default enabled)
- keyword/weight mapping inputs needed by default scorer

#### Scenario: Startup with file and env scoring overrides
- **WHEN** runtime starts with skill trigger scoring fields set in YAML and overlapping environment values
- **THEN** effective scoring policy follows `env > file > default`

#### Scenario: Startup with default scoring policy
- **WHEN** runtime starts without explicit skill trigger scoring overrides
- **THEN** effective policy uses lexical weighted-keyword scorer, tie-break `highest-priority`, and suppression enabled

### Requirement: Runtime SHALL fail fast on invalid skill trigger scoring configuration
Runtime MUST validate skill trigger scoring configuration during startup and hot reload; invalid enum values, out-of-range thresholds, or malformed weight entries MUST fail fast and block activation.

#### Scenario: Invalid tie-break mode
- **WHEN** configuration sets unsupported tie-break mode
- **THEN** runtime returns validation error and does not activate the snapshot

#### Scenario: Invalid confidence threshold range
- **WHEN** configuration sets confidence threshold outside supported range
- **THEN** runtime returns validation error and does not activate the snapshot

#### Scenario: Malformed keyword weight mapping
- **WHEN** configuration contains malformed or duplicate-conflicting keyword weights
- **THEN** runtime returns validation error and does not activate the snapshot
