## ADDED Requirements

### Requirement: Runtime SHALL expose CA3 pressure-control configuration with deterministic precedence
Runtime configuration MUST support CA3 pressure-control fields with deterministic precedence `env > file > default`, including tier thresholds, absolute token limits, emergency protection behavior, and spill/swap file backend parameters.

#### Scenario: Startup with CA3 threshold overrides
- **WHEN** YAML and environment variables both define CA3 pressure thresholds
- **THEN** effective CA3 configuration resolves with `env > file > default` precedence

#### Scenario: Invalid CA3 threshold configuration
- **WHEN** CA3 thresholds are malformed, overlapping, or out of range
- **THEN** runtime fails fast during startup or hot reload and retains previous valid snapshot

### Requirement: Runtime diagnostics SHALL include CA3 pressure and recovery aggregates
Run diagnostics MUST include CA3 observability fields at minimum for zone residency duration, trigger counts, compression ratio, spill count, and swap-back count.

#### Scenario: Consumer inspects run diagnostics after CA3 pressure event
- **WHEN** a run triggers CA3 pressure controls
- **THEN** diagnostics contain CA3 aggregate fields sufficient to identify zone transitions and mitigation actions

#### Scenario: Consumer inspects run diagnostics after replay with recovery
- **WHEN** replay executes for a run that previously triggered spill/swap
- **THEN** diagnostics include recovery-related counters and preserve consistent aggregate semantics
