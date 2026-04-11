# mapreduce-large-batch (minimal)

## Purpose
用真实语义链路演示 `mapreduce-large-batch` 的最小闭环：shard fan-out、reduce 聚合、retry 分类。

## Run
go run ./examples/agent-modes/mapreduce-large-batch/minimal

## Prerequisites
- Go 1.22+ and module dependencies resolved (`go mod tidy`).
- Writable local cache for Go build artifacts (`GOCACHE`).
- No external network service is required.

## Real Runtime Path
- Semantic anchor: `mapreduce.shard_reduce_retry`.
- Classification: `mapreduce.large_batch`.
- Runtime path evidence: `core/runner,tool/local,runtime/config,orchestration/teams,runtime/diagnostics`.
- Semantic flow:
  - `mapreduce_shards_fanned_out`: 将批任务按分区稳定映射到 shard，并输出 hot-shard 证据。
  - `mapreduce_reduce_aggregated`: 聚合 shard 输出，保留 partial reduce 的失败统计。
  - `mapreduce_retry_classified`: 按失败 shard 与预算输出 `retry_class` / `retry_shards`。
- Related contracts: `composed-orchestration-contract`.
- Required gates: `check-multi-agent-shared-contract.*`.
- Replay fixtures: `cross-domain-primary-reason-arbitration-contract.v1`.

## Expected Output/Verification
- `verification.mainline_runtime_path=ok`
- `verification.semantic.phase=P1`
- `verification.semantic.anchor=mapreduce.shard_reduce_retry`
- `verification.semantic.classification=mapreduce.large_batch`
- `verification.semantic.runtime_path=core/runner,tool/local,runtime/config,orchestration/teams,runtime/diagnostics`
- `verification.semantic.governance=baseline`
- `verification.semantic.expected_markers=mapreduce_shards_fanned_out,mapreduce_reduce_aggregated,mapreduce_retry_classified`
- one line per marker: `verification.semantic.marker.<token>=ok`
- `result.final_answer=` and `result.signature=`
- `result.final_answer` 包含 `shards/hot/aggregate_sum/retry_class/retry_shards` 等真实状态字段。

## Failure/Rollback Notes
- If runtime path check fails, verify local registry wiring and rerun this variant.
- If reduce/retry behavior is unexpected, compare `failed_shards`、`retry_class` 与 `retry_shards` 的输出一致性。
- If semantic markers are missing, run `pwsh -File scripts/check-agent-mode-real-runtime-semantic-contract.ps1`.
- If README diverges from runtime behavior, run `pwsh -File scripts/check-agent-mode-readme-runtime-sync-contract.ps1`.
- For rollback, revert this directory (`main.go` + `README.md`) together to keep code/docs synchronized.
