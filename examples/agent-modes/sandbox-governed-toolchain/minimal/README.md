# sandbox-governed-toolchain (minimal)

## Purpose
Real runtime semantic example for `sandbox-governed-toolchain` with `minimal` evidence profile.
This variant executes a concrete sandbox chain: tool allow/deny classification, egress allowlist check, and fallback path emission.

## Run
go run ./examples/agent-modes/sandbox-governed-toolchain/minimal

## Prerequisites
- Go 1.22+ and module dependencies resolved (`go mod tidy`).
- Writable local cache for Go build artifacts (`GOCACHE`).
- No external network service is required.

## Real Runtime Path
- Semantic anchor: `sandbox.allow_deny_egress_fallback`.
- Classification: `sandbox.toolchain_governance`.
- Runtime path evidence: `core/runner,tool/local,runtime/config,runtime/security`.
- Related contracts: `security-sandbox-contract`.
- Required gates: `check-security-sandbox-contract.*; check-sandbox-egress-allowlist-contract.*`.
- Replay fixtures: `sandbox_egress.v1`.

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.phase=P0`
- `verification.semantic.anchor=sandbox.allow_deny_egress_fallback`
- `verification.semantic.classification=sandbox.toolchain_governance`
- `verification.semantic.runtime_path=core/runner,tool/local,runtime/config,runtime/security`
- `verification.semantic.governance=baseline`
- `verification.semantic.expected_markers=sandbox_allow_deny_classified,sandbox_egress_allowlist_checked,sandbox_fallback_path_emitted`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If allow/deny or fallback output is unexpected, inspect policy/request fixtures in `semantic_example.go`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.
