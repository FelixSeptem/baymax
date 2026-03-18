## ADDED Requirements

### Requirement: Runtime config SHALL expose recovery controls with default disabled
Runtime configuration MUST expose recovery controls and default `recovery.enabled` to false.

#### Scenario: Runtime loads default config without recovery settings
- **WHEN** runtime starts using default config values
- **THEN** recovery is disabled unless explicitly enabled by config input

### Requirement: Runtime config SHALL expose recovery conflict policy and enforce fail-fast
Runtime configuration MUST expose recovery conflict policy and MUST enforce `fail_fast` for state reconciliation conflicts.

#### Scenario: Recovery conflict policy is evaluated
- **WHEN** state reconciliation conflict occurs during recovery
- **THEN** runtime applies fail-fast handling and terminates recovery flow with deterministic conflict reporting

### Requirement: Diagnostics SHALL expose additive recovery markers
Run diagnostics MUST include additive recovery markers (enabled/recovered/replay/conflict/fallback indicators) while preserving compatibility-window semantics.

#### Scenario: Legacy consumer reads run summary after recovery rollout
- **WHEN** legacy consumer parses run summary produced with recovery-capable runtime
- **THEN** pre-existing fields remain stable and newly added recovery markers are optional with nullable/default fallback behavior
