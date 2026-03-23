## MODIFIED Requirements

### Requirement: Quality gate SHALL include golangci-lint in standard verification flow
The standard validation flow MUST include `golangci-lint`, `go test ./...`, `go test -race ./...`, `govulncheck`, and mainline contract test suites so style, correctness, concurrency regressions, dependency vulnerability risks, and cross-module semantic regressions are detected before merge.

`govulncheck` MUST run in strict mode by default, and vulnerability findings MUST fail validation unless explicitly downgraded by controlled configuration.

For both shell and PowerShell gate implementations, each required check MUST propagate failure deterministically (non-zero exit). Quality gate MUST NOT continue with success reporting after an unhandled required-check failure.

#### Scenario: Validation in CI or local pre-merge checks
- **WHEN** a change is validated before merge
- **THEN** linter execution, unit tests, race tests, vulnerability scan, and required mainline contract tests are all required checks and failures block completion

#### Scenario: govulncheck finds vulnerabilities in strict mode
- **WHEN** validation runs with default strict scan mode and vulnerabilities are reported
- **THEN** quality gate exits non-zero and blocks merge

#### Scenario: Required native command fails in PowerShell gate
- **WHEN** a required command in `check-quality-gate.ps1` exits non-zero
- **THEN** the script exits non-zero deterministically and does not report overall gate success

### Requirement: Quality-gate scripts SHALL provide cross-platform security scan parity
Repository-provided quality-gate scripts for Linux and PowerShell MUST both execute the same vulnerability scan semantics as CI.

Cross-platform parity MUST include deterministic failure propagation semantics: equivalent check failures in shell and PowerShell MUST produce equivalent blocking outcomes.

#### Scenario: Linux and PowerShell scripts are executed
- **WHEN** contributors run quality-gate scripts on different platforms
- **THEN** both flows execute equivalent test/lint/race/vuln checks and produce consistent pass/fail semantics

#### Scenario: Docs consistency check fails under PowerShell flow
- **WHEN** `check-docs-consistency.ps1` detects status-parity drift or contract-doc mismatch
- **THEN** PowerShell gate returns non-zero and quality gate treats it as blocking failure
