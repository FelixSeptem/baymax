## ADDED Requirements

### Requirement: State Snapshot Contract Gate Integration
Quality gate MUST include `check-state-snapshot-contract.sh/.ps1` and MUST fail fast on contract suite failure in both shell and PowerShell paths.

#### Scenario: Shell gate fail-fast
- **WHEN** `check-state-snapshot-contract.sh` exits non-zero
- **THEN** `check-quality-gate.sh` MUST fail immediately without soft fallback

#### Scenario: PowerShell gate fail-fast parity
- **WHEN** `check-state-snapshot-contract.ps1` exits non-zero
- **THEN** `check-quality-gate.ps1` MUST fail with equivalent blocking semantics

### Requirement: Snapshot Impacted Suite Enforcement
Gate execution MUST enforce impacted suites for A66 scope changes and MUST reject merges when required contract/replay suites are missing.

#### Scenario: Recovery scope requires shared multi-agent suites
- **WHEN** A66 changes touch scheduler/composer recovery and session restore paths
- **THEN** gate MUST require corresponding shared multi-agent contract suites before merge

#### Scenario: Snapshot replay scope requires replay suites
- **WHEN** A66 changes touch diagnostics replay fixture or drift classification logic
- **THEN** gate MUST require replay contract suites before merge
