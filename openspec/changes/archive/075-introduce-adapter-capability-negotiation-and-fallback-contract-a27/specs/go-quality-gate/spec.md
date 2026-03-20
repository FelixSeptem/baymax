## ADDED Requirements

### Requirement: Quality gate SHALL include adapter capability negotiation contract checks
The standard quality gate MUST execute adapter capability negotiation contract checks and treat failures as blocking.

This check MUST run in both shell and PowerShell quality-gate flows.

#### Scenario: Contributor runs shell quality gate
- **WHEN** contributor executes `bash scripts/check-quality-gate.sh`
- **THEN** adapter capability negotiation contract checks run as required blocking step

#### Scenario: Contributor runs PowerShell quality gate
- **WHEN** contributor executes `pwsh -File scripts/check-quality-gate.ps1`
- **THEN** equivalent adapter capability negotiation contract checks run as required blocking step

### Requirement: Capability negotiation gate SHALL fail fast on semantic drift
If required-capability fail-fast behavior, optional-downgrade behavior, strategy override semantics, or Run/Stream equivalence regresses, capability negotiation validation MUST fail fast and return deterministic non-zero status.

#### Scenario: Regression changes required-capability failure semantics
- **WHEN** contract checks detect required capability missing no longer fails deterministically
- **THEN** quality gate exits non-zero and blocks merge

#### Scenario: Regression changes Run/Stream negotiation equivalence
- **WHEN** contract checks detect negotiation outcome divergence between Run and Stream
- **THEN** quality gate exits non-zero and blocks merge
