# a71 Acceptance Report

Date: 2026-04-09
Change: introduce-real-runtime-agent-mode-examples-contract-a71

## 1. Scope Summary

- Completed per-mode semantic ownership migration for all 28 agent-mode patterns.
- Both `minimal` and `production-ish` variants use mode-owned semantic implementations.
- Removed shared entrypoint usage (`runtimeexample.MustRun(pattern, variant)`) from mode `main.go` files.
- Added shared helper package only for non-semantic common concerns.

## 2. Implementation Evidence

- Mode-owned semantic files: `examples/agent-modes/<pattern>/semantic_example.go` (28 files).
- Variant entrypoints: `examples/agent-modes/<pattern>/{minimal,production-ish}/main.go` (56 files) now call mode-owned `RunMinimal/RunProduction`.
- Shared non-semantic helper: `examples/agent-modes/internal/modecommon/common.go`.
- A71 gate update:
  - `scripts/check-agent-mode-real-runtime-semantic-contract.ps1`
  - `scripts/check-agent-mode-real-runtime-semantic-contract.sh`
  - Added failure classes:
    - `agent-mode-shared-semantic-engine-detected`
    - `agent-mode-semantic-ownership-missing`
    - `agent-mode-missing-runtime-path-evidence`

## 3. Validation Results

### A71-specific gate results

- PASS: `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`
- PASS: `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`
- PASS: `pwsh -File scripts/check-agent-mode-examples-smoke.ps1`
  - Coverage: 28 patterns * 2 variants = 56 runs
  - Required semantic evidence tokens detected for all runs
  - Variant distinction checks passed (expected markers, governance, signature divergence)

### Code/Test/Lint results

- PASS: `go test ./examples/... -count=1`
- PASS: `go test ./tool/contributioncheck -count=1`
- PASS: `go test ./...`
- PASS: `go test -race ./...`
- PASS: `golangci-lint run --config .golangci.yml`
- PASS: `pwsh -File scripts/check-docs-consistency.ps1`

### Quality gate

- PASS: `BAYMAX_QUALITY_GATE_SCOPE=general pwsh -File scripts/check-quality-gate.ps1`
- PASS: `BAYMAX_QUALITY_GATE_TOTAL_TIMEOUT_SECONDS=2400 BAYMAX_QUALITY_GATE_STEP_TIMEOUT_SECONDS=1800 BAYMAX_SECURITY_SCAN_MODE=warn pwsh -File scripts/check-quality-gate.ps1` (`scope=full`)
- BLOCKED (non-a71 blocker): `BAYMAX_QUALITY_GATE_TOTAL_TIMEOUT_SECONDS=2400 BAYMAX_QUALITY_GATE_STEP_TIMEOUT_SECONDS=1800 pwsh -File scripts/check-quality-gate.ps1` (`scope=full`, strict)
  - Failure class: `[native-strict] command failed: govulncheck ./... (exit=3)`
  - Cause: Go stdlib vulnerabilities detected on local toolchain `go1.26.1`
  - Affected packages: `crypto/x509`, `crypto/tls`, `html/template`
  - Upstream fix version reported by scanner: `go1.26.2`

## 4. Failure Classification Summary

- `agent-mode-shared-semantic-engine-detected`: no hit after migration
- `agent-mode-semantic-ownership-missing`: no hit after migration
- `agent-mode-missing-runtime-path-evidence`: no hit after migration
- strict full-scope `govulncheck` blocker: hit on Go stdlib vulnerability scan only

## 5. Coverage Matrix Summary

- Pattern coverage: 28 / 28
- Variant coverage: 56 / 56
- Runtime path evidence: present for all patterns in smoke output and matrix mapping
- README required sections: validated by readme-runtime-sync gate

## 6. Residual Risk and Follow-up

- Residual item for full closure:
  - Upgrade local/CI Go toolchain from `go1.26.1` to at least `go1.26.2`, then re-run strict `pwsh -File scripts/check-quality-gate.ps1`.
- This residual does not indicate a71 semantic regression; it is a strict security scan blocker from Go stdlib CVEs on the local toolchain version.
