# hooks-middleware-extension-pipeline (production-ish)

## Purpose
Real runtime semantic example for `hooks-middleware-extension-pipeline` with `production-ish` evidence profile.

## Run
go run ./examples/agent-modes/hooks-middleware-extension-pipeline/production-ish

## Prerequisites
- Go 1.22+ and module dependencies resolved (`go mod tidy`).
- Writable local cache for Go build artifacts (`GOCACHE`).
- No external network service is required.

## Real Runtime Path
- Semantic anchor: `middleware.onion_bubble_passthrough`.
- Classification: `hooks.middleware_pipeline`.
- Runtime path evidence: `core/runner,tool/local,runtime/config`.
- Related contracts: `agent-lifecycle-hooks-and-tool-middleware-contract`.
- Required gates: `check-hooks-middleware-contract.*`.
- Replay fixtures: `hooks_middleware.v1`.

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.phase=P1`
- `verification.semantic.anchor=middleware.onion_bubble_passthrough`
- `verification.semantic.classification=hooks.middleware_pipeline`
- `verification.semantic.runtime_path=core/runner,tool/local,runtime/config`
- `verification.semantic.governance=enforced`
- `verification.semantic.expected_markers=middleware_onion_order_verified,middleware_error_bubbled,middleware_extension_passthrough,governance_hooks_gate_enforced,governance_hooks_replay_bound`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.
