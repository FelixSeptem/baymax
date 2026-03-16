## Why

当前 Action Timeline H1.5 已提供单 run 的 phase 聚合字段，但缺少跨 run 维度的窗口趋势聚合，导致运维侧难以回答以下问题：
- 最近一段时间哪个 phase 的失败/取消在上升；
- Run 与 Stream 在窗口级别是否持续等价；
- latency p95 是否发生回归。

仓库文档已明确该能力为后续 TODO。需要在不改变现有主流程语义的前提下，补齐可配置窗口趋势聚合，并保持库接口优先与兼容增量扩展。

## What Changes

- 新增跨 run Action Timeline 趋势聚合能力（H16）：
  - 支持两种窗口模式：`last_n_runs` 与 `time_window`（均可用）。
  - 默认窗口：`N=100`，`T=15m`（可配置）。
  - 聚合维度支持：`phase` 与 `status` 双维度。
  - 指标最小集包含：`count_total`、`failed_total`、`canceled_total`、`skipped_total`、`latency_avg_ms`、`latency_p95_ms`、`window_start`、`window_end`。
- 能力默认启用；保持内存窗口实现（单进程），本期不做持久化历史聚合。
- 保持 single-writer + idempotency，避免 replay/重试导致趋势统计重复计数。
- 保持 Run/Stream 语义等价，不改 fail-fast 行为与既有 run 级聚合字段。
- 增补契约测试、race 测试与 benchmark smoke。
- 文档一次性收敛：README + roadmap + runtime-config-diagnostics + v1-acceptance + contract test index。

## Capabilities

### Modified Capabilities
- `runtime-config-and-diagnostics-api`: 增加跨 run 窗口趋势聚合配置与查询契约。
- `action-timeline-events`: 增加跨 run 趋势聚合在 Run/Stream 等价与 phase/status 维度上的语义约束。
- `diagnostics-single-writer-idempotency`: 明确跨 run 聚合在 replay/duplicate 下保持幂等统计。

### New Capabilities
- None.

## Impact

- Affected code:
  - `runtime/config/*`
  - `runtime/diagnostics/*`
  - `observability/event/*`
  - `core/runner/*`（仅在事件/聚合接入点）
  - `integration/*`（contract + benchmark smoke）
- Affected docs:
  - `README.md`
  - `docs/runtime-config-diagnostics.md`
  - `docs/development-roadmap.md`
  - `docs/v1-acceptance.md`
  - `docs/mainline-contract-test-index.md`
- Compatibility:
  - 为增量字段与增量查询能力，既有 diagnostics API 使用方可继续按旧字段读取。
  - 本期不新增 CLI，保持 library-first。
