## ADDED Requirements

### Requirement: Shared multi-agent gate SHALL include Task Board query contract suites
The shared multi-agent contract gate MUST execute Task Board query contract suites as blocking checks.

The gate MUST cover at least filter semantics, pagination/cursor determinism, invalid-input fail-fast behavior, and memory/file backend parity.

#### Scenario: Contributor runs shared multi-agent gate in shell
- **WHEN** contributor executes `scripts/check-multi-agent-shared-contract.sh`
- **THEN** Task Board query contract suites are executed as required blocking checks

#### Scenario: Contributor runs shared multi-agent gate in PowerShell
- **WHEN** contributor executes `scripts/check-multi-agent-shared-contract.ps1`
- **THEN** equivalent Task Board query contract suites are executed as required blocking checks
