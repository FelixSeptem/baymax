# react-plan-notebook-loop (production-ish)

## Purpose
用真实语义链路演示 `react-plan-notebook-loop` 的生产治理闭环：在最小链路上增加 gate 与 replay 绑定。

## Variant Delta (vs minimal)
- 生产场景会将 `change_hook_type` 提升为 `guardrail_patch`，从而触发不同的 loop 行为。
- 在 loop 输出后增加治理决策：`allow / allow_with_guardrails / deny`。
- 追加 replay 绑定，确保 plan/notebook 仲裁结果可审计复放。

## Run
go run ./examples/agent-modes/react-plan-notebook-loop/production-ish

## Prerequisites
- Go 1.22+ and module dependencies resolved (`go mod tidy`).
- Writable local cache for Go build artifacts (`GOCACHE`).
- No external network service is required.

## Real Runtime Path
- Semantic anchor: `react.plan_notebook_change_hooks`.
- Classification: `react.plan_notebook_loop`.
- Runtime path evidence: `core/runner,tool/local,runtime/config,runtime/diagnostics`.
- Semantic flow:
  - minimal 的 3 步交互链路；
  - 追加 `governance_react_gate_enforced` 与 `governance_react_replay_bound` 两步治理链路。
- Related contracts: `react-plan-notebook-and-plan-change-hook-contract`.
- Required gates: `check-react-plan-notebook-contract.*`.
- Replay fixtures: `react_plan_notebook.v1`.

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.phase=P1`
- `verification.semantic.anchor=react.plan_notebook_change_hooks`
- `verification.semantic.classification=react.plan_notebook_loop`
- `verification.semantic.runtime_path=core/runner,tool/local,runtime/config,runtime/diagnostics`
- `verification.semantic.governance=enforced`
- `verification.semantic.expected_markers=react_plan_notebook_synced,react_change_hook_emitted,react_tool_loop_closed,governance_react_gate_enforced,governance_react_replay_bound`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`
- `result.final_answer` 包含 `governance/ticket/replay`，且签名必须与 minimal 不同。

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If governance decision is unexpected,检查 `tool_action` 与 `change_hook_type` 是否匹配 gate 结果。
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.
