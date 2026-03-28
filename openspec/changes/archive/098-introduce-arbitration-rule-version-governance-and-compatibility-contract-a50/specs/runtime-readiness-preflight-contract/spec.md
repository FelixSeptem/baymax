## ADDED Requirements

### Requirement: Readiness preflight SHALL classify arbitration-version compatibility deterministically
Readiness preflight MUST evaluate arbitration version-governance compatibility and emit canonical findings for unsupported-version and compatibility-mismatch paths.

Readiness findings for version governance MUST remain machine-assertable and deterministic under equivalent inputs.

#### Scenario: Preflight detects unsupported arbitration rule version
- **WHEN** runtime preflight receives requested arbitration version that is not supported
- **THEN** readiness output includes canonical unsupported-version finding and deterministic blocking classification

#### Scenario: Preflight detects compatibility-window mismatch
- **WHEN** requested arbitration version is registered but outside configured compatibility window
- **THEN** readiness output includes canonical mismatch finding and deterministic classification aligned with policy

### Requirement: Readiness preflight SHALL expose arbitration-version explainability fields
Readiness preflight output MUST expose arbitration-version explainability fields that align with arbitration diagnostics:
- requested version,
- effective version,
- version source,
- policy action.

#### Scenario: Preflight uses default arbitration version
- **WHEN** preflight runs without per-request version override
- **THEN** readiness output includes effective default version and deterministic source metadata

#### Scenario: Preflight uses requested arbitration version
- **WHEN** preflight runs with supported requested version override
- **THEN** readiness output includes requested/effective version alignment without reclassification drift
