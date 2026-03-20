# Performance Regression Policy

更新时间：2026-03-20

## Scope

本策略用于评估运行时性能回归风险，覆盖 runner/tool/mcp 与 multi-agent 主链路关键路径。

## Baseline

- 基线由主分支最近一次稳定 benchmark 结果生成。
- 结果建议存储为 JSON 或 markdown 报告，并附测试环境信息（CPU、Go 版本、提交号）。

## Acceptance Rule (Relative Percentage)

- 吞吐类指标（越高越好）采用相对下降阈值：
  - `degradation_pct = (baseline - candidate) / baseline * 100`
- 延迟类指标（越低越好）采用相对上升阈值：
  - `degradation_pct = (candidate - baseline) / baseline * 100`

默认建议阈值（可按模块再细化）：
- 吞吐下降不超过 `5%`
- P95/P99 延迟上升不超过 `8%`

超过阈值时，变更必须附带说明和缓解计划，或在评审中明确接受风险。

## Required Benchmarks

- `BenchmarkToolFanOutHighConcurrency`
- `BenchmarkToolFanOutSlowCall`
- `BenchmarkToolFanOutCancelStorm`
- `BenchmarkMultiAgentMainlineSyncInvocation`（需同时关注 `ns/op`、`p95-ns/op`、`allocs/op`）
- `BenchmarkMultiAgentMainlineAsyncReporting`（需同时关注 `ns/op`、`p95-ns/op`、`allocs/op`）
- `BenchmarkMultiAgentMainlineDelayedDispatch`（需同时关注 `ns/op`、`p95-ns/op`、`allocs/op`）
- `BenchmarkMultiAgentMainlineRecoveryReplay`（需同时关注 `ns/op`、`p95-ns/op`、`allocs/op`）
- `BenchmarkCA4PressureEvaluation`（需同时关注 `ns/op` 与 `p95-ns/op`）
- `BenchmarkCA3SemanticCompactionLatency`（需同时关注 `ns/op` 与 `p95-ns/op`）
- `BenchmarkCA3SemanticCompactionLatencyEmbeddingEnabled`（需同时关注 `ns/op` 与 `p95-ns/op`）
- 现有关键基准（iteration latency、MCP reconnect overhead）

Multi-agent 主链路回归门禁（本地/CI 一致）：

```bash
bash scripts/check-multi-agent-performance-regression.sh
```

```powershell
pwsh -File scripts/check-multi-agent-performance-regression.ps1
```

默认参数（可通过环境变量覆盖）：
- `BAYMAX_MULTI_AGENT_BENCH_BENCHTIME=200ms`
- `BAYMAX_MULTI_AGENT_BENCH_COUNT=5`

默认阈值（可通过环境变量覆盖）：
- `BAYMAX_MULTI_AGENT_BENCH_MAX_NS_DEGRADATION_PCT=8`
- `BAYMAX_MULTI_AGENT_BENCH_MAX_P95_DEGRADATION_PCT=12`
- `BAYMAX_MULTI_AGENT_BENCH_MAX_ALLOCS_DEGRADATION_PCT=10`

CA4 回归门禁（本地/CI 一致）：

```bash
bash scripts/check-ca4-benchmark-regression.sh
```

```powershell
pwsh -File scripts/check-ca4-benchmark-regression.ps1
```

默认阈值（可通过环境变量覆盖）：
- `BAYMAX_CA4_BENCH_MAX_DEGRADATION_PCT=5`
- `BAYMAX_CA4_BENCH_MAX_P95_DEGRADATION_PCT=8`

## Reporting

每次性能相关改动需提交：
- 基线与候选对比表（相对百分比）
- 指标异常解释
- 后续优化 TODO（若未达目标）

## MCP 重构重复逻辑度量（结构性指标）

对于 `mcp/http` 与 `mcp/stdio` 的共享核心重构，使用脚本输出重复逻辑占比与下降比例：

```powershell
pwsh -File scripts/report-mcp-duplication.ps1
```

验收建议（当前仓库基线）：`-MinReductionPct 5`。

```powershell
pwsh -File scripts/report-mcp-duplication.ps1 -MinReductionPct 5
```

- 生成字段：`duplicated_lines`、`duplicate_pct`、`baseline_duplicate_pct`、`reduction_pct`
- 如需更新基线：

```powershell
pwsh -File scripts/report-mcp-duplication.ps1 -WriteBaseline
```
