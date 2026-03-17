## ADDED Requirements

### Requirement: Runtime config SHALL expose Workflow baseline settings
Runtime configuration MUST expose workflow baseline settings with deterministic precedence `env > file > default`, including workflow enablement, planner validation mode, default step timeout, and checkpoint backend selector.

#### Scenario: Startup applies workflow config overrides
- **WHEN** workflow config is provided through both file and environment variables
- **THEN** runtime resolves effective workflow settings with `env > file > default` and rejects invalid configuration values

### Requirement: Runtime diagnostics SHALL expose workflow run summary fields
Runtime diagnostics MUST include additive workflow summary fields, including `workflow_id`, `workflow_status`, `workflow_step_total`, `workflow_step_failed`, and `workflow_resume_count`.

#### Scenario: Consumer queries workflow diagnostics
- **WHEN** application queries diagnostics for a workflow-enabled run
- **THEN** diagnostics return normalized workflow summary fields and preserve existing run summary compatibility

### Requirement: Workflow diagnostics SHALL remain idempotent under replay
Replay or duplicate workflow events for the same run MUST NOT increase logical workflow aggregates more than once.

#### Scenario: Workflow replay is ingested multiple times
- **WHEN** identical workflow timeline records are replayed for a completed run
- **THEN** diagnostics keep stable workflow aggregate counters after first logical write
