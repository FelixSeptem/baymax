# runtime-readiness-admission-guard-contract Specification

## Purpose
TBD - created by archiving change introduce-runtime-readiness-admission-guard-and-degradation-policy-contract-a44. Update Purpose after archive.
## Requirements
### Requirement: Runtime SHALL provide readiness-admission guard before managed execution
Runtime MUST provide a readiness-admission guard that evaluates readiness status before managed Run/Stream execution begins.

Admission evaluation MUST consume readiness preflight result and MUST produce deterministic decision:
- `allow`
- `deny`

#### Scenario: Admission guard evaluates blocked readiness
- **WHEN** readiness preflight result is `blocked` and admission feature is enabled
- **THEN** admission decision is `deny` and managed execution does not start

#### Scenario: Admission guard evaluates ready readiness
- **WHEN** readiness preflight result is `ready` and admission feature is enabled
- **THEN** admission decision is `allow` and managed execution can proceed

### Requirement: Readiness-admission deny path SHALL be side-effect free
When admission decision is `deny`, runtime MUST fail fast before any scheduler enqueue, mailbox publish, or task lifecycle mutation.

#### Scenario: Deny path rejects run without task mutation
- **WHEN** admission guard returns `deny` for a managed run request
- **THEN** runtime returns deterministic admission error and scheduler/mailbox state remains unchanged

#### Scenario: Equivalent deny path under Run and Stream
- **WHEN** equivalent requests under Run and Stream both hit admission deny
- **THEN** both paths return semantically equivalent admission classification with no lifecycle side effects

### Requirement: Degraded readiness SHALL support policy-controlled admission
Runtime MUST support policy-controlled handling for `degraded` readiness:
- `allow_and_record`
- `fail_fast`

#### Scenario: Degraded readiness with allow-and-record policy
- **WHEN** readiness result is `degraded` and degraded policy is `allow_and_record`
- **THEN** admission allows execution and records degraded-admission observability markers

#### Scenario: Degraded readiness with fail-fast policy
- **WHEN** readiness result is `degraded` and degraded policy is `fail_fast`
- **THEN** admission denies execution with deterministic degraded-admission reason classification

### Requirement: Admission guard SHALL consume arbitration-aligned primary reason without reclassification drift
Runtime readiness admission guard MUST consume primary reason output from cross-domain arbitration without introducing per-path reclassification drift.

Admission decision explanation fields MUST preserve:
- primary domain,
- primary code,
- primary source.

#### Scenario: Admission deny consumes blocked primary reason
- **WHEN** admission guard receives blocked-class primary reason from arbitration
- **THEN** deny decision explanation preserves the same primary domain/code/source semantics

#### Scenario: Admission allow-and-record consumes degraded primary reason
- **WHEN** admission guard receives degraded-class primary reason under allow-and-record policy
- **THEN** allow decision explanation preserves arbitration primary reason without remapping

### Requirement: Admission guard SHALL preserve arbitration explainability semantics
Admission guard decisions MUST preserve arbitration explainability semantics without per-path remapping drift.

Admission explanation MUST include:
- primary reason fields,
- bounded secondary reason fields,
- remediation hint fields.

#### Scenario: Admission deny path includes explainability output
- **WHEN** admission guard denies execution using arbitration result
- **THEN** deny explanation preserves canonical primary and secondary explainability fields

#### Scenario: Admission allow-and-record path includes explainability output
- **WHEN** admission guard allows degraded execution with record policy
- **THEN** allow explanation preserves canonical explainability fields without reclassification drift

