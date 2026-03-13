## MODIFIED Requirements

### Requirement: Quality gate SHALL include golangci-lint in standard verification flow
The standard validation flow MUST include `golangci-lint`, `go test ./...`, `go test -race ./...`, `govulncheck`, and mainline contract test suites so style, correctness, concurrency regressions, dependency vulnerability risks, and cross-module semantic regressions are detected before merge.

`govulncheck` MUST run in strict mode by default, and vulnerability findings MUST fail validation unless explicitly downgraded by controlled configuration.

#### Scenario: Validation in CI or local pre-merge checks
- **WHEN** a change is validated before merge
- **THEN** linter execution, unit tests, race tests, vulnerability scan, and required mainline contract tests are all required checks and failures block completion

#### Scenario: govulncheck finds vulnerabilities in strict mode
- **WHEN** validation runs with default strict scan mode and vulnerabilities are reported
- **THEN** quality gate exits non-zero and blocks merge

## ADDED Requirements

### Requirement: Quality gate SHALL enforce repository hygiene checks
The standard validation flow MUST include repository hygiene checks that reject temporary backup artifacts and stale generated-by-accident files that are outside committed source-of-truth conventions.

#### Scenario: Temporary backup file is tracked
- **WHEN** repository hygiene checks detect files matching banned temporary patterns
- **THEN** validation fails and requires cleanup before merge

### Requirement: Mainline contract coverage SHALL be explicitly traceable
The repository MUST maintain a traceable mapping between required mainline flows and their corresponding contract test cases.

#### Scenario: Contributor reviews test coverage for a critical chain
- **WHEN** contributor inspects quality-gate documentation or test index
- **THEN** contributor can identify which contract test covers each required mainline flow
