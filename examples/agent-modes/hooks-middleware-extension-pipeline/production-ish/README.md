# hooks-middleware-extension-pipeline (production-ish)

## Purpose
用真实语义链路演示 `hooks-middleware-extension-pipeline` 的生产治理闭环：在最小链路上增加治理门控与 replay 绑定。

## Variant Delta (vs minimal)
- 生产链路会增加 `security-hook`，middleware 深度提升，且更容易触发 `critical` bubble。
- 在 bubble/passthrough 结果后增加治理 gate，输出 `allow / allow_with_guardrails / deny`。
- 追加 replay 绑定，确保 hooks 仲裁结果可审计复放。

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
- Semantic flow:
  - minimal 的 3 步 middleware 链路；
  - 追加 `governance_hooks_gate_enforced` 与 `governance_hooks_replay_bound` 两步治理链路。
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
- `result.final_answer` 包含 `governance/ticket/replay`，且签名必须与 minimal 不同。

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If governance result is unexpected,检查 `bubble_severity`/`retryable` 与 gate 决策是否一致。
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.
