## ADDED Requirements

### Requirement: Quality Gate SHALL Enforce Semantic Labeling Regression Checks
Standard quality-gate validation MUST execute semantic-labeling regression checks for active paths in both shell and PowerShell flows.

Checks MUST block reintroduction of legacy Context Assembler stage wording and MUST block `Axx` in any non-`openspec/**` content/path/file-name.

#### Scenario: Shell quality gate detects legacy naming regression
- **WHEN** `bash scripts/check-quality-gate.sh` detects forbidden naming patterns in governed active paths
- **THEN** gate MUST fail fast with non-zero exit and block merge

#### Scenario: PowerShell quality gate detects legacy naming regression
- **WHEN** `pwsh -File scripts/check-quality-gate.ps1` detects forbidden naming patterns in governed active paths
- **THEN** gate MUST fail with equivalent blocking semantics

### Requirement: Quality Gate SHALL Enforce Canonical Mapping Consistency
Validation MUST ensure semantic-to-legacy mapping is maintained in one canonical source and active docs do not maintain divergent duplicate mappings.

#### Scenario: Duplicate mapping appears in active documentation
- **WHEN** validation detects duplicated legacy-mapping definitions outside canonical source
- **THEN** gate MUST fail and require mapping consolidation

#### Scenario: Mapping source remains canonical and consistent
- **WHEN** all governed references resolve through canonical mapping source
- **THEN** quality gate passes mapping-consistency checks

### Requirement: Quality Gate SHALL Use Unified Governed-Path Matrix for Naming Scan
Naming-regression validation MUST consume one canonical governed-path matrix with this invariant:
- `openspec/**`: `Axx` allowed for historical traceability,
- non-`openspec/**`: `Axx` forbidden in content/path/file-name.

Shell and PowerShell scripts MUST use semantically equivalent matrix inputs to avoid drift between platforms.

#### Scenario: Shell and PowerShell use different governed-path matrices
- **WHEN** gate execution detects matrix mismatch across shell and PowerShell implementations
- **THEN** validation MUST fail and require matrix convergence

#### Scenario: Governed-path matrix is aligned
- **WHEN** shell and PowerShell naming scans run with the same matrix
- **THEN** gate semantics remain equivalent and deterministic

#### Scenario: Non-openspec file path or name contains Axx
- **WHEN** naming scan detects `A[0-9]{2,3}` in non-`openspec/**` path or file-name
- **THEN** quality gate MUST fail and block merge

#### Scenario: Non-openspec file content contains Axx
- **WHEN** naming scan detects `A[0-9]{2,3}` in non-`openspec/**` file content
- **THEN** quality gate MUST fail and block merge

### Requirement: Quality Gate SHALL Block Stale Temporary Asset Regression
Quality gate MUST reject stale temporary assets in active source/documentation surface, including accidental timestamp backup files and non-indexed offline scaffold bulk directories.

#### Scenario: Timestamp backup source file appears in active source tree
- **WHEN** validation detects source files matching accidental timestamp backup naming pattern
- **THEN** gate MUST fail and require cleanup

#### Scenario: Offline scaffold bulk directory lacks retention index
- **WHEN** validation finds offline scaffold bulk copies not covered by retained-sample policy
- **THEN** gate MUST fail until assets are removed or archived with index traceability

### Requirement: Quality Gate SHALL Enforce Single-File Code Size Budget
Quality gate MUST execute single-file line-budget checks for governed `*.go` files outside `openspec/**` in both shell and PowerShell flows.

Checks MUST enforce:
- hard threshold blocking,
- controlled exceptions from canonical exception list,
- debt non-expansion rule for already oversized files.

#### Scenario: Shell gate detects oversized code file
- **WHEN** `bash scripts/check-quality-gate.sh` detects a governed `*.go` file exceeding hard line threshold without valid exception
- **THEN** gate MUST fail fast and block merge

#### Scenario: PowerShell gate detects oversized code file
- **WHEN** `pwsh -File scripts/check-quality-gate.ps1` detects a governed `*.go` file exceeding hard line threshold without valid exception
- **THEN** gate MUST fail with equivalent blocking semantics

#### Scenario: Oversized-file exception is expired
- **WHEN** line-budget check finds exception entry past expiry date
- **THEN** gate MUST fail and require split or exception renewal review

### Requirement: Quality Gate SHALL Strongly Validate Semantic Equivalence for Go File Splits
When a change performs `*.go` file split/refactor for size governance, quality gate MUST treat it as semantic-preserving refactor and run strong equivalence checks.

Strong checks MUST include:
- Run/Stream parity suites,
- impacted contract suites for touched modules,
- diagnostics replay idempotency and drift-class stability.

Any failure MUST block merge (no soft-pass).

#### Scenario: Go split passes parity but fails replay stability
- **WHEN** `*.go` split change passes compile/tests but replay drift-class changes unexpectedly
- **THEN** quality gate MUST fail and block merge

#### Scenario: Go split strong checks are all green
- **WHEN** parity, impacted contracts, and replay stability checks all pass
- **THEN** gate may allow merge for split change

### Requirement: Consolidation Validation SHALL Preserve Contract and Replay Stability
A63 naming/documentation consolidation checks MUST run together with impacted contract/replay suites so semantic compatibility is continuously verified.

#### Scenario: Naming cleanup accidentally changes contract behavior
- **WHEN** impacted contract or replay suite detects semantic drift after consolidation edits
- **THEN** quality gate MUST fail and block merge even if naming scans pass

#### Scenario: Consolidation changes are behavior-neutral
- **WHEN** naming/documentation scans and impacted suites both pass
- **THEN** validation confirms consolidation is semantics-preserving
