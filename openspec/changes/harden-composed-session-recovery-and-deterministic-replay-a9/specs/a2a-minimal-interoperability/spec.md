## ADDED Requirements

### Requirement: A2A in-flight state SHALL be included in recovery model
A2A interoperability contract MUST include in-flight task state in composed recovery snapshots to preserve remote collaboration continuity.

#### Scenario: Recovery resumes with pending A2A task
- **WHEN** composed recovery snapshot contains A2A in-flight task not yet terminal
- **THEN** recovery restores A2A task correlation and continues terminal convergence without creating duplicate logical tasks

### Requirement: Recovered A2A replay SHALL preserve error-layer normalization
Recovered A2A task replay MUST preserve existing error-layer normalization and reason taxonomy semantics.

#### Scenario: Recovery replays failed A2A terminal state
- **WHEN** recovered A2A failure is replayed into composed runtime
- **THEN** error layer and canonical reason mapping remain consistent with non-recovery execution paths
