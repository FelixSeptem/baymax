## ADDED Requirements

### Requirement: Runtime config SHALL expose A2A baseline settings with deterministic precedence
Runtime configuration MUST expose A2A baseline settings using precedence `env > file > default`, including enablement, client timeout, callback retry budget, and capability-discovery controls.

For this milestone, A2A configuration keys MUST remain domain-scoped under `a2a.*` namespace and MUST NOT overlap with teams/workflow keys.

#### Scenario: Startup applies A2A overrides
- **WHEN** runtime starts with overlapping A2A settings from config file and environment
- **THEN** effective A2A settings resolve with `env > file > default` and invalid values fail fast

### Requirement: Runtime diagnostics SHALL expose normalized A2A run summary fields
Runtime diagnostics MUST include additive A2A summary fields, including `a2a_task_total`, `a2a_task_failed`, `peer_id`, and `a2a_error_layer`.

#### Scenario: Consumer inspects A2A-enabled run
- **WHEN** application queries diagnostics for a run that invoked A2A interactions
- **THEN** diagnostics return normalized A2A fields without breaking existing run summary schema

### Requirement: A2A diagnostics SHALL remain replay-idempotent
Repeated ingestion of identical A2A events for the same run MUST NOT inflate logical A2A counters or trend aggregates.

#### Scenario: Duplicate A2A events are replayed
- **WHEN** A2A event stream is replayed more than once for a completed run
- **THEN** diagnostics keep stable A2A aggregate counters after first logical write
