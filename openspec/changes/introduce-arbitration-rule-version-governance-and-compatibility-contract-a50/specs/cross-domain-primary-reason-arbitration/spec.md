## ADDED Requirements

### Requirement: Primary-reason arbitration SHALL include version-governance traceability
Cross-domain primary-reason arbitration MUST include version-governance traceability fields for each arbitration decision:
- requested rule version,
- effective rule version,
- rule-version source,
- policy action.

These fields MUST remain deterministic for equivalent inputs and equivalent configuration.

#### Scenario: Arbitration uses caller-requested version
- **WHEN** caller provides supported requested arbitration version
- **THEN** arbitration output preserves canonical primary reason and includes requested/effective/source version fields

#### Scenario: Arbitration falls back to runtime default version
- **WHEN** caller does not provide requested arbitration version
- **THEN** arbitration output includes effective default version and deterministic source classification

### Requirement: Primary-reason arbitration SHALL enforce version-policy fail-fast semantics
When arbitration version policy is configured as fail-fast, arbitration MUST reject unsupported or compatibility-mismatch versions before producing primary reason output.

#### Scenario: Unsupported version is requested
- **WHEN** requested version is not in runtime arbitration rule registry and `on_unsupported=fail_fast`
- **THEN** arbitration returns deterministic unsupported-version failure without emitting non-canonical primary reason

#### Scenario: Compatibility-mismatch version is requested
- **WHEN** requested version exists but violates configured compatibility window and `on_mismatch=fail_fast`
- **THEN** arbitration returns deterministic mismatch failure and does not silently downgrade to another version
