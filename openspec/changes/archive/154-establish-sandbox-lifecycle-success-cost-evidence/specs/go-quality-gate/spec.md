## ADDED Requirements

### Requirement: Quality gate SHALL enforce sandbox lifecycle-success-cost evidence contract checks
The quality gate MUST execute `check-sandbox-lifecycle-success-cost-evidence-contract.sh` and `check-sandbox-lifecycle-success-cost-evidence-contract.ps1` as blocking checks for this contract scope. The checks MUST validate lifecycle-success-cost replay fixtures, deterministic verdict and drift classifications, Run/Stream parity, privacy-field rejection, mixed-fixture compatibility, and repository boundaries that keep the evidence path reference-only.

The checks MUST NOT require OS sandbox privileges, launch a platform sandbox, use a network proxy, create an account, manage credentials, or depend on a live host or provider service.

#### Scenario: Evidence contract fixture diverges
- **WHEN** the lifecycle-success-cost contract check detects fixture, verdict, parity, or privacy classification drift
- **THEN** the quality gate exits non-zero and blocks completion with the deterministic failure classification

#### Scenario: Evidence gate runs in an unprivileged environment
- **WHEN** the contract check runs without platform sandbox privileges or a live sandbox backend
- **THEN** it validates the offline evidence contract and returns its deterministic result without attempting platform execution

### Requirement: Lifecycle-success-cost evidence gates SHALL preserve shell and PowerShell parity
For the same repository state, the shell and PowerShell lifecycle-success-cost evidence gate paths MUST execute equivalent assertions and produce equivalent pass or fail semantics.

#### Scenario: Both gate implementations validate the same fixture suite
- **WHEN** shell and PowerShell gates run against the same lifecycle-success-cost fixtures and boundary checks
- **THEN** both report equivalent pass or fail outcomes and no platform-specific sandbox dependency is introduced
