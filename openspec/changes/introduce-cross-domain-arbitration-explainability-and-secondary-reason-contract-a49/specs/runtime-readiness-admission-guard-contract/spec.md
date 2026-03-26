## ADDED Requirements

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
