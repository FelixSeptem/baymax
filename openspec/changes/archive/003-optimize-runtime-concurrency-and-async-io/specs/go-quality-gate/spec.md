## MODIFIED Requirements

### Requirement: Quality gate SHALL include golangci-lint in standard verification flow
The standard validation flow MUST include `golangci-lint` and concurrency safety checks so regressions are detected before merge.

#### Scenario: Validation in CI or local pre-merge checks
- **WHEN** a change is validated before merge
- **THEN** linter execution and `go test -race ./...` are both part of required checks and failures block completion

### Requirement: Lint profile SHALL align with Go style and safety priorities
The configured quality profile MUST enforce formatting/import conventions, detect common correctness risks, and include concurrency safety auditing practices.

#### Scenario: Code violates configured style or safety rules
- **WHEN** code introduces issues covered by enabled linters or race detection
- **THEN** validation output reports actionable diagnostics tied to file and line locations

## ADDED Requirements

### Requirement: Performance regression gate SHALL use relative percentage thresholds
Performance validation MUST evaluate benchmark outcomes using relative percentage change against a documented baseline.

#### Scenario: Benchmark comparison is executed
- **WHEN** benchmark results are compared for a candidate change
- **THEN** acceptance is decided by relative percentage thresholds for throughput and latency metrics

### Requirement: Concurrency safety SHALL be treated as a baseline quality requirement
Concurrency safety checks MUST be mandatory and cannot be bypassed in standard merge flow.

#### Scenario: Concurrency safety check fails
- **WHEN** race detection or goroutine-leak validation fails
- **THEN** the change is rejected from merge until safety checks pass
