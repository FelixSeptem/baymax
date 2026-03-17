## ADDED Requirements

### Requirement: Model I/O filter deny outcomes SHALL map to S3 security-event taxonomy
Input/output filter deny outcomes MUST map to S3 security-event taxonomy with normalized `filter_stage`, `reason_code`, and `severity` fields.

#### Scenario: Input filter deny maps to S3 event
- **WHEN** input filter returns blocking deny decision
- **THEN** emitted S3 event includes `filter_stage=input` and normalized taxonomy fields

#### Scenario: Output filter deny maps to S3 event
- **WHEN** output filter returns blocking deny decision
- **THEN** emitted S3 event includes `filter_stage=output` and normalized taxonomy fields

### Requirement: Model I/O deny outcomes SHALL trigger deny-only callback alerts
I/O filtering deny outcomes MUST invoke registered callback alert sink and non-deny outcomes MUST NOT invoke callback.

#### Scenario: Output deny triggers callback alert
- **WHEN** output filter decision is `deny`
- **THEN** runtime dispatches callback alert with normalized S3 event payload

#### Scenario: Match outcome does not trigger callback alert
- **WHEN** filter decision is `match`
- **THEN** runtime records observability data but does not dispatch callback alert
