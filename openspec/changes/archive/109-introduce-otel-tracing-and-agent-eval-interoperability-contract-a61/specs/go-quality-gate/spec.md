## ADDED Requirements

### Requirement: Quality gate SHALL include tracing and eval interoperability contract checks
Standard quality gate flow MUST include tracing and eval interoperability contract checks as blocking validations.

Repository MUST provide:
- `scripts/check-agent-eval-and-tracing-interop-contract.sh`
- `scripts/check-agent-eval-and-tracing-interop-contract.ps1`

Shell and PowerShell implementations MUST preserve equivalent blocking semantics (`non-zero exit` => gate failure).

#### Scenario: Tracing and eval interop contract check fails
- **WHEN** gate detects OTel semconv or eval contract drift
- **THEN** quality gate exits non-zero and blocks merge

#### Scenario: Cross-platform parity for tracing and eval checks
- **WHEN** contributors run shell and PowerShell quality-gate flows
- **THEN** both flows execute equivalent tracing and eval interop checks with consistent pass/fail semantics

### Requirement: Tracing and eval gate SHALL be exposed as independent required-check candidate
CI workflow MUST expose tracing and eval interoperability validation as an independent status check suitable for branch-protection configuration.

#### Scenario: Maintainer configures branch protection for A61
- **WHEN** maintainer reviews available CI status checks
- **THEN** `agent-eval-tracing-interop-gate` appears as an independent candidate required check

### Requirement: Tracing and eval gate SHALL enforce control-plane absence boundary
Tracing and eval interoperability contract gate MUST enforce `control_plane_absent` assertion.

This assertion MUST fail when runtime introduces hosted eval control-plane dependency or service-based distributed execution dependency.

#### Scenario: Gate detects hosted evaluator dependency
- **WHEN** tracing and eval contract checks detect hosted evaluation control-plane dependency
- **THEN** gate exits non-zero and blocks merge

#### Scenario: Embedded distributed execution passes boundary assertion
- **WHEN** distributed execution remains library-embedded without hosted scheduler dependency
- **THEN** `control_plane_absent` assertion passes
