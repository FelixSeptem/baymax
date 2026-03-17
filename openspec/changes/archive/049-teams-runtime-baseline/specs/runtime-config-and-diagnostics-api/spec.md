## ADDED Requirements

### Requirement: Runtime SHALL expose Teams baseline configuration with deterministic precedence
Runtime configuration MUST expose Teams baseline fields with precedence `env > file > default`, including at minimum enablement, default strategy, task timeout, and strategy-specific guardrails.

For this milestone, Teams configuration keys MUST remain domain-scoped under `teams.*` namespace and MUST NOT overlap with workflow/a2a keys.

#### Scenario: Startup loads Teams defaults with environment override
- **WHEN** runtime starts with Teams config values in both YAML and environment variables
- **THEN** effective Teams configuration resolves with `env > file > default` and invalid values fail fast

### Requirement: Runtime diagnostics SHALL expose Teams run-level summary fields
Runtime diagnostics MUST include additive Teams fields for run summaries, including `team_id`, `team_strategy`, `team_task_total`, `team_task_failed`, and `team_task_canceled`.

#### Scenario: Consumer inspects team run summary
- **WHEN** application queries diagnostics for a run executed through Teams orchestration
- **THEN** diagnostics return normalized Teams summary fields without breaking existing run summary contracts

### Requirement: Runtime diagnostics SHALL preserve idempotent Teams aggregates under replay
Replayed or duplicated Teams events for the same run MUST NOT inflate logical team aggregate counters more than once.

#### Scenario: Duplicate team events are replayed
- **WHEN** a completed run replays the same Teams event stream
- **THEN** diagnostics keep stable `team_task_*` aggregates after first logical ingestion
