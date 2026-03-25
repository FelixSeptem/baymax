## ADDED Requirements

### Requirement: Quality gate SHALL include diagnostics-cardinality contract suites
The standard quality gate MUST execute diagnostics-cardinality contract suites as blocking checks in both shell and PowerShell workflows.

The suites MUST cover at minimum:
- diagnostics cardinality config validation fail-fast and hot-reload rollback behavior,
- overflow policy semantics (`truncate_and_record` and `fail_fast`),
- deterministic truncation output semantics,
- Run/Stream truncation equivalence,
- replay-idempotent cardinality aggregate behavior.

#### Scenario: Contributor runs shell quality gate
- **WHEN** contributor executes `bash scripts/check-quality-gate.sh`
- **THEN** diagnostics-cardinality contract suites run as required blocking checks

#### Scenario: Contributor runs PowerShell quality gate
- **WHEN** contributor executes `pwsh -File scripts/check-quality-gate.ps1`
- **THEN** equivalent diagnostics-cardinality contract suites run as required blocking checks

### Requirement: Quality gate SHALL block merge on diagnostics-cardinality semantic drift
When diagnostics-cardinality suites detect non-deterministic truncation output, non-canonical overflow policy behavior, or replay-idempotency regressions, quality gate MUST fail and block merge.

#### Scenario: Regression changes truncation output ordering
- **WHEN** contract suites detect equivalent payloads produce different truncated field summaries
- **THEN** quality gate exits non-zero and blocks validation

#### Scenario: Diagnostics-cardinality semantics remain aligned
- **WHEN** diagnostics-cardinality suites pass canonical semantic assertions
- **THEN** quality gate proceeds without diagnostics-cardinality failure
