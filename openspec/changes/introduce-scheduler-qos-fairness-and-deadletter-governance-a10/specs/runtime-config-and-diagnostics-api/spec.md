## ADDED Requirements

### Requirement: Runtime config SHALL expose scheduler QoS controls with FIFO default
Runtime config MUST expose scheduler QoS settings and default mode to FIFO.

#### Scenario: Config snapshot uses defaults
- **WHEN** runtime loads default scheduler settings
- **THEN** scheduler QoS mode resolves to FIFO without requiring explicit config

### Requirement: Runtime config SHALL expose fairness threshold and dead-letter toggles
Runtime config MUST expose fairness threshold and dead-letter enablement, with dead-letter disabled by default.

#### Scenario: Dead-letter settings are not configured
- **WHEN** runtime starts without explicit dead-letter configuration
- **THEN** dead-letter behavior is disabled and tasks follow standard retry path

### Requirement: Runtime config SHALL expose exponential retry backoff and jitter parameters
Runtime config MUST expose exponential retry backoff controls and jitter bounds for scheduler retry governance.

#### Scenario: Retry governance parameters are configured
- **WHEN** scheduler retry backoff and jitter params are provided
- **THEN** runtime validates ranges and applies exponential+jitter retry delays deterministically under same seed/input

### Requirement: Run diagnostics SHALL include additive QoS and dead-letter summaries
Run diagnostics MUST include additive scheduler QoS/fairness/dead-letter counters while preserving compatibility-window semantics.

#### Scenario: Legacy consumer reads run summary after A10 rollout
- **WHEN** legacy consumer parses run diagnostics with QoS fields present
- **THEN** existing fields keep prior semantics and new QoS/DLQ fields are optional with nullable/default behavior
