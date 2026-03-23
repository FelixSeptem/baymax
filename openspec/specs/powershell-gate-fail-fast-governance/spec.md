# powershell-gate-fail-fast-governance Specification

## Purpose
TBD - created by archiving change harden-windows-gate-fail-fast-parity-and-status-convergence-a37. Update Purpose after archive.
## Requirements
### Requirement: PowerShell gate scripts SHALL execute native commands with strict fail-fast semantics
Repository-provided PowerShell gate scripts MUST execute native commands through a strict execution path that treats any non-zero exit code as blocking failure.

This strict path MUST produce deterministic non-zero process exit and MUST include command context in error output.

#### Scenario: Native test command fails in PowerShell gate
- **WHEN** a gate script executes a native command (for example `go test`) and the command exits non-zero
- **THEN** the script terminates with deterministic non-zero exit and does not continue to subsequent success logs

#### Scenario: Native lint command fails in PowerShell gate
- **WHEN** a gate script executes `golangci-lint` and lint exits non-zero
- **THEN** the script reports failure context and blocks completion

### Requirement: PowerShell strict-failure policy SHALL allow only explicit governance exceptions
PowerShell gate scripts MUST default to strict blocking failure semantics for all native commands.

If a non-blocking policy is required, it MUST be explicitly governed and limited to documented exceptions.

#### Scenario: Security scan runs in strict mode
- **WHEN** quality gate executes vulnerability scan with strict mode
- **THEN** vulnerability findings cause deterministic non-zero gate failure

#### Scenario: Security scan runs in warn mode
- **WHEN** quality gate executes vulnerability scan with `warn` policy configured by governance
- **THEN** warnings are emitted without changing strict-failure semantics for other gate commands

