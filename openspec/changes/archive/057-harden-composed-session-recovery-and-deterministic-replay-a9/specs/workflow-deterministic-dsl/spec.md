## ADDED Requirements

### Requirement: Workflow checkpoint recovery SHALL compose with scheduler and A2A restore
Workflow resume semantics MUST compose with scheduler and A2A recovery so checkpoint-based continuation remains deterministic in composed runs.

#### Scenario: Workflow resume follows scheduler and A2A restore
- **WHEN** workflow resumes from checkpoint after composed recovery initialization
- **THEN** step scheduling and dependency execution remain deterministic and aligned with restored scheduler/A2A state

### Requirement: Workflow recovery replay SHALL preserve deterministic execution order
Recovered workflow execution MUST preserve deterministic ordering guarantees for resumed steps under equivalent inputs and effective config.

#### Scenario: Equivalent recovery replay is executed twice
- **WHEN** the same workflow recovery snapshot is replayed under equivalent conditions
- **THEN** resumed execution order and terminal category remain semantically equivalent
