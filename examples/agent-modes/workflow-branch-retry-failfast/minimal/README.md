# workflow-branch-retry-failfast (minimal)

## Purpose
用真实语义链路演示 `workflow-branch-retry-failfast` 的最小闭环：分支路由、重试预算、fail-fast 分类。

## Run
go run ./examples/agent-modes/workflow-branch-retry-failfast/minimal

## Prerequisites
- Go 1.22+ and module dependencies resolved (`go mod tidy`).
- Writable local cache for Go build artifacts (`GOCACHE`).
- No external network service is required.

## Real Runtime Path
- Semantic anchor: `workflow.branch_retry_failfast`.
- Classification: `workflow.retry_failfast`.
- Runtime path evidence: `core/runner,tool/local,runtime/config,orchestration/workflow`.
- Semantic flow:
  - `workflow_branch_routed`: 根据 `error_ratio + latency_slo_ms` 在 `fast-path/safe-path` 间路由。
  - `workflow_retry_budgeted`: 按分支与瞬时失败数计算预算与消耗。
  - `workflow_failfast_classified`: 按预算耗尽/错误类型输出 `failfast_class` 与 `escalation_lane`。
- Related contracts: `workflow-graph-composability-contract`.
- Required gates: `check-multi-agent-shared-contract.*`.
- Replay fixtures: `cross-domain-primary-reason-arbitration-contract.v1`.

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.phase=P1`
- `verification.semantic.anchor=workflow.branch_retry_failfast`
- `verification.semantic.classification=workflow.retry_failfast`
- `verification.semantic.runtime_path=core/runner,tool/local,runtime/config,orchestration/workflow`
- `verification.semantic.governance=baseline`
- `verification.semantic.expected_markers=workflow_branch_routed,workflow_retry_budgeted,workflow_failfast_classified`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`
- `result.final_answer` 中包含 `branch=... retry=... failfast=... failfast_class=... escalation=...`

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If routing or retry output is unexpected, compare `result.final_answer` fields with `workflow_branch_routed` and `workflow_retry_budgeted` markers.
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.
