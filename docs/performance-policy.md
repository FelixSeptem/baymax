# Performance Regression Policy

更新时间：2026-03-11

## Scope

本策略用于评估并发与异步运行时改造的性能回归风险，覆盖 runner/tool/mcp 关键路径。

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
- 现有关键基准（iteration latency、MCP reconnect overhead）

## Reporting

每次性能相关改动需提交：
- 基线与候选对比表（相对百分比）
- 指标异常解释
- 后续优化 TODO（若未达目标）
