# workflow-branch-retry-failfast (production-ish)

## Purpose
用真实语义链路演示 `workflow-branch-retry-failfast` 的生产治理闭环：在 minimal 基础上增加治理门控与回放绑定。

## Variant Delta (vs minimal)
- 分支路由阈值更严格（更倾向 `safe-path`），同一输入下可能与 minimal 走不同路由。
- 重试预算按生产策略扩容并保留治理上下文，不仅是 marker 数量变化。
- 新增两步治理链路：
  - `governance_workflow_gate_enforced`：对 fail-fast 结果做 allow/deny/guardrails 仲裁并生成 ticket。
  - `governance_workflow_replay_bound`：将分支、预算、治理决策绑定 replay signature。

## Run
go run ./examples/agent-modes/workflow-branch-retry-failfast/production-ish

## Prerequisites
- Go 1.22+ and module dependencies resolved (`go mod tidy`).
- Writable local cache for Go build artifacts (`GOCACHE`).
- No external network service is required.

## Real Runtime Path
- Semantic anchor: `workflow.branch_retry_failfast`.
- Classification: `workflow.retry_failfast`.
- Runtime path evidence: `core/runner,tool/local,runtime/config,orchestration/workflow`.
- Semantic flow:
  - minimal 的 3 步基础链路；
  - 追加治理 gate 与 replay 绑定两步，形成 5 步闭环。
- Related contracts: `workflow-graph-composability-contract`.
- Required gates: `check-multi-agent-shared-contract.*`.
- Replay fixtures: `cross-domain-primary-reason-arbitration-contract.v1`.

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.phase=P1`
- `verification.semantic.anchor=workflow.branch_retry_failfast`
- `verification.semantic.classification=workflow.retry_failfast`
- `verification.semantic.runtime_path=core/runner,tool/local,runtime/config,orchestration/workflow`
- `verification.semantic.governance=enforced`
- `verification.semantic.expected_markers=workflow_branch_routed,workflow_retry_budgeted,workflow_failfast_classified,governance_workflow_gate_enforced,governance_workflow_replay_bound`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`
- `result.final_answer` 中包含 `governance=... ticket=... replay=...`，且与 minimal 的签名不同。

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If governance decision/replay output is unexpected, inspect `governance_workflow_gate_enforced` and `governance_workflow_replay_bound` marker output.
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.
