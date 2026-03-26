## ADDED Requirements

### Requirement: Adapter health probe contract SHALL include backoff and circuit governance semantics
Adapter health probe contract MUST define governed probing semantics with:
- exponential backoff (`initial`, `max`, `multiplier`, `jitter_ratio`)
- circuit breaker controls (`failure_threshold`, `open_duration`, `half_open_max_probe`, `half_open_success_threshold`)

Probe governance MUST preserve existing status semantics (`healthy|degraded|unavailable`) and MUST NOT introduce incompatible status values.

#### Scenario: Probe governance enabled with canonical defaults
- **WHEN** runtime uses default adapter health governance settings
- **THEN** probe execution applies configured backoff/circuit controls while keeping status output in canonical three-state model

#### Scenario: Invalid governance config is rejected
- **WHEN** startup or hot reload provides unsupported backoff/circuit values
- **THEN** runtime fails fast and preserves previous valid active snapshot
