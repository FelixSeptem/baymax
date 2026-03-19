## ADDED Requirements

### Requirement: Shared multi-agent gate SHALL include A12/A13 cross-mode closure matrix suites
Quality gate MUST include blocking cross-mode contract suites that cover sync/async/delayed communication semantics under Run/Stream and required qos/recovery key paths.

#### Scenario: CI executes shared gate after A14 closure
- **WHEN** CI runs `check-multi-agent-shared-contract.*`
- **THEN** required A12/A13 cross-mode suites execute as blocking checks in the shared gate path

### Requirement: Shared gate SHALL block async and delayed reason-taxonomy drift
Quality gate contract checks MUST fail when required A12/A13 canonical reason taxonomy is incomplete or renamed without synchronized contract updates.

#### Scenario: Contract drift removes delayed reason from shared snapshot
- **WHEN** delayed canonical reason is missing from shared contract snapshot validation
- **THEN** gate fails with explicit taxonomy-drift classification and blocks merge

### Requirement: Mainline contract index SHALL trace A12/A13 closure matrix coverage
Repository contract index MUST map A12/A13 closure matrix flows to concrete test entries and remain synchronized with gate suites.

#### Scenario: Contributor audits A12/A13 closure coverage
- **WHEN** contributor reviews mainline contract index and shared-gate suites
- **THEN** each required cross-mode matrix row has traceable test mapping and no missing index entry
