# mapreduce-large-batch (production-ish)

## Purpose
用真实语义链路演示 `mapreduce-large-batch` 的生产治理闭环：在 minimal 基础上增加治理 gate 与 replay 绑定。

## Variant Delta (vs minimal)
- 使用更严格的 hot-shard 阈值与更多 shard 切分，路径不只是 marker 增量。
- 在 retry 分类之后增加治理决策：`allow / allow_with_throttle / deny`。
- 追加 replay 绑定，确保治理决策可复放审计。

## Run
go run ./examples/agent-modes/mapreduce-large-batch/production-ish

## Prerequisites
- Go 1.22+ and module dependencies resolved (`go mod tidy`).
- Writable local cache for Go build artifacts (`GOCACHE`).
- No external network service is required.

## Real Runtime Path
- Semantic anchor: `mapreduce.shard_reduce_retry`.
- Classification: `mapreduce.large_batch`.
- Runtime path evidence: `core/runner,tool/local,runtime/config,orchestration/teams,runtime/diagnostics`.
- Semantic flow:
  - minimal 的 3 步数据处理链路；
  - 追加 `governance_mapreduce_gate_enforced` 与 `governance_mapreduce_replay_bound` 两步治理链路。
- Related contracts: `composed-orchestration-contract`.
- Required gates: `check-multi-agent-shared-contract.*`.
- Replay fixtures: `cross-domain-primary-reason-arbitration-contract.v1`.

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.phase=P1`
- `verification.semantic.anchor=mapreduce.shard_reduce_retry`
- `verification.semantic.classification=mapreduce.large_batch`
- `verification.semantic.runtime_path=core/runner,tool/local,runtime/config,orchestration/teams,runtime/diagnostics`
- `verification.semantic.governance=enforced`
- `verification.semantic.expected_markers=mapreduce_shards_fanned_out,mapreduce_reduce_aggregated,mapreduce_retry_classified,governance_mapreduce_gate_enforced,governance_mapreduce_replay_bound`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`
- `result.final_answer` 包含 `governance/ticket/replay` 字段，且签名必须与 minimal 不同。

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If governance result is unexpected,重点检查 `retry_class`、`hot_shards` 与 gate 决策的对应关系。
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.
