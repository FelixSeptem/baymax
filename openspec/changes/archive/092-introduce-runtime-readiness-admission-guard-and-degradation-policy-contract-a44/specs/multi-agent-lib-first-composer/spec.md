## ADDED Requirements

### Requirement: Composer managed Run and Stream SHALL enforce readiness-admission guard when enabled
When readiness-admission is enabled in managed runtime configuration, composer MUST execute admission guard before starting Run or Stream orchestration path.

Admission deny MUST return deterministic error classification and MUST NOT mutate scheduler task state.

#### Scenario: Managed Run request denied by readiness admission
- **WHEN** composer Run entrypoint receives request under enabled admission and readiness maps to deny
- **THEN** composer returns admission-denied result and no child task is enqueued

#### Scenario: Managed Stream request denied by readiness admission
- **WHEN** composer Stream entrypoint receives equivalent deny condition
- **THEN** composer returns semantically equivalent admission-denied classification with no orchestration mutation

### Requirement: Composer readiness-admission behavior SHALL remain mode-equivalent
For equivalent configuration and runtime snapshot, readiness-admission outcomes in Run and Stream MUST remain semantically equivalent.

#### Scenario: Equivalent degraded allow-and-record behavior in Run and Stream
- **WHEN** degraded policy is `allow_and_record` and equivalent requests execute under Run and Stream
- **THEN** both paths allow execution and emit semantically equivalent admission observability fields

#### Scenario: Equivalent blocked fail-fast behavior in Run and Stream
- **WHEN** equivalent requests hit blocked readiness condition with admission enabled
- **THEN** both paths deny execution with semantically equivalent primary reason code
