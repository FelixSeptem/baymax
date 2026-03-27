## ADDED Requirements

### Requirement: Admission guard SHALL enforce arbitration-version governance policy before execution
Runtime readiness-admission guard MUST evaluate arbitration-version governance outcomes before managed execution begins.

When version policy requires fail-fast (`on_unsupported=fail_fast` or `on_mismatch=fail_fast`), admission guard MUST deny execution deterministically.

#### Scenario: Admission guard denies unsupported-version request
- **WHEN** admission evaluates readiness result containing unsupported arbitration version finding under fail-fast policy
- **THEN** admission decision is `deny` and execution does not start

#### Scenario: Admission guard denies compatibility-mismatch request
- **WHEN** admission evaluates readiness result containing compatibility-mismatch finding under fail-fast policy
- **THEN** admission decision is `deny` with deterministic reason classification

### Requirement: Admission decision SHALL preserve arbitration-version explainability fields
Admission explanation fields MUST preserve arbitration-version explainability metadata without per-path remap:
- requested version,
- effective version,
- version source,
- policy action.

#### Scenario: Admission allow path preserves version metadata
- **WHEN** readiness result passes version-governance checks and admission decision is `allow`
- **THEN** admission explanation exposes deterministic arbitration-version metadata

#### Scenario: Admission deny path preserves version metadata
- **WHEN** readiness result fails version-governance checks and admission decision is `deny`
- **THEN** admission explanation exposes same canonical version metadata used by readiness/arbitration outputs
