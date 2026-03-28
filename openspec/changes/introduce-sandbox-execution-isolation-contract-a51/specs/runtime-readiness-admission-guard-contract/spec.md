## ADDED Requirements

### Requirement: Admission guard SHALL deny managed execution when required sandbox dependency is unavailable
Managed Run/Stream admission MUST deny execution when readiness reports required sandbox dependency as unavailable or invalid.

The deny path MUST remain side-effect free and preserve deterministic admission error classification.

#### Scenario: Managed run denied by required sandbox-unavailable finding
- **WHEN** admission guard receives blocked readiness primary reason indicating required sandbox dependency unavailable
- **THEN** admission decision is `deny` and scheduler/mailbox/task lifecycle state remains unchanged

#### Scenario: Managed stream denied by required sandbox-profile-invalid finding
- **WHEN** admission guard receives blocked readiness primary reason indicating sandbox profile invalid
- **THEN** admission decision is `deny` with semantically equivalent classification and no lifecycle mutation

#### Scenario: Managed run denied by sandbox capability mismatch
- **WHEN** admission guard receives blocked readiness primary reason indicating sandbox capability mismatch
- **THEN** admission decision is `deny` with deterministic capability-mismatch classification and no lifecycle mutation

### Requirement: Admission explainability SHALL preserve sandbox-related arbitration fields
Admission outputs for sandbox-driven deny/allow decisions MUST preserve canonical arbitration explainability fields without remapping drift.

#### Scenario: Sandbox-driven deny includes explainability payload
- **WHEN** admission denies execution due to sandbox-required readiness finding
- **THEN** output includes canonical primary reason and bounded explainability fields

#### Scenario: Equivalent sandbox-driven decisions in Run and Stream
- **WHEN** equivalent managed Run and Stream requests hit same sandbox admission outcome
- **THEN** outputs preserve semantically equivalent explainability and reason taxonomy
