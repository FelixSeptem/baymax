# a72 Acceptance Report

Date: 2026-04-11
Change: introduce-agent-mode-anti-template-doc-first-delivery-contract-a72

## 1. Scope Summary

- Completed mode-owned semantic implementation replacement for 28 `agent-modes` patterns across `minimal` and `production-ish`.
- Enforced doc-first delivery baseline (`MATRIX.md`, `PLAYBOOK.md`, per-variant README required sections) before mode implementation convergence.
- Added/connected A72 gate set into quality-gate path:
  - anti-template contract
  - doc-first delivery contract
  - migration playbook consistency checks

## 2. Key Fixes in This Round

- Fixed migration playbook consistency gate parser drift after matrix column expansion:
  - `scripts/check-agent-mode-migration-playbook-consistency.ps1`
  - `scripts/check-agent-mode-migration-playbook-consistency.sh`
- Changes:
  - parse `pattern/gates` by matrix header name instead of hard-coded column index;
  - fix PowerShell interpolation bug for `${pattern}:...` failure classification strings.
- Result: migration playbook consistency gate is now green.

## 3. Validation Results

### Agent-mode semantic/doc gates

- PASS: `pwsh -File scripts/check-agent-mode-examples-smoke.ps1`
- PASS: `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`
- PASS: `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`
- PASS: `pwsh -File scripts/check-agent-mode-anti-template-contract.ps1`
- PASS: `pwsh -File scripts/check-agent-mode-doc-first-delivery-contract.ps1`
- PASS: `pwsh -File scripts/check-agent-mode-migration-playbook-consistency.ps1`

### A72 gate-test evidence (T17/T18)

- PASS: `go test ./tool/contributioncheck -run 'TestAgentModeAntiTemplateContractExecutionPassFailAndExceptionalBranches|TestAgentModeDocFirstDeliveryContractExecutionPassFailAndExceptionalBranches|TestAgentModeA72ShellPowerShellParityOnClassifications' -count=1`
  - covers pass/fail classification and exceptional branch assertions for A72 new gates.
  - shell/PowerShell parity is executed with same inputs and same classification assertions.
- PASS: `go test -race ./tool/contributioncheck -count=1` (includes above tests).

### Code/test/lint baseline

- PASS: `go test ./...`
- PASS: `go test -race ./...` (executed with `GOFLAGS=-timeout=30m -p=2`)
- PASS: `golangci-lint run --config .golangci.yml`
- PASS: `pwsh -File scripts/check-docs-consistency.ps1`
- PASS: `go test ./integration -run 'TestAgentModeP0RegressionPaths|TestAgentModeP1RegressionPaths|TestAgentModeP2RegressionPaths' -count=1 -v`
- PASS: `go test ./integration -count=1`

### Full quality gate

- PASS (all pre-security strict checks): `pwsh -File scripts/check-quality-gate.ps1` reached final security scan stage.
  - A72 gate steps executed and passed in full run:
    - `agent mode pattern coverage`
    - `agent mode examples smoke`
    - `agent mode migration playbook consistency`
    - `agent mode real runtime semantic contract`
    - `agent mode readme runtime sync contract`
    - `agent mode anti-template contract`
    - `agent mode doc-first delivery contract`
- BLOCKED (strict security scan):
  - failure: `[native-strict] command failed: govulncheck ./... (exit=3)`
  - cause: local Go toolchain `go1.26.1` standard library vulnerability findings.
  - scanner-reported fixed version: `go1.26.2`.

## 4. Task Evidence Alignment

- Marked done with concrete evidence:
  - `A72-T17` gate test coverage (pass/fail/exception)
  - `A72-T18` shell/powershell parity runtime coverage
  - `A72-T60` P0 positive + degraded integration coverage
  - `A72-T61` P1 positive + degraded/failure integration coverage
  - `A72-T62` P2 positive + degraded/failure integration coverage
  - `A72-T63` smoke coverage evidence
  - `A72-T64` `go test` / `go test -race` / `golangci-lint` evidence
  - `A72-T65` docs consistency evidence
  - `A72-T66` full quality-gate execution evidence (A72 blocking path exercised in full pipeline)
  - `A72-T69` pre-archive self-check evidence:
    - PASS: `openspec validate --changes --strict --json` (A72 change valid, no schema issues)
    - PASS: `pwsh -File scripts/check-docs-consistency.ps1` (roadmap/archive status consistency green before archive)
  - `A72-T70` archive sequencing evidence:
    - PASS: `pwsh -File scripts/openspec-archive-seq.ps1 -ChangeName "introduce-agent-mode-anti-template-doc-first-delivery-contract-a72"` (OpenSpec archive flow executed)
    - PASS: `pwsh -File scripts/openspec-archive-seq.ps1` (archive naming normalized and `openspec/changes/archive/INDEX.md` regenerated)
    - PASS: archive directory finalized as `openspec/changes/archive/121-introduce-agent-mode-anti-template-doc-first-delivery-contract-a72`
    - Note: initial Windows ACL denied delete/rename on archive cleanup; resolved by ownership/ACL repair and then re-running archive sequencing.

All A72 tasks are complete.

## 5. Residual Risks and Next Closure Conditions

- Security strict gate residual:
  - upgrade Go toolchain from `go1.26.1` to at least `go1.26.2`;
  - rerun strict full `pwsh -File scripts/check-quality-gate.ps1` to clear final govulncheck blocker in release-quality report.
- Gate parity residual (`T17/T18`):
  - closed in this round by executable assertions in `tool/contributioncheck/agent_mode_example_pack_gate_test.go`.
