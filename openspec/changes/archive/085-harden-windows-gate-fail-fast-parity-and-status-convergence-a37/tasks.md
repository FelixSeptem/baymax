## 1. Baseline and Strict-Execution Foundation

- [x] 1.1 Audit `scripts/check-*.ps1` native command invocation paths and mark fail-fast gaps.
- [x] 1.2 Introduce a shared strict native execution helper for PowerShell gate scripts (non-zero -> deterministic throw/exit).
- [x] 1.3 Define and document the single governance exception path (`govulncheck` in warn mode only).

## 2. PowerShell Gate Fail-Fast Rollout

- [x] 2.1 Refactor `scripts/check-docs-consistency.ps1` to use strict native execution and remove false-pass behavior.
- [x] 2.2 Refactor `scripts/check-quality-gate.ps1` required checks to enforce deterministic fail-fast semantics.
- [x] 2.3 Refactor `scripts/check-multi-agent-shared-contract.ps1` required checks to enforce deterministic fail-fast semantics.
- [x] 2.4 Refactor adapter/security PowerShell gate scripts (`check-adapter-*.ps1`, `check-security-*.ps1`) to the same strict execution path.

## 3. Status-Parity Convergence

- [x] 3.1 Align `README.md` milestone snapshot with OpenSpec authority status (A35 archived, A36 active).
- [x] 3.2 Align `docs/development-roadmap.md` in-progress/archived entries with OpenSpec authority status.
- [x] 3.3 Ensure docs consistency flow fails on status-parity drift and does not emit pass logs after failure.

## 4. Contract and Gate Regression Coverage

- [x] 4.1 Add or update contributioncheck tests to cover status-parity drift detection in current authority state.
- [x] 4.2 Add or update gate regression tests/fixtures to cover PowerShell native-command failure propagation.
- [x] 4.3 Add guard assertions preventing key gate scripts from bypassing strict native execution paths.
- [x] 4.4 Update `docs/mainline-contract-test-index.md` with A37 fail-fast parity and status-convergence mappings.

## 5. Documentation Sync

- [x] 5.1 Update roadmap and README governance wording to reflect convergence-phase gate hardening.
- [x] 5.2 Update related module/readme entries for gate execution semantics where referenced.
- [x] 5.3 Verify docs and script descriptions keep shell/PowerShell parity semantics consistent.

## 6. Validation

- [x] 6.1 Run `go test ./tool/contributioncheck -run '^(TestReleaseStatusParityDocsConsistency|TestValidateStatusParityDetectsConflict)$' -count=1`.
- [x] 6.2 Run `pwsh -File scripts/check-docs-consistency.ps1`.
- [x] 6.3 Run `pwsh -File scripts/check-multi-agent-shared-contract.ps1`.
- [x] 6.4 Run `pwsh -File scripts/check-quality-gate.ps1`.
- [x] 6.5 Run `go test ./...`.
- [x] 6.6 Run `go test -race ./...`.
- [x] 6.7 Run `golangci-lint run --config .golangci.yml`.
- [x] 6.8 Run `openspec validate harden-windows-gate-fail-fast-parity-and-status-convergence-a37 --strict`.
