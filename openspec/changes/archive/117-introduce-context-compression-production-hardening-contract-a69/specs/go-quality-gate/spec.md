## ADDED Requirements

### Requirement: Quality Gate SHALL Include A69 Context Compression Contract Checks
Standard quality gate MUST execute A69 context compression production contract checks as blocking validations in both shell and PowerShell paths.

Required commands:
- `scripts/check-context-compression-production-contract.sh`
- `scripts/check-context-compression-production-contract.ps1`

#### Scenario: Shell quality gate executes A69 contract checks
- **WHEN** contributor runs shell quality gate on A69-impacted changes
- **THEN** A69 contract checks run as required blocking steps and fail fast on non-zero exit

#### Scenario: PowerShell quality gate executes A69 contract checks
- **WHEN** contributor runs PowerShell quality gate on A69-impacted changes
- **THEN** equivalent A69 contract checks run with the same blocking semantics

### Requirement: A69 Gate SHALL Enforce Replay and Benchmark Regression Suites for Impacted Context Paths
A69 gate execution MUST enforce replay suites and context benchmark regression suites when context compression hotpaths are touched.

At minimum, impacted validation MUST include:
- diagnostics replay suites for A69 fixture taxonomy,
- `check-context-production-hardening-benchmark-regression.sh/.ps1`,
- impacted contract suites mapped from touched context/runtime modules.

#### Scenario: A69 impacted change omits required suite
- **WHEN** changed-file mapping indicates A69 impacted suites but required replay/benchmark suite is missing
- **THEN** quality gate fails and blocks merge

#### Scenario: Benchmark regression threshold breach blocks merge
- **WHEN** A69 context benchmark regression suite exceeds configured thresholds
- **THEN** quality gate exits non-zero and blocks merge

### Requirement: A69 Gate SHALL Preserve Shell PowerShell Parity
A69 gate pass/fail semantics MUST be equivalent between shell and PowerShell for the same repository state and fixtures.

#### Scenario: Equivalent failure under shell and PowerShell
- **WHEN** A69 contract or replay validation fails in one gate path
- **THEN** the other gate path fails with equivalent blocking outcome
