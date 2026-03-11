## MODIFIED Requirements

### Requirement: Quality gate SHALL include golangci-lint in standard verification flow
The standard validation flow MUST include `golangci-lint`, `go test ./...`, and `go test -race ./...` so style, correctness, and concurrency regressions are detected before merge.

#### Scenario: Validation in CI or local pre-merge checks
- **WHEN** a change is validated before merge
- **THEN** linter execution, unit tests, and race tests are all required checks and failures block completion

### Requirement: Concurrency safety SHALL be treated as a baseline quality requirement
Concurrency safety checks MUST be mandatory and cannot be bypassed in standard merge flow, including race detection and targeted concurrent diagnostics tests.

#### Scenario: Concurrency safety check fails
- **WHEN** race detection or required concurrent diagnostics tests fail
- **THEN** the change is rejected from merge until safety checks pass

## ADDED Requirements

### Requirement: Quality gate SHALL include diagnostics concurrency test coverage
The repository MUST maintain explicit tests for concurrent diagnostics writes, duplicate event replay, and idempotent persistence behavior.

#### Scenario: Diagnostics concurrency suite is executed
- **WHEN** diagnostics-focused concurrent tests run
- **THEN** write deduplication and data integrity guarantees are verified under parallel workloads