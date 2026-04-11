# sandbox-governed-toolchain (production-ish)

## Purpose
Real runtime semantic example for `sandbox-governed-toolchain` with `production-ish` evidence profile.

## Variant Delta (vs minimal)
- Reuses the same semantic anchor and runtime path baseline as minimal.
- Adds `governance_sandbox_gate_enforced`: classify sandbox result into `allow|allow_with_record|block`.
- Adds `governance_sandbox_replay_bound`: bind replay signature from policy version and decision tuple.
- Preserves minimal allow/deny/egress/fallback chain and appends governance enforcement.
- Requires verification.semantic.governance=enforced.
- Requires verification.semantic.expected_markers and result.signature to differ from minimal.

## Run
go run ./examples/agent-modes/sandbox-governed-toolchain/production-ish

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
- `verification.semantic.governance=enforced`
- `verification.semantic.expected_markers=sandbox_allow_deny_classified,sandbox_egress_allowlist_checked,sandbox_fallback_path_emitted,governance_sandbox_gate_enforced,governance_sandbox_replay_bound`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If governance/replay output is unexpected, inspect `governance_sandbox_*` branches in `semantic_example.go`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.


