# react-plan-notebook-loop (minimal)

## Purpose
用真实语义链路演示 `react-plan-notebook-loop` 的最小闭环：plan/notebook 同步、change hook 触发、tool loop 收敛。

## Run
go run ./examples/agent-modes/react-plan-notebook-loop/minimal

## Prerequisites
- Go 1.22+ and module dependencies resolved (`go mod tidy`).
- Writable local cache for Go build artifacts (`GOCACHE`).
- No external network service is required.

## Real Runtime Path
- Semantic anchor: `react.plan_notebook_change_hooks`.
- Classification: `react.plan_notebook_loop`.
- Runtime path evidence: `core/runner,tool/local,runtime/config,runtime/diagnostics`.
- Semantic flow:
  - `react_plan_notebook_synced`: 对齐计划与笔记本状态并产出 `pending_steps/notebook_digest`。
  - `react_change_hook_emitted`: 基于 pending 与 drift 输出 `change_hook_type`。
  - `react_tool_loop_closed`: 按 hook 和 pending 选择下一步工具动作并判断是否闭环。
- Related contracts: `react-plan-notebook-and-plan-change-hook-contract`.
- Required gates: `check-react-plan-notebook-contract.*`.
- Replay fixtures: `react_plan_notebook.v1`.

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.phase=P1`
- `verification.semantic.anchor=react.plan_notebook_change_hooks`
- `verification.semantic.classification=react.plan_notebook_loop`
- `verification.semantic.runtime_path=core/runner,tool/local,runtime/config,runtime/diagnostics`
- `verification.semantic.governance=baseline`
- `verification.semantic.expected_markers=react_plan_notebook_synced,react_change_hook_emitted,react_tool_loop_closed`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`
- `result.final_answer` 包含 `pending/hook/action/loop_closed` 等真实交互字段。

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If loop action is unexpected,优先核对 `pending_steps` 与 `change_hook_type` 的对应关系。
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.
